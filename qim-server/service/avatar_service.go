package service

import (
	"context"
	"fmt"
	"log"
	"qim-server/ai"
	"qim-server/model"
	"qim-server/utils"
	"strings"
	"time"

	"gorm.io/gorm"
)

// AvatarService 分身服务
type AvatarService struct {
	db            *gorm.DB
	aiService     *ai.AIService
	workerPool    *AvatarWorkerPool
	noteVectorSvc *NoteVectorService    // 笔记向量检索（RAG）
	memorySvc     *AvatarMemoryService  // 长期记忆
	triggerSvc    *AvatarTriggerService // 智能触发
	groupDocSvc   *GroupDocumentService // 群文档知识检索
	replyGraph    *AvatarReplyGraph     // Eino Graph 编排
	wsNotify      func(userID uint, eventType string, data map[string]interface{})
}

// SetWebSocketNotify 设置 WebSocket 通知回调
func (s *AvatarService) SetWebSocketNotify(fn func(userID uint, eventType string, data map[string]interface{})) {
	s.wsNotify = fn
}

// LearningData 多来源学习数据结构
type LearningData struct {
	Messages      []string
	BotConfigs    []string
	AIActions     []string
	MessageWeight float64
	BotWeight     float64
	ActionWeight  float64
}

// NewAvatarService 创建分身服务实例
func NewAvatarService(db *gorm.DB, aiService *ai.AIService) *AvatarService {
	service := &AvatarService{
		db:        db,
		aiService: aiService,
	}
	service.workerPool = NewAvatarWorkerPool(5, 30, service)
	service.replyGraph = NewAvatarReplyGraph(aiService, db, nil, nil, nil)
	if err := service.replyGraph.BuildGraph(); err != nil {
		log.Printf("[AvatarService] BuildGraph 失败: %v", err)
	}
	return service
}

// SetRAGServices 设置 RAG 相关服务（可选）
func (s *AvatarService) SetRAGServices(noteVectorSvc *NoteVectorService, memorySvc *AvatarMemoryService, triggerSvc *AvatarTriggerService) {
	s.noteVectorSvc = noteVectorSvc
	s.memorySvc = memorySvc
	s.triggerSvc = triggerSvc

	s.replyGraph = NewAvatarReplyGraph(s.aiService, s.db, noteVectorSvc, memorySvc, s.groupDocSvc)
	if err := s.replyGraph.BuildGraph(); err != nil {
		log.Printf("[AvatarService] BuildGraph 失败 (SetRAGServices): %v", err)
	}
}

func (s *AvatarService) SetGroupDocumentService(groupDocSvc *GroupDocumentService) {
	s.groupDocSvc = groupDocSvc

	s.replyGraph = NewAvatarReplyGraph(s.aiService, s.db, s.noteVectorSvc, s.memorySvc, groupDocSvc)
	if err := s.replyGraph.BuildGraph(); err != nil {
		log.Printf("[AvatarService] BuildGraph 失败 (SetGroupDocumentService): %v", err)
	}
}

func (s *AvatarService) SetAIService(aiService *ai.AIService) {
	s.aiService = aiService
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

	// 1. 开始处理
	s.db.Model(&task).Updates(map[string]interface{}{
		"status":     "processing",
		"started_at": time.Now(),
		"progress":   10,
	})

	// 2. 查询历史消息
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

	s.db.Model(&task).Updates(map[string]interface{}{
		"message_count": len(messages),
		"progress":      30,
	})

	if len(messages) < 10 {
		s.db.Model(&task).Updates(map[string]interface{}{
			"status":       "failed",
			"error":        "历史消息不足，无法学习风格",
			"completed_at": time.Now(),
		})
		return
	}

	// 3. 处理消息内容
	var contents []string
	for _, msg := range messages {
		if len(msg.Content) > 20 && len(msg.Content) < 500 {
			contents = append(contents, msg.Content)
		}
	}
	s.db.Model(&task).Update("progress", 50)

	// 4. 准备 AI 调用
	sampleText := strings.Join(contents[:min(50, len(contents))], "\n\n")
	s.db.Model(&task).Update("progress", 70)

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

	// 5. AI 分析
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

	s.db.Model(&task).Update("progress", 90)

	// 6. 保存结果
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

// GenerateReply 生成分身回复（使用 Eino Graph 编排）
func (s *AvatarService) GenerateReply(userID uint, conversationID uint, triggerMessage string) (string, error) {
	if s.replyGraph == nil {
		return "", fmt.Errorf("回复 Graph 未初始化")
	}

	ctx := context.Background()
	return s.replyGraph.Execute(ctx, userID, conversationID, triggerMessage)
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
	var config model.AIConfig
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

// LearnFromMultipleSources 从多个来源学习用户风格
func (s *AvatarService) LearnFromMultipleSources(userID uint) error {
	db := s.db

	messages := make([]model.Message, 0)
	db.Where("sender_id = ?", userID).Order("created_at DESC").Limit(500).Find(&messages)

	var botConfigs []model.Bot
	db.Where("creator_id = ?", userID).Find(&botConfigs)

	var aiActions []model.AIUsageLog
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(100).Find(&aiActions)

	learningData := LearningData{
		Messages:      processMessages(messages),
		BotConfigs:    processBotConfigs(botConfigs),
		AIActions:     processAIActions(aiActions),
		MessageWeight: 0.6,
		BotWeight:     0.2,
		ActionWeight:  0.2,
	}

	return s.UpdatePersona(userID, learningData)
}

// UpdatePersona 根据学习数据更新人设
func (s *AvatarService) UpdatePersona(userID uint, data LearningData) error {
	sampleText := buildLearningPrompt(data)

	prompt := fmt.Sprintf(`分析以下从多个来源收集的用户数据，总结这个用户的说话风格和特征。

%s

请从以下维度分析：
1. 语气特点（正式/随意/幽默/严肃等）
2. 常用表达方式和口头禅
3. 回复长度偏好（简短/详细）
4. 表情符号使用习惯
5. 专业领域或兴趣话题
6. 其他显著的说话风格特征

请用简洁的中文描述，不超过200字。`, sampleText)

	aiMessages := []ai.Message{
		{Role: "user", Content: prompt},
	}
	persona, err := s.aiService.GetCompletion(aiMessages)
	if err != nil {
		return err
	}

	now := time.Now()
	err = s.db.Model(&model.AvatarConfig{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"auto_learned_persona": persona,
		"persona_version":      gorm.Expr("persona_version + 1"),
		"last_learned_at":      now,
	}).Error

	if err == nil && s.wsNotify != nil {
		s.wsNotify(userID, "avatar_learning_completed", map[string]interface{}{
			"persona_version": persona,
		})
	}

	return err
}

// buildLearningPrompt 构建学习提示词
func buildLearningPrompt(data LearningData) string {
	var sb strings.Builder

	if len(data.Messages) > 0 {
		sb.WriteString("【聊天消息样本】（权重 60%%）\n")
		for i, msg := range data.Messages {
			if i >= 30 {
				break
			}
			sb.WriteString("- " + msg + "\n")
		}
		sb.WriteString("\n")
	}

	if len(data.BotConfigs) > 0 {
		sb.WriteString("【机器人配置】（权重 20%%）\n")
		for _, config := range data.BotConfigs {
			sb.WriteString("- " + config + "\n")
		}
		sb.WriteString("\n")
	}

	if len(data.AIActions) > 0 {
		sb.WriteString("【AI使用行为】（权重 20%%）\n")
		for _, action := range data.AIActions {
			sb.WriteString("- " + action + "\n")
		}
	}

	return sb.String()
}

// processMessages 处理消息数据
func processMessages(messages []model.Message) []string {
	var result []string
	for _, msg := range messages {
		if len(msg.Content) > 10 && len(msg.Content) < 500 {
			result = append(result, msg.Content)
		}
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

// processBotConfigs 处理机器人配置数据
func processBotConfigs(bots []model.Bot) []string {
	var result []string
	for _, bot := range bots {
		desc := bot.Description
		if desc == "" {
			desc = bot.Name
		}
		result = append(result, fmt.Sprintf("机器人[%s]: %s", bot.Name, desc))
	}
	return result
}

// processAIActions 处理AI使用行为数据
func processAIActions(actions []model.AIUsageLog) []string {
	var result []string
	for _, action := range actions {
		result = append(result, action.MessagePreview)
	}
	return result
}
