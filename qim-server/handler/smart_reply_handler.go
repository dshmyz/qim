package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/service"
	"qim-server/ws"
	"strings"
	"sync"
	"time"
)

// SmartReplyEngine 智能回复引擎
type SmartReplyEngine struct {
	aiService        *ai.AIService
	intentDetector   *ai.IntentDetector
	knowledgeSvc     *KnowledgeService
	unifiedKnowledge *service.UnifiedKnowledgeService
	memorySvc        *service.AvatarMemoryService
	promptBuilder    *SmartPromptBuilder
	messageSender    *WebSocketMessageSender
	avatarWorkerPool *service.AvatarWorkerPool
	smartReplyGraph  *service.SmartReplyGraph
}

// NewSmartReplyEngine 创建智能回复引擎
func NewSmartReplyEngine(aiService *ai.AIService, detector *ai.IntentDetector) *SmartReplyEngine {
	knowledgeSvc := NewKnowledgeService(aiService)
	return &SmartReplyEngine{
		aiService:      aiService,
		intentDetector: detector,
		knowledgeSvc:   knowledgeSvc,
		promptBuilder:  NewSmartPromptBuilder(knowledgeSvc),
		messageSender:  NewWebSocketMessageSender(ws.GlobalHub, di.GlobalContainer.UserService),
	}
}

// SetUnifiedKnowledge 设置统一知识检索服务（向量库+MySQL兜底）
func (e *SmartReplyEngine) SetUnifiedKnowledge(uk *service.UnifiedKnowledgeService) {
	e.unifiedKnowledge = uk
}

// SetAvatarWorkerPool 设置分身工作池
func (e *SmartReplyEngine) SetAvatarWorkerPool(pool *service.AvatarWorkerPool) {
	e.avatarWorkerPool = pool
}

// SetMemoryService sets the avatar memory service for the smart reply engine
func (e *SmartReplyEngine) SetMemoryService(ms *service.AvatarMemoryService) {
	e.memorySvc = ms
}

func (e *SmartReplyEngine) InitSmartReplyGraph() error {
	log.Printf("[SmartReplyGraph] 创建 SmartReplyGraph 实例...")
	e.smartReplyGraph = service.NewSmartReplyGraph(
		e.aiService,
		database.GetDB(),
		e.unifiedKnowledge,
		e.knowledgeSvc,
		e.memorySvc,
		di.GlobalContainer.UserService,
	)

	log.Printf("[SmartReplyGraph] 开始编译 Graph...")
	err := e.smartReplyGraph.BuildGraph()
	if err != nil {
		log.Printf("[SmartReplyGraph] BuildGraph 失败: %v", err)
	} else {
		log.Printf("[SmartReplyGraph] BuildGraph 成功")
	}
	return err
}

// HandleMessage 处理消息并决定是否需要智能回复
func (e *SmartReplyEngine) HandleMessage(userID uint, conversationID uint, content string, mentionUserIDs []uint) {
	if e.aiService == nil || !e.aiService.IsConfigured() {
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return
	}

	if e.avatarWorkerPool != nil {
		e.checkAvatarTriggers(userID, &conv, content, mentionUserIDs)
	}

	if conv.Type == "bot" {
		return
	}

	var group *model.Group
	if conv.Type == "group" || conv.Type == "discussion" {
		var g model.Group
		if err := db.Where("conversation_id = ?", conversationID).First(&g).Error; err != nil {
			return
		}
		group = &g
		aiConfig := group.GetAIConfig()
		if !aiConfig.Enabled {
			return
		}

		if aiConfig.TriggerKeywords != "" {
			keywords := strings.Split(aiConfig.TriggerKeywords, ",")
			hasKeyword := false
			for _, kw := range keywords {
				kw = strings.TrimSpace(kw)
				if kw != "" && strings.Contains(strings.ToLower(content), strings.ToLower(kw)) {
					hasKeyword = true
					break
				}
			}
			if !hasKeyword {
				return
			}
		}

		if aiConfig.AntiSpamInterval > 0 {
			userSvc := service.NewUserService(db)
			systemUserID := userSvc.GetSystemUserID()
			var lastAIMsg model.Message
			err := db.Where("conversation_id = ? AND sender_id = ? AND created_at > ?",
				conversationID, systemUserID, time.Now().Add(-time.Duration(aiConfig.AntiSpamInterval)*time.Minute)).
				Order("created_at DESC").First(&lastAIMsg).Error
			if err == nil {
				return
			}
		}
	}

	if group != nil {
		aiConfig := group.GetAIConfig()
		assistantName := "AI助手"
		if aiConfig.AssistantName != "" {
			assistantName = aiConfig.AssistantName
		}

		if e.isAIMention(content, assistantName) {
			question := extractAIQuestion(content, assistantName)
			e.handleAIMention(userID, conversationID, question, content, &conv, assistantName)
			return
		}

		if aiConfig.ReplyMode == "off" {
			return
		}

		if aiConfig.ReplyMode == "mention_only" {
			return
		}
	}

	intent, err := e.intentDetector.Detect(content, userID, conversationID)
	if err != nil {
		return
	}

	shouldReply := e.intentDetector.ShouldTriggerAIReply(intent, conv.Type)

	if shouldReply {
		go e.generateAndSendReply(userID, conversationID, content, intent)
	}

	// 待办提取（所有群聊消息都尝试提取）
	if todoExtractor != nil && conv.Type == "group" {
		go todoExtractor.ExtractAndCreateTodos(content, userID, conversationID)
	}
}

// generateAndSendReply 生成并发送智能回复
func (e *SmartReplyEngine) generateAndSendReply(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	if e.smartReplyGraph != nil {
		e.generateAndSendReplyWithGraph(userID, conversationID, userContent, intent)
		return
	}
	e.generateAndSendReplyLegacy(userID, conversationID, userContent, intent)
}

func (e *SmartReplyEngine) generateAndSendReplyWithGraph(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	ctx := context.Background()
	input := &service.SmartReplyContext{
		Message:         userContent,
		OriginalContent: userContent,
		UserID:          userID,
		ConversationID:  conversationID,
		Intent:          intent,
		IsAIMention:     false,
	}

	result, err := e.smartReplyGraph.Execute(ctx, input)
	if err != nil {
		log.Printf("[SmartReplyGraph] AI 回复生成失败: %v", err)
		return
	}

	log.Printf("[SmartReplyGraph] 生成回复长度: %d 字符", len(result.Reply))

	err = e.messageSender.SendAIMessage(conversationID, result.Reply, "AI助手")
	if err != nil {
		log.Printf("[SmartReply] 发送 AI 消息失败: %v", err)
		return
	}

	log.Printf("[SmartReplyGraph] 已发送智能回复到会话 %d", conversationID)
}

func (e *SmartReplyEngine) generateAndSendReplyLegacy(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	if e.unifiedKnowledge != nil && ctx.Group != nil {
		knowledgeCtx := e.unifiedKnowledge.BuildContext(userContent, ctx.Group.ID)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	} else if e.knowledgeSvc != nil {
		knowledgeCtx := e.knowledgeSvc.BuildKnowledgeContext(userContent)
		if knowledgeCtx != "" {
			systemPrompt += "\n\n" + knowledgeCtx
		}
	}

	if e.memorySvc != nil {
		memoryResults, err := e.memorySvc.Recall(userID, userContent, 2)
		if err == nil && len(memoryResults) > 0 {
			var parts []string
			for _, r := range memoryResults {
				parts = append(parts, r.Content)
			}
			memoryCtx := "💡 用户历史记忆：\n" + strings.Join(parts, "\n")
			systemPrompt += "\n\n" + memoryCtx
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
func (e *SmartReplyEngine) handleAIMention(userID uint, conversationID uint, question string, originalContent string, conv *model.Conversation, assistantName string) {
	log.Printf("[SmartReply] @AI 触发回复: userID=%d, convID=%d, question=%s", userID, conversationID, question[:min(50, len(question))])

	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置")
		return
	}

	if e.smartReplyGraph != nil {
		e.handleAIMentionWithGraph(userID, conversationID, question, originalContent, assistantName)
		return
	}

	e.handleAIMentionLegacy(userID, conversationID, question, originalContent, conv, assistantName)
}

func (e *SmartReplyEngine) handleAIMentionWithGraph(userID uint, conversationID uint, question string, originalContent string, assistantName string) {
	ctx := context.Background()
	input := &service.SmartReplyContext{
		Message:         question,
		OriginalContent: originalContent,
		UserID:          userID,
		ConversationID:  conversationID,
		IsAIMention:     true,
		AssistantName:   assistantName,
	}

	stream, err := e.smartReplyGraph.ExecuteStream(ctx, input)
	if err != nil {
		log.Printf("[SmartReplyGraph] @AI 流式回复失败: %v", err)
		return
	}

	sendChunk, finish, err := e.messageSender.SendStreamingAIMessage(conversationID, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 创建流式消息失败: %v", err)
		return
	}

	chunkCount := 0
	totalLen := 0
	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}
		chunkCount++
		totalLen += len(msg.Content)
		sendChunk(msg.Content)
	}

	log.Printf("[SmartReplyGraph] @AI 流式回复完成: %d 个 chunk, 总长度 %d 字符", chunkCount, totalLen)

	_ = finish()

	log.Printf("[SmartReplyGraph] @AI 流式回复已完成")
}

func (e *SmartReplyEngine) handleAIMentionLegacy(userID uint, conversationID uint, question string, originalContent string, conv *model.Conversation, assistantName string) {
	ctx := e.promptBuilder.BuildPromptContext(conversationID, userID)
	if ctx == nil {
		log.Printf("[SmartReply] 构建提示词上下文失败")
		return
	}

	systemPrompt := e.promptBuilder.BuildSystemPrompt(ctx)

	if e.unifiedKnowledge != nil && conv.Type == "group" {
		var g model.Group
		if err := database.GetDB().Where("conversation_id = ?", conversationID).First(&g).Error; err == nil {
			knowledgeCtx := e.unifiedKnowledge.BuildContext(question, g.ID)
			if knowledgeCtx != "" {
				systemPrompt += "\n\n" + knowledgeCtx
			}
		}
	} else if e.knowledgeSvc != nil {
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
		Limit(20).
		Find(&recentMessages)

	for i, j := 0, len(recentMessages)-1; i < j; i, j = i+1, j-1 {
		recentMessages[i], recentMessages[j] = recentMessages[j], recentMessages[i]
	}

	systemUserID := service.NewUserService(db).GetSystemUserID()

	var messages []ai.Message
	messages = append(messages, ai.Message{Role: "system", Content: systemPrompt})

	for _, msg := range recentMessages {
		if originalContent != "" && msg.SenderID == userID && msg.Content == originalContent {
			continue
		}

		if msg.SenderID == systemUserID {
			messages = append(messages, ai.Message{
				Role:    "assistant",
				Content: msg.Content,
			})
		} else {
			senderName := msg.Sender.Nickname
			if senderName == "" {
				senderName = msg.Sender.Username
			}
			messages = append(messages, ai.Message{
				Role:    "user",
				Content: fmt.Sprintf("[%s]: %s", senderName, msg.Content),
			})
		}
	}

	messages = append(messages, ai.Message{Role: "user", Content: fmt.Sprintf("💬 请回答：%s", question)})

	sendChunk, finish, err := e.messageSender.SendStreamingAIMessage(conversationID, assistantName)
	if err != nil {
		log.Printf("[SmartReply] 创建流式消息失败: %v", err)
		return
	}

	err = e.aiService.GetCompletionStream(messages, func(chunk ai.StreamChunk) error {
		return sendChunk(chunk.Content)
	})

	if err != nil {
		log.Printf("[SmartReply] AI 流式回复失败: %v", err)
		return
	}

	err = finish()
	if err != nil {
		log.Printf("[SmartReply] 完成流式消息失败: %v", err)
		return
	}

	log.Printf("[SmartReply] @AI 流式回复已完成")
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

	const workerCount = 5
	sem := make(chan struct{}, workerCount)
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for _, group := range groups {
		wg.Add(1)
		sem <- struct{}{}
		go func(g model.Conversation) {
			defer wg.Done()
			defer func() { <-sem }()

			if j.generateGroupSummary(&g) {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				mu.Lock()
				failCount++
				mu.Unlock()
			}

			time.Sleep(2 * time.Second) // 避免 API 限流
		}(group)
	}

	wg.Wait()
	log.Printf("[GroupSummary] 每日总结生成完成，成功: %d, 失败: %d", successCount, failCount)
}

// generateGroupSummary 生成单个群的总结
func (j *GroupSummaryJob) generateGroupSummary(group *model.Conversation) bool {
	db := database.GetDB()

	var groupInfo model.Group
	if err := db.Where("conversation_id = ?", group.ID).First(&groupInfo).Error; err != nil {
		log.Printf("[GroupSummary] 获取群聊信息失败: %v", err)
		return false
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
		return false
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
		return false
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
	return true
}

// checkAvatarTriggers 检查是否有用户的分身需要触发
func (e *SmartReplyEngine) checkAvatarTriggers(senderID uint, conv *model.Conversation, content string, mentionUserIDs []uint) {
	db := database.GetDB()

	// 查找当前会话中启用了分身的用户
	var sessions []model.AvatarSession
	db.Where("conversation_id = ? AND avatar_enabled = ?", conv.ID, true).Find(&sessions)

	for _, session := range sessions {
		// 不触发自己的分身
		if session.UserID == senderID {
			continue
		}

		isGroupChat := conv.Type == "group" || conv.Type == "discussion"
		if e.shouldTriggerAvatar(&session, senderID, content, isGroupChat, mentionUserIDs) {
			// 获取发送者信息
			var sender model.User
			db.First(&sender, senderID)

			// 获取群聊名称（如果是群聊）
			groupName := ""
			if isGroupChat {
				var group model.Group
				if err := db.Where("conversation_id = ?", conv.ID).First(&group).Error; err == nil {
					groupName = group.Name
				}
			}

			task := service.AvatarTask{
				UserID:         session.UserID,
				ConversationID: conv.ID,
				TriggerMessage: content,
				TriggerUserID:  senderID,
				IsGroupChat:    isGroupChat,
				GroupName:      groupName,
				TriggerName:    sender.Nickname,
			}

			if err := e.avatarWorkerPool.Submit(task); err != nil {
				log.Printf("[SmartReply] 提交分身任务失败: %v", err)
			} else {
				log.Printf("[SmartReply] 已触发用户 %d 的分身", session.UserID)
			}
		}
	}
}

// shouldTriggerAvatar 判断是否应该触发分身
func (e *SmartReplyEngine) shouldTriggerAvatar(session *model.AvatarSession, senderID uint, content string, isGroupChat bool, mentionUserIDs []uint) bool {
	db := database.GetDB()

	// 检查是否在接管期内
	if session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		return false
	}

	// 获取分身配置
	var config model.AvatarConfig
	if err := db.Where("user_id = ?", session.UserID).First(&config).Error; err != nil {
		return false
	}

	// 检查分身是否启用
	if !config.Enabled {
		return false
	}

	// 解析触发规则
	var triggerRules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules); err != nil {
			log.Printf("[SmartReply] 解析触发规则失败: %v", err)
			return false
		}
	}

	// 私聊场景：如果是 mention 模式，默认改为 all（因为私聊中不会有 @ 操作）
	if !isGroupChat && triggerRules.Mode == "mention" {
		return true
	}

	// 根据触发模式判断
	switch triggerRules.Mode {
	case "mention":
		// @触发：检查 mention_user_ids 是否包含该用户
		for _, uid := range mentionUserIDs {
			if uid == session.UserID {
				return true
			}
		}
		return false
	case "offline":
		// 离线触发：检查用户是否离线
		var user model.User
		if err := db.First(&user, session.UserID).Error; err != nil {
			return false
		}
		return user.Status == "offline"
	case "keyword":
		// 关键词触发：检查是否包含关键词
		for _, kw := range triggerRules.Keywords {
			if strings.Contains(content, kw) {
				return true
			}
		}
		return false
	case "all":
		// 全部触发
		return true
	default:
		return false
	}
}
