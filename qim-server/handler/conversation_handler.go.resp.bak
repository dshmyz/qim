package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")

	db := database.GetDB()
	var convMembers []model.ConversationMember
	db.Where("user_id = ?", userID).Preload("Conversation").Preload("Conversation.LastMessage").Preload("Conversation.Members").Preload("Conversation.Members.User").Find(&convMembers)

	type ConversationWithPin struct {
		model.Conversation
		IsPinned        bool   `json:"is_pinned"`
		IP              string `json:"ip,omitempty"`
		Status          string `json:"status,omitempty"`
		Signature       string `json:"signature,omitempty"`
		OtherMemberID   uint   `json:"other_member_id,omitempty"`
		OtherMemberName string `json:"other_member_name,omitempty"`
	}

	var conversations []ConversationWithPin
	for _, cm := range convMembers {
		var session model.ConversationSession
		db.Where("user_id = ? AND conversation_id = ?", userID, cm.Conversation.ID).FirstOrCreate(&session, model.ConversationSession{
			IsPinned:      false,
			LastVisitedAt: time.Now(),
		})

		convWithPin := ConversationWithPin{
			Conversation: cm.Conversation,
			IsPinned:     session.IsPinned,
		}

		if cm.Conversation.Type == "single" {
			var otherMember model.ConversationMember
			db.Where("conversation_id = ? AND user_id != ?", cm.Conversation.ID, userID).First(&otherMember)
			if otherMember.UserID > 0 {
				var otherUser model.User
				db.First(&otherUser, otherMember.UserID)
				convWithPin.IP = otherUser.IP
				convWithPin.Status = otherUser.Status
				convWithPin.Signature = otherUser.Signature
				convWithPin.OtherMemberID = otherUser.ID
				convWithPin.OtherMemberName = otherUser.Nickname
			}
		}

		conversations = append(conversations, convWithPin)
	}

	response.Success(c, conversations)
}

func GetConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	db := database.GetDB()
	var conv model.Conversation
	if err := db.Preload("Members").Preload("Members.User").First(&conv, convID).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", conv.ID, userID).First(&member).Error; err != nil {
		response.Forbidden(c, "无权限访问")
		return
	}

	response.Success(c, conv)
}

func CreateSingleConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		UserID uint `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var existingConv model.Conversation
	db.Raw(`
		SELECT c.* FROM conversations c
		JOIN conversation_members cm1 ON c.id = cm1.conversation_id
		JOIN conversation_members cm2 ON c.id = cm2.conversation_id
		WHERE c.type = 'single'
		AND cm1.user_id = ? AND cm2.user_id = ?
	`, userID, req.UserID).Scan(&existingConv)

	if existingConv.ID > 0 {
		response.Success(c, existingConv)
		return
	}

	if userID.(uint) == req.UserID {
		var targetUser model.User
		db.First(&targetUser, req.UserID)

		conv := model.Conversation{
			Type:   "single",
			Name:   targetUser.Nickname + "（自己）",
			Avatar: targetUser.Avatar,
		}
		db.Create(&conv)

		db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})

		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)

		response.Success(c, conv)
		return
	}

	var targetUser model.User
	db.First(&targetUser, req.UserID)

	conv := model.Conversation{
		Type:   "single",
		Name:   targetUser.Nickname,
		Avatar: targetUser.Avatar,
	}
	db.Create(&conv)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: req.UserID, Role: "member"})

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)

	response.Success(c, conv)
}

func CreateGroupConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name      string `json:"name" binding:"required"`
		Avatar    string `json:"avatar"`
		MemberIDs []uint `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	conv := model.Conversation{
		Type:      "group",
		Name:      req.Name,
		Avatar:    req.Avatar,
		CreatorID: userID.(uint),
	}
	db.Create(&conv)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner"})

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	response.Success(c, conv)
}

func CreateDiscussionConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		Name      string `json:"name" binding:"required"`
		Avatar    string `json:"avatar"`
		MemberIDs []uint `json:"member_ids"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	conv := model.Conversation{
		Type:      "discussion",
		Name:      req.Name,
		Avatar:    req.Avatar,
		CreatorID: userID.(uint),
	}
	db.Create(&conv)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner"})

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	response.Success(c, conv)
}

func PinConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	var req struct {
		IsPinned bool `json:"is_pinned"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		response.Forbidden(c, "无权限操作")
		return
	}

	var session model.ConversationSession
	result := db.Where("user_id = ? AND conversation_id = ?", userID, uint(convID)).First(&session)

	if result.Error != nil {
		session = model.ConversationSession{
			UserID:         userID.(uint),
			ConversationID: uint(convID),
			IsPinned:       req.IsPinned,
			LastVisitedAt:  time.Now(),
		}
		if req.IsPinned {
			now := time.Now()
			session.PinnedAt = &now
		}
		db.Create(&session)
	} else {
		session.IsPinned = req.IsPinned
		if req.IsPinned {
			now := time.Now()
			session.PinnedAt = &now
		} else {
			session.PinnedAt = nil
		}
		db.Save(&session)
	}

	response.Success(c, gin.H{
		"message": "操作成功",
		"data":    session,
	})
}

func SetConversationMute(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	var req struct {
		Muted bool `json:"muted" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		response.Forbidden(c, "无权限操作")
		return
	}

	member.Muted = req.Muted
	db.Save(&member)

	response.Success(c, gin.H{
		"message": "操作成功",
		"data":    member,
	})
}

func DeleteConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	db := database.GetDB()

	convIDUint, err := strconv.ParseUint(convID, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的会话ID")
		return
	}

	var conv model.Conversation
	if err := db.First(&conv, uint(convIDUint)).Error; err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能解散群聊或讨论组")
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convIDUint), userID).First(&currentMember).Error; err != nil {
		response.Forbidden(c, "无权限操作")
		return
	}

	if currentMember.Role != "owner" {
		response.Forbidden(c, "只有群主可以解散群聊")
		return
	}

	tx := db.Begin()

	if err := tx.Where("conversation_id = ?", uint(convIDUint)).Delete(&model.ConversationMember{}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "删除成员失败")
		return
	}

	conv.Name = "[已解散] " + conv.Name
	conv.IsDeleted = true
	if err := tx.Save(&conv).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "更新会话状态失败")
		return
	}

	tx.Commit()

	if ws.GlobalHub != nil {
		dissolveMsg := ws.WSMessage{
			Type: "conversation_deleted",
			Data: gin.H{
				"conversation_id": conv.ID,
				"message":         "群聊已被解散",
			},
		}
		jsonMsg, _ := json.Marshal(dissolveMsg)

		ws.GlobalHub.SendToConversation(uint(convIDUint), 0, jsonMsg)
	}

	response.Success(c, gin.H{
		"message": "群聊解散成功",
	})
}
