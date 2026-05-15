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
	db      *gorm.DB
	hub     *ws.Hub
	userSvc *service.UserService
}

func NewWebSocketMessageSender(hub *ws.Hub, userSvc *service.UserService) *WebSocketMessageSender {
	return &WebSocketMessageSender{
		db:      database.GetDB(),
		hub:     hub,
		userSvc: userSvc,
	}
}

func resolveAISender(db *gorm.DB, userSvc *service.UserService, conversationID uint, assistantName string) (*model.User, uint, error) {
	var conv model.Conversation
	if err := db.First(&conv, conversationID).Error; err != nil {
		return nil, 0, fmt.Errorf("会话不存在")
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		aiUser, err := userSvc.GetDefaultAIAssistant()
		if err != nil {
			return nil, 0, fmt.Errorf("获取默认 AI 助手失败: %w", err)
		}
		return aiUser, 0, nil
	}

	var group model.Group
	if err := db.Where("conversation_id = ?", conversationID).First(&group).Error; err != nil {
		return nil, 0, fmt.Errorf("获取群信息失败")
	}

	aiUser, err := userSvc.EnsureGroupAIAssistant(group.ID, assistantName)
	if err != nil {
		return nil, 0, fmt.Errorf("获取 AI 助手用户失败: %w", err)
	}
	return aiUser, group.ID, nil
}

func (s *WebSocketMessageSender) resolveAISender(conversationID uint, assistantName string) (*model.User, uint, error) {
	return resolveAISender(s.db, s.userSvc, conversationID, assistantName)
}

func (s *WebSocketMessageSender) SendAIMessage(conversationID uint, content string, assistantName string) error {
	aiUser, groupID, err := s.resolveAISender(conversationID, assistantName)
	if err != nil {
		return err
	}

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       aiUser.ID,
		Type:           "markdown",
		Content:        content,
		IsRead:         false,
		AIType:         "assistant",
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiMessage.Sender = *aiUser

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		log.Printf("[MessageSender] 获取会话信息失败: %v", err)
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	broadcastNewMessage(&aiMessage, 0, &conv)

	log.Printf("[MessageSender] AI 消息已发送到会话 %d (group=%d), msgID=%d, sender=%s", conversationID, groupID, aiMessage.ID, aiUser.Nickname)
	return nil
}

func (s *WebSocketMessageSender) SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() error, error) {
	aiUser, _, err := s.resolveAISender(conversationID, assistantName)
	if err != nil {
		return nil, nil, err
	}

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       aiUser.ID,
		Type:           "text",
		Content:        "",
		IsRead:         false,
		AIType:         "assistant",
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return nil, nil, fmt.Errorf("保存 AI 消息失败: %w", err)
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

		aiMessage.Sender = *aiUser

		if s.hub != nil {
			msgData := gin.H{
				"id":                aiMessage.ID,
				"conversation_id":   conversationID,
				"sender_id":         aiUser.ID,
				"type":              "markdown",
				"content":           accumulatedContent,
				"is_ai_message":     true,
				"ai_assistant_name": assistantName,
				"is_streaming":      true,
				"is_avatar_reply":   aiMessage.AIType == "avatar",
				"ai_type":           aiMessage.AIType,
				"created_at":        aiMessage.CreatedAt,
				"sender":            aiUser,
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
		aiMessage.Type = "markdown"
		if err := s.db.Save(&aiMessage).Error; err != nil {
			log.Printf("[MessageSender] 完成流式消息失败: %v", err)
			return err
		}

		aiMessage.Sender = *aiUser
		broadcastNewMessage(&aiMessage, 0, &conv)

		log.Printf("[MessageSender] 流式 AI 消息已完成，会话 %d, msgID=%d, sender=%s", conversationID, aiMessage.ID, aiUser.Nickname)
		return nil
	}

	return sendChunk, finish, nil
}

func (s *WebSocketMessageSender) SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error {
	aiUser, _, err := s.resolveAISender(conversationID, assistantName)
	if err != nil {
		return err
	}

	if msg == nil {
		aiMessage := model.Message{
			ConversationID: conversationID,
			SenderID:       aiUser.ID,
			Type:           "markdown",
			Content:        content,
			IsRead:         false,
			AIType:         "assistant",
		}

		if err := s.db.Create(&aiMessage).Error; err != nil {
			return fmt.Errorf("保存 AI 消息失败: %w", err)
		}

		msg = &aiMessage
	}

	msg.Sender = *aiUser

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

	aiUser, groupID, err := resolveAISender(db, userSvc, conversationID, assistantName)
	if err != nil {
		return err
	}

	aiMessage := model.Message{
		ConversationID: conversationID,
		SenderID:       aiUser.ID,
		Type:           "markdown",
		Content:        content,
		IsRead:         false,
		AIType:         "assistant",
	}

	if err := db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiMessage.Sender = *aiUser

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
			"sender_id":         aiUser.ID,
			"type":              "text",
			"content":           content,
			"is_ai_message":     true,
			"ai_assistant_name": assistantName,
			"ai_type":           aiMessage.AIType,
			"is_avatar_reply":   aiMessage.AIType == "avatar",
			"created_at":        aiMessage.CreatedAt,
			"sender":            aiUser,
		}

		wsMsg := ws.WSMessage{
			Type: "new_message",
			Data: msgData,
		}

		jsonMsg, _ := json.Marshal(wsMsg)
		ws.GlobalHub.SendToConversation(conversationID, 0, jsonMsg)
	}

	log.Printf("[BroadcastAIMessage] AI 消息已推送到会话 %d (group=%d), msgID=%d", conversationID, groupID, aiMessage.ID)
	return nil
}
