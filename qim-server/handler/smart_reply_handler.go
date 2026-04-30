package handler

import (
	"fmt"
	"log"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"
	"strings"
	"time"
)

// SmartReplyEngine 智能回复引擎
type SmartReplyEngine struct {
	aiService      *ai.AIService
	intentDetector *ai.IntentDetector
	knowledgeSvc   *KnowledgeService
	promptBuilder  *SmartPromptBuilder
	messageSender  *WebSocketMessageSender
}

// NewSmartReplyEngine 创建智能回复引擎
func NewSmartReplyEngine(aiService *ai.AIService, detector *ai.IntentDetector) *SmartReplyEngine {
	knowledgeSvc := NewKnowledgeService(aiService)
	return &SmartReplyEngine{
		aiService:      aiService,
		intentDetector: detector,
		knowledgeSvc:   knowledgeSvc,
		promptBuilder:  NewSmartPromptBuilder(knowledgeSvc),
		messageSender:  NewWebSocketMessageSender(ws.GlobalHub),
	}
}

// HandleMessage 处理消息并决定是否需要智能回复
func (e *SmartReplyEngine) HandleMessage(userID uint, conversationID uint, content string) {
	log.Printf("[SmartReply] HandleMessage: userID=%d, convID=%d, content=%s", userID, conversationID, content[:min(30, len(content))])

	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置，跳过")
		return
	}
	log.Printf("[SmartReply] AI 服务已配置")

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		log.Printf("[SmartReply] 获取会话失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 会话类型: %s", conv.Type)

	if conv.Type == "bot" {
		log.Printf("[SmartReply] 机器人会话，由现有逻辑处理")
		return // 机器人会话由现有逻辑处理
	}

	// 检查群聊是否开启了AI助手
	var group *model.Group
	if conv.Type == "group" || conv.Type == "discussion" {
		var g model.Group
		if err := db.Where("conversation_id = ?", conversationID).First(&g).Error; err != nil {
			log.Printf("[SmartReply] 获取群聊配置失败: %v", err)
			return
		}
		group = &g
		log.Printf("[SmartReply] 群聊类型: %s, AI启用状态: %v", group.GroupType, group.AIEnabled)
		if !group.AIEnabled {
			log.Printf("[SmartReply] 群聊未开启AI助手，跳过")
			return
		}

		// 触发关键词过滤
		if group.AITriggerKeywords != "" {
			keywords := strings.Split(group.AITriggerKeywords, ",")
			hasKeyword := false
			for _, kw := range keywords {
				kw = strings.TrimSpace(kw)
				if kw != "" && strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
					hasKeyword = true
					break
				}
			}
			if !hasKeyword {
				log.Printf("[SmartReply] 消息不包含触发关键词，跳过")
				return
			}
		}

		// 防刷屏检查
		if group.AIAntiSpamInterval > 0 {
			var lastAIMsg model.Message
			err := db.Where("conversation_id = ? AND sender_id = 0 AND created_at > ?",
				conversationID, time.Now().Add(-time.Duration(group.AIAntiSpamInterval)*time.Minute)).
				Order("created_at DESC").First(&lastAIMsg).Error
			if err == nil {
				log.Printf("[SmartReply] 防刷屏间隔内，跳过回复")
				return
			}
		}
	}

	// 群聊场景下：获取 AI 助手名称，检测 @AI
	if group != nil {
		assistantName := "AI助手"
		if group.AIAssistantName != "" {
			assistantName = group.AIAssistantName
		}

		// 检测是否 @AI
		if e.isAIMention(content, assistantName) {
			question := extractAIQuestion(content, assistantName)
			e.handleAIMention(userID, conversationID, question, &conv, assistantName)
			return
		}

		// 根据回复模式决定是否自动回复
		if group.AIReplyMode == "off" {
			log.Printf("[SmartReply] AI 回复已关闭")
			return
		}
	}

	intent, err := e.intentDetector.Detect(content, userID, conversationID)
	if err != nil {
		log.Printf("[SmartReply] 意图检测失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 检测到意图: type=%s, confidence=%.2f", intent.Type, intent.Confidence)

	shouldReply := e.intentDetector.ShouldTriggerAIReply(intent, conv.Type)
	log.Printf("[SmartReply] 是否触发 AI 回复: %v", shouldReply)

	if shouldReply {
		go e.generateAndSendReply(userID, conversationID, content, intent)
	} else {
		log.Printf("[SmartReply] 不满足触发条件，不发送回复")
	}

	// 待办提取（所有群聊消息都尝试提取）
	if todoExtractor != nil && conv.Type == "group" {
		go todoExtractor.ExtractAndCreateTodos(content, userID, conversationID)
	}
}

// generateAndSendReply 生成并发送智能回复
func (e *SmartReplyEngine) generateAndSendReply(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	if e.knowledgeSvc != nil {
		knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(userContent)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	}

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userContent},
	}

	callerCtx := &ai.CallerContext{
		UserID: userID,
	}
	reply, err := e.aiService.GetCompletionWithTools(messages, callerCtx)
	if err != nil {
		log.Printf("[SmartReply] AI 回复生成失败: %v", err)
		return
	}

	err = e.messageSender.SendAIMessage(conversationID, reply, "AI助手")
	if err != nil {
		log.Printf("[SmartReply] 发送 AI 消息失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 已发送智能回复到会话 %d", conversationID)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isAIMention 检测是否 @AI 或 @AI助手
func (e *SmartReplyEngine) isAIMention(content string, assistantName string) bool {
	patterns := []string{
		"@AI",
		"@Ai",
		"@ai",
		"@" + assistantName,
	}

	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// extractAIQuestion 提取 @AI 后的问题内容
func extractAIQuestion(content string, assistantName string) string {
	patterns := []string{"@AI", "@Ai", "@ai", "@" + assistantName}

	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			question := content[idx+len(pattern):]
			return strings.TrimSpace(question)
		}
	}
	return content
}

// handleAIMention 处理 @AI 触发回复
func (e *SmartReplyEngine) handleAIMention(userID uint, conversationID uint, question string, conv *model.Conversation, assistantName string) {
	log.Printf("[SmartReply] @AI 触发回复: userID=%d, convID=%d, question=%s", userID, conversationID, question[:min(50, len(question))])

	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置")
		return
	}

	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	if e.knowledgeSvc != nil {
		knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(question)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	}

	db := database.GetDB()
	var recentMessages []model.Message
	db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(15).
		Find(&recentMessages)

	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	var messages []ai.Message
	messages = append(messages, ai.Message{Role: "system", Content: systemPrompt})

	for _, msg := range recentMessages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		messages = append(messages, ai.Message{
			Role:    "user",
			Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
		})
	}

	messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("[用户提问]: %s", question)})

	var reply string
	var err error

	if e.aiService.GetMCPServer() != nil {
		callerCtx := &ai.CallerContext{UserID: userID}
		reply, err = e.aiService.GetCompletionWithTools(messages, callerCtx)
	} else {
		reply, err = e.aiService.GetCompletion(messages)
	}

	if err != nil {
		log.Printf("[SmartReply] AI 回复生成失败: %v", err)
		return
	}

	reply = e.aiService.FilterOutput(reply, "ai_reply")

	err = e.messageSender.SendAIMessage(conversationID, reply, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 发送 AI 消息失败: %v", err)
		return
	}

	log.Printf("[SmartReply] @AI 回复已发送，消息ID已记录")
}

// GroupSummaryJob 群聊总结定时任务
type GroupSummaryJob struct {
	aiService *ai.AIService
}

// NewGroupSummaryJob 创建群聊总结任务
func NewGroupSummaryJob(aiService *ai.AIService) *GroupSummaryJob {
	return &GroupSummaryJob{
		aiService: aiService,
	}
}

// GenerateDailySummaries 生成所有群的每日总结
func (j *GroupSummaryJob) GenerateDailySummaries() {
	if j.aiService == nil || !j.aiService.IsConfigured() {
		log.Printf("[GroupSummary] AI 服务未配置，跳过总结")
		return
	}

	db := database.GetDB()

	var groups []model.Conversation
	db.Where("type = ?", "group").Find(&groups)

	log.Printf("[GroupSummary] 开始为 %d 个群生成每日总结", len(groups))

	for _, group := range groups {
		j.generateGroupSummary(&group)
		time.Sleep(2 * time.Second) // 避免 API 限流
	}
}

// generateGroupSummary 生成单个群的总结
func (j *GroupSummaryJob) generateGroupSummary(group *model.Conversation) {
	db := database.GetDB()

	// 获取群聊详细信息
	var groupInfo model.Group
	if err := db.Where("conversation_id = ?", group.ID).First(&groupInfo).Error; err != nil {
		log.Printf("[GroupSummary] 获取群聊信息失败: %v", err)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	var messages []model.Message
	db.Where("conversation_id = ? AND created_at >= ? AND created_at < ?",
		group.ID, yesterday, today).
		Preload("Sender").
		Order("created_at ASC").
		Limit(200).
		Find(&messages)

	if len(messages) < 5 {
		return // 消息太少，不需要总结
	}

	messagesText := ""
	for _, msg := range messages {
		senderName := msg.Sender.Nickname
		if senderName == "" {
			senderName = msg.Sender.Username
		}
		messagesText += senderName + ": " + msg.Content + "\n"
	}

	systemPrompt := `你是一个群聊总结助手。请分析以下群聊记录，生成简洁的每日总结。

总结格式：
📋 【群聊日报】- {日期}

📊 概览
- 今日消息数：X 条
- 活跃成员：X 人

🔥 热门话题
1. 话题一（参与人数）
2. 话题二（参与人数）

✅ 待办事项
- [ ] 待办一（负责人）
- [ ] 待办二（负责人）

💡 重要决策
- 决策一
- 决策二

请只输出总结内容，不要其他说明。`

	messages_input := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: messagesText},
	}

	summary, err := j.aiService.GetCompletion(messages_input)
	if err != nil {
		log.Printf("[GroupSummary] 群 %d 总结生成失败: %v", group.ID, err)
		return
	}

	summaryMsg := model.SystemMessage{
		Title:      "📋 群聊日报 - " + groupInfo.Name,
		Content:    summary,
		SenderID:   1,
		Status:     "active",
		TargetType: "group",
		TargetID:   &group.ID,
		CreatedAt:  time.Now(),
	}
	db.Create(&summaryMsg)

	log.Printf("[GroupSummary] 群 %d (%s) 总结已生成", group.ID, groupInfo.Name)
}
