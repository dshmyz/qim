package ws

import (
	"encoding/json"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/utils"

	"gorm.io/gorm"
)

func handleSendMessage(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)
	msgType, _ := msgData["type"].(string)
	content, _ := msgData["content"].(string)

	var quotedMessageID *uint
	if quotedID, ok := msgData["quoted_message_id"].(float64); ok {
		quotedIDUint := uint(quotedID)
		quotedMessageID = &quotedIDUint
	}

	// 统一调用外部注册的 MessageService 处理
	// service.SendMessage 内部已完成：db.Create、广播、未读计数、OnMessageSent 触发
	if c.hub.HandleMessage != nil {
		_, err := c.hub.HandleMessage(convID, c.userID, msgType, content, quotedMessageID)
		if err != nil {
			errMsg := WSMessage{
				Type: "error",
				Data: map[string]interface{}{"code": "send_failed", "message": err.Error()},
			}
			jsonErr, _ := json.Marshal(errMsg)
			safeSend(c, jsonErr)
		}
		return
	}

	// 降级：无 HandleMessage 时使用原有逻辑（兼容测试场景）
	fallbackHandleMessage(c, convID, msgType, content, quotedMessageID)
}

// fallbackHandleMessage 仅当 Hub.HandleMessage 未注册（nil）时触发的降级路径。
// 生产环境由 InitWSHandlers 必然注册 MessageService.SendMessage（auth_handler.go），
// 因此该路径实际不可达；其载荷构建仍是手写（历史遗留，未走 service.BuildMessageResponse），
// 保留仅作防御。字段集可能落后于统一构建函数——改动消息字段时请同步此处或直接删除。
func fallbackHandleMessage(c *Client, convID uint, msgType, content string, quotedMessageID *uint) {
	db := c.hub.db

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		errMsg := WSMessage{
			Type: "error",
			Data: map[string]interface{}{"code": "forbidden", "message": "你不是该会话的成员"},
		}
		jsonErr, _ := json.Marshal(errMsg)
		safeSend(c, jsonErr)
		return
	}

	msg := model.Message{
		ConversationID:  convID,
		SenderID:        c.userID,
		Type:            msgType,
		Content:         content,
		QuotedMessageID: quotedMessageID,
	}
	db.Create(&msg)

	db.Preload("Sender").Preload("QuotedMessage").First(&msg, msg.ID)

	if msg.QuotedMessage != nil {
		db.Model(&msg.QuotedMessage).Association("Sender").Find(&msg.QuotedMessage.Sender)
	}

	now := time.Now()
	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return
	}
	conv.LastMessageID = &msg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	// 恢复会话显示：新消息到来时，如果会话被隐藏则恢复显示
	db.Model(&model.ConversationSession{}).
		Where("conversation_id = ? AND is_hidden = ?", convID, true).
		Update("is_hidden", false)

	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, c.userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	mentions := mention.Parse(msg.Content)
	var allMembers []model.ConversationMember
	db.Where("conversation_id = ?", convID).Find(&allMembers)
	allMemberIDs := make([]uint, 0, len(allMembers))
	for _, m := range allMembers {
		allMemberIDs = append(allMemberIDs, m.UserID)
	}
	mentionUserIDs := mention.ExtractUserIDs(mentions, allMemberIDs, c.userID)

	// 更新被提及成员的未读 @ 计数
	if len(mentionUserIDs) > 0 {
		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id IN ?", convID, mentionUserIDs).
			UpdateColumn("unread_at_mention_count", gorm.Expr("unread_at_mention_count + 1"))
	}

	wsMsg := WSMessage{
		Type: "new_message",
		Data: map[string]interface{}{
			"id":                msg.ID,
			"conversation_id":   msg.ConversationID,
			"sender_id":         msg.SenderID,
			"type":              msg.Type,
			"content":           msg.Content,
			"quoted_message_id": msg.QuotedMessageID,
			"is_recalled":       msg.IsRecalled,
			"is_read":           msg.IsRead,
			"is_avatar_reply":   msg.Origin == "avatar",
			"is_ai_message":     msg.Sender.Type == "bot" || msg.Sender.Type == "system",
			"recalled_at":       msg.RecalledAt,
			"created_at":        msg.CreatedAt,
			"sender":            msg.Sender,
			"quoted_message":    msg.QuotedMessage,
			"mention_user_ids":  mentionUserIDs,
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)
	c.hub.SendToConversationAsync(convID, c.userID, jsonMsg)

	if c.hub.OnMessageSent != nil && !mention.IsAllMentioned(mentions) {
		utils.SafeGo(func() { c.hub.OnMessageSent(&msg, mentionUserIDs) })
	}
}

func handleReadMessage(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)

	// 统一调用外部注册的 MarkAsRead 处理
	if c.hub.HandleReadMessage != nil {
		c.hub.HandleReadMessage(convID, c.userID)
		return
	}

	// 降级：无回调时使用原有逻辑
	fallbackHandleReadMessage(c, convID)
}

func fallbackHandleReadMessage(c *Client, convID uint) {
	db := c.hub.db

	// 检查会话成员资格
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 用 per-user 的 message_read_receipts 表判断"该用户尚未读过的消息"，
	// 不能用 messages.is_read 全局字段（详见 MessageService.MarkAsRead 注释）
	now := time.Now()

	var unreadMsgIDs []uint
	db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ?", convID, c.userID).
		Where("id NOT IN (?)", db.Model(&model.MessageReadReceipt{}).Select("message_id").Where("user_id = ?", c.userID)).
		Pluck("id", &unreadMsgIDs)

	// 即使没有未读，也要清零 unread_count 和推进 last_read_at
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, c.userID).
		Updates(map[string]interface{}{
			"unread_count":            0,
			"unread_at_mention_count": 0,
			"last_read_at":            now,
		})

	if len(unreadMsgIDs) == 0 {
		return
	}

	if database.D.Type() == "mysql" {
		db.Exec(`
			INSERT IGNORE INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ?
			  AND id NOT IN (SELECT message_id FROM message_read_receipts WHERE user_id = ?)
		`, convID, c.userID, now, convID, c.userID, c.userID)
	} else {
		db.Exec(`
			INSERT INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ?
			  AND id NOT IN (SELECT message_id FROM message_read_receipts WHERE user_id = ?)
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, convID, c.userID, now, convID, c.userID, c.userID)
	}

	// messages.is_read 仅作为缓存标志
	db.Model(&model.Message{}).
		Where("id IN ? AND is_read = false", unreadMsgIDs).
		UpdateColumn("is_read", true)

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return
	}

	readMsg := WSMessage{
		Type: "message_read",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"message_ids":     unreadMsgIDs,
			"timestamp":       now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(readMsg)

	if conv.Type == "single" {
		var otherMember model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).First(&otherMember)
		c.hub.SendToUser(otherMember.UserID, jsonMsg)
	} else if conv.Type == "group" {
		var members []model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, c.userID).Find(&members)

		for _, member := range members {
			c.hub.SendToUser(member.UserID, jsonMsg)
		}
	}
}
