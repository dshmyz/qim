package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MessageSender interface {
	SendAIMessage(conversationID uint, content string, assistantName string, extra map[string]interface{}) error
	SendMessageWithContext(conversationID uint, content string, assistantName string, msg *model.Message) error
	// SendStreamingAIMessage 创建一个流式 AI 消息并返回：
	//   sendChunk 追加正文块（首个非空块才真正落库创建消息，空块为 no-op）；
	//   getMsg 获取流式中的消息（含已落库的 ID，供关联事件用；尚未创建时触发创建，
	//         便于工具卡片在首个正文前锚定消息 ID）；
	//   finish 收尾并返回最终消息（全程无内容无工具则返回 nil）。
	SendStreamingAIMessage(conversationID uint, assistantName string) (sendChunk func(string) error, getMsg func() *model.Message, finish func() *model.Message, err error)
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

	// Bot 会话：使用 Bot 关联的虚拟用户
	if conv.Type == "bot" {
		var botConv model.BotConversation
		if err := db.Where("conversation_id = ?", conversationID).Preload("Bot").First(&botConv).Error; err == nil {
			if botConv.Bot.VirtualUserID != nil {
				var botUser model.User
				if err := db.First(&botUser, *botConv.Bot.VirtualUserID).Error; err == nil {
					return &botUser, 0, nil
				}
			}
		}
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

func (s *WebSocketMessageSender) SendAIMessage(conversationID uint, content string, assistantName string, extra map[string]interface{}) error {
	// 空/纯空白内容不落库不广播，避免 AI 不可用时出现空白气泡
	if strings.TrimSpace(content) == "" {
		logger.WithModule("MessageSender").Info("AI 消息内容为空，跳过发送",
			"conversationID", conversationID, "assistantName", assistantName)
		return nil
	}

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
		Origin:         "assistant",
	}
	// 附加元数据（如 knowledge_sources）若有内容则写入 Extra，供刷新/回放后渲染徽章。
	if len(extra) > 0 {
		if b, jerr := json.Marshal(extra); jerr == nil {
			aiMessage.Extra = string(b)
		}
	}

	if err := s.db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiMessage.Sender = *aiUser

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		logger.WithModule("MessageSender").Error("获取会话信息失败", "error", err)
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	broadcastNewMessage(&aiMessage, 0, &conv)

	logger.WithModule("MessageSender").Info("AI 消息已发送到会话", "conversationID", conversationID, "groupID", groupID, "msgID", aiMessage.ID, "sender", aiUser.Nickname)
	return nil
}

func (s *WebSocketMessageSender) SendStreamingAIMessage(conversationID uint, assistantName string) (func(string) error, func() *model.Message, func() *model.Message, error) {
	// 消息采用「懒创建」：不在一进入就落库空消息，而是延迟到首个非空正文块
	// （sendChunk）或首个工具调用（getMsg）才真正创建。这样 AI 在思考/调用工具阶段
	// 不会提前插入一个空的流式占位，避免它排在用户后续消息之上造成顺序错乱；
	// 而一旦有正文或工具调用，消息仍在同一 goroutine 内顺序创建，ID 稳定、卡片可锚定。
	var created *model.Message
	var conv *model.Conversation
	accumulatedContent := ""

	// 幂等创建：已创建则直接返回当前消息；未创建则取 AI 发送者、加载会话、
	// 构造空流式消息并落库，随后缓存 conv 供 finish 广播复用。全程同一 goroutine 顺序调用。
	ensureCreated := func() (*model.Message, error) {
		if created != nil {
			return created, nil
		}
		aiUser, _, err := s.resolveAISender(conversationID, assistantName)
		if err != nil {
			return nil, err
		}
		aiMessage := model.Message{
			ConversationID: conversationID,
			SenderID:       aiUser.ID,
			Type:           "text",
			Content:        "",
			IsRead:         false,
			Origin:         "assistant",
			Sender:         *aiUser,
		}
		if err := s.db.Create(&aiMessage).Error; err != nil {
			return nil, fmt.Errorf("保存 AI 消息失败: %w", err)
		}
		var c model.Conversation
		if err := s.db.Preload("Members.User").First(&c, conversationID).Error; err != nil {
			logger.WithModule("MessageSender").Error("获取会话信息失败", "error", err)
			return nil, fmt.Errorf("获取会话信息失败: %w", err)
		}
		created = &aiMessage
		conv = &c
		return created, nil
	}

	getMsg := func() *model.Message {
		// 工具回调（ai_tool_call）经此拿消息 ID；尚未创建则此处触发创建，保证卡片可锚定。
		m, err := ensureCreated()
		if err != nil {
			logger.WithModule("MessageSender").Error("确保流式消息已创建失败", "conversationID", conversationID, "error", err)
			return nil
		}
		return m
	}

	sendChunk := func(chunk string) error {
		accumulatedContent += chunk
		// 空块（如点亮占位用的 sendChunk("")）为 no-op：不落库、不广播，避免超前空占位。
		if strings.TrimSpace(accumulatedContent) == "" {
			return nil
		}
		msg, err := ensureCreated()
		if err != nil {
			return err
		}
		msg.Content = accumulatedContent
		if err := s.db.Save(msg).Error; err != nil {
			logger.WithModule("MessageSender").Error("保存流式消息失败", "error", err)
			return err
		}
		if s.hub != nil {
			msgData := gin.H{
				"id":                msg.ID,
				"conversation_id":   conversationID,
				"sender_id":         msg.SenderID,
				"type":              "markdown",
				"content":           accumulatedContent,
				"is_ai_message":     true,
				"ai_assistant_name": assistantName,
				"is_streaming":      true,
				"is_avatar_reply":   msg.Origin == "avatar",
				"origin":            msg.Origin,
				"created_at":        msg.CreatedAt,
				"sender":            &msg.Sender,
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

	finish := func() *model.Message {
		// 全程无内容无工具调用：消息从未被创建（sendChunk 正文或 getMsg 工具回调都未触发）
		// 则直接返回 nil，由调用方已有的 if finish()==nil 保护跳过，避免残留空白空消息卡住气泡。
		// 一旦有正文/工具，ensureCreated 已被上述路径置 created，此处不会重复创建空消息。
		if created == nil {
			return nil
		}
		msg, err := ensureCreated()
		if err != nil {
			logger.WithModule("MessageSender").Error("完成流式消息失败", "error", err)
			return nil
		}
		msg.Content = accumulatedContent
		msg.Type = "markdown"
		if err := s.db.Save(msg).Error; err != nil {
			logger.WithModule("MessageSender").Error("完成流式消息失败", "error", err)
			return nil
		}
		broadcastNewMessage(msg, 0, conv)
		logger.WithModule("MessageSender").Info("流式 AI 消息已完成", "conversationID", conversationID, "msgID", msg.ID, "sender", msg.Sender.Nickname)
		return msg
	}

	return sendChunk, getMsg, finish, nil
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
			Origin:         "assistant",
		}

		if err := s.db.Create(&aiMessage).Error; err != nil {
			return fmt.Errorf("保存 AI 消息失败: %w", err)
		}

		msg = &aiMessage
	}

	msg.Sender = *aiUser

	var conv model.Conversation
	if err := s.db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		logger.WithModule("MessageSender").Error("获取会话信息失败", "error", err)
		return fmt.Errorf("获取会话信息失败: %w", err)
	}

	broadcastNewMessage(msg, 0, &conv)

	logger.WithModule("MessageSender").Info("AI 消息已发送到会话", "conversationID", conversationID, "msgID", msg.ID)
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
		Origin:         "assistant",
	}

	if err := db.Create(&aiMessage).Error; err != nil {
		return fmt.Errorf("保存 AI 消息失败: %w", err)
	}

	aiMessage.Sender = *aiUser

	var conv model.Conversation
	if err := db.Preload("Members.User").First(&conv, conversationID).Error; err != nil {
		logger.WithModule("BroadcastAIMessage").Error("获取会话信息失败", "error", err)
		return nil
	}

	broadcastNewMessage(&aiMessage, 0, &conv)

	logger.WithModule("BroadcastAIMessage").Info("AI 消息已推送到会话", "conversationID", conversationID, "groupID", groupID, "msgID", aiMessage.ID)
	return nil
}

// broadcastMessageContentUpdate 广播消息内容更新（不更新会话/未读数）
func broadcastMessageContentUpdate(msg *model.Message) {
	if ws.GlobalHub == nil {
		return
	}
	msgData := gin.H{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"content":         msg.Content,
		"type":            msg.Type,
	}
	wsMsg := ws.WSMessage{Type: "message_updated", Data: msgData}
	jsonMsg, _ := json.Marshal(wsMsg)
	ws.GlobalHub.SendToConversation(msg.ConversationID, 0, jsonMsg)
}

// ToolCallRecord 是 service.ToolCallRecord 的类型别名。
// 该结构体已下沉到 service 包（service/streaming_ai_sender.go），使 service 层
// （专属机器人回复路径）能在不 import handler 的前提下复用流式 + 工具调用基建。
// 此处保留别名，handler 内所有引用零改动；WebSocketMessageSender.SendToolCallEvent
// 的形参类型即 service.ToolCallRecord，天然满足 service.StreamingAISender 接口。
type ToolCallRecord = service.ToolCallRecord

// SendToolCallEvent 把一条工具调用作为独立 WS 事件推给会话（type=ai_tool_call），
// 前端按 message_id 把卡片关联到对应流式 AI 消息。实时推送与 Extra 持久化分离：
// 落库由调用方写 Message.Extra（见 smart_reply_handler）。
func (s *WebSocketMessageSender) SendToolCallEvent(conversationID uint, msgID uint, record ToolCallRecord) {
	if s.hub == nil {
		return
	}
	data := gin.H{
		"message_id":      msgID,
		"conversation_id": conversationID,
		"id":              record.ID,
		"tool_name":       record.ToolName,
		"tool_label":      record.ToolLabel,
		"args":            record.Args,
		"status":          record.Status,
	}
	wsMsg := ws.WSMessage{Type: "ai_tool_call", Data: data}
	jsonMsg, _ := json.Marshal(wsMsg)
	s.hub.SendToConversation(conversationID, 0, jsonMsg)
}
