package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/ws"

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

	rtSvc := di.GlobalContainer.RealtimeService

	session := model.RealtimeSession{
		ID:             uuid.New().String(),
		Type:           req.Type,
		InitiatorID:    userID,
		ConversationID: req.ConversationID,
		Status:         "pending",
		Metadata:       req.Metadata,
	}

	if err := rtSvc.CreateSession(&session); err != nil {
		response.InternalServerError(c, "创建会话失败")
		return
	}

	now := time.Now()
	participant := model.RealtimeParticipant{
		ID:          uuid.New().String(),
		SessionID:   session.ID,
		UserID:      userID,
		Role:        "initiator",
		Status:      "approved",
		RequestedAt: time.Now(),
		ApprovedAt:  &now,
	}

	if err := rtSvc.CreateParticipant(&participant); err != nil {
		response.InternalServerError(c, "创建参与者失败")
		return
	}

	rtSvc.GetSession(session.ID)

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

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if _, err := rtSvc.GetParticipant(sessionID, userID); err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	response.Success(c, session)
}

func GetActiveSessions(c *gin.Context) {
	userID := c.GetUint("user_id")

	rtSvc := di.GlobalContainer.RealtimeService
	sessions, err := rtSvc.GetActiveSessions(userID)
	if err != nil {
		response.InternalServerError(c, "获取会话列表失败")
		return
	}

	response.Success(c, sessions)
}

func EndSession(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
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

	if err := rtSvc.UpdateSession(session); err != nil {
		response.InternalServerError(c, "结束会话失败")
		return
	}

	rtSvc.UpdateParticipantsStatus(sessionID, "left", now)

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

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.Status == "ended" {
		response.BadRequest(c, "会话已结束")
		return
	}

	if _, err := rtSvc.GetParticipant(sessionID, userID); err == nil {
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

	if err := rtSvc.CreateParticipant(&participant); err != nil {
		response.InternalServerError(c, "申请失败")
		return
	}

	rtSvc.GetParticipantByID(participant.ID)

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

func ApproveJoin(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.InitiatorID != userID {
		response.Forbidden(c, "只有发起者可以审批")
		return
	}

	participant, err := rtSvc.GetParticipant(sessionID, uint(targetUserID))
	if err != nil {
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

	if err := rtSvc.UpdateParticipant(participant); err != nil {
		response.InternalServerError(c, "审批失败")
		return
	}

	if session.Status == "pending" {
		session.Status = "active"
		session.StartedAt = &now
		rtSvc.UpdateSession(session)
	}

	rtSvc.GetParticipantByID(participant.ID)

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

func RejectJoin(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")
	targetUserIDStr := c.Param("user_id")

	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if session.InitiatorID != userID {
		response.Forbidden(c, "只有发起者可以拒绝")
		return
	}

	participant, err := rtSvc.GetParticipant(sessionID, uint(targetUserID))
	if err != nil {
		response.NotFound(c, "申请不存在")
		return
	}

	if participant.Status != "pending" {
		response.BadRequest(c, "该申请已处理")
		return
	}

	participant.Status = "rejected"

	if err := rtSvc.UpdateParticipant(participant); err != nil {
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

func LeaveSession(c *gin.Context) {
	userID := c.GetUint("user_id")
	sessionID := c.Param("id")

	rtSvc := di.GlobalContainer.RealtimeService

	session, err := rtSvc.GetSession(sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	participant, err := rtSvc.GetParticipant(sessionID, userID)
	if err != nil {
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

	if err := rtSvc.UpdateParticipant(participant); err != nil {
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

	rtSvc := di.GlobalContainer.RealtimeService

	participants, err := rtSvc.GetPendingRequests(userID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	type PendingRequest struct {
		ID              string    `json:"id"`
		SessionID       string    `json:"session_id"`
		SessionType     string    `json:"session_type"`
		ConversationID  uint      `json:"conversation_id"`
		InitiatorID     uint      `json:"initiator_id"`
		InitiatorName   string    `json:"initiator_name"`
		RequestedAt     time.Time `json:"requested_at"`
	}

	var requests []PendingRequest
	for _, p := range participants {
		if p.Session.Status == "ended" {
			rtSvc.UpdateParticipant(&model.RealtimeParticipant{
				ID:     p.ID,
				Status: "left",
				LeftAt: &[]time.Time{time.Now()}[0],
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

func RespondToShareRequest(c *gin.Context) {
	userID := c.GetUint("user_id")
	participantID := c.Param("id")

	var req struct {
		Action string `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		response.BadRequest(c, "无效的操作")
		return
	}

	rtSvc := di.GlobalContainer.RealtimeService

	participant, err := rtSvc.GetParticipantWithSession(participantID, userID)
	if err != nil {
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

		if participant.Session.Status == "pending" {
			participant.Session.Status = "active"
			participant.Session.StartedAt = &now
			rtSvc.UpdateSession(&participant.Session)
		}

		msg, _ := json.Marshal(ws.WSMessage{
			Type: "screen-share-accepted",
			Data: map[string]interface{}{
				"session_id":      participant.SessionID,
				"user_id":         userID,
				"conversation_id": participant.Session.ConversationID,
			},
		})
		ws.GlobalHub.SendToUser(participant.Session.InitiatorID, msg)
	} else {
		participant.Status = "rejected"

		msg, _ := json.Marshal(ws.WSMessage{
			Type: "screen-share-rejected",
			Data: map[string]interface{}{
				"session_id":      participant.SessionID,
				"user_id":         userID,
				"conversation_id": participant.Session.ConversationID,
			},
		})
		ws.GlobalHub.SendToUser(participant.Session.InitiatorID, msg)
	}

	if err := rtSvc.UpdateParticipant(participant); err != nil {
		response.InternalServerError(c, "操作失败")
		return
	}

	response.Success(c, gin.H{"message": "操作成功"})
}
