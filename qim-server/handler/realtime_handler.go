package handler

import (
	"encoding/json"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateSession 创建实时会话
func CreateSession(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Type           string `json:"type" binding:"required"`
		ConversationID uint   `json:"conversation_id" binding:"required"`
		Metadata       string `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	session := model.RealtimeSession{
		ID:             uuid.New().String(),
		Type:           req.Type,
		InitiatorID:    userID,
		ConversationID: req.ConversationID,
		Status:         "pending",
		Metadata:       req.Metadata,
	}

	if err := db.Create(&session).Error; err != nil {
		response.InternalServerError(c, "创建会话失败")
		return
	}

	participant := model.RealtimeParticipant{
		ID:          uuid.New().String(),
		SessionID:   session.ID,
		UserID:      userID,
		Role:        "initiator",
		Status:      "approved",
		RequestedAt: time.Now(),
		ApprovedAt:  &[]time.Time{time.Now()}[0],
	}

	if err := db.Create(&participant).Error; err != nil {
		response.InternalServerError(c, "创建参与者失败")
		return
	}

	db.Preload("Initiator").Preload("Participants").Preload("Participants.User").First(&session, session.ID)

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:session:created",
		Data: map[string]interface{}{
			"session": session,
		},
	})
	ws.GlobalHub.SendToConversation(session.ConversationID, 0, msg)

	response.Success(c, session)
}

// GetSession 获取会话详情
func GetSession(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.Preload("Initiator").Preload("Participants").Preload("Participants.User").
		First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error; err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	response.Success(c, session)
}

// GetActiveSessions 获取活跃会话列表
func GetActiveSessions(c *gin.Context) {
	userID := c.GetUint("user_id")

	db := database.GetDB()

	var participants []model.RealtimeParticipant
	db.Where("user_id = ? AND status IN ?", userID, []string{"approved", "joined"}).
		Find(&participants)

	var sessions []model.RealtimeSession
	for _, p := range participants {
		var session model.RealtimeSession
		if err := db.Preload("Initiator").Preload("Participants").Preload("Participants.User").
			First(&session, "id = ?", p.SessionID).Error; err == nil {
			if session.Status == "active" || session.Status == "pending" {
				sessions = append(sessions, session)
			}
		}
	}

	response.Success(c, sessions)
}

// EndSession 结束会话
func EndSession(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.InitiatorID != userID {
		response.Forbidden(c, "只有发起者可以结束会话")
		return
	}

	now := time.Now()
	session.Status = "ended"
	session.EndedAt = &now

	if err := db.Save(&session).Error; err != nil {
		response.InternalServerError(c, "结束会话失败")
		return
	}

	db.Model(&model.RealtimeParticipant{}).
		Where("session_id = ? AND status IN ?", sessionID, []string{"approved", "joined", "pending"}).
		Updates(map[string]interface{}{
			"status":  "left",
			"left_at": now,
		})

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:session:ended",
		Data: map[string]interface{}{
			"session": session,
		},
	})
	ws.GlobalHub.SendToConversation(session.ConversationID, 0, msg)

	response.Success(c, session)
}

// RequestJoin 申请加入会话
func RequestJoin(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.Status == "ended" {
		response.BadRequest(c, "会话已结束")
		return
	}

	var existingParticipant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&existingParticipant).Error; err == nil {
		response.Conflict(c, "已申请或已加入")
		return
	}

	participant := model.RealtimeParticipant{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		UserID:      userID,
		Role:        "viewer",
		Status:      "pending",
		RequestedAt: time.Now(),
	}

	if err := db.Create(&participant).Error; err != nil {
		response.InternalServerError(c, "申请失败")
		return
	}

	db.Preload("User").First(&participant, participant.ID)

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:join:requested",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"participant": participant,
		},
	})
	ws.GlobalHub.SendToUser(session.InitiatorID, msg)

	response.Success(c, participant)
}

// ApproveJoin 审批加入请求
func ApproveJoin(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")
	targetUserID := c.Param("user_id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.InitiatorID != userID {
		response.Forbidden(c, "只有发起者可以审批")
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}

	if participant.Status != "pending" {
		response.BadRequest(c, "该申请已处理")
		return
	}

	now := time.Now()
	participant.Status = "approved"
	participant.ApprovedAt = &now

	if err := db.Save(&participant).Error; err != nil {
		response.InternalServerError(c, "审批失败")
		return
	}

	if session.Status == "pending" {
		session.Status = "active"
		session.StartedAt = &now
		db.Save(&session)
	}

	db.Preload("User").First(&participant, participant.ID)

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:join:approved",
		Data: map[string]interface{}{
			"session_id":  sessionID,
			"participant": participant,
		},
	})
	ws.GlobalHub.SendToUser(participant.UserID, msg)

	response.Success(c, participant)
}

// RejectJoin 拒绝加入请求
func RejectJoin(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")
	targetUserID := c.Param("user_id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.InitiatorID != userID {
		response.Forbidden(c, "只有发起者可以拒绝")
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, targetUserID).First(&participant).Error; err != nil {
		response.NotFound(c, "申请不存在")
		return
	}

	if participant.Status != "pending" {
		response.BadRequest(c, "该申请已处理")
		return
	}

	participant.Status = "rejected"

	if err := db.Save(&participant).Error; err != nil {
		response.InternalServerError(c, "拒绝失败")
		return
	}

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:join:rejected",
		Data: map[string]interface{}{
			"session_id": sessionID,
		},
	})
	ws.GlobalHub.SendToUser(participant.UserID, msg)

	response.Success(c, gin.H{"message": "已拒绝"})
}

// LeaveSession 离开会话
func LeaveSession(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	db := database.GetDB()

	var session model.RealtimeSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	var participant model.RealtimeParticipant
	if err := db.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&participant).Error; err != nil {
		response.NotFound(c, "未加入会话")
		return
	}

	if participant.Status == "left" {
		response.BadRequest(c, "已离开会话")
		return
	}

	now := time.Now()
	participant.Status = "left"
	participant.LeftAt = &now

	if err := db.Save(&participant).Error; err != nil {
		response.InternalServerError(c, "离开失败")
		return
	}

	msg, _ := json.Marshal(ws.WSMessage{
		Type: "realtime:participant:left",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"user_id":    userID,
			"left_at":    now,
		},
	})
	ws.GlobalHub.SendToConversation(session.ConversationID, 0, msg)

	response.Success(c, gin.H{"message": "已离开会话"})
}

// GetPendingRequests 获取待处理的共享请求（用户登录后调用）
func GetPendingRequests(c *gin.Context) {
	userID := c.GetUint("user_id")

	db := database.GetDB()

	// 查询该用户所有待处理的参与请求
	var participants []model.RealtimeParticipant
	if err := db.Where("user_id = ? AND status = ?", userID, "pending").
		Preload("User").
		Preload("Session").
		Preload("Session.Initiator").
		Find(&participants).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	// 构建返回数据
	type PendingRequest struct {
		ID              string                 `json:"id"`
		SessionID       string                 `json:"session_id"`
		SessionType     string                 `json:"session_type"`
		ConversationID  uint                   `json:"conversation_id"`
		InitiatorID     uint                   `json:"initiator_id"`
		InitiatorName   string                 `json:"initiator_name"`
		RequestedAt     time.Time              `json:"requested_at"`
	}

	var requests []PendingRequest
	for _, p := range participants {
		// 检查会话是否还有效（未结束）
		if p.Session.Status == "ended" {
			// 标记为已离开
			db.Model(&p).Updates(map[string]interface{}{
				"status": "left",
				"left_at": time.Now(),
			})
			continue
		}

		requests = append(requests, PendingRequest{
			ID:             p.ID,
			SessionID:      p.SessionID,
			SessionType:    p.Session.Type,
			ConversationID: p.Session.ConversationID,
			InitiatorID:    p.Session.InitiatorID,
			InitiatorName:  p.Session.Initiator.Nickname,
			RequestedAt:    p.RequestedAt,
		})
	}

	response.Success(c, requests)
}

// RespondToShareRequest 响应共享请求（接受/拒绝）
func RespondToShareRequest(c *gin.Context) {
	userID := c.GetUint("user_id")
	participantID := c.Param("id")

	var req struct {
		Action string `json:"action" binding:"required"` // accept 或 reject
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		response.BadRequest(c, "无效的操作")
		return
	}

	db := database.GetDB()

	var participant model.RealtimeParticipant
	if err := db.Where("id = ? AND user_id = ?", participantID, userID).
		Preload("Session").
		First(&participant).Error; err != nil {
		response.NotFound(c, "请求不存在")
		return
	}

	if participant.Status != "pending" {
		response.BadRequest(c, "该请求已处理")
		return
	}

	now := time.Now()

	if req.Action == "accept" {
		participant.Status = "approved"
		participant.ApprovedAt = &now

		// 激活会话
		if participant.Session.Status == "pending" {
			participant.Session.Status = "active"
			participant.Session.StartedAt = &now
			db.Save(&participant.Session)
		}

		msg, _ := json.Marshal(ws.WSMessage{
			Type: "screen-share-accepted",
			Data: map[string]interface{}{
				"session_id":    participant.SessionID,
				"user_id":       userID,
				"conversation_id": participant.Session.ConversationID,
			},
		})
		ws.GlobalHub.SendToUser(participant.Session.InitiatorID, msg)
	} else {
		participant.Status = "rejected"

		msg, _ := json.Marshal(ws.WSMessage{
			Type: "screen-share-rejected",
			Data: map[string]interface{}{
				"session_id":    participant.SessionID,
				"user_id":       userID,
				"conversation_id": participant.Session.ConversationID,
			},
		})
		ws.GlobalHub.SendToUser(participant.Session.InitiatorID, msg)
	}

	if err := db.Save(&participant).Error; err != nil {
		response.InternalServerError(c, "操作失败")
		return
	}

	response.Success(c, gin.H{"message": "操作成功"})
}
