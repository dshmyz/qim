package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageSender interface {
	SendAIMessage(conversationID uint, content string, assistantName string) error
	SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error
}

type WebSocketMessageSender struct {
	db  *gorm.DB
	hub *ws.Hub
}

func NewWebSocketMessageSender(hub *ws.Hub) *WebSocketMessageSender {
	return &WebSocketMessageSender{
		db:  database.GetDB(),
		hub: hub,
	}
}

func (s *WebSocketMessageSender) SendAIMessage(conversationID uint, content string, assistantName string) error {
	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       0,
		Type:           "text",
		Content:        content,
		IsRead:         false,
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
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

func (s *WebSocketMessageSender) SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error {
	if msg == nil {
		aiMessage := model.Message{
			ConversationID: conversationID,
			SenderID:       0,
			Type:           "text",
			Content:        content,
			IsRead:         false,
		}

		if err := s.db.Create(&aiMessage).Error; err != nil {
			return fmt.Errorf("保存 AI 消息失败: %w", err)
		}

		msg = &aiMessage
	}

	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
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

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       0,
		Type:           "text",
		Content:        content,
		IsRead:         false,
	}

	if err := db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiSender := model.User{
		ID:       0,
		Username: "ai_assistant",
		Nickname: "🤖 AI 助手",
		Avatar:   "",
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
			"sender_id":         0,
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
