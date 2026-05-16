package handler

import (
	"context"
	"encoding/json"
	"log"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/service"
	"qim-server/utils"
	"qim-server/ws"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// SmartReplyEngine 智能回复引擎
type SmartReplyEngine struct {
	aiService        *ai.AIService
	intentDetector   *ai.IntentDetector
	unifiedKnowledge *service.UnifiedKnowledgeService
	memorySvc        *service.AvatarMemoryService
	messageSender    *WebSocketMessageSender
	avatarWorkerPool *service.AvatarWorkerPool
	smartReplyGraph  *service.SmartReplyGraph
}

// NewSmartReplyEngine 创建智能回复引擎
func NewSmartReplyEngine(aiService *ai.AIService, detector *ai.IntentDetector) *SmartReplyEngine {
	return &SmartReplyEngine{
		aiService:      aiService,
		intentDetector: detector,
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

// InitSmartReplyGraph initializes the Eino Graph for smart reply
func (e *SmartReplyEngine) InitSmartReplyGraph() error {
	log.Printf("[SmartReplyGraph] 创建 SmartReplyGraph 实例...")
	e.smartReplyGraph = service.NewSmartReplyGraph(
		e.aiService,
		database.GetDB(),
		e.unifiedKnowledge,
		nil, // LegacyKnowledgeService no longer needed
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
	log.Printf("[HandleMessage] sender=%d conv=%d, aiConfigured=%v, avatarPool=%v", userID, conversationID, e.aiService != nil && e.aiService.IsConfigured(), e.avatarWorkerPool != nil)
	if e.aiService == nil || !e.aiService.IsConfigured() {
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return
	}

	if conv.Type == "bot" {
		return
	}

	// 按会话类型分别处理
	if conv.Type == "group" || conv.Type == "discussion" {
		e.handleGroupMessage(userID, conversationID, content, mentionUserIDs, &conv)
	} else {
		e.handlePrivateMessage(userID, conversationID, content, &conv)
	}
}

// handleGroupMessage 处理群聊消息
func (e *SmartReplyEngine) handleGroupMessage(userID uint, conversationID uint, content string, mentionUserIDs []uint, conv *model.Conversation) {
	db := database.GetDB()

	// 1. 分身触发：只在 @ 恰好 1 个人时才触发（避免 @ 多人打架）
	if e.avatarWorkerPool != nil && len(mentionUserIDs) == 1 {
		triggered := e.checkAvatarTriggers(userID, conv, content, mentionUserIDs)
		if triggered {
			return
		}
	}

	// 2. 获取群 AI 配置
	var group model.Group
	if err := db.Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
		return
	}
	aiConfig := group.GetAIConfig()
	if !aiConfig.Enabled {
		return
	}

	assistantName := "AI助手"
	if aiConfig.AssistantName != "" {
		assistantName = aiConfig.AssistantName
	}

	// 2a. @AI 提及 → 群助手回复
	if e.isAIMention(content, assistantName) {
		question := extractAIQuestion(content, assistantName)
		e.handleAIMention(userID, conversationID, question, content, conv, assistantName)
		return
	}

	// 2b. 反垃圾检查
	if aiConfig.AntiSpamInterval > 0 {
		var lastAIMsg model.Message
		err := db.Where("conversation_id = ? AND ai_type != '' AND created_at > ?",
			conversationID, time.Now().Add(-time.Duration(aiConfig.AntiSpamInterval)*time.Minute)).
			Order("created_at DESC").First(&lastAIMsg).Error
		if err == nil {
			return
		}
	}

	// 2c. 已配置群 AI Bot 时，SmartReply 不自动回复
	var hasGroupAssistant bool
	if err := db.Model(&model.Bot{}).
		Where("type = ? AND group_id = ? AND is_active = ?", "ai", group.ID, true).
		First(&model.Bot{}).Error; err == nil {
		hasGroupAssistant = true
	}
	if hasGroupAssistant {
		return
	}

	// 2d. 回复模式检查
	if aiConfig.ReplyMode == "off" || aiConfig.ReplyMode == "mention_only" {
		return
	}

	// 2e. 关键词检查
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

	// 2f. 通过所有检查，走智能回复
	e.smartReply(userID, conversationID, content, conv)
}

// handlePrivateMessage 处理私聊消息
func (e *SmartReplyEngine) handlePrivateMessage(userID uint, conversationID uint, content string, conv *model.Conversation) {
	if e.avatarWorkerPool == nil {
		return
	}

	// 私聊中只有消息的接收者（非发送者）的分身才应该回复
	// 从会话成员中找到对方用户
	db := database.GetDB()
	var members []model.ConversationMember
	db.Where("conversation_id = ?", conversationID).Find(&members)
	for _, m := range members {
		if m.UserID == userID {
			continue // 跳过发送者自己
		}
		// m.UserID 是对方用户，检查是否启用了分身
		var cfg model.AvatarConfig
		if err := db.Where("user_id = ? AND enabled = ?", m.UserID, true).First(&cfg).Error; err != nil {
			continue
		}
		// 检查是否在该会话中明确关闭了分身
		var session model.AvatarSession
		if err := db.Where("user_id = ? AND conversation_id = ? AND avatar_enabled = ?", m.UserID, conversationID, false).First(&session).Error; err == nil {
			continue
		}
		e.triggerSingleAvatar(db, cfg.UserID, userID, conv, content, false)
		break
	}
}

// smartReply 意图检测 + 生成回复
func (e *SmartReplyEngine) smartReply(userID uint, conversationID uint, content string, conv *model.Conversation) {
	intent, err := e.intentDetector.Detect(content, userID, conversationID)
	if err != nil {
		return
	}
	if e.intentDetector.ShouldTriggerAIReply(intent, conv.Type) {
		utils.SafeGo(func() { e.generateAndSendReply(userID, conversationID, content, intent) })
	}
}

// generateAndSendReply 生成并发送智能回复
func (e *SmartReplyEngine) generateAndSendReply(userID uint, conversationID uint, userContent string, intent *ai.MessageIntent) {
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
		g := group
		utils.SafeGo(func() {
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
		})
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
func (e *SmartReplyEngine) checkAvatarTriggers(senderID uint, conv *model.Conversation, content string, mentionUserIDs []uint) bool {
	db := database.GetDB()

	isGroupChat := conv.Type == "group" || conv.Type == "discussion"

	// 群聊：调用方已保证 len(mentionUserIDs) == 1，检查被 @ 的那个用户是否启用了分身
	if isGroupChat {
		mentionedUserID := mentionUserIDs[0]
		var cfg model.AvatarConfig
		if err := db.Where("user_id = ? AND enabled = ?", mentionedUserID, true).First(&cfg).Error; err != nil {
			return false
		}
		if cfg.UserID == senderID {
			return false
		}
		var session model.AvatarSession
		err := db.Where("user_id = ? AND conversation_id = ? AND avatar_enabled = ?", cfg.UserID, conv.ID, false).First(&session).Error
		if err == nil {
			return false
		}
		return e.triggerSingleAvatar(db, cfg.UserID, senderID, conv, content, isGroupChat)
	}

	// 私聊：遍历所有启用分身，触发匹配的
	var allConfigs []model.AvatarConfig
	db.Where("enabled = ?", true).Find(&allConfigs)
	for _, cfg := range allConfigs {
		if cfg.UserID == senderID {
			continue
		}
		var session model.AvatarSession
		err := db.Where("user_id = ? AND conversation_id = ? AND avatar_enabled = ?", cfg.UserID, conv.ID, false).First(&session).Error
		if err == nil {
			continue
		}
		if e.shouldTriggerAvatar(&session, senderID, content, isGroupChat, mentionUserIDs) {
			return e.triggerSingleAvatar(db, cfg.UserID, senderID, conv, content, isGroupChat)
		}
	}
	return false
}

// triggerSingleAvatar 触发单个分身任务
func (e *SmartReplyEngine) triggerSingleAvatar(db *gorm.DB, userID uint, senderID uint, conv *model.Conversation, content string, isGroupChat bool) bool {
	var sender model.User
	db.First(&sender, senderID)

	groupName := ""
	if isGroupChat {
		var group model.Group
		if err := db.Where("conversation_id = ?", conv.ID).First(&group).Error; err == nil {
			groupName = group.Name
		}
	}

	task := service.AvatarTask{
		UserID:         userID,
		ConversationID: conv.ID,
		TriggerMessage: content,
		TriggerUserID:  senderID,
		IsGroupChat:    isGroupChat,
		GroupName:      groupName,
		TriggerName:    sender.Nickname,
	}

	if err := e.avatarWorkerPool.Submit(task); err != nil {
		log.Printf("[SmartReply] 提交分身任务失败: %v", err)
		return false
	}
	log.Printf("[SmartReply] 已触发用户 %d 的分身", userID)
	return true
}

// shouldTriggerAvatar 判断是否应该触发分身
func (e *SmartReplyEngine) shouldTriggerAvatar(session *model.AvatarSession, senderID uint, content string, isGroupChat bool, mentionUserIDs []uint) bool {
	db := database.GetDB()

	// 检查是否在接管期内
	if session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		log.Printf("[shouldTriggerAvatar] user=%d 处于接管期内", session.UserID)
		return false
	}

	// 获取分身配置
	var config model.AvatarConfig
	if err := db.Where("user_id = ?", session.UserID).First(&config).Error; err != nil {
		log.Printf("[shouldTriggerAvatar] user=%d 未找到 AvatarConfig: %v", session.UserID, err)
		return false
	}

	// 检查分身是否启用
	if !config.Enabled {
		log.Printf("[shouldTriggerAvatar] user=%d 分身未启用", session.UserID)
		return false
	}

	// 解析触发规则
	var triggerRules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules); err != nil {
			log.Printf("[shouldTriggerAvatar] user=%d 解析触发规则失败: %v", session.UserID, err)
			return false
		}
	}
	log.Printf("[shouldTriggerAvatar] user=%d triggerRules.Mode=%q, isGroupChat=%v", session.UserID, triggerRules.Mode, isGroupChat)

	// 群聊场景：先检查是否 @了自己，没 @直接不触发
	if isGroupChat {
		mentioned := false
		for _, uid := range mentionUserIDs {
			if uid == session.UserID {
				mentioned = true
				break
			}
		}
		if !mentioned {
			log.Printf("[shouldTriggerAvatar] user=%d 群聊未@，不触发", session.UserID)
			return false
		}
		// @命中后，根据回复策略决定是否回复
		switch triggerRules.Mode {
		case "off":
			log.Printf("[shouldTriggerAvatar] user=%d 群聊@命中但回复模式为off", session.UserID)
			return false
		default:
			// mention/smart/always/auto/off 以外的模式，@命中后都回复
			log.Printf("[shouldTriggerAvatar] user=%d 群聊@命中，回复模式=%q，触发", session.UserID, triggerRules.Mode)
			return true
		}
	}

	// 私聊场景：非 mention 模式下直接触发（私聊中不会有 @ 操作）
	if triggerRules.Mode != "mention" {
		log.Printf("[shouldTriggerAvatar] user=%d 私聊非mention模式，触发", session.UserID)
		return true
	}

	// 私聊 mention 模式：始终触发（私聊就是发给本人的）
	return true
}
