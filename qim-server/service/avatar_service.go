package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"qim-server/ai"
	"qim-server/model"
	"qim-server/utils"

	"gorm.io/gorm"
)

// AvatarService 分身服务
type AvatarService struct {
	db         *gorm.DB
	aiService  *ai.AIService
	workerPool *AvatarWorkerPool
}

// NewAvatarService 创建分身服务实例
func NewAvatarService(db *gorm.DB, aiService *ai.AIService) *AvatarService {
	service := &AvatarService{
		db:        db,
		aiService: aiService,
	}
	service.workerPool = NewAvatarWorkerPool(5, 30, service)
	return service
}

// GetWorkerPool 获取 Worker Pool
func (s *AvatarService) GetWorkerPool() *AvatarWorkerPool {
	return s.workerPool
}

// LearnPersona 学习用户人设（异步）
func (s *AvatarService) LearnPersona(userID uint, taskID uint) {
	var task model.AvatarLearnTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return
	}

	s.db.Model(&task).Updates(map[string]interface{}{
		"status":     "processing",
		"started_at": time.Now(),
	})

	var messages []model.Message
	s.db.Table("messages m").
		Joins("JOIN conversation_members cm ON m.conversation_id = cm.conversation_id").
		Where("cm.user_id = ? AND m.sender_id = ?", userID, userID).
		Where("m.type = ?", "text").
		Where("m.created_at > ?", time.Now().AddDate(0, -3, 0)).
		Order("m.created_at DESC").
		Limit(500).
		Select("m.content").
		Find(&messages)

	s.db.Model(&task).Update("message_count", len(messages))

	if len(messages) < 10 {
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":       "failed",
			"error":        "历史消息不足，无法学习风格",
			"completed_at": time.Now(),
		})
		return
	}

	var contents []string
	for _, msg := range messages {
		if len(msg.Content) > 20 && len(msg.Content) < 500 {
			contents = append(contents, msg.Content)
		}
	}

	sampleText := strings.Join(contents[:min(50, len(contents))], "\n\n")

	prompt := fmt.Sprintf(`分析以下用户发送的消息样本，总结这个用户的说话风格特点。

消息样本：
%s

请从以下维度分析：
1. 语气特点（正式/随意/幽默/严肃等）
2. 常用表达方式和口头禅
3. 回复长度偏好（简短/详细）
4. 表情符号使用习惯
5. 专业领域或兴趣话题
6. 其他显著的说话风格特征

请用简洁的中文描述，不超过200字。`, sampleText)

	// 使用现有的 AI 服务调用模式
	aiMessages := []ai.Message{
		{Role: "user", Content: prompt},
	}
	persona, err := s.aiService.GetCompletion(aiMessages)
	if err != nil {
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":       "failed",
			"error":        err.Error(),
			"completed_at": time.Now(),
		})
		return
	}

	now := time.Now()
	s.db.Model(&model.AvatarConfig{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"auto_learned_persona": persona,
		"persona_version":      gorm.Expr("persona_version + 1"),
		"last_learned_at":      now,
	})

	s.db.Model(&task).Updates(map[string]interface{}{
		"status":       "completed",
		"progress":     100,
		"completed_at": now,
	})
}

// GenerateReply 生成分身回复
func (s *AvatarService) GenerateReply(userID uint, conversationID uint, triggerMessage string) (string, error) {
	var config model.AvatarConfig
	if err := s.db.Where("user_id = ?", userID).First(&config).Error; err != nil {
		return "", fmt.Errorf("分身配置不存在")
	}

	var knowledgeScope model.AvatarKnowledgeScope
	if config.KnowledgeScopeJSON != "" {
		json.Unmarshal([]byte(config.KnowledgeScopeJSON), &knowledgeScope)
	}

	var replyStrategy model.AvatarReplyStrategy
	if config.ReplyStrategyJSON != "" {
		json.Unmarshal([]byte(config.ReplyStrategyJSON), &replyStrategy)
	}

	var user model.User
	s.db.First(&user, userID)

	systemPrompt := s.buildSystemPrompt(config, user, replyStrategy)

	contextParts := []string{}
	if knowledgeScope.ConversationHistory && conversationID > 0 {
		history := s.getConversationHistory(conversationID, 10)
		if history != "" {
			contextParts = append(contextParts, "【对话历史】\n"+history)
		}
	}

	contextStr := strings.Join(contextParts, "\n\n")

	prompt := ""
	if contextStr != "" {
		prompt = contextStr + "\n\n"
	}
	prompt += fmt.Sprintf("对方说：%s\n\n请以我的身份回复：", triggerMessage)

	var reply string
	var err error

	if config.UseSystemConfig {
		// 使用系统默认 AI 配置
		aiMessages := []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		}
		reply, err = s.aiService.GetCompletion(aiMessages)
	} else if config.ModelConfigID != nil {
		// 使用用户自定义模型配置
		reply, err = s.generateWithUserProvider(*config.ModelConfigID, systemPrompt, prompt)
	} else {
		return "", fmt.Errorf("未配置 AI 模型")
	}

	if err != nil {
		return "", err
	}

	// 根据回复策略截断
	if replyStrategy.MaxReplyLength == "short" && len(reply) > 200 {
		reply = reply[:200] + "..."
	} else if replyStrategy.MaxReplyLength == "medium" && len(reply) > 500 {
		reply = reply[:500] + "..."
	}

	return reply, nil
}

// buildSystemPrompt 构建系统提示词
func (s *AvatarService) buildSystemPrompt(config model.AvatarConfig, user model.User, strategy model.AvatarReplyStrategy) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("你是%s的AI分身，需要以TA的身份回复消息。\n\n", user.Nickname))

	if config.AutoLearnedPersona != "" {
		sb.WriteString("【你的说话风格】\n")
		sb.WriteString(config.AutoLearnedPersona)
		sb.WriteString("\n\n")
	}

	if config.CustomPersonaAddon != "" {
		sb.WriteString("【补充说明】\n")
		sb.WriteString(config.CustomPersonaAddon)
		sb.WriteString("\n\n")
	}

	sb.WriteString("【回复要求】\n")
	sb.WriteString("- 以第一人称回复，就像你就是这个人\n")
	sb.WriteString("- 保持自然的对话风格\n")
	sb.WriteString("- 不要暴露你是AI\n")
	sb.WriteString("- 回复要简洁，不要过长\n")

	return sb.String()
}

// getConversationHistory 获取对话历史
func (s *AvatarService) getConversationHistory(conversationID uint, limit int) string {
	var messages []model.Message
	s.db.Where("conversation_id = ?", conversationID).
		Where("type = ?", "text").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages)

	if len(messages) == 0 {
		return ""
	}

	var parts []string
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		var sender model.User
		s.db.First(&sender, msg.SenderID)
		parts = append(parts, fmt.Sprintf("%s: %s", sender.Nickname, msg.Content))
	}

	return strings.Join(parts, "\n")
}

// generateWithUserProvider 使用用户自定义模型配置生成回复
func (s *AvatarService) generateWithUserProvider(configID uint, systemPrompt string, prompt string) (string, error) {
	var config model.UserAIConfig
	if err := s.db.First(&config, configID).Error; err != nil {
		return "", err
	}

	apiKey, err := utils.DecryptAPIKey(config.APIKeyEncrypted)
	if err != nil {
		return "", err
	}

	// 创建 Provider
	provider := s.createProvider(config.Provider, apiKey, config.ModelName, config.BaseURL, config.MaxTokens)

	// 构建消息
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: prompt},
	}

	return provider.Chat(messages)
}

// createProvider 根据配置创建 Provider
func (s *AvatarService) createProvider(providerName, apiKey, model, baseURL string, maxTokens int) ai.Provider {
	config := ai.ProviderConfig{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		ExtraParams: map[string]interface{}{
			"max_tokens": maxTokens,
		},
	}

	switch providerName {
	case "openai":
		return ai.NewOpenAIProvider(config)
	case "baidu":
		return ai.NewBaiduProvider(config)
	case "alibaba":
		return ai.NewAlibabaProvider(config)
	case "tencent":
		return ai.NewTencentProvider(config)
	case "bytedance":
		return ai.NewBytedanceProvider(config)
	case "anthropic":
		return ai.NewAnthropicProvider(config)
	default:
		return ai.NewOpenAIProvider(config)
	}
}

// PreviewReply 预览回复
func (s *AvatarService) PreviewReply(userID uint, message string) (string, error) {
	return s.GenerateReply(userID, 0, message)
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
