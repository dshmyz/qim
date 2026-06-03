package ws

import (
	"encoding/json"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HandleRealtimeSessionCreate 处理创建实时会话
func HandleRealtimeSessionCreate(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("创建实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionType, _ := msgData["type"].(string)
	convIDFloat, _ := msgData["conversation_id"].(float64)
	convID := uint(convIDFloat)

	if sessionType == "" || convID == 0 {
		logger.WithModule("Realtime").Error("创建实时会话缺少必要参数", "type", sessionType, "conversation_id", convID)
		return
	}

	// 验证用户是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, c.userID).First(&member).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是会话成员", "userID", c.userID, "conversationID", convID)
		return
	}

	// 检查是否已存在活跃的实时会话
	var existingSession model.RealtimeSession
	err := db.Where("conversation_id = ? AND status IN ?", convID, []string{"pending", "active"}).First(&existingSession).Error
	if err == nil {
		// 已存在活跃会话，返回错误
		errorMsg := WSMessage{
			Type: "realtime:session:error",
			Data: map[string]interface{}{
				"error":        "session_already_exists",
				"message":      "该会话已存在活跃的实时会话",
				"session_id":   existingSession.ID,
				"session_type": existingSession.Type,
			},
		}
		jsonMsg, _ := json.Marshal(errorMsg)
		c.hub.SendToUser(c.userID, jsonMsg)
		return
	}

	// 创建实时会话
	sessionID := uuid.New().String()
	now := time.Now()
	session := model.RealtimeSession{
		ID:             sessionID,
		Type:           sessionType,
		InitiatorID:    c.userID,
		ConversationID: convID,
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := db.Create(&session).Error; err != nil {
		logger.WithModule("Realtime").Error("创建实时会话失败", "error", err)
		return
	}

	// 创建发起者参与者记录
	participantID := uuid.New().String()
	participant := model.RealtimeParticipant{
		ID:          participantID,
		SessionID:   sessionID,
		UserID:      c.userID,
		Role:        "initiator",
		Status:      "approved",
		RequestedAt: now,
		ApprovedAt:  &now,
	}

	if err := db.Create(&participant).Error; err != nil {
		logger.WithModule("Realtime").Error("创建参与者记录失败", "error", err)
		return
	}

	// 预加载用户信息
	db.Model(&participant).Association("User").Find(&participant.User)
	db.Model(&session).Association("Initiator").Find(&session.Initiator)

	// 通知会话成员
	wsMsg := WSMessage{
		Type: "realtime:session:created",
		Data: map[string]interface{}{
			"session":     session,
			"participant": participant,
			"timestamp":   now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(wsMsg)
	c.hub.SendToConversation(convID, c.userID, jsonMsg)

	// 同时发送给创建者
	creatorMsg := WSMessage{
		Type: "realtime:session:created",
		Data: map[string]interface{}{
			"session":     session,
			"participant": participant,
			"timestamp":   now.Unix(),
		},
	}
	creatorJsonMsg, _ := json.Marshal(creatorMsg)
	c.hub.SendToUser(c.userID, creatorJsonMsg)

	logger.WithModule("Realtime").Info("用户创建实时会话", "userID", c.userID, "sessionID", sessionID, "type", sessionType, "conversationID", convID)
}

// HandleRealtimeJoinRequest 处理申请加入实时会话
func HandleRealtimeJoinRequest(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("申请加入实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	if sessionID == "" {
		logger.WithModule("Realtime").Error("申请加入实时会话缺少 session_id")
		return
	}

	// 查询实时会话
	var session model.RealtimeSession
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		logger.WithModule("Realtime").Warn("实时会话不存在", "sessionID", sessionID)
		return
	}

	// 检查会话状态
	if session.Status != "pending" && session.Status != "active" {
		errorMsg := WSMessage{
			Type: "realtime:join:error",
			Data: map[string]interface{}{
				"error":   "session_not_active",
				"message": "实时会话已结束",
			},
		}
		jsonMsg, _ := json.Marshal(errorMsg)
		c.hub.SendToUser(c.userID, jsonMsg)
		return
	}

	// 检查用户是否为会话成员
	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", session.ConversationID, c.userID).First(&member).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是会话成员", "userID", c.userID, "conversationID", session.ConversationID)
		return
	}

	// 检查是否已经是参与者
	var existingParticipant model.RealtimeParticipant
	err := db.Where("session_id = ? AND user_id = ?", sessionID, c.userID).First(&existingParticipant).Error
	if err == nil {
		// 已存在参与者记录
		if existingParticipant.Status == "approved" || existingParticipant.Status == "joined" {
			// 已经是参与者，直接通知
			approvedMsg := WSMessage{
				Type: "realtime:join:approved",
				Data: map[string]interface{}{
					"session":     session,
					"participant": existingParticipant,
					"timestamp":   time.Now().Unix(),
				},
			}
			jsonMsg, _ := json.Marshal(approvedMsg)
			c.hub.SendToUser(c.userID, jsonMsg)
			return
		}
		// 更新现有记录
		now := time.Now()
		db.Model(&existingParticipant).Updates(map[string]interface{}{
			"status":       "pending",
			"requested_at": now,
		})
		existingParticipant.Status = "pending"
		existingParticipant.RequestedAt = now
	} else {
		// 创建新的参与者申请
		now := time.Now()
		participantID := uuid.New().String()
		existingParticipant = model.RealtimeParticipant{
			ID:          participantID,
			SessionID:   sessionID,
			UserID:      c.userID,
			Role:        "viewer",
			Status:      "pending",
			RequestedAt: now,
		}

		if err := db.Create(&existingParticipant).Error; err != nil {
			logger.WithModule("Realtime").Error("创建参与者申请失败", "error", err)
			return
		}
	}

	// 预加载用户信息
	db.Model(&existingParticipant).Association("User").Find(&existingParticipant.User)

	// 通知发起者
	requestMsg := WSMessage{
		Type: "realtime:join:requested",
		Data: map[string]interface{}{
			"session":     session,
			"participant": existingParticipant,
			"timestamp":   time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(requestMsg)
	c.hub.SendToUser(session.InitiatorID, jsonMsg)

	logger.WithModule("Realtime").Info("用户申请加入实时会话", "userID", c.userID, "sessionID", sessionID)
}

// HandleRealtimeJoinApprove 处理批准加入实时会话
func HandleRealtimeJoinApprove(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("批准加入实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	userIDFloat, _ := msgData["user_id"].(float64)
	targetUserID := uint(userIDFloat)

	if sessionID == "" || targetUserID == 0 {
		logger.WithModule("Realtime").Error("批准加入实时会话缺少必要参数")
		return
	}

	// 查询实时会话
	var session model.RealtimeSession
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		logger.WithModule("Realtime").Warn("实时会话不存在", "sessionID", sessionID)
		return
	}

	// 验证是否为发起者
	if session.InitiatorID != c.userID {
		logger.WithModule("Realtime").Warn("用户不是发起者，无权批准", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 查询参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("参与者不存在", "sessionID", sessionID, "userID", targetUserID)
		return
	}

	// 更新参与者状态
	now := time.Now()
	if err := db.Model(&participant).Updates(map[string]interface{}{
		"status":      "approved",
		"approved_at": now,
	}).Error; err != nil {
		logger.WithModule("Realtime").Error("更新参与者状态失败", "error", err)
		return
	}

	participant.Status = "approved"
	participant.ApprovedAt = &now

	// 预加载用户信息
	db.Model(&participant).Association("User").Find(&participant.User)

	// 通知被批准的用户
	approvedMsg := WSMessage{
		Type: "realtime:join:approved",
		Data: map[string]interface{}{
			"session":     session,
			"participant": participant,
			"timestamp":   now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(approvedMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)

	// 通知其他参与者有新成员加入
	newMemberMsg := WSMessage{
		Type: "realtime:participant:joined",
		Data: map[string]interface{}{
			"session":     session,
			"participant": participant,
			"timestamp":   now.Unix(),
		},
	}
	newMemberJsonMsg, _ := json.Marshal(newMemberMsg)

	// 查询所有已批准的参与者
	var participants []model.RealtimeParticipant
	db.Where("session_id = ? AND status IN ? AND user_id != ?", sessionID, []string{"approved", "joined"}, targetUserID).Find(&participants)
	for _, p := range participants {
		c.hub.SendToUser(p.UserID, newMemberJsonMsg)
	}

	logger.WithModule("Realtime").Info("用户批准用户加入实时会话", "approverID", c.userID, "targetUserID", targetUserID, "sessionID", sessionID)
}

// HandleRealtimeJoinReject 处理拒绝加入实时会话
func HandleRealtimeJoinReject(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("拒绝加入实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	userIDFloat, _ := msgData["user_id"].(float64)
	targetUserID := uint(userIDFloat)

	if sessionID == "" || targetUserID == 0 {
		logger.WithModule("Realtime").Error("拒绝加入实时会话缺少必要参数")
		return
	}

	// 查询实时会话
	var session model.RealtimeSession
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		logger.WithModule("Realtime").Warn("实时会话不存在", "sessionID", sessionID)
		return
	}

	// 验证是否为发起者
	if session.InitiatorID != c.userID {
		logger.WithModule("Realtime").Warn("用户不是发起者，无权拒绝", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 查询参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("参与者不存在", "sessionID", sessionID, "userID", targetUserID)
		return
	}

	// 更新参与者状态
	now := time.Now()
	if err := db.Model(&participant).Updates(map[string]interface{}{
		"status": "rejected",
	}).Error; err != nil {
		logger.WithModule("Realtime").Error("更新参与者状态失败", "error", err)
		return
	}

	// 通知被拒绝的用户
	rejectedMsg := WSMessage{
		Type: "realtime:join:rejected",
		Data: map[string]interface{}{
			"session":   session,
			"timestamp": now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(rejectedMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)

	logger.WithModule("Realtime").Info("用户拒绝用户加入实时会话", "rejecterID", c.userID, "targetUserID", targetUserID, "sessionID", sessionID)
}

// HandleRealtimeLeave 处理离开实时会话
func HandleRealtimeLeave(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("离开实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	if sessionID == "" {
		logger.WithModule("Realtime").Error("离开实时会话缺少 session_id")
		return
	}

	// 查询实时会话
	var session model.RealtimeSession
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		logger.WithModule("Realtime").Warn("实时会话不存在", "sessionID", sessionID)
		return
	}

	// 查询参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, c.userID).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是实时会话的参与者", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 更新参与者状态
	now := time.Now()
	if err := db.Model(&participant).Updates(map[string]interface{}{
		"status":  "left",
		"left_at": now,
	}).Error; err != nil {
		logger.WithModule("Realtime").Error("更新参与者状态失败", "error", err)
		return
	}

	// 通知其他参与者
	leaveMsg := WSMessage{
		Type: "realtime:participant:left",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"user_id":    c.userID,
			"timestamp":  now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(leaveMsg)

	// 查询所有参与者
	var participants []model.RealtimeParticipant
	db.Where("session_id = ? AND status IN ? AND user_id != ?", sessionID, []string{"approved", "joined"}, c.userID).Find(&participants)
	for _, p := range participants {
		c.hub.SendToUser(p.UserID, jsonMsg)
	}

	// 如果是发起者离开，结束会话
	if session.InitiatorID == c.userID {
		handleEndSession(db, &session)
	}

	logger.WithModule("Realtime").Info("用户离开实时会话", "userID", c.userID, "sessionID", sessionID)
}

// HandleRealtimeSessionEnd 处理结束实时会话
func HandleRealtimeSessionEnd(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("结束实时会话数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	if sessionID == "" {
		logger.WithModule("Realtime").Error("结束实时会话缺少 session_id")
		return
	}

	// 查询实时会话
	var session model.RealtimeSession
	if err := db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		logger.WithModule("Realtime").Warn("实时会话不存在", "sessionID", sessionID)
		return
	}

	// 验证是否为发起者
	if session.InitiatorID != c.userID {
		logger.WithModule("Realtime").Warn("用户不是发起者，无权结束", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 结束会话
	handleEndSession(db, &session)

	logger.WithModule("Realtime").Info("用户结束实时会话", "userID", c.userID, "sessionID", sessionID)
}

// handleEndSession 内部函数：结束实时会话
func handleEndSession(db *gorm.DB, session *model.RealtimeSession) {
	// 更新会话状态
	now := time.Now()
	if err := db.Model(session).Updates(map[string]interface{}{
		"status":   "ended",
		"ended_at": now,
	}).Error; err != nil {
		logger.WithModule("Realtime").Error("更新会话状态失败", "error", err)
		return
	}

	// 更新所有参与者状态
	db.Model(&model.RealtimeParticipant{}).
		Where("session_id = ? AND status IN ?", session.ID, []string{"pending", "approved", "joined"}).
		Updates(map[string]interface{}{
			"status":  "left",
			"left_at": now,
		})

	// 通知所有参与者
	endMsg := WSMessage{
		Type: "realtime:session:ended",
		Data: map[string]interface{}{
			"session_id": session.ID,
			"timestamp":  now.Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(endMsg)

	var participants []model.RealtimeParticipant
	db.Where("session_id = ?", session.ID).Find(&participants)
	for _, p := range participants {
		GlobalHub.SendToUser(p.UserID, jsonMsg)
	}
}

// HandleRealtimeWebRTCOffer 处理 WebRTC offer
func HandleRealtimeWebRTCOffer(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("WebRTC offer 数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	targetUserIDFloat, _ := msgData["target_user_id"].(float64)
	targetUserID := uint(targetUserIDFloat)
	offer := msgData["offer"]

	if sessionID == "" || targetUserID == 0 || offer == nil {
		logger.WithModule("Realtime").Error("WebRTC offer 缺少必要参数")
		return
	}

	// 验证用户是否为会话参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ? AND status IN ?", sessionID, c.userID, []string{"approved", "joined"}).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是实时会话的有效参与者", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 转发 offer 给目标用户
	offerMsg := WSMessage{
		Type: "realtime:webrtc:offer",
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"from_user_id": c.userID,
			"offer":        offer,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(offerMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)

	logger.WithModule("Realtime").Info("转发 WebRTC offer", "fromUserID", c.userID, "toUserID", targetUserID, "sessionID", sessionID)
}

// HandleRealtimeWebRTCAnswer 处理 WebRTC answer
func HandleRealtimeWebRTCAnswer(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("WebRTC answer 数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	targetUserIDFloat, _ := msgData["target_user_id"].(float64)
	targetUserID := uint(targetUserIDFloat)
	answer := msgData["answer"]

	if sessionID == "" || targetUserID == 0 || answer == nil {
		logger.WithModule("Realtime").Error("WebRTC answer 缺少必要参数")
		return
	}

	// 验证用户是否为会话参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ? AND status IN ?", sessionID, c.userID, []string{"approved", "joined"}).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是实时会话的有效参与者", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 转发 answer 给目标用户
	answerMsg := WSMessage{
		Type: "realtime:webrtc:answer",
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"from_user_id": c.userID,
			"answer":       answer,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(answerMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)

	logger.WithModule("Realtime").Info("转发 WebRTC answer", "fromUserID", c.userID, "toUserID", targetUserID, "sessionID", sessionID)
}

// HandleRealtimeWebRTCIce 处理 WebRTC ICE candidate
func HandleRealtimeWebRTCIce(c *Client, data interface{}) {
	db := c.hub.db

	msgData, ok := data.(map[string]interface{})
	if !ok {
		logger.WithModule("Realtime").Error("WebRTC ICE 数据格式错误", "data", data)
		return
	}

	// 获取参数
	sessionID, _ := msgData["session_id"].(string)
	targetUserIDFloat, _ := msgData["target_user_id"].(float64)
	targetUserID := uint(targetUserIDFloat)
	candidate := msgData["candidate"]

	if sessionID == "" || targetUserID == 0 || candidate == nil {
		logger.WithModule("Realtime").Error("WebRTC ICE 缺少必要参数")
		return
	}

	// 验证用户是否为会话参与者
	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ? AND status IN ?", sessionID, c.userID, []string{"approved", "joined"}).First(&participant).Error; err != nil {
		logger.WithModule("Realtime").Warn("用户不是实时会话的有效参与者", "userID", c.userID, "sessionID", sessionID)
		return
	}

	// 转发 ICE candidate 给目标用户
	iceMsg := WSMessage{
		Type: "realtime:webrtc:ice",
		Data: map[string]interface{}{
			"session_id":   sessionID,
			"from_user_id": c.userID,
			"candidate":    candidate,
			"timestamp":    time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(iceMsg)
	c.hub.SendToUser(targetUserID, jsonMsg)

	logger.WithModule("Realtime").Info("转发 WebRTC ICE candidate", "fromUserID", c.userID, "toUserID", targetUserID, "sessionID", sessionID)
}
