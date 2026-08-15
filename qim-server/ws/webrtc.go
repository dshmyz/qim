package ws

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

func handleWebRTCSignal(c *Client, data interface{}, signalType string) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		return
	}
	logger.WithModule("WS").Debug("收到信令消息", "signalType", signalType, "data", msgData)
	var targetUserID uint

	// 尝试将 target_user_id 转换为 float64 (数字类型)
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		// 尝试将 target_user_id 转换为 string 类型，然后转换为 uint
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	// 构建转发的信令消息
	// ICE 候选者使用 candidate 字段，其他信令使用 signal 字段
	signalData := msgData["signal"]
	if signalType == "webrtc.ice-candidate" {
		signalData = msgData["candidate"]
	}

	// 构建转发的数据，包含原始消息中的所有字段
	forwardData := map[string]interface{}{
		"from_user_id": c.userID,
		"signal":       signalData,
	}

	// 转发原始消息中的其他字段
	// 优先使用新的 media_type 字段
	if mediaType, ok := msgData["media_type"]; ok {
		forwardData["media_type"] = mediaType
	} else {
		// 兼容旧的 share_type 和 call_type 字段
		// 如果存在 share_type 或 call_type，同时设置 media_type
		if shareType, ok := msgData["share_type"]; ok {
			forwardData["share_type"] = shareType
			forwardData["media_type"] = shareType // 同时设置 media_type
		}
		if callType, ok := msgData["call_type"]; ok {
			forwardData["call_type"] = callType
			forwardData["media_type"] = callType // 同时设置 media_type
		}
	}

	// 如果有 media_type，也转发原始的 share_type 和 call_type（向后兼容）
	if mediaType, ok := forwardData["media_type"]; ok {
		// 如果是新格式（只有 media_type），也设置 share_type 或 call_type
		if _, hasShareType := forwardData["share_type"]; !hasShareType {
			if mediaTypeStr, ok := mediaType.(string); ok {
				if mediaTypeStr == "screen" {
					forwardData["share_type"] = mediaTypeStr
				} else if mediaTypeStr == "video" || mediaTypeStr == "audio" {
					forwardData["call_type"] = mediaTypeStr
				}
			}
		}
	}

	signalMsg := WSMessage{
		Type: signalType,
		Data: forwardData,
	}

	jsonMsg, _ := json.Marshal(signalMsg)

	// 发送给目标用户
	c.hub.SendToUser(targetUserID, jsonMsg)
	logger.WithModule("WS").Debug("转发WebRTC信令", "signalType", signalType, "fromUserID", c.userID, "targetUserID", targetUserID)
}

func handleCallInvite(c *Client, data interface{}) {
	db := c.hub.db
	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("通话邀请数据格式错误", "data", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			logger.WithModule("WS").Warn("解析target_user_id失败", "value", targetUserIDStr)
			return
		}
	} else {
		logger.WithModule("WS").Warn("通话邀请缺少target_user_id")
		return
	}

	callType, _ := msgData["call_type"].(string)
	signal := msgData["signal"]

	logger.WithModule("WS").Info("用户发起通话邀请", "fromUserID", c.userID, "targetUserID", targetUserID, "callType", callType)

	// 查询发起者昵称用于来电通知展示
	var callerNickname string
	if err := db.Model(&model.User{}).Where("id = ?", c.userID).Select("nickname").First(&callerNickname).Error; err != nil {
		logger.WithModule("WS").Warn("查询发起者昵称失败，使用默认值", "error", err)
		callerNickname = "对方"
	}

	// 转发通话邀请给目标用户
	callMsg := WSMessage{
		Type: "call.start",
		Data: map[string]interface{}{
			"from_user_id":   c.userID,
			"from_user_name": callerNickname,
			"call_type":      callType,
			"signal":         signal,
			"timestamp":      time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话接听

func handleCallAccept(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("通话接听数据格式错误", "data", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	signal := msgData["signal"]

	logger.WithModule("WS").Info("用户接听通话", "userID", c.userID, "targetUserID", targetUserID)

	// 转发接听消息给发起方
	callMsg := WSMessage{
		Type: "call.answer",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"signal":       signal,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话拒绝

func handleCallReject(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("通话拒绝数据格式错误", "data", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	logger.WithModule("WS").Info("用户拒绝通话", "userID", c.userID, "targetUserID", targetUserID)

	// 转发拒绝消息给发起方
	callMsg := WSMessage{
		Type: "call.reject",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// 处理视频通话结束

func handleCallEnd(c *Client, data interface{}) {
	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("WS").Warn("通话结束数据格式错误", "data", data)
		return
	}

	var targetUserID uint
	if targetUserIDFloat, ok := msgData["target_user_id"].(float64); ok {
		targetUserID = uint(targetUserIDFloat)
	} else if targetUserIDStr, ok := msgData["target_user_id"].(string); ok {
		if id, err := strconv.ParseUint(targetUserIDStr, 10, 32); err == nil {
			targetUserID = uint(id)
		} else {
			return
		}
	} else {
		return
	}

	logger.WithModule("WS").Info("用户结束通话", "userID", c.userID, "targetUserID", targetUserID)

	// 转发通话结束消息给对方
	callMsg := WSMessage{
		Type: "call.end",
		Data: map[string]interface{}{
			"from_user_id": c.userID,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(callMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)
}

// StatusDebouncer 状态变更防抖器
