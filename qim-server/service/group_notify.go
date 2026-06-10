package service

import (
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// NotifyMembersJoined 统一处理成员加入群聊时的通知（系统消息+WebSocket广播）
// 供 AddMemberToGroup、AI助手加入、审批通过后加入等场景复用
func NotifyMembersJoined(db *gorm.DB, convID uint, senderID uint, content string, addedMembers []model.User) {
	// 创建系统消息
	systemMsg := &model.Message{
		ConversationID: convID,
		SenderID:       senderID,
		Type:           "system",
		Content:        content,
		IsRead:         true,
	}
	if err := db.Create(systemMsg).Error; err != nil {
		logger.WithModule("GroupNotify").Error("创建系统消息失败", "error", err)
	}

	// 更新会话最后消息
	now := time.Now()
	db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
		"last_message_id": systemMsg.ID,
		"last_message_at": now,
	})

	// WebSocket 广播
	if ws.GlobalHub == nil {
		return
	}

	for _, member := range addedMembers {
		joinMsg := ws.WSMessage{
			Type: "group_member_joined",
			Data: gin.H{
				"conversation_id": convID,
				"member": gin.H{
					"id":       member.ID,
					"nickname": member.Nickname,
					"username": member.Username,
					"avatar":   member.Avatar,
					"type":     member.Type,
				},
			},
		}
		jsonMsg, _ := json.Marshal(joinMsg)
		ws.GlobalHub.SendToConversation(convID, 0, jsonMsg)
	}

	// 获取发送者信息填充 systemMsg.Sender
	var sender model.User
	if err := db.First(&sender, senderID).Error; err == nil {
		systemMsg.Sender = sender
	}

	newMsg := ws.WSMessage{
		Type: "new_message",
		Data: gin.H{
			"id":              systemMsg.ID,
			"conversation_id": systemMsg.ConversationID,
			"sender_id":       systemMsg.SenderID,
			"type":            systemMsg.Type,
			"content":         systemMsg.Content,
			"is_read":         systemMsg.IsRead,
			"created_at":      systemMsg.CreatedAt,
			"sender":          systemMsg.Sender,
		},
	}
	newMsgJson, _ := json.Marshal(newMsg)
	ws.GlobalHub.SendToConversation(convID, 0, newMsgJson)

	ws.GlobalHub.UpdateConversationMembers(convID)
}
