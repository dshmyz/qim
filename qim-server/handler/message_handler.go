package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	pageStr := c.Query("page")
	pageSizeStr := c.Query("page_size")

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

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		var count int64
		db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", convID, userID).Count(&count)
		if count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
			return
		}
	}

	var total int64
	db.Model(&model.Message{}).Where("conversation_id = ?", convID).Count(&total)

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	var messages []model.Message
	db.Where("conversation_id = ?", convID).Preload("Sender").Preload("QuotedMessage").Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&messages)

	for i := range messages {
		if messages[i].QuotedMessage != nil {
			db.Model(&messages[i].QuotedMessage).Association("Sender").Find(&messages[i].QuotedMessage.Sender)
		}
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	var responseMessages []gin.H
	for _, msg := range messages {

		responseMsg := gin.H{
			"id":                msg.ID,
			"conversation_id":   msg.ConversationID,
			"sender_id":         msg.SenderID,
			"type":              msg.Type,
			"content":           msg.Content,
			"quoted_message_id": msg.QuotedMessageID,
			"is_recalled":       msg.IsRecalled,
			"is_read":           msg.IsRead,
			"recalled_at":       msg.RecalledAt,
			"created_at":        msg.CreatedAt,
			"sender":            msg.Sender,
			"quoted_message":    msg.QuotedMessage,
		}
		responseMessages = append(responseMessages, responseMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": responseMessages,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total":        total,
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "会话ID不能为空"})
		return
	}

	db := database.GetDB()
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
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

	query := db.Where("conversation_id = ?", convID)

	if messageType != "" {
		query = query.Where("type = ?", messageType)
	}

	if search != "" {
		query = query.Where("content LIKE ?", "%"+search+"%")
	}

	if startDate != "" {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("created_at <= ?", endDate+" 23:59:59")
	}

	var total int64
	query.Model(&model.Message{}).Count(&total)

	var messages []model.Message
	query.Preload("Sender").Preload("QuotedMessage").Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&messages)

	for i := range messages {
		if messages[i].QuotedMessage != nil {
			db.Model(&messages[i].QuotedMessage).Association("Sender").Find(&messages[i].QuotedMessage.Sender)
		}
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"messages": messages,
			"total":    total,
		},
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
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发送消息"})
		return
	}

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	content := req.Content

	msg := model.Message{
		ConversationID:  uint(convIDUint),
		SenderID:        userID.(uint),
		Type:            req.Type,
		Content:         content,
		QuotedMessageID: req.QuotedMessageID,
		IsRead:          false,
	}
	db.Create(&msg)

	db.Preload("Sender").Preload("QuotedMessage").First(&msg, msg.ID)

	if msg.QuotedMessage != nil {
		db.Model(&msg.QuotedMessage).Association("Sender").Find(&msg.QuotedMessage.Sender)
	}

	responseData := gin.H{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
	}

	now := time.Now()
	var conv model.Conversation
	db.First(&conv, convID)
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	if conv.Type == "bot" {
		go HandleBotMessage(userID.(uint), uint(convIDUint), req.Content)
	} else {
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, userID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		if ws.GlobalHub != nil {
			newMsg := ws.WSMessage{
				Type: "new_message",
				Data: responseData,
			}
			jsonMsg, _ := json.Marshal(newMsg)

			log.Printf("发送WebSocket消息到会话 %d，排除用户 %d", uint(convIDUint), userID.(uint))
			ws.GlobalHub.SendToConversation(uint(convIDUint), userID.(uint), jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": responseData})
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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发送消息"})
		return
	}

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "bot" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "仅支持机器人会话的流式消息"})
		return
	}

	convIDUint, _ := strconv.ParseUint(convID, 10, 32)

	content := req.Content

	msg := model.Message{
		ConversationID:  uint(convIDUint),
		SenderID:        userID.(uint),
		Type:            req.Type,
		Content:         content,
		QuotedMessageID: req.QuotedMessageID,
		IsRead:          false,
	}
	db.Create(&msg)

	now := time.Now()
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	responseChan := make(chan ai.StreamChunk)
	doneChan := make(chan bool)

	go func() {
		var botConv model.BotConversation
		if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
			log.Printf("[StreamMessage] 查找机器人会话关联失败: %v", err)
			close(responseChan)
			doneChan <- true
			return
		}

		var bot model.Bot
		if err := db.First(&bot, botConv.BotID).Error; err != nil {
			log.Printf("[StreamMessage] 查找机器人信息失败: %v", err)
			close(responseChan)
			doneChan <- true
			return
		}

		var messages []model.Message
		db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

		var aiMessages []ai.Message
		for _, msg := range messages {
			role := "user"
			if msg.SenderID == 0 {
				role = "assistant"
			}
			aiMessages = append(aiMessages, ai.Message{
				Role:    role,
				Content: msg.Content,
			})
		}

		var fullResponse string
		err := aiService.GetCompletionStream(aiMessages, func(chunk ai.StreamChunk) error {
			responseChan <- chunk
			fullResponse += chunk.Content
			return nil
		})

		if err != nil {
			log.Printf("[StreamMessage] AI API 调用失败: %v", err)
			errorMsg := "抱歉，AI 服务暂时不可用，请稍后再试。"
			responseChan <- ai.StreamChunk{Content: errorMsg}
			fullResponse = errorMsg
		}

		close(responseChan)
		doneChan <- true

		botReply := model.Message{
			ConversationID: uint(convIDUint),
			SenderID:       0,
			Type:           "markdown",
			Content:        fullResponse,
		}
		db.Create(&botReply)

		logLength := 100
		if len(fullResponse) < logLength {
			logLength = len(fullResponse)
		}
		log.Printf("[StreamMessage] 机器人回复保存成功: %s", fullResponse[:logLength])
	}()

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
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	// 检查是否在2分钟内可撤回
	if time.Since(msg.CreatedAt) > 2*time.Minute {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "消息已超过2分钟，无法撤回",
		})
		return
	}

	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能撤回自己发送的消息"})
		return
	}

	if msg.IsRecalled {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "消息已经被撤回"})
		return
	}

	msg.IsRecalled = true
	msg.Content = "[消息已撤回]"
	db.Save(&msg)

	db.Preload("Sender").First(&msg, msg.ID)

	if ws.GlobalHub != nil {
		recallMsg := ws.WSMessage{
			Type: "message_recalled",
			Data: msg,
		}
		jsonMsg, _ := json.Marshal(recallMsg)

		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "消息撤回成功", "data": msg})
}

func RemindMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限发送提醒"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "提醒已发送",
	})
}

func DeleteMessage(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	if msg.SenderID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只能删除自己发送的消息"})
		return
	}

	if err := db.Delete(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除消息失败"})
		return
	}

	if ws.GlobalHub != nil {
		deleteMsg := ws.WSMessage{
			Type: "message_deleted",
			Data: gin.H{
				"message_id":      msg.ID,
				"conversation_id": msg.ConversationID,
			},
		}
		jsonMsg, _ := json.Marshal(deleteMsg)

		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "消息删除成功",
	})
}

func GetMessageReadUsers(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	var readReceipts []model.MessageReadReceipt
	db.Where("message_id = ?", msgID).Preload("User").Order("created_at DESC").Find(&readReceipts)

	var readUsers []map[string]interface{}
	for _, receipt := range readReceipts {
		if receipt.User != nil && receipt.User.ID != userID.(uint) {
			name := receipt.User.Nickname
			if name == "" {
				name = receipt.User.Username
			}
			readUsers = append(readUsers, map[string]interface{}{
				"id":       receipt.User.ID,
				"name":     name,
				"username": receipt.User.Username,
				"avatar":   receipt.User.Avatar,
			})
		}
	}

	var totalMembers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"read_users":    readUsers,
			"total_members": totalMembers,
		},
	})
}

func MarkConversationAsRead(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	var messages []model.Message
	db.Where("conversation_id = ?", uint(convID)).Find(&messages)

	for _, msg := range messages {
		var existingReceipt model.MessageReadReceipt
		err := db.Where("message_id = ? AND user_id = ?", msg.ID, userID).First(&existingReceipt).Error
		if err != nil {
			receipt := model.MessageReadReceipt{
				MessageID:      msg.ID,
				ConversationID: msg.ConversationID,
				UserID:         userID.(uint),
				CreatedAt:      time.Now(),
			}
			db.Create(&receipt)
		}
	}

	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", uint(convID), userID).
		UpdateColumn("is_read", true)

	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("unread_count", 0)

	now := time.Now()
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", uint(convID), userID).
		UpdateColumn("last_read_at", now)

	if result.RowsAffected > 0 {
		var conv model.Conversation
		db.First(&conv, uint(convID))

		if conv.Type == "single" {
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", uint(convID), userID).First(&otherMember)

			if ws.GlobalHub != nil {
				readMsg := ws.WSMessage{
					Type: "message_read",
					Data: map[string]interface{}{
						"conversation_id": convID,
						"user_id":         userID,
						"timestamp":       time.Now().Unix(),
					},
				}
				jsonMsg, _ := json.Marshal(readMsg)

				ws.GlobalHub.SendToUser(otherMember.UserID, jsonMsg)
			}
		} else if conv.Type == "group" {
			var members []model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", uint(convID), userID).Find(&members)

			if ws.GlobalHub != nil {
				readMsg := ws.WSMessage{
					Type: "message_read",
					Data: map[string]interface{}{
						"conversation_id": convID,
						"user_id":         userID,
						"timestamp":       time.Now().Unix(),
					},
				}
				jsonMsg, _ := json.Marshal(readMsg)

				for _, member := range members {
					ws.GlobalHub.SendToUser(member.UserID, jsonMsg)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "标记已读成功"})
}

func SearchMessages(c *gin.Context) {
	userID, _ := c.Get("user_id")

	keyword := c.Query("keyword")
	convID := c.Query("conv_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	msgType := c.Query("type")

	db := database.GetDB()

	query := db.Model(&model.Message{}).Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").Where("conversation_members.user_id = ?", userID)

	if keyword != "" {
		query = query.Where("messages.content LIKE ?", "%"+keyword+"%")
	}

	if convID != "" {
		query = query.Where("messages.conversation_id = ?", convID)
	}

	if startDate != "" {
		query = query.Where("messages.created_at >= ?", startDate)
	}

	if endDate != "" {
		query = query.Where("messages.created_at <= ?", endDate)
	}

	if msgType != "" {
		query = query.Where("messages.type = ?", msgType)
	}

	var messages []model.Message
	query.Preload("Sender").Preload("Conversation").Order("messages.created_at DESC").Find(&messages)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": messages,
	})
}

func GetMessageQuoteChain(c *gin.Context) {
	userID, _ := c.Get("user_id")
	msgIDStr := c.Param("id")

	msgID, err := strconv.ParseUint(msgIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的消息ID"})
		return
	}

	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, uint(msgID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "消息不存在"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限访问"})
		return
	}

	var quoteChain []model.Message
	currentMsg := msg

	for i := 0; i < 3 && currentMsg.QuotedMessageID != nil; i++ {
		var quotedMsg model.Message
		if err := db.Preload("Sender").First(&quotedMsg, *currentMsg.QuotedMessageID).Error; err == nil {
			quoteChain = append(quoteChain, quotedMsg)
			currentMsg = quotedMsg
		} else {
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"messages": quoteChain,
		},
	})
}
