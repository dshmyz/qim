package service

import (
	"encoding/json"
	"errors"
	"time"

	"qim-server/ai"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrMessageNotFound = errors.New("message not found")
var ErrMessageForbidden = errors.New("access forbidden")
var ErrMessageAlreadyRecalled = errors.New("message already recalled")

type MessageService struct{}

func NewMessageService() *MessageService {
	return &MessageService{}
}

type MessageQuery struct {
	ConvID      uint
	UserID      uint
	BeforeMsgID uint
	Limit       int
	Offset      int
	MessageType string
	Keyword     string
	StartDate   string
	EndDate     string
}

type MessageResult struct {
	Messages    []model.Message
	Total       int64
	TotalPages  int
	CurrentPage int
	PageSize    int
}

func (s *MessageService) SendMessage(convID, senderID uint, msgType, content string, quotedMessageID *uint) (*model.Message, error) {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, senderID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	msg := model.Message{
		ConversationID:  convID,
		SenderID:        senderID,
		Type:            msgType,
		Content:         content,
		QuotedMessageID: quotedMessageID,
		IsRead:          false,
	}
	if err := db.Create(&msg).Error; err != nil {
		return nil, err
	}

	db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msg.ID)

	now := time.Now()
	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return nil, err
	}
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	if err := db.Save(&conv).Error; err != nil {
		return nil, err
	}

	if conv.Type == "bot" {
		go s.handleBotMessage(senderID, convID, content)
	} else {
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, senderID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		if ws.GlobalHub != nil {
			newMsg := ws.WSMessage{
				Type: "new_message",
				Data: s.buildMessageResponse(msg),
			}
			jsonMsg, _ := json.Marshal(newMsg)
			ws.GlobalHub.SendToConversation(convID, senderID, jsonMsg)
		}
	}

	return &msg, nil
}

func (s *MessageService) handleBotMessage(userID, convID uint, content string) {
	db := database.GetDB()

	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		return
	}

	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		return
	}

	var messages []model.Message
	db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

	// 获取系统用户 ID
	systemUserID := model.GetSystemUserID(db)

	var aiMessages []ai.Message
	for _, msg := range messages {
		role := "user"
		if msg.SenderID == systemUserID {
			role = "assistant"
		}
		aiMessages = append(aiMessages, ai.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	var fullResponse string
	responseChan := make(chan ai.StreamChunk)

	go func() {
		err := aiSvc.GetCompletionStream(aiMessages, func(chunk ai.StreamChunk) error {
			responseChan <- chunk
			fullResponse += chunk.Content
			return nil
		})

		if err != nil {
			fullResponse = "抱歉，AI 服务暂时不可用，请稍后再试。"
		}

		// 获取系统用户 ID
		senderID := model.GetSystemUserID(db)

		botReply := model.Message{
			ConversationID: convID,
			SenderID:       senderID,
			Type:           "markdown",
			Content:        fullResponse,
		}
		db.Create(&botReply)
	}()
}

func (s *MessageService) GetMessages(query MessageQuery) (*MessageResult, error) {
	db := database.GetDB()

	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		var count int64
		db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", query.ConvID, query.UserID).Count(&count)
		if count == 0 {
			return nil, ErrMessageForbidden
		}
	}

	var total int64
	db.Model(&model.Message{}).Where("conversation_id = ?", query.ConvID).Count(&total)

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	var messages []model.Message
	q := db.Where("conversation_id = ?", query.ConvID)

	if query.BeforeMsgID > 0 {
		var beforeMsg model.Message
		if err := db.First(&beforeMsg, query.BeforeMsgID).Error; err == nil {
			q = q.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	q.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) GetMessagesByFilter(query MessageQuery) (*MessageResult, error) {
	db := database.GetDB()

	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	dbQuery := db.Where("conversation_id = ?", query.ConvID)

	if query.MessageType != "" {
		dbQuery = dbQuery.Where("type = ?", query.MessageType)
	}

	if query.Keyword != "" {
		dbQuery = dbQuery.Where("content LIKE ?", "%"+query.Keyword+"%")
	}

	if query.StartDate != "" {
		dbQuery = dbQuery.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		dbQuery = dbQuery.Where("created_at <= ?", query.EndDate+" 23:59:59")
	}

	var total int64
	dbQuery.Model(&model.Message{}).Count(&total)

	var messages []model.Message
	dbQuery.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) SearchMessages(userID uint, keyword string, convID *uint, limit, offset int) ([]model.Message, error) {
	db := database.GetDB()

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := db.Model(&model.Message{}).Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").Where("conversation_members.user_id = ?", userID)

	if keyword != "" {
		query = query.Where("messages.content LIKE ?", "%"+keyword+"%")
	}

	if convID != nil {
		query = query.Where("messages.conversation_id = ?", *convID)
	}

	var messages []model.Message
	if err := query.Preload("Sender").Preload("Conversation").Preload("Conversation.Members").Preload("Conversation.Members.User").Order("messages.created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *MessageService) RecallMessage(msgID, userID uint) (*model.Message, error) {
	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, ErrMessageForbidden
	}

	if msg.IsRecalled {
		return nil, ErrMessageAlreadyRecalled
	}

	msg.IsRecalled = true
	msg.Content = "[消息已撤回]"
	if err := db.Save(&msg).Error; err != nil {
		return nil, err
	}

	db.Preload("Sender").First(&msg, msg.ID)

	if ws.GlobalHub != nil {
		recallMsg := ws.WSMessage{
			Type: "message_recalled",
			Data: msg,
		}
		jsonMsg, _ := json.Marshal(recallMsg)
		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return &msg, nil
}

func (s *MessageService) DeleteMessage(msgID, userID uint) error {
	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMessageNotFound
		}
		return err
	}

	if msg.SenderID != userID {
		return ErrMessageForbidden
	}

	if err := db.Delete(&msg).Error; err != nil {
		return err
	}

	if ws.GlobalHub != nil {
		deleteMsg := ws.WSMessage{
			Type: "message_deleted",
			Data: map[string]interface{}{
				"message_id":      msg.ID,
				"conversation_id": msg.ConversationID,
			},
		}
		jsonMsg, _ := json.Marshal(deleteMsg)
		ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return nil
}

func (s *MessageService) MarkAsRead(convID, userID uint) error {
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return ErrMessageForbidden
	}

	var unreadMsgIDs []uint
	if err := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ?", convID, userID).
		Pluck("id", &unreadMsgIDs).Error; err != nil {
		return err
	}

	if len(unreadMsgIDs) > 0 {
		batchSize := 500
		for i := 0; i < len(unreadMsgIDs); i += batchSize {
			end := i + batchSize
			if end > len(unreadMsgIDs) {
				end = len(unreadMsgIDs)
			}
			batch := unreadMsgIDs[i:end]

			receipts := make([]model.MessageReadReceipt, 0, len(batch))
			now := time.Now()
			for _, msgID := range batch {
				receipts = append(receipts, model.MessageReadReceipt{
					MessageID:      msgID,
					ConversationID: convID,
					UserID:         userID,
					CreatedAt:      now,
				})
			}

			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "message_id"}, {Name: "user_id"}},
				DoNothing: true,
			}).Create(&receipts).Error; err != nil {
				return err
			}
		}
	}

	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", convID, userID).
		UpdateColumn("is_read", true)

	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumn("unread_count", 0)

	now := time.Now()
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumn("last_read_at", now)

	if result.RowsAffected > 0 {
		s.notifyMessageRead(convID, userID)
	}

	return nil
}

func (s *MessageService) notifyMessageRead(convID, userID uint) {
	if ws.GlobalHub == nil {
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return
	}

	readMsg := ws.WSMessage{
		Type: "message_read",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         userID,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(readMsg)

	if conv.Type == "single" {
		var otherMember model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, userID).First(&otherMember)
		ws.GlobalHub.SendToUser(otherMember.UserID, jsonMsg)
	} else if conv.Type == "group" {
		var members []model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, userID).Find(&members)

		for _, member := range members {
			ws.GlobalHub.SendToUser(member.UserID, jsonMsg)
		}
	}
}

func (s *MessageService) GetMessageByID(msgID uint) (*model.Message, error) {
	db := database.GetDB()

	var msg model.Message
	if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return &msg, nil
}

func (s *MessageService) GetMessageQuoteChain(msgID, userID uint) ([]model.Message, error) {
	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	var quoteChain []model.Message
	currentMsg := msg

	for i := 0; i < 3 && currentMsg.QuotedMessageID != nil; i++ {
		var quotedMsg model.Message
		if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&quotedMsg, *currentMsg.QuotedMessageID).Error; err == nil {
			quoteChain = append(quoteChain, quotedMsg)
			currentMsg = quotedMsg
		} else {
			break
		}
	}

	return quoteChain, nil
}

func (s *MessageService) GetMessageReadUsers(msgID, userID uint) ([]model.User, int64, error) {
	db := database.GetDB()

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		return nil, 0, ErrMessageNotFound
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, 0, ErrMessageForbidden
	}

	var readReceipts []model.MessageReadReceipt
	if err := db.Where("message_id = ?", msgID).Preload("User").Order("created_at DESC").Find(&readReceipts).Error; err != nil {
		return nil, 0, err
	}

	var readUsers []model.User
	for _, receipt := range readReceipts {
		if receipt.User != nil && receipt.User.ID != userID {
			readUsers = append(readUsers, *receipt.User)
		}
	}

	var totalMembers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

	return readUsers, totalMembers, nil
}

func (s *MessageService) buildMessageResponse(msg model.Message) map[string]interface{} {
	return map[string]interface{}{
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
}

func (s *MessageService) CreateMessage(msg *model.Message) error {
	db := database.GetDB()
	return db.Create(msg).Error
}
