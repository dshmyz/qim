package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"qim-server/database"
	"qim-server/model"
	"qim-server/service"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageSender interface {
	SendAIMessage(conversationID uint, content string, assistantName string) error
	SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error
	SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() error, error)
}

type WebSocketMessageSender struct {
	db        *gorm.DB
	hub       *ws.Hub
	userSvc   *service.UserService
}

func NewWebSocketMessageSender(hub *ws.Hub, userSvc *service.UserService) *WebSocketMessageSender {
	return &WebSocketMessageSender{
		db:      database.GetDB(),
		hub:     hub,
		userSvc: userSvc,
	}
}

func (s *WebSocketMessageSender) SendAIMessage(conversationID uint, content string, assistantName string) error {
	senderID := s.userSvc.GetSystemUserID()

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           "text",
		Content:        content,
		IsRead:         false,
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	systemUser := s.userSvc.GetSystemUser()
	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
	}
	if systemUser != nil {
		aiSender = *systemUser
		if aiSender.Nickname == "" || aiSender.Nickname == "系统" {
			aiSender.Nickname = "🤖 AI 助手"
		}
	}
	aiMessage.Sender = aiSender

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[MessageSender] 获取会话信息失败: %v", err)
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	broadcastNewMessage(&aiMessage, 0, &conv)

	log.Printf("[MessageSender] AI 消息已发送到会话 %d, msgID=%d", conversationID, aiMessage.ID)
	return nil
}

func (s *WebSocketMessageSender) SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() error, error) {
	senderID := s.userSvc.GetSystemUserID()

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           "text",
		Content:        "",
		IsRead:         false,
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return nil, nil, fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	systemUser := s.userSvc.GetSystemUser()
	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
	}
	if systemUser != nil {
		aiSender = *systemUser
		if aiSender.Nickname == "" || aiSender.Nickname == "系统" {
			aiSender.Nickname = "🤖 AI 助手"
		}
	}

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[MessageSender] 获取会话信息失败: %v", err)
		return nil, nil, fmt.Errorf("获取会话信息失败: %w", err)
	}

	accumulatedContent := ""

	sendChunk := func(chunk string) error {
		accumulatedContent += chunk
		aiMessage.Content = accumulatedContent

		if err := s.db.Save(&aiMessage).Error; err != nil {
			log.Printf("[MessageSender] 保存流式消息失败: %v", err)
			return err
		}

		aiMessage.Sender = aiSender

		if s.hub != nil {
			msgData := gin.H{
				"id":                aiMessage.ID,
				"conversation_id":   conversationID,
				"sender_id":         senderID,
				"type":              "text",
				"content":           accumulatedContent,
				"is_ai_message":     true,
				"ai_assistant_name": assistantName,
				"is_streaming":      true,
				"created_at":        aiMessage.CreatedAt,
				"sender":            aiSender,
			}

			wsMsg := ws.WSMessage{
				Type: "new_message",
				Data: msgData,
			}

			jsonMsg, _ := json.Marshal(wsMsg)
			s.hub.SendToConversation(conversationID, 0, jsonMsg)
		}

		return nil
	}

	finish := func() error {
		aiMessage.Content = accumulatedContent
		if err := s.db.Save(&aiMessage).Error; err != nil {
			log.Printf("[MessageSender] 完成流式消息失败: %v", err)
			return err
		}

		aiMessage.Sender = aiSender
		broadcastNewMessage(&aiMessage, 0, &conv)

		log.Printf("[MessageSender] 流式 AI 消息已完成，会话 %d, msgID=%d", conversationID, aiMessage.ID)
		return nil
	}

	return sendChunk, finish, nil
}

func (s *WebSocketMessageSender) SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error {
	senderID := s.userSvc.GetSystemUserID()

	if msg == nil {
		aiMessage := model.Message{
			ConversationID: conversationID,
			SenderID:       senderID,
			Type:           "text",
			Content:        content,
			IsRead:         false,
		}

		if err := s.db.Create(&aiMessage).Error; err != nil {
			return fmt.Errorf("保存 AI 消息失败: %w", err)
		}

		msg = &aiMessage
	}

	systemUser := s.userSvc.GetSystemUser()
	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
	}
	if systemUser != nil {
		aiSender = *systemUser
		if aiSender.Nickname == "" || aiSender.Nickname == "系统" {
			aiSender.Nickname = "🤖 AI 助手"
		}
	}
	msg.Sender = aiSender

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[MessageSender] 获取会话信息失败: %v", err)
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	broadcastNewMessage(msg, 0, &conv)

	log.Printf("[MessageSender] AI 消息已发送到会话 %d, msgID=%d", conversationID, msg.ID)
	return nil
}

func BroadcastAIMessage(conversationID uint, content string, assistantName string) error {
	db := database.GetDB()
	userSvc := service.NewUserService(db)

	senderID := userSvc.GetSystemUserID()

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       senderID,
		Type:           "text",
		Content:        content,
		IsRead:         false,
	}

	if err := db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	systemUser := userSvc.GetSystemUser()
	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
	}
	if systemUser != nil {
		aiSender = *systemUser
		if aiSender.Nickname == "" || aiSender.Nickname == "系统" {
			aiSender.Nickname = "🤖 AI 助手"
		}
	}
	aiMessage.Sender = aiSender

	var conv model.Conversation
	if err := db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[BroadcastAIMessage] 获取会话信息失败: %v", err)
		return nil
	}

	broadcastNewMessage(&aiMessage, 0, &conv)

	if ws.GlobalHub != nil {
		msgData := gin.H{
			"id":                aiMessage.ID,
			"conversation_id":   conversationID,
			"sender_id":         senderID,
			"type":              "text",
			"content":           content,
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

	log.Printf("[BroadcastAIMessage] AI 消息已推送到会话 %d, msgID=%d", conversationID, aiMessage.ID)
	return nil
}
