package service

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// AvatarService 分身服务
type AvatarService struct {
	db            *gorm.DB
	aiService     *ai.AIService
	workerPool    *AvatarWorkerPool
	noteVectorSvc *NoteVectorService               // 笔记向量检索（RAG）
	memorySvc     *AvatarMemoryService             // 长期记忆
	groupDocSvc   *GroupDocumentService            // 群文档知识检索
	replyGraph    atomic.Pointer[AvatarReplyGraph] // Eino Graph 编排，原子读写避免重建与调用竞争
	thresholdSvc  *AiThresholdService              // AI 阈值（记忆召回门槛等），nil 时 graph 用默认值
	wsNotify      func(userID uint, eventType string, data map[string]interface{})
}

// SetThresholdService 注入 AI 阈值服务并重建 graph（阈值在记忆召回门槛处生效）。
func (s *AvatarService) SetThresholdService(t *AiThresholdService) {
	s.thresholdSvc = t
	s.rebuildReplyGraph("SetThresholdService")
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
	graph := NewAvatarReplyGraph(aiService, db, nil, nil, nil)
	if err := graph.BuildGraph(); err != nil {
		logger.WithModule("AvatarService").Error("BuildGraph 失败", "error", err)
	}
	service.replyGraph.Store(graph)
	return service
}

// SetRAGServices 设置 RAG 相关服务（可选）
func (s *AvatarService) SetRAGServices(noteVectorSvc *NoteVectorService, memorySvc *AvatarMemoryService) {
	s.noteVectorSvc = noteVectorSvc
	s.memorySvc = memorySvc
	s.rebuildReplyGraph("SetRAGServices")
}

func (s *AvatarService) SetGroupDocumentService(groupDocSvc *GroupDocumentService) {
	s.groupDocSvc = groupDocSvc
	s.rebuildReplyGraph("SetGroupDocumentService")
}

func (s *AvatarService) SetAIService(aiService *ai.AIService) {
	s.aiService = aiService
	s.rebuildReplyGraph("SetAIService")
}

func (s *AvatarService) rebuildReplyGraph(source string) {
	graph := NewAvatarReplyGraph(s.aiService, s.db, s.noteVectorSvc, s.memorySvc, s.groupDocSvc)
	graph.SetThresholdService(s.thresholdSvc)
	if err := graph.BuildGraph(); err != nil {
		logger.WithModule("AvatarService").Error("BuildGraph 失败", "source", source, "error", err)
		return
	}
	// 先编译成功再原子替换，调用方始终拿到完整可用的 graph
	s.replyGraph.Store(graph)
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
2. 回复长度偏好（简短/详细）
3. 表情符号使用习惯
4. 专业领域或兴趣话题
5. 其他显著的说话风格特征

注意：
- 只分析用户的回复风格，忽略对AI的请求（如"帮我写"、"查一下"等指令）
- 不要把用户的请求表达当作说话风格
- 重点关注用户在对话中的自然表达方式

请用简洁的中文描述，不超过200字。`, sampleText)

	// 5. AI 分析
	aiMessages := []ai.Message{
		{Role: "user", Content: prompt},
	}
	persona, err := s.aiService.GetCompletion(ai.TaskTypeAnalysis, aiMessages)
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

// GenerateReply 生成分身回复（使用 Eino Graph 编排）。
// config 非 nil 时复用调用方已加载的配置，避免一次回复流程内重复查 avatar_configs。
func (s *AvatarService) GenerateReply(userID uint, conversationID uint, triggerMessage string, config *model.AvatarConfig) (string, error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return "", fmt.Errorf("回复 Graph 未初始化")
	}

	ctx := context.Background()
	return graph.Execute(ctx, userID, conversationID, triggerMessage, config)
}

// GenerateReplyWithSources 与 GenerateReply 等价，额外返回本条回复命中的知识来源
// （笔记/群知识/记忆标题与摘要），供 worker 随 WS 下发供前端展示「依据」。
func (s *AvatarService) GenerateReplyWithSources(userID uint, conversationID uint, triggerMessage string, config *model.AvatarConfig) (string, []KnowledgeSource, error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return "", nil, fmt.Errorf("回复 Graph 未初始化")
	}

	ctx := context.Background()
	return graph.ExecuteWithSources(ctx, userID, conversationID, triggerMessage, config)
}

// GenerateReplyWithImageSources 供分身识别图片触发消息：按 fileID 读图（base64 data URL）后走
// 多模态生成路径（ExecuteWithImageSources）。返回回复 + 命中的知识来源。
// 读图失败（存储不可用/图片过大/读取错误）时返回错误，由调用方按"尽力而为失败则跳过"跳过本次回复；
// 模型不支持视觉导致生成报错同样以 error 返回，调用方跳过。
func (s *AvatarService) GenerateReplyWithImageSources(userID uint, conversationID uint, triggerMessage string, imageName string, fileID uint, config *model.AvatarConfig) (string, []KnowledgeSource, error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return "", nil, fmt.Errorf("回复 Graph 未初始化")
	}
	if s.groupDocSvc == nil {
		return "", nil, fmt.Errorf("分身服务未接入群文档服务，无法读取图片")
	}
	name, dataURL, err := s.groupDocSvc.ImageURLForContext(fileID)
	if err != nil {
		return "", nil, fmt.Errorf("读取分身图片失败: %w", err)
	}
	if dataURL == "" {
		return "", nil, fmt.Errorf("分身图片内容为空")
	}
	if imageName == "" {
		imageName = name
	}
	ctx := context.Background()
	return graph.ExecuteWithImageSources(ctx, userID, conversationID, triggerMessage, dataURL, imageName, config)
}

// GenerateReplyBatchWithImageSources 供分身对「合并窗口内连发的一批消息」生成一条合并回复。
// 批内可含纯文本（照常拼入 prompt）与图片消息（逐张读图转 base64 data URL 注入多模态）。
// 任一图片读图失败（存储不可用/图片过大/读取错误）即返回错误，由调用方按"尽力而为失败
// 则跳过"跳过整批，避免部分看图的不一致回复。
func (s *AvatarService) GenerateReplyBatchWithImageSources(userID uint, conversationID uint, batch []AvatarBatchItem, config *model.AvatarConfig) (string, []KnowledgeSource, error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return "", nil, fmt.Errorf("回复 Graph 未初始化")
	}

	var orderTexts []string
	var imageURLs []string
	var imageNames []string
	for _, item := range batch {
		if item.MsgType == "image" {
			if s.groupDocSvc == nil {
				return "", nil, fmt.Errorf("分身服务未接入群文档服务，无法读取图片")
			}
			if item.FileID == 0 {
				return "", nil, fmt.Errorf("分身批量图片消息缺少文件 id")
			}
			name, dataURL, err := s.groupDocSvc.ImageURLForContext(item.FileID)
			if err != nil {
				return "", nil, fmt.Errorf("读取分身批量图片失败: %w", err)
			}
			if dataURL == "" {
				return "", nil, fmt.Errorf("分身批量图片内容为空")
			}
			if item.Name == "" {
				item.Name = name
			}
			imageURLs = append(imageURLs, dataURL)
			imageNames = append(imageNames, item.Name)
		} else {
			orderTexts = append(orderTexts, item.Msg)
		}
	}

	// 批内全是图片（无文本）时，给模型一个占位文本提示；否则照常拼批内文本
	if len(orderTexts) == 0 {
		orderTexts = []string{"对方发来了一张/多张图片，请识别其内容并结合对话回复。"}
	}

	ctx := context.Background()
	return graph.ExecuteBatchWithImagesSources(ctx, userID, conversationID, orderTexts, imageURLs, imageNames, config)
}

// PreviewReply 预览回复
func (s *AvatarService) PreviewReply(userID uint, message string) (string, error) {
	return s.GenerateReply(userID, 0, message, nil)
}

// GenerateReplyStream 流式生成分身回复（供"帮我回复"草稿模式：复用分身全套上下文，
// 但把流式输出交给调用方，不落库不发送）。config 非 nil 时复用调用方已加载的配置。
// historyBefore 为对话历史锚点（目标消息的 CreatedAt）：非 nil 时只取该时间之前的消息作
// 上下文，避免目标不是会话最新一条时把后续对话混进历史导致答非所问。
// ctx 透传自 HTTP 请求，客户端断开时取消上游 AI 请求，避免空跑浪费 token。
func (s *AvatarService) GenerateReplyStream(ctx context.Context, userID uint, conversationID uint, triggerMessage string, historyBefore *time.Time, config *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return nil, fmt.Errorf("回复 Graph 未初始化")
	}

	return graph.ExecuteStream(ctx, userID, conversationID, triggerMessage, historyBefore, config)
}

// GenerateReplyStreamWithImageSources 流式生成分身「帮我回复」草稿：目标消息是图片时按 fileID
// 读图（base64 data URL）走流式多模态生成，草稿基于图片内容。读图失败（存储不可用/图片过大/
// 读取错误）或未接入群文档服务时降级为纯文本草稿——换诚实文案说明看不到图片内容，不让草稿
// 功能对图片消息失效、也不把 {"url":...} JSON 泄漏给模型。
func (s *AvatarService) GenerateReplyStreamWithImageSources(ctx context.Context, userID uint, conversationID uint, imageName string, fileID uint, historyBefore *time.Time, config *model.AvatarConfig) (*schema.StreamReader[*schema.Message], error) {
	graph := s.replyGraph.Load()
	if graph == nil {
		return nil, fmt.Errorf("回复 Graph 未初始化")
	}

	if s.groupDocSvc != nil {
		if name, dataURL, err := s.groupDocSvc.ImageURLForContext(fileID); err == nil && dataURL != "" {
			if imageName == "" {
				imageName = name
			}
			return graph.ExecuteStreamWithImageSources(ctx, userID, conversationID,
				"对方发来了一张图片，请起草一条回复。", dataURL, imageName, historyBefore, config)
		}
	}

	fallbackMsg := "对方发来了一张图片，但无法读取图片内容，请根据对话上下文起草一条回复。"
	if imageName != "" {
		fallbackMsg = fmt.Sprintf("对方发来了一张图片「%s」，但无法读取图片内容，请根据对话上下文起草一条回复。", imageName)
	}
	return graph.ExecuteStream(ctx, userID, conversationID, fallbackMsg, historyBefore, config)
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
	persona, err := s.aiService.GetCompletion(ai.TaskTypeAnalysis, aiMessages)
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
		// 回读自增后的版本号，避免把人设文本误塞进 persona_version 字段
		var updated model.AvatarConfig
		personaVersion := 0
		if readErr := s.db.Select("persona_version").Where("user_id = ?", userID).First(&updated).Error; readErr == nil {
			personaVersion = updated.PersonaVersion
		}
		s.wsNotify(userID, "avatar_learning_completed", map[string]interface{}{
			"persona_version": personaVersion,
			"learned_at":      now,
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
