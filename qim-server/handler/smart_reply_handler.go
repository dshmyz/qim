package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"
	"log"
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
	avatarTriggerSvc AvatarTriggerDecider
	smartReplyGraph  *service.SmartReplyGraph
}

// AvatarTriggerDecider decides whether a configured avatar should reply.
// The interface keeps the real-time trigger path independent from a concrete AI provider.
type AvatarTriggerDecider interface {
	ShouldReply(userID uint, conversationID uint, message string, senderName string) (bool, string, error)
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

// SetAvatarTriggerService injects the smart-avatar decision service.
func (e *SmartReplyEngine) SetAvatarTriggerService(triggerSvc AvatarTriggerDecider) {
	e.avatarTriggerSvc = triggerSvc
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

func (e *SmartReplyEngine) HandleMessage(userID uint, conversationID uint, content string, mentionUserIDs []uint) {
	if e.aiService == nil || !e.aiService.IsConfigured() {
		log.Printf("[SmartReply] AI 服务未配置，跳过处理")
		return
	}

	configSvc := service.NewSystemConfigService(database.GetDB())
	publicConfigs, err := configSvc.GetPublicConfigs()
	if err == nil {
		if enableAI, ok := publicConfigs["enableAI"]; ok {
			if !enableAI.(bool) {
				log.Printf("[SmartReply] AI 功能已关闭 (enableAI=false)，跳过处理")
				return
			}
		}
	}

	log.Printf("[SmartReply] 开始处理消息: userID=%d convID=%d content=%s", userID, conversationID, content[:min(50, len(content))])

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		log.Printf("[SmartReply] 查找会话失败: convID=%d err=%v", conversationID, err)
		return
	}

	log.Printf("[SmartReply] 会话类型: %s", conv.Type)

	if e.avatarWorkerPool != nil {
		e.checkAvatarTriggers(userID, &conv, content, mentionUserIDs)
	}

	if conv.Type == "bot" {
		log.Printf("[SmartReply] bot 会话，跳过 AI 助手回复")
		return
	}

	if conv.Type == "single" {
		var avatarSessions []model.AvatarSession
		db.Where("conversation_id = ? AND avatar_enabled = ? AND user_id != ?", conv.ID, true, userID).Find(&avatarSessions)
		if len(avatarSessions) > 0 {
			log.Printf("[SmartReply] 私聊中对方分身已激活，跳过 AI 助手回复 (对方userID=%d)", avatarSessions[0].UserID)
			return
		}
	}

	var group *model.Group
	if conv.Type == "group" || conv.Type == "discussion" {
		var g model.Group
		if err := db.Where("conversation_id = ?", conversationID).First(&g).Error; err != nil {
			log.Printf("[SmartReply] 查找群聊失败: convID=%d err=%v", conversationID, err)
			return
		}
		group = &g
		aiConfig := group.GetAIConfig()
		log.Printf("[SmartReply] 群聊 AI 配置: enabled=%v replyMode=%s triggerKeywords=%s", aiConfig.Enabled, aiConfig.ReplyMode, aiConfig.TriggerKeywords)
		if !aiConfig.Enabled {
			log.Printf("[SmartReply] 群聊 AI 未启用，跳过处理")
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
				log.Printf("[SmartReply] 消息不包含触发关键词，跳过处理")
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
				log.Printf("[SmartReply] 反垃圾策略：AI 最近已回复，跳过 (interval=%dmin)", aiConfig.AntiSpamInterval)
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
		log.Printf("[SmartReply] 意图检测失败: %v", err)
		return
	}

	log.Printf("[SmartReply] 意图检测结果: type=%s confidence=%.2f", intent.Type, intent.Confidence)

	shouldReply := e.intentDetector.ShouldTriggerAIReply(intent, conv.Type)

	if !shouldReply {
		log.Printf("[SmartReply] 意图类型 %s (confidence=%.2f) 不触发 AI 回复", intent.Type, intent.Confidence)
	}

	if shouldReply {
		go e.generateAndSendReply(userID, conversationID, content, intent)
	}

	// 待办提取（所有群聊消息都尝试提取）
	if todoExtractor != nil && conv.Type == "group" {
		if group != nil && group.GetAIConfig().ExtractTodos {
			go todoExtractor.ExtractAndCreateTodos(content, userID, conversationID)
		}
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
	reply, err := e.aiService.GetCompletionWithTools(ai.TaskTypeChat, messages, callerCtx)
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

// RateLimiter 简易令牌桶限流器
type RateLimiter struct {
	ticker *time.Ticker
	ch     chan struct{}
	once   sync.Once
}

// NewRateLimiter 创建限流器，interval 为两次放行间隔，burst 为突发上限
func NewRateLimiter(interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		ticker: time.NewTicker(interval),
		ch:     make(chan struct{}, burst),
	}
	// 预填令牌
	for i := 0; i < burst; i++ {
		rl.ch <- struct{}{}
	}
	go func() {
		for range rl.ticker.C {
			select {
			case rl.ch <- struct{}{}:
			default:
			}
		}
	}()
	return rl
}

// Wait 阻塞直到获取一个令牌
func (rl *RateLimiter) Wait() {
	<-rl.ch
}

// Stop 停止限流器
func (rl *RateLimiter) Stop() {
	rl.once.Do(func() {
		rl.ticker.Stop()
		close(rl.ch)
	})
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

	err = e.aiService.GetCompletionStream(ai.TaskTypeChat, messages, func(chunk ai.StreamChunk) error {
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
	// 共享令牌桶：每秒 2 次 AI 调用，突发上限 1
	rl := NewRateLimiter(500*time.Millisecond, 1)
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

			rl.Wait() // 等待令牌，只在真正调用 AI 前阻塞
			if j.generateGroupSummary(&g) {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else {
				mu.Lock()
				failCount++
				mu.Unlock()
			}
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

	summary, err := j.aiService.GetCompletion(ai.TaskTypeAnalysis, messages_input)
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

	var sessions []model.AvatarSession
	db.Where("conversation_id = ? AND avatar_enabled = ?", conv.ID, true).Find(&sessions)

	log.Printf("[AvatarTrigger] 查找分身会话: convID=%d senderID=%d 找到%d个激活的分身会话", conv.ID, senderID, len(sessions))

	var convMembers []model.ConversationMember
	db.Where("conversation_id = ? AND user_id != ?", conv.ID, senderID).Find(&convMembers)

	for _, member := range convMembers {
		hasSession := false
		for _, session := range sessions {
			if session.UserID == member.UserID {
				hasSession = true
				break
			}
		}

		if !hasSession {
			var avatarConfig model.AvatarConfig
			if err := db.Where("user_id = ? AND enabled = ?", member.UserID, true).First(&avatarConfig).Error; err == nil {
				log.Printf("[AvatarTrigger] 用户 %d 全局分身已启用但无会话级记录，自动创建 (convID=%d)", member.UserID, conv.ID)
				newSession := model.AvatarSession{
					ConversationID: conv.ID,
					UserID:         member.UserID,
					AvatarEnabled:  true,
				}
				if err := db.Create(&newSession).Error; err != nil {
					log.Printf("[AvatarTrigger] 自动创建分身会话失败: userID=%d err=%v", member.UserID, err)
				} else {
					sessions = append(sessions, newSession)
				}
			}
		}
	}

	for _, session := range sessions {
		log.Printf("[AvatarTrigger] 检查分身会话: userID=%d avatarEnabled=%v", session.UserID, session.AvatarEnabled)

		if session.UserID == senderID {
			log.Printf("[AvatarTrigger] 跳过自己的分身: userID=%d == senderID=%d", session.UserID, senderID)
			continue
		}

		isGroupChat := conv.Type == "group" || conv.Type == "discussion"
		triggered := e.shouldTriggerAvatar(&session, senderID, content, isGroupChat, mentionUserIDs)
		log.Printf("[AvatarTrigger] 触发判断结果: userID=%d triggered=%v (isGroupChat=%v mentionUserIDs=%v)", session.UserID, triggered, isGroupChat, mentionUserIDs)

		if triggered {
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
				UserID:         session.UserID,
				ConversationID: conv.ID,
				TriggerMessage: content,
				TriggerUserID:  senderID,
				IsGroupChat:    isGroupChat,
				GroupName:      groupName,
				TriggerName:    sender.Nickname,
			}

			if err := e.avatarWorkerPool.Submit(task); err != nil {
				log.Printf("[AvatarTrigger] 提交分身任务失败: %v", err)
			} else {
				log.Printf("[AvatarTrigger] 已触发用户 %d 的分身 (convID=%d triggerUserID=%d)", session.UserID, conv.ID, senderID)
			}
		}
	}
}

// shouldTriggerAvatar 判断是否应该触发分身
func (e *SmartReplyEngine) shouldTriggerAvatar(session *model.AvatarSession, senderID uint, content string, isGroupChat bool, mentionUserIDs []uint) bool {
	db := database.GetDB()

	if session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		log.Printf("[AvatarTrigger] 分身接管期内，不触发: userID=%d takeoverUntil=%v", session.UserID, session.TakeoverUntil)
		return false
	}

	var config model.AvatarConfig
	if err := db.Where("user_id = ?", session.UserID).First(&config).Error; err != nil {
		log.Printf("[AvatarTrigger] 分身配置未找到: userID=%d err=%v", session.UserID, err)
		return false
	}

	if !config.Enabled {
		log.Printf("[AvatarTrigger] 分身配置未启用: userID=%d enabled=%v", session.UserID, config.Enabled)
		return false
	}

	log.Printf("[AvatarTrigger] 分身配置: userID=%d enabled=%v triggerMode=%s triggerRulesJSON=%s", session.UserID, config.Enabled, config.TriggerRulesJSON, config.TriggerRulesJSON)

	var triggerRules model.AvatarTriggerRules
	if config.TriggerRulesJSON != "" {
		if err := json.Unmarshal([]byte(config.TriggerRulesJSON), &triggerRules); err != nil {
			log.Printf("[AvatarTrigger] 解析触发规则失败: userID=%d err=%v", session.UserID, err)
			return false
		}
	}

	effectiveMode := triggerRules.Mode
	if effectiveMode == "" {
		effectiveMode = "mention"
	}

	log.Printf("[AvatarTrigger] 触发规则: userID=%d mode=%s effectiveMode=%s keywords=%v", session.UserID, triggerRules.Mode, effectiveMode, triggerRules.Keywords)

	if !isGroupChat && effectiveMode == "mention" {
		log.Printf("[AvatarTrigger] 私聊中 mention 模式自动触发: userID=%d", session.UserID)
		return true
	}

	switch effectiveMode {
	case "mention":
		for _, uid := range mentionUserIDs {
			if uid == session.UserID {
				log.Printf("[AvatarTrigger] mention 触发成功: userID=%d 在 mentionUserIDs 中", session.UserID)
				return true
			}
		}
		log.Printf("[AvatarTrigger] mention 触发失败: userID=%d 不在 mentionUserIDs(%v) 中", session.UserID, mentionUserIDs)
		return false
	case "offline":
		var user model.User
		if err := db.First(&user, session.UserID).Error; err != nil {
			log.Printf("[AvatarTrigger] offline 触发失败: 查找用户失败 userID=%d err=%v", session.UserID, err)
			return false
		}
		log.Printf("[AvatarTrigger] offline 触发判断: userID=%d status=%s", session.UserID, user.Status)
		return user.Status == "offline"
	case "keyword":
		for _, kw := range triggerRules.Keywords {
			if strings.Contains(content, kw) {
				log.Printf("[AvatarTrigger] keyword 触发成功: userID=%d keyword=%s", session.UserID, kw)
				return true
			}
		}
		if len(triggerRules.Keywords) == 0 {
			log.Printf("[AvatarTrigger] keyword 模式无关键词，默认触发: userID=%d", session.UserID)
			return true
		}
		log.Printf("[AvatarTrigger] keyword 触发失败: userID=%d 消息不包含任何关键词 %v", session.UserID, triggerRules.Keywords)
		return false
	case "all":
		log.Printf("[AvatarTrigger] all 模式触发: userID=%d", session.UserID)
		return true
	case "smart":
		if e.avatarTriggerSvc == nil {
			log.Printf("[AvatarTrigger] smart 模式跳过：意图判断服务未初始化 userID=%d", session.UserID)
			return false
		}

		var sender model.User
		if err := db.Select("nickname", "username").First(&sender, senderID).Error; err != nil {
			log.Printf("[AvatarTrigger] smart 模式跳过：获取发送者失败 userID=%d err=%v", senderID, err)
			return false
		}
		senderName := sender.Nickname
		if senderName == "" {
			senderName = sender.Username
		}

		shouldReply, reason, err := e.avatarTriggerSvc.ShouldReply(session.UserID, session.ConversationID, content, senderName)
		if err != nil {
			log.Printf("[AvatarTrigger] smart 意图判断失败，跳过回复: userID=%d err=%v", session.UserID, err)
			return false
		}
		log.Printf("[AvatarTrigger] smart 意图判断: userID=%d shouldReply=%v reason=%s", session.UserID, shouldReply, reason)
		return shouldReply
	default:
		log.Printf("[AvatarTrigger] 未知触发模式: userID=%d mode=%s，按 mention 处理", session.UserID, effectiveMode)
		return isGroupChat && len(mentionUserIDs) > 0
	}
}
