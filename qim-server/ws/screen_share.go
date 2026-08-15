package ws

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

func handleScreenShareStart(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 支持两种命名格式：下划线和驼峰
	var userIdFloat float64
	if val, ok := msgData["user_id"].(float64); ok {
		userIdFloat = val
	} else if val, ok := msgData["userId"].(float64); ok {
		userIdFloat = val
	} else {
		// 如果没有提供userId，使用当前用户ID
		userIdFloat = float64(c.userID)
	}
	userId := uint(userIdFloat)

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 查询分享者昵称用于屏幕共享通知展示
	var sharerNickname string
	if err := db.Model(&model.User{}).Where("id = ?", c.userID).Select("nickname").First(&sharerNickname).Error; err != nil {
		logger.WithModule("WS").Warn("查询屏幕共享者昵称失败，使用默认值", "error", err)
		sharerNickname = "对方"
	}

	// 构建屏幕共享开始消息
	wsMsg := WSMessage{
		Type: "screen-share.start",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         userId,
			"from_user_id":    c.userID,
			"from_user_name":  sharerNickname,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversationAsync(convID, c.userID, jsonMsg)
	logger.WithModule("WS").Info("用户开始屏幕共享", "userID", c.userID, "convID", convID)
}

// 处理屏幕共享停止

func handleScreenShareStop(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 构建屏幕共享停止消息
	wsMsg := WSMessage{
		Type: "screen-share.stop",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	c.hub.SendToConversationAsync(convID, c.userID, jsonMsg)
	logger.WithModule("WS").Info("用户停止屏幕共享", "userID", c.userID, "convID", convID)
}

// 处理屏幕共享数据

func handleScreenShareData(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("屏幕共享数据格式错误", "data", data)
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convID uint
	var found bool

	// 尝试从 conversation_id 获取
	if val, ok := msgData["conversation_id"]; ok {
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	// 尝试从 conversationId 获取
	if !found && msgData["conversationId"] != nil {
		val := msgData["conversationId"]
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	if !found {
		logger.WithModule("WS").Warn("屏幕共享数据缺少会话ID", "data", msgData)
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}

	// 构建屏幕共享数据消息。转发时沿用客户端发送的 dot 类型 "screen-share.data"，
	// 保证接收方按同一事件名注册的 handler 能收到（此前误用下划线 "screen-share-data"
	// 导致与前端注册名不一致、中继帧被静默丢弃）。
	wsMsg := WSMessage{
		Type: "screen-share.data",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"data":            msgData["data"],
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员
	logger.WithModule("WS").Debug("准备推送屏幕共享请求", "convID", convID, "senderID", c.userID)
	c.hub.SendToConversationAsync(convID, c.userID, jsonMsg)
	logger.WithModule("WS").Info("用户屏幕共享数据转发", "userID", c.userID, "convID", convID)
}

// 处理屏幕共享请求（支持离线用户）

func handleScreenShareRequest(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("屏幕共享请求数据格式错误", "data", data)
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convID uint
	var found bool

	// 尝试从 conversation_id 获取
	if val, ok := msgData["conversation_id"]; ok {
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	// 尝试从 conversationId 获取
	if !found && msgData["conversationId"] != nil {
		val := msgData["conversationId"]
		switch v := val.(type) {
		case float64:
			convID = uint(v)
			found = true
		case int:
			convID = uint(v)
			found = true
		case int64:
			convID = uint(v)
			found = true
		case string:
			if id, err := strconv.Atoi(v); err == nil {
				convID = uint(id)
				found = true
			}
		}
	}

	if !found {
		logger.WithModule("WS").Warn("屏幕共享请求缺少会话ID", "data", msgData)
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		logger.WithModule("WS").Warn("用户不是会话成员", "userID", c.userID, "convID", convID)
		return
	}

	// 查询发送者昵称
	var senderNickname string
	if err := db.Model(&model.User{}).Where("id = ?", c.userID).Select("nickname").First(&senderNickname).Error; err != nil {
		logger.WithModule("WS").Warn("查询用户昵称失败，使用默认值", "error", err)
		senderNickname = "未知用户"
	}

	// 构建屏幕共享请求消息
	wsMsg := WSMessage{
		Type: "screen-share.request",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         c.userID,
			"from_user_id":    c.userID,
			"from_user_name":  senderNickname,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)

	// 推送给会话其他成员（复用原有的 SendToConversation 逻辑）
	logger.WithModule("WS").Debug("准备推送屏幕共享请求", "convID", convID, "senderID", c.userID)
	c.hub.SendToConversationAsync(convID, c.userID, jsonMsg)
	logger.WithModule("WS").Info("用户请求屏幕共享", "userID", c.userID, "convID", convID)
}

// 处理屏幕共享响应

func handleScreenShareResponse(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	// 支持两种命名格式：下划线和驼峰
	var convIDFloat float64
	if val, ok := msgData["conversation_id"].(float64); ok {
		convIDFloat = val
	} else if val, ok := msgData["conversationId"].(float64); ok {
		convIDFloat = val
	} else {
		return
	}
	convID := uint(convIDFloat)

	// 获取请求者ID
	var requesterIDFloat float64
	if val, ok := msgData["requester_id"].(float64); ok {
		requesterIDFloat = val
	} else if val, ok := msgData["requesterId"].(float64); ok {
		requesterIDFloat = val
	} else {
		return
	}
	requesterID := uint(requesterIDFloat)

	// 获取响应状态
	status, ok := msgData["status"].(string)
	if !ok {
		return
	}

	// 验证是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		return
	}
	logger.WithModule("WS").Info("用户响应屏幕共享请求", "userID", c.userID, "convID", convID, "requesterID", requesterID, "status", status)
	if status == "accepted" {
		// 向请求者发送接受消息
		acceptMsg := WSMessage{
			Type: "screen-share.accepted",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		acceptJson, _ := json.Marshal(acceptMsg)
		c.hub.SendToUser(requesterID, acceptJson)

		// 向响应者发送开始消息
		var requesterName string
		if err := db.Model(&model.User{}).Where("id = ?", requesterID).Select("nickname").First(&requesterName).Error; err != nil {
			logger.WithModule("WS").Warn("查询请求者昵称失败，使用默认值", "error", err)
			requesterName = "对方"
		}
		startMsg := WSMessage{
			Type: "screen-share.start",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         requesterID,
				"from_user_id":    requesterID,
				"from_user_name":  requesterName,
				"timestamp":       time.Now().Unix(),
			},
		}
		startJson, _ := json.Marshal(startMsg)
		c.hub.SendToUser(c.userID, startJson)
	} else if status == "rejected" {
		// 向请求者发送拒绝消息
		rejectMsg := WSMessage{
			Type: "screen-share.rejected",
			Data: map[string]interface{}{
				"conversation_id": convID,
				"user_id":         c.userID,
				"timestamp":       time.Now().Unix(),
			},
		}
		rejectJson, _ := json.Marshal(rejectMsg)
		c.hub.SendToUser(requesterID, rejectJson)

		logger.WithModule("WS").Info("用户拒绝屏幕共享请求", "userID", c.userID, "convID", convID)
	}
}

// 处理视频通话邀请
