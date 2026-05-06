package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/ws"

	"github.com/gin-gonic/gin"
)

func AddMemberToGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convID := c.Param("id")

	if strings.HasPrefix(convID, "conv_") {
		convID = strings.TrimPrefix(convID, "conv_")
	}

	var req struct {
		MemberIDs []uint `json:"member_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	convIDUint, err := strconv.ParseUint(convID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var conv model.Conversation
	if err := db.First(&conv, uint(convIDUint)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊或讨论组添加成员"})
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convIDUint), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权限操作"})
		return
	}

	if conv.Type == "group" {
		// 获取群聊信息
		var group model.Group
		db.Where("conversation_id = ?", uint(convIDUint)).First(&group)
		if group.InvitePermission == "owner_admin" && currentMember.Role != "owner" && currentMember.Role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主和管理员可以邀请成员"})
			return
		}
	}

	var addedMembers []model.User
	for _, memberID := range req.MemberIDs {
		var existingMember model.ConversationMember
		if err := db.Where("conversation_id = ? AND user_id = ?", uint(convIDUint), memberID).First(&existingMember).Error; err == nil {
			continue
		}

		var user model.User
		if err := db.First(&user, memberID).Error; err != nil {
			continue
		}

		newMember := model.ConversationMember{
			ConversationID: uint(convIDUint),
			UserID:         memberID,
			Role:           "member",
			UnreadCount:    0,
			Muted:          false,
			JoinedAt:       time.Now(),
		}
		db.Create(&newMember)

		// 获取群聊信息
		var group model.Group
		db.Where("conversation_id = ?", uint(convIDUint)).First(&group)

		notification := model.Notification{
			UserID:        memberID,
			Type:          "group_invitation",
			Title:         "群聊邀请",
			Content:       fmt.Sprintf("您被邀请加入群聊 %s", group.Name),
			Priority:      "important",
			ActionType:    "accept_ignore",
			ActionPayload: fmt.Sprintf(`{"conversation_id":%d}`, uint(convIDUint)),
		}
		db.Create(&notification)

		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(memberID, jsonMsg)
		}

		addedMembers = append(addedMembers, user)
	}

	if len(addedMembers) > 0 {
		var existingMembers []model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", uint(convIDUint), userID).Find(&existingMembers)

		var memberNames []string
		for _, member := range addedMembers {
			memberNames = append(memberNames, member.Nickname)
		}

		for _, member := range existingMembers {
			notification := model.Notification{
				UserID:  member.UserID,
				Type:    "group_member_added",
				Title:   "新成员加入",
				Content: fmt.Sprintf("新成员 %s 加入了群聊", strings.Join(memberNames, "、")),
			}
			db.Create(&notification)

			if ws.GlobalHub != nil {
				notificationMsg := ws.WSMessage{
					Type: "notification",
					Data: notification,
				}
				jsonMsg, _ := json.Marshal(notificationMsg)
				ws.GlobalHub.SendToUser(member.UserID, jsonMsg)
			}
		}
	}

	if len(addedMembers) > 0 {
		var memberNames []string
		for _, member := range addedMembers {
			memberNames = append(memberNames, member.Nickname)
		}

		var currentUser model.User
		if err := db.First(&currentUser, userID).Error; err != nil {
			log.Printf("获取用户信息失败: %v", err)
		}

		systemMessageContent := fmt.Sprintf("%s 添加了新成员 %s", currentUser.Nickname, strings.Join(memberNames, "、"))
		if currentUser.Nickname == "" {
			systemMessageContent = fmt.Sprintf("%s 添加了新成员 %s", currentUser.Username, strings.Join(memberNames, "、"))
		}

		systemMsg := model.Message{
			ConversationID: conv.ID,
			SenderID:       0,
			Type:           "system",
			Content:        systemMessageContent,
			IsRead:         true,
		}
		if err := db.Create(&systemMsg).Error; err != nil {
			log.Printf("创建系统消息失败: %v", err)
		}

		systemUser := model.User{
			ID:       0,
			Username: "system",
			Nickname: "系统",
			Avatar:   "",
		}
		systemMsg.Sender = systemUser

		now := time.Now()
		conv.LastMessageID = &systemMsg.ID
		conv.LastMessageAt = &now
		db.Save(&conv)

		responseData := gin.H{
			"id":                systemMsg.ID,
			"conversation_id":   systemMsg.ConversationID,
			"sender_id":         systemMsg.SenderID,
			"type":              systemMsg.Type,
			"content":           systemMsg.Content,
			"quoted_message_id": systemMsg.QuotedMessageID,
			"is_recalled":       systemMsg.IsRecalled,
			"is_read":           systemMsg.IsRead,
			"recalled_at":       systemMsg.RecalledAt,
			"created_at":        systemMsg.CreatedAt,
			"sender":            systemMsg.Sender,
		}

		if ws.GlobalHub != nil {
			for _, member := range addedMembers {
				joinMsg := ws.WSMessage{
					Type: "group_member_joined",
					Data: gin.H{
						"conversation_id": conv.ID,
						"member": gin.H{
							"id":       member.ID,
							"nickname": member.Nickname,
							"username": member.Username,
							"avatar":   member.Avatar,
						},
					},
				}
				jsonMsg, _ := json.Marshal(joinMsg)

				ws.GlobalHub.SendToConversation(uint(convIDUint), 0, jsonMsg)
			}

			newMsg := ws.WSMessage{
				Type: "new_message",
				Data: responseData,
			}
			newMsgJson, _ := json.Marshal(newMsg)
			ws.GlobalHub.SendToConversation(uint(convIDUint), 0, newMsgJson)
		}
	}

	if ws.GlobalHub != nil {
		var groupMembers []model.ConversationMember
		db.Where("conversation_id = ?", uint(convIDUint)).Preload("User").Find(&groupMembers)

		var members []gin.H
		for _, gm := range groupMembers {
			members = append(members, gin.H{
				"id":       gm.User.ID,
				"nickname": gm.User.Nickname,
				"username": gm.User.Username,
				"avatar":   gm.User.Avatar,
			})
		}

		for _, member := range addedMembers {
			// 获取群聊信息
			var group model.Group
			db.Where("conversation_id = ?", uint(convIDUint)).First(&group)

			addedMsg := ws.WSMessage{
				Type: "added_to_group",
				Data: gin.H{
					"conversation_id": conv.ID,
					"group_name":      group.Name,
					"group_avatar":    group.Avatar,
					"members":         members,
				},
			}
			jsonMsg, _ := json.Marshal(addedMsg)

			ws.GlobalHub.SendToUser(member.ID, jsonMsg)
		}

		ws.GlobalHub.UpdateConversationMembers(uint(convIDUint))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "添加成员成功",
		"data":    addedMembers,
	})
}

func RemoveMemberFromGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能移除群聊成员"})
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以移除成员"})
		return
	}

	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(memberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	if targetMember.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "群主不能被移除"})
		return
	}

	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(memberID)).Delete(&model.ConversationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "移除成员失败"})
		return
	}

	if ws.GlobalHub != nil {
		removeMsg := ws.WSMessage{
			Type: "group_member_left",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         uint(memberID),
			},
		}
		jsonMsg, _ := json.Marshal(removeMsg)

		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)

		ws.GlobalHub.UpdateConversationMembers(uint(convID))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "移除成员成功",
	})
}

func ExitGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能退出群聊"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).Delete(&model.ConversationMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "退出群聊失败"})
		return
	}

	if ws.GlobalHub != nil {
		exitMsg := ws.WSMessage{
			Type: "group_member_left",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         userID,
			},
		}
		jsonMsg, _ := json.Marshal(exitMsg)

		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)

		ws.GlobalHub.UpdateConversationMembers(uint(convID))
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "退出群聊成功"})
}

func UpdateGroupInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		Name             string `json:"name"`
		Avatar           string `json:"avatar"`
		InvitePermission string `json:"invite_permission"`
		AIEnabled        *bool  `json:"ai_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能更新群聊或讨论组信息"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	// 获取群聊信息
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊信息不存在"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以更新群聊信息"})
		return
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Avatar != "" {
		group.Avatar = req.Avatar
	}
	if req.InvitePermission != "" {
		if req.InvitePermission == "owner_admin" || req.InvitePermission == "all" {
			group.InvitePermission = req.InvitePermission
		}
	}
	if req.AIEnabled != nil {
		group.AIEnabled = *req.AIEnabled
	}
	db.Save(&group)

	if ws.GlobalHub != nil {
		updateMsg := ws.WSMessage{
			Type: "conversation_updated",
			Data: gin.H{
				"id":                conv.ID,
				"type":              conv.Type,
				"name":              group.Name,
				"avatar":            group.Avatar,
				"announcement":      group.Announcement,
				"invite_permission": group.InvitePermission,
				"ai_enabled":        group.AIEnabled,
				"last_message_id":   conv.LastMessageID,
				"last_message_at":   conv.LastMessageAt,
				"created_at":        conv.CreatedAt,
				"updated_at":        conv.UpdatedAt,
			},
		}
		jsonMsg, _ := json.Marshal(updateMsg)

		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "群聊信息更新成功",
		"data":    group,
	})
}

func UpdateGroupAISettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		AIEnabled          *bool   `json:"ai_enabled"`
		AIReplyMode        *string `json:"ai_reply_mode"`
		AIAssistantName    *string `json:"ai_assistant_name"`
		AIPersonality      *string `json:"ai_personality"`
		AICustomPrompt     *string `json:"ai_custom_prompt"`
		AILanguage         *string `json:"ai_language"`
		AIMaxLength        *string `json:"ai_max_length"`
		AIMentionReplyMode *string `json:"ai_mention_reply_mode"`
		AIAntiSpamInterval *int    `json:"ai_anti_spam_interval"`
		AITriggerKeywords  *string `json:"ai_trigger_keywords"`
		AILearnEnabled     *bool   `json:"ai_learn_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能更新群聊或讨论组的 AI 设置"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是成员"})
		return
	}

	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊信息不存在"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以更新 AI 设置"})
		return
	}

	if req.AIEnabled != nil {
		group.AIEnabled = *req.AIEnabled
	}
	if req.AIReplyMode != nil {
		validModes := map[string]bool{"always": true, "mention_only": true, "smart": true, "off": true}
		if validModes[*req.AIReplyMode] {
			group.AIReplyMode = *req.AIReplyMode
		}
	}
	if req.AIAssistantName != nil {
		group.AIAssistantName = *req.AIAssistantName
	}
	if req.AIPersonality != nil {
		group.AIPersonality = *req.AIPersonality
	}
	if req.AICustomPrompt != nil {
		group.AICustomPrompt = *req.AICustomPrompt
	}
	if req.AILanguage != nil {
		group.AILanguage = *req.AILanguage
	}
	if req.AIMaxLength != nil {
		group.AIMaxLength = *req.AIMaxLength
	}
	if req.AIMentionReplyMode != nil {
		group.AIMentionReplyMode = *req.AIMentionReplyMode
	}
	if req.AIAntiSpamInterval != nil {
		group.AIAntiSpamInterval = *req.AIAntiSpamInterval
	}
	if req.AITriggerKeywords != nil {
		group.AITriggerKeywords = *req.AITriggerKeywords
	}
	if req.AILearnEnabled != nil {
		group.AILearnEnabled = *req.AILearnEnabled
	}
	db.Save(&group)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "AI 设置更新成功",
		"data":    group,
	})
}

func SetMemberRole(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin member"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊设置管理员"})
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主可以设置管理员"})
		return
	}

	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(targetMemberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	if targetMember.Role == "owner" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能修改群主角色"})
		return
	}

	targetMember.Role = req.Role
	db.Save(&targetMember)

	if ws.GlobalHub != nil {
		updateMsg := ws.WSMessage{
			Type: "group_member_role_updated",
			Data: gin.H{
				"conversation_id": conv.ID,
				"user_id":         targetMember.UserID,
				"role":            req.Role,
			},
		}
		jsonMsg, _ := json.Marshal(updateMsg)
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "角色设置成功",
		"data":    targetMember,
	})
}

func TransferOwner(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能转让群主给群聊成员"})
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主可以转让群主"})
		return
	}

	var targetMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), uint(targetMemberID)).First(&targetMember).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "目标用户不是群成员"})
		return
	}

	tx := db.Begin()

	currentMember.Role = "admin"
	if err := tx.Save(&currentMember).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	targetMember.Role = "owner"
	if err := tx.Save(&targetMember).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	// 更新群聊创建者
	var group model.Group
	if err := tx.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取群聊信息失败"})
		return
	}
	group.CreatorID = targetMember.UserID
	if err := tx.Save(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "转让失败"})
		return
	}

	tx.Commit()

	if ws.GlobalHub != nil {
		transferMsg := ws.WSMessage{
			Type: "group_owner_transferred",
			Data: gin.H{
				"conversation_id": conv.ID,
				"old_owner_id":    userID,
				"new_owner_id":    targetMember.UserID,
			},
		}
		jsonMsg, _ := json.Marshal(transferMsg)
		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "群主转让成功",
		"data":    targetMember,
	})
}

func UpdateAnnouncement(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	var req struct {
		Announcement string `json:"announcement"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能为群聊或讨论组设置公告"})
		return
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&member).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	// 获取群聊信息
	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊信息不存在"})
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以设置群公告"})
		return
	}

	group.Announcement = req.Announcement
	if err := db.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新群公告失败"})
		return
	}

	var currentUser model.User
	if err := db.First(&currentUser, userID).Error; err != nil {
		log.Printf("获取用户信息失败: %v", err)
	}

	systemMessageContent := fmt.Sprintf("%s 修改群公告", currentUser.Nickname)
	if currentUser.Nickname == "" {
		systemMessageContent = fmt.Sprintf("%s 修改群公告", currentUser.Username)
	}

	systemMsg := model.Message{
		ConversationID: conv.ID,
		SenderID:       0,
		Type:           "system",
		Content:        systemMessageContent,
		IsRead:         true,
	}
	if err := db.Create(&systemMsg).Error; err != nil {
		log.Printf("创建系统消息失败: %v", err)
	}

	systemUser := model.User{
		ID:       0,
		Username: "system",
		Nickname: "系统",
		Avatar:   "",
	}
	systemMsg.Sender = systemUser

	now := time.Now()
	conv.LastMessageID = &systemMsg.ID
	conv.LastMessageAt = &now
	db.Save(&conv)

	responseData := gin.H{
		"id":                systemMsg.ID,
		"conversation_id":   systemMsg.ConversationID,
		"sender_id":         systemMsg.SenderID,
		"type":              systemMsg.Type,
		"content":           systemMsg.Content,
		"quoted_message_id": systemMsg.QuotedMessageID,
		"is_recalled":       systemMsg.IsRecalled,
		"is_read":           systemMsg.IsRead,
		"recalled_at":       systemMsg.RecalledAt,
		"created_at":        systemMsg.CreatedAt,
		"sender":            systemMsg.Sender,
	}

	if ws.GlobalHub != nil {
		updaterName := currentUser.Nickname
		if updaterName == "" {
			updaterName = currentUser.Username
		}

		announcementMsg := ws.WSMessage{
			Type: "group_announcement_updated",
			Data: gin.H{
				"conversation_id": conv.ID,
				"announcement":    req.Announcement,
				"updater_name":    updaterName,
			},
		}
		jsonMsg, _ := json.Marshal(announcementMsg)

		ws.GlobalHub.SendToConversation(uint(convID), 0, jsonMsg)

		newMsg := ws.WSMessage{
			Type: "new_message",
			Data: responseData,
		}
		newMsgJson, _ := json.Marshal(newMsg)
		ws.GlobalHub.SendToConversation(uint(convID), 0, newMsgJson)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "群公告更新成功", "data": group})
}

func ApplyJoinGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "只能申请加入群聊或讨论组"})
		return
	}

	var existingMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&existingMember).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"code": 409, "message": "您已经是群成员"})
		return
	}

	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊信息不存在"})
		return
	}

	var currentUser model.User
	if err := db.First(&currentUser, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户信息失败"})
		return
	}

	var ownersAndAdmins []model.ConversationMember
	db.Where("conversation_id = ? AND role IN ?", uint(convID), []string{"owner", "admin"}).Find(&ownersAndAdmins)

	for _, admin := range ownersAndAdmins {
		notification := model.Notification{
			UserID:        admin.UserID,
			Type:          "group_join_request",
			Title:         "入群申请",
			Content:       fmt.Sprintf("%s 申请加入群聊 %s", currentUser.Nickname, group.Name),
			Priority:      "important",
			ActionType:    "approve_reject",
			ActionPayload: fmt.Sprintf(`{"conversation_id":%d,"user_id":%d}`, uint(convID), userID.(uint)),
		}
		db.Create(&notification)

		if ws.GlobalHub != nil {
			notificationMsg := ws.WSMessage{
				Type: "notification",
				Data: notification,
			}
			jsonMsg, _ := json.Marshal(notificationMsg)
			ws.GlobalHub.SendToUser(admin.UserID, jsonMsg)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "申请已发送，请等待管理员审批"})
}

func RejectJoinRequest(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	targetUserIDStr := c.Param("user_id")

	if strings.HasPrefix(convIDStr, "conv_") {
		convIDStr = strings.TrimPrefix(convIDStr, "conv_")
	}

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的会话ID"})
		return
	}

	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	db := database.GetDB()

	var conv model.Conversation
	if err := db.First(&conv, uint(convID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "会话不存在"})
		return
	}

	var currentMember model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", uint(convID), userID).First(&currentMember).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "您不是群成员"})
		return
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "只有群主或管理员可以拒绝加入请求"})
		return
	}

	var targetUser model.User
	if err := db.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	var group model.Group
	if err := db.Where("conversation_id = ?", uint(convID)).First(&group).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "群聊信息不存在"})
		return
	}

	notification := model.Notification{
		UserID:   uint(targetUserID),
		Type:     "group_join_rejected",
		Title:    "入群申请被拒绝",
		Content:  fmt.Sprintf("您加入群聊 %s 的申请已被拒绝", group.Name),
		Priority: "normal",
	}
	db.Create(&notification)

	if ws.GlobalHub != nil {
		notificationMsg := ws.WSMessage{
			Type: "notification",
			Data: notification,
		}
		jsonMsg, _ := json.Marshal(notificationMsg)
		ws.GlobalHub.SendToUser(uint(targetUserID), jsonMsg)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已拒绝加入请求"})
}
