package handler

import (
	"encoding/json"
	"fmt"
	"qim-server/ai"
	"qim-server/database"
	"qim-server/di"
	"qim-server/model"
	"qim-server/pkg/logger"
	"qim-server/pkg/mention"
	"qim-server/pkg/response"
	"qim-server/service"
	"qim-server/utils"
	"qim-server/ws"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Global smart reply engine instance
var smartReplyEngine *SmartReplyEngine

// Global todo extractor instance
var todoExtractor *TodoExtractor

// Global anomaly detector instance
var anomalyDetector *AnomalyDetector

// InitSmartReplyEngine initializes the smart reply engine with the given AI service
func InitSmartReplyEngine(aiService *ai.AIService) {
	detector := ai.NewIntentDetector(aiService)
	smartReplyEngine = NewSmartReplyEngine(aiService, detector)
	todoExtractor = NewTodoExtractor(aiService)
}

// SetAvatarWorkerPool sets the avatar worker pool for the smart reply engine
func SetAvatarWorkerPool(pool *service.AvatarWorkerPool) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetAvatarWorkerPool(pool)
	}
}

// SetUnifiedKnowledge sets the unified knowledge service for the smart reply engine
func SetUnifiedKnowledge(uk *service.UnifiedKnowledgeService) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetUnifiedKnowledge(uk)
	}
}

// SetMemoryService sets the avatar memory service for the smart reply engine
func SetMemoryService(ms *service.AvatarMemoryService) {
	if smartReplyEngine != nil {
		smartReplyEngine.SetMemoryService(ms)
	}
}

// InitSmartReplyGraph initializes the Eino Graph for smart reply
func InitSmartReplyGraph() error {
	if smartReplyEngine == nil {
		return fmt.Errorf("SmartReplyEngine not initialized")
	}

	logger.WithModule("SmartReplyGraph").Info("开始初始化...")
	logger.WithModule("SmartReplyGraph").Info("aiService", "initialized", smartReplyEngine.aiService != nil)
	logger.WithModule("SmartReplyGraph").Info("unifiedKnowledge", "initialized", smartReplyEngine.unifiedKnowledge != nil)
	logger.WithModule("SmartReplyGraph").Info("memorySvc", "initialized", smartReplyEngine.memorySvc != nil)

	err := smartReplyEngine.InitSmartReplyGraph()
	if err != nil {
		logger.WithModule("SmartReplyGraph").Error("初始化失败", "error", err)
		return err
	}

	logger.WithModule("SmartReplyGraph").Info("初始化成功")
	return nil
}

// InitAnomalyDetector initializes the anomaly detector
func InitAnomalyDetector() {
	anomalyDetector = NewAnomalyDetector()
	StartAnomalyDetection(anomalyDetector)
}

// GetSmartReplyEngine returns the smart reply engine instance
func GetSmartReplyEngine() *SmartReplyEngine {
	return smartReplyEngine
}

func resolveAIName(msg model.Message) string {
	nameCache := service.GetAINameCache()
	db := database.GetDB()

	if msg.AIType == "assistant" {
		var group model.Group
		if err := db.Select("ai_config").
			Where("conversation_id = ?", msg.ConversationID).
			First(&group).Error; err == nil && group.AIConfigJSON != "" {
			aiConfig := group.GetAIConfig()
			if aiConfig.AssistantName != "" {
				return aiConfig.AssistantName
			}
		}
	}
	if msg.AIType == "avatar" {
		if name := nameCache.GetAvatarName(msg.SenderID); name != "" {
			return name
		}
	}
	if msg.Sender.Type == "bot" || msg.Sender.Type == "system" {
		return msg.Sender.Nickname
	}
	return "AI助手"
}

func buildMessageResponse(msg model.Message, currentUserID uint) gin.H {
	isAtMention := msg.SenderID != currentUserID && mention.IsMentioned(msg.Content, currentUserID)

	aiName := resolveAIName(msg)

	resp := gin.H{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"is_avatar_reply":   msg.AIType == "avatar",
		"is_ai_message":     msg.AIType == "assistant" || msg.AIType == "avatar" || msg.Sender.Type == "bot" || msg.Sender.Type == "system",
		"ai_assistant_name": aiName,
		"ai_type":           msg.AIType,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"is_at_mention":     isAtMention,
	}

	// 分身消息：透出分身名称
	if msg.AIType == "avatar" {
		resp["avatar_name"] = service.GetAINameCache().GetAvatarName(msg.SenderID)
	}

	return resp
}

func GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	afterIDStr := c.Query("after_id")

	page := 1
	pageSize := 20
	afterID := uint(0)

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	if afterIDStr != "" {
		if id, err := strconv.ParseUint(afterIDStr, 10, 64); err == nil {
			afterID = uint(id)
		}
	}

	offset := (page - 1) * pageSize

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), userID.(uint))
	if !isMember {
		query := service.MessageQuery{
			ConvID: uint(convIDUint),
			UserID: userID.(uint),
			Limit:  1,
		}
		result, _ := msgSvc.GetMessages(query)
		if result == nil || result.Total == 0 {
			response.Forbidden(c, "无权限访问")
			return
		}
	}

	query := service.MessageQuery{
		ConvID: uint(convIDUint),
		UserID: userID.(uint),
		Limit:  pageSize,
		Offset: offset,
	}
	if afterID > 0 {
		query.BeforeMsgID = afterID
	}
	result, err := msgSvc.GetMessages(query)
	if err != nil {
		response.InternalServerError(c, "获取消息失败")
		return
	}

	var responseMessages []gin.H
	for _, msg := range result.Messages {
		responseMessages = append(responseMessages, buildMessageResponse(msg, userID.(uint)))
	}

	totalPages := int(result.Total) / pageSize
	if int(result.Total)%pageSize > 0 {
		totalPages++
	}

	response.Success(c, gin.H{
		"messages": responseMessages,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total":        result.Total,
			"total_pages":  totalPages,
		},
	})
}

func GetMessagesByFilter(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Query("conversation_id")
	messageType := c.Query("type")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")
	search := c.Query("search")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if convID == "" {
		response.BadRequest(c, "会话ID不能为空")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)
	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), userID.(uint))
	if !isMember {
		response.Forbidden(c, "无权限访问")
		return
	}

	page := 1
	pageSize := 10
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}
	offset := (page - 1) * pageSize

	query := service.MessageQuery{
		ConvID:      uint(convIDUint),
		UserID:      userID.(uint),
		Limit:       pageSize,
		Offset:      offset,
		MessageType: messageType,
		Keyword:     search,
		StartDate:   startDate,
		EndDate:     endDate,
	}
	result, err := msgSvc.GetMessagesByFilter(query)
	if err != nil {
		response.InternalServerError(c, "获取消息失败")
		return
	}

	response.Success(c, gin.H{
		"messages": result.Messages,
		"total":    result.Total,
	})
}

func SendMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Content         string                 `json:"content" binding:"required"`
		QuotedMessageID *uint                  `json:"quoted_message_id"`
		ShareData       map[string]interface{} `json:"share_data"`
		MentionUserIDs  []uint                 `json:"mention_user_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	msgSvc := di.GlobalContainer.MessageService
	convSvc := di.GlobalContainer.ConversationService
	fileSvc := di.GlobalContainer.FileService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	msg, err := msgSvc.SendMessage(uint(convIDUint), userID.(uint), req.Type, req.Content, req.QuotedMessageID)
	if err != nil {
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限发送消息")
			return
		}
		response.InternalServerError(c, "消息发送失败")
		return
	}

	if req.Type == "file" || req.Type == "image" {
		var fileData struct {
			URL string `json:"url"`
			ID  uint   `json:"id"`
		}
		if err := json.Unmarshal([]byte(req.Content), &fileData); err == nil && fileData.ID > 0 {
			fileSvc.UpdateFileSource(fileData.ID, userID.(uint), "chat")
		}
	}

	responseData := buildMessageResponse(*msg, userID.(uint))

	conv, _ := convSvc.GetConversation(uint(convIDUint))
	if conv != nil && conv.Type != "bot" {
		// AI 生成的消息不触发其他 AI 回复
		if msg.Sender.Type != "bot" && msg.Sender.Type != "system" {
			if smartReplyEngine != nil {
				utils.SafeGo(func() {
					smartReplyEngine.HandleMessage(userID.(uint), uint(convIDUint), req.Content, req.MentionUserIDs)
				})
			}
		}

		if anomalyDetector != nil {
			utils.SafeGo(func() {
				anomalyDetector.RecordMessage(uint(convIDUint))

				if alert := anomalyDetector.CheckSensitiveContent(req.Content); alert != nil {
					anomalyDetector.SendAlert(userID.(uint), alert)
				}

				if alert := anomalyDetector.CheckMessageFrequency(userID.(uint), uint(convIDUint)); alert != nil {
					anomalyDetector.SendAlert(userID.(uint), alert)
				}
			})
		}
	}

	response.Success(c, responseData)
}

// broadcastNewMessage 广播新消息到会话并更新相关状态
func broadcastNewMessage(msg *model.Message, excludeUserID uint, conv *model.Conversation) {
	convSvc := di.GlobalContainer.ConversationService

	now := time.Now()
	convSvc.UpdateConversation(msg.ConversationID, map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	if excludeUserID > 0 {
		convSvc.IncrementUnreadCount(msg.ConversationID, excludeUserID)
	}

	mentions := mention.ExtractMentions(msg.Content)
	mentionUserIDs := make([]uint, 0, len(mentions))
	for _, m := range mentions {
		mentionUserIDs = append(mentionUserIDs, m.UserID)
	}

	aiName := resolveAIName(*msg)

	responseData := gin.H{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"is_avatar_reply":   msg.AIType == "avatar",
		"is_ai_message":     msg.AIType == "assistant" || msg.AIType == "avatar" || msg.Sender.Type == "bot" || msg.Sender.Type == "system",
		"ai_assistant_name": aiName,
		"ai_type":           msg.AIType,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"mention_user_ids":  mentionUserIDs,
	}

	// 分身消息：透出分身名称
	if msg.AIType == "avatar" {
		responseData["avatar_name"] = service.GetAINameCache().GetAvatarName(msg.SenderID)
	}

	if ws.GlobalHub != nil {
		newMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		jsonMsg, _ := json.Marshal(newMsg)
		utils.SafeGo(func() { ws.GlobalHub.SendToConversationAsync(msg.ConversationID, excludeUserID, jsonMsg) })
	}
}

func StreamMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		Type            string                 `json:"type" binding:"required"`
		Content         string                 `json:"content" binding:"required"`
		QuotedMessageID *uint                  `json:"quoted_message_id"`
		ShareData       map[string]interface{} `json:"share_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	msgSvc := di.GlobalContainer.MessageService

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	isMember, _ := convSvc.IsConversationMember(uint(convIDUint), userID.(uint))
	if !isMember {
		response.Forbidden(c, "无权限发送消息")
		return
	}

	conv, err := convSvc.GetConversation(uint(convIDUint))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "bot" {
		response.BadRequest(c, "仅支持机器人会话的流式消息")
		return
	}

	content := req.Content

	msg := model.Message{
		ConversationID:  uint(convIDUint),
		SenderID:        userID.(uint),
		Type:            req.Type,
		Content:         content,
		QuotedMessageID: req.QuotedMessageID,
		IsRead:          false,
	}
	msgSvc.CreateMessage(&msg)

	now := time.Now()
	convSvc.UpdateConversation(conv.ID, map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	})

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	responseChan := make(chan ai.StreamChunk)
	doneChan := make(chan bool)

	utils.SafeGoWithLabel("stream-message", func() {
		db := database.GetDB()
		var botConv model.BotConversation
		if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
			logger.WithModule("StreamMessage").Error("查找机器人会话关联失败", "error", err)
			close(responseChan)
			doneChan <- true
			return
		}

		var bot model.Bot
		if err := db.First(&bot, botConv.BotID).Error; err != nil {
			logger.WithModule("StreamMessage").Error("查找机器人信息失败", "error", err)
			close(responseChan)
			doneChan <- true
			return
		}

		systemPrompt := "你是一个智能助手，帮助用户解决问题。"
		if bot.Config != "" {
			var botConfig map[string]interface{}
			if err := json.Unmarshal([]byte(bot.Config), &botConfig); err == nil {
				if prompt, ok := botConfig["system_prompt"].(string); ok && prompt != "" {
					systemPrompt = prompt
				}
			}
		}

		var messages []model.Message
		db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

		logger.WithModule("StreamMessage").Info("加载历史消息",
			"conversationID", convID,
			"messageCount", len(messages),
			"botID", botConv.BotID,
			"virtualUserID", bot.VirtualUserID,
		)

		var aiMessages []ai.Message
		aiMessages = append(aiMessages, ai.Message{
			Role:    "system",
			Content: systemPrompt,
		})

		for _, msg := range messages {
			role := "user"
			if msg.SenderID == 0 || (bot.VirtualUserID != nil && msg.SenderID == *bot.VirtualUserID) {
				role = "assistant"
			}
			aiMessages = append(aiMessages, ai.Message{
				Role:    role,
				Content: msg.Content,
			})
		}

		// 确保最新的用户问题包含在消息中（防止数据库查询延迟导致新消息未被加载）
		// 检查最后一条消息是否是用户的新问题
		if len(aiMessages) == 1 || aiMessages[len(aiMessages)-1].Role != "user" || aiMessages[len(aiMessages)-1].Content != content {
			aiMessages = append(aiMessages, ai.Message{
				Role:    "user",
				Content: content,
			})
		}

		logger.WithModule("StreamMessage").Info("准备调用 AI", "messageCount", len(aiMessages))
		for i, aiMsg := range aiMessages {
			if i == 0 {
				logger.WithModule("StreamMessage").Info("AI消息 - system", "index", i, "role", aiMsg.Role)
			} else {
				logger.WithModule("StreamMessage").Info("AI消息", "index", i, "role", aiMsg.Role, "content", aiMsg.Content[:min(50, len(aiMsg.Content))])
			}
		}

		var fullResponse string
		err := di.GlobalContainer.AIService.GetCompletionStream(ai.TaskTypeChat, aiMessages, func(chunk ai.StreamChunk) error {
			responseChan <- chunk
			fullResponse += chunk.Content
			return nil
		})

		if err != nil {
			logger.WithModule("StreamMessage").Error("AI API 调用失败", "error", err)
			errorMsg := "抱歉，AI 服务暂时不可用，请稍后再试。"
			responseChan <- ai.StreamChunk{Content: errorMsg}
			fullResponse = errorMsg
		}

		close(responseChan)
		doneChan <- true

		senderID := uint(0)
		if bot.VirtualUserID != nil {
			senderID = *bot.VirtualUserID
		}

		botReply := model.Message{
			ConversationID: uint(convIDUint),
			SenderID:       senderID,
			Type:           "markdown",
			Content:        fullResponse,
		}
		db.Create(&botReply)

		logLength := 100
		if len(fullResponse) < logLength {
			logLength = len(fullResponse)
		}
		logger.WithModule("StreamMessage").Info("机器人回复保存成功", "content", fullResponse[:logLength])
	})

	c.Writer.Write([]byte("data: \n\n"))
	c.Writer.Flush()

	for {
		select {
		case chunk, ok := <-responseChan:
			if !ok {
				finish := "stop"
				doneData, _ := json.Marshal(ai.StreamChunk{Finish: &finish})
				c.Writer.Write([]byte("data: " + string(doneData) + "\n\n"))
				c.Writer.Flush()
				return
			}
			data, _ := json.Marshal(chunk)
			c.Writer.Write([]byte("data: " + string(data) + "\n\n"))
			c.Writer.Flush()
		case <-doneChan:
			return
		case <-c.Request.Context().Done():
			return
		}
	}
}

func RecallMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	msg, err := msgSvc.RecallMessage(uint(msgID), userID.(uint))
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "只能撤回自己发送的消息")
			return
		}
		if err == service.ErrMessageAlreadyRecalled {
			response.BadRequest(c, "消息已经被撤回")
			return
		}
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "消息撤回成功", "data": msg})
}

func RemindMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	msg, err := msgSvc.GetMessageByID(uint(msgID))
	if err != nil {
		response.NotFound(c, "消息不存在")
		return
	}

	if msg.SenderID != userID.(uint) {
		response.Forbidden(c, "无权限发送提醒")
		return
	}

	response.Success(c, gin.H{
		"message": "提醒已发送",
	})
}

func DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	err = msgSvc.DeleteMessage(uint(msgID), userID.(uint))
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "只能删除自己发送的消息")
			return
		}
		response.InternalServerError(c, "删除消息失败")
		return
	}

	response.Success(c, gin.H{
		"message": "消息删除成功",
	})
}

func GetMessageReadUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	readUsers, totalMembers, err := msgSvc.GetMessageReadUsers(uint(msgID), userID.(uint))
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "获取已读用户失败")
		return
	}

	response.Success(c, gin.H{
		"read_users":    readUsers,
		"read_count":    int64(len(readUsers)),
		"total_members": totalMembers,
	})
}

func BatchGetMessageReadUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		MessageIDs []uint `json:"message_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.MessageIDs) == 0 {
		response.BadRequest(c, "消息ID列表不能为空")
		return
	}

	if len(req.MessageIDs) > 50 {
		response.BadRequest(c, "一次最多查询50条消息")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	result, err := msgSvc.BatchGetMessageReadUsers(req.MessageIDs, userID.(uint))
	if err != nil {
		response.InternalServerError(c, "批量获取已读用户失败")
		return
	}

	response.Success(c, result)
}

func MarkConversationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	err = msgSvc.MarkAsRead(uint(convID), userID.(uint))
	if err != nil {
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "标记已读失败")
		return
	}

	response.Success(c, gin.H{
		"message": "标记已读成功",
	})
}

func SearchMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")

	keyword := c.Query("keyword")
	convID := c.Query("conv_id")
	pageStr := c.Query("page")
	pageSizeStr := c.Query("pageSize")

	page := 1
	pageSize := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	offset := (page - 1) * pageSize

	msgSvc := di.GlobalContainer.MessageService

	var convIDPtr *uint
	if convID != "" {
		id, _ := strconv.ParseUint(convID, 10, 32)
		cid := uint(id)
		convIDPtr = &cid
	}

	messages, err := msgSvc.SearchMessages(userID.(uint), keyword, convIDPtr, pageSize, offset)
	if err != nil {
		response.InternalServerError(c, "搜索消息失败")
		return
	}

	response.Success(c, gin.H{
		"list":  messages,
		"total": len(messages),
		"page":  page,
	})
}

func GetMessageQuoteChain(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的消息ID")
		return
	}

	msgSvc := di.GlobalContainer.MessageService

	quoteChain, err := msgSvc.GetMessageQuoteChain(uint(msgID), userID.(uint))
	if err != nil {
		if err == service.ErrMessageNotFound {
			response.NotFound(c, "消息不存在")
			return
		}
		if err == service.ErrMessageForbidden {
			response.Forbidden(c, "无权限访问")
			return
		}
		response.InternalServerError(c, "获取引用链失败")
		return
	}

	response.Success(c, gin.H{
		"messages": quoteChain,
	})
}
