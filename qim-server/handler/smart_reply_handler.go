package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SmartReplyEngine 智能回复引擎
type SmartReplyEngine struct {
	aiService      *ai.AIService
	intentDetector *ai.IntentDetector
	knowledgeSvc   *KnowledgeService
}

// NewSmartReplyEngine 创建智能回复引擎
func NewSmartReplyEngine(aiService *ai.AIService, detector *ai.IntentDetector) *SmartReplyEngine {
	return &SmartReplyEngine{
		aiService:      aiService,
		intentDetector: detector,
		knowledgeSvc:   NewKnowledgeService(aiService),
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
	db := database.GetDB()

	var conv model.Conversation
	if err := db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[SmartReply] 获取会话信息失败: %v", err)
		return
	}

	// 始终构建完整的系统提示（包含用户上下文）
	systemPrompt := e.buildSystemPrompt(conversationID, &conv, userID)

	// 如果有知识库内容，追加到 prompt 中
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

	// 使用带工具调用的 AI 完成（支持自动执行管理操作）
	callerCtx := &ai.CallerContext{
		UserID: userID,
	}
	reply, err := e.aiService.GetCompletionWithTools(messages, callerCtx)
	if err != nil {
		log.Printf("[SmartReply] AI 回复生成失败: %v", err)
		return
	}

	botReply := model.Message{
		ConversationID: conversationID,
		SenderID:       0,
		Type:           "text",
		Content:        reply,
		IsRead:         false,
	}
	db.Create(&botReply)

	// 构建 AI 发送者信息
	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
	}
	botReply.Sender = aiSender

	// 复用 broadcastNewMessage 统一处理广播和状态更新
	broadcastNewMessage(&botReply, 0, &conv)

	log.Printf("[SmartReply] 已发送智能回复到会话 %d, msgID=%d", conversationID, botReply.ID)
}

// buildSystemPrompt 构建系统提示词
func (e *SmartReplyEngine) buildSystemPrompt(conversationID uint, conv *model.Conversation, userID uint) string {
	db := database.GetDB()

	prompt := "你是 QIM 企业即时通讯系统中的智能助手。"

	// 1. 当前时间
	now := time.Now()
	prompt += fmt.Sprintf("\n当前时间：%s (%s)", now.Format("2006-01-02 15:04"), now.Weekday().String())

	// 2. 群组性质和历史
	var groupName string
	if conv.Type == "group" || conv.Type == "discussion" {
		// 获取群聊信息
		var group model.Group
		db.Where("conversation_id = ?", conversationID).First(&group)
		groupName = group.Name
		prompt += "\n\n📋 群组信息："
		prompt += fmt.Sprintf("\n- 群名：%s", group.Name)
		prompt += fmt.Sprintf("\n- 群ID：%d", conv.ID)
		prompt += fmt.Sprintf("\n- 成员数：%d", len(conv.Members))
		prompt += "\n- 群成员："
		for _, m := range conv.Members {
			name := m.User.Nickname
			if name == "" {
				name = m.User.Username
			}
			prompt += name + "、"
		}
		prompt = prompt[:len(prompt)-1] + "。"

		// 最近消息历史（最近 20 条）
		var recentMessages []model.Message
		db.Where("conversation_id = ?", conversationID).
			Preload("Sender").
			Order("created_at DESC").
			Limit(20).
			Find(&recentMessages)

		if len(recentMessages) > 0 {
			prompt += "\n\n📝 最近对话历史（按时间倒序）："
			for i := len(recentMessages) - 1; i >= 0; i-- {
				msg := recentMessages[i]
				senderName := msg.Sender.Nickname
				if senderName == "" {
					senderName = msg.Sender.Username
				}
				content := msg.Content
				if len(content) > 100 {
					content = content[:100] + "..."
				}
				prompt += fmt.Sprintf("\n[%s] %s: %s", msg.CreatedAt.Format("15:04"), senderName, content)
			}
		}
	}

	// 3. 用户角色和权限
	var user model.User
	if err := db.First(&user, userID).Error; err == nil {
		prompt += fmt.Sprintf("\n\n👤 当前提问用户：%s", user.Nickname)
		if user.Status == "disabled" {
			prompt += "（账号已禁用）"
		}
	}

	// 3.1 待办任务
	var pendingTasks []model.Task
	db.Where("user_id = ? AND status = 'todo'", userID).Order("due_date ASC").Limit(5).Find(&pendingTasks)
	if len(pendingTasks) > 0 {
		prompt += "\n\n📋 用户待办任务（未完成）："
		for _, task := range pendingTasks {
			dueStr := "无截止日期"
			if task.DueDate != nil {
				dueStr = task.DueDate.Format("2006-01-02")
			}
			prompt += fmt.Sprintf("\n- [%s] %s (截止: %s)", strings.ToUpper(task.Priority[:1]), task.Title, dueStr)
		}
		if len(pendingTasks) >= 5 {
			prompt += "\n- ... 还有更多未显示"
		}
	}

	// 4. 相关文档和知识库
	knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(groupName)
	if knowledgeCtx != "" {
		prompt += "\n\n📚 群组相关文档：\n" + knowledgeCtx
	}

	// 5. 系统状态和指标
	var totalMessages int64
	db.Model(&model.Message{}).Where("conversation_id = ?", conversationID).Count(&totalMessages)
	var onlineUsers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", conversationID).Count(&onlineUsers)

	prompt += fmt.Sprintf("\n\n📊 当前群状态：")
	prompt += fmt.Sprintf("\n- 总消息数：%d", totalMessages)
	prompt += fmt.Sprintf("\n- 成员数：%d", onlineUsers)

	prompt += `\n\n回复规则：
- 优先使用知识库中的内容回答
- 如果知识库中有相关内容，基于该内容给出准确答案
- 如果知识库中没有相关内容，使用你的通用知识回答，但明确说明"以下回答基于通用知识，建议核实"
- 回答要简洁、专业、准确
- 使用中文回复，除非用户用其他语言
- 如果是管理员提问，可以提供更详细的管理操作建议`

	return prompt
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

	db := database.GetDB()

	// 构建系统提示
	systemPrompt := e.buildSystemPrompt(conversationID, conv, userID)

	// 获取最近消息作为上下文(最近 15 条)
	var recentMessages []model.Message
	db.Where("conversation_id = ?", conversationID).
		Preload("Sender").
		Order("created_at DESC").
		Limit(15).
		Find(&recentMessages)

	// 反转消息顺序(按时间正序)
	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	// 构建消息列表
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

	// 添加用户问题
	messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("[用户提问]: %s", question)})

	// 调用 AI
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

	// 过滤输出长度
	reply = e.aiService.FilterOutput(reply, "ai_reply")

	// 创建 AI 回复消息
	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       0, // 0 表示 AI/系统
		Type:           "text",
		Content:        reply,
		IsRead:         false,
	}

	if err := db.Create(&aiMessage).Error; err != nil {
		log.Printf("[SmartReply] 保存 AI 回复失败: %v", err)
		return
	}

	// 通过 WebSocket 推送消息
	if ws.GlobalHub != nil {
		msgData := gin.H{
			"id":                aiMessage.ID,
			"conversation_id":   conversationID,
			"sender_id":         0,
			"type":              "text",
			"content":           reply,
			"is_ai_message":     true,
			"ai_assistant_name": assistantName,
			"created_at":        aiMessage.CreatedAt,
		}

		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: msgData,
		}

		jsonMsg, _ := json.Marshal(wsMsg)
		ws.GlobalHub.SendToConversation(conversationID, 0, jsonMsg)
	}

	log.Printf("[SmartReply] @AI 回复已发送,消息ID=%d", aiMessage.ID)
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
