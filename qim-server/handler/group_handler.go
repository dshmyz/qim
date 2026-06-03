package handler

import (
	"encoding/json"
	"fmt"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/di"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/dshmyz/qim/qim-server/pkg/validation"
	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/ws"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func AddMemberToGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	var req struct {
		MemberIDs []uint `json:"member_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	userSvc := di.GlobalContainer.UserService
	notifSvc := di.GlobalContainer.NotificationService
	msgSvc := di.GlobalContainer.MessageService
	groupSvc := di.GlobalContainer.GroupService

	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能为群聊或讨论组添加成员")
		return
	}

	currentMember, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "无权限操作")
		return
	}

	if conv.Type == "group" {
		if group.InvitePermission == "owner_admin" && currentMember.Role != "owner" && currentMember.Role != "admin" {
			response.Forbidden(c, "只有群主和管理员可以邀请成员")
			return
		}
	}

	var addedMembers []model.User
	for _, memberID := range req.MemberIDs {
		existingMember, _ := convSvc.GetMember(convIDUint, memberID)
		if existingMember != nil {
			continue
		}

		user, err := userSvc.GetUser(memberID)
		if err != nil {
			continue
		}

		newMember := &model.ConversationMember{
			ConversationID: uint(convIDUint),
			UserID:         memberID,
			Role:           "member",
			UnreadCount:    0,
			Muted:          false,
			JoinedAt:       time.Now(),
		}
		convSvc.CreateMember(newMember)

		// 恢复会话显示：用户被添加到群聊时，如果会话被隐藏则恢复显示
		database.GetDB().Model(&model.ConversationSession{}).
			Where("user_id = ? AND conversation_id = ? AND is_hidden = ?", memberID, uint(convIDUint), true).
			Update("is_hidden", false)

		group, _ := groupSvc.GetGroupByConversationID(uint(convIDUint))
		groupName := ""
		if group != nil {
			groupName = group.Name
		}

		notifSvc.Create(&model.Notification{
			UserID:     memberID,
			Type:       "group_invitation",
			Title:      "群聊邀请",
			Content:    fmt.Sprintf("您被邀请加入群聊 %s", groupName),
			Priority:   "important",
			ActionType: "accept_ignore",
			ActionPayload: func() string {
				b, _ := json.Marshal(map[string]interface{}{"conversation_id": uint(convIDUint)})
				return string(b)
			}(),
		})

		addedMembers = append(addedMembers, *user)
	}

	if len(addedMembers) > 0 {
		existingMembers, _ := convSvc.GetMembersExcept(uint(convIDUint), userID.(uint))

		var memberNames []string
		for _, member := range addedMembers {
			memberNames = append(memberNames, member.Nickname)
		}

		for _, member := range existingMembers {
			notifSvc.Create(&model.Notification{
				UserID:  member.UserID,
				Type:    "group_member_added",
				Title:   "新成员加入",
				Content: fmt.Sprintf("新成员 %s 加入了群聊", strings.Join(memberNames, "、")),
			})
		}
	}

	if len(addedMembers) > 0 {
		var memberNames []string
		for _, member := range addedMembers {
			memberNames = append(memberNames, member.Nickname)
		}

		currentUser, err := userSvc.GetUser(userID.(uint))
		if err != nil {
			logger.WithModule("GroupHandler").Error("获取用户信息失败", "error", err)
			currentUser = &model.User{
				ID:       userSvc.GetSystemUserID(),
				Username: "system",
				Nickname: "系统",
			}
		}

		systemMessageContent := fmt.Sprintf("%s 添加了新成员 %s", currentUser.Nickname, strings.Join(memberNames, "、"))
		if currentUser.Nickname == "" {
			systemMessageContent = fmt.Sprintf("%s 添加了新成员 %s", currentUser.Username, strings.Join(memberNames, "、"))
		}

		systemMsg := &model.Message{
			ConversationID: conv.ID,
			SenderID:       currentUser.ID,
			Type:           "system",
			Content:        systemMessageContent,
			IsRead:         true,
		}
		if err := msgSvc.CreateMessage(systemMsg); err != nil {
			logger.WithModule("GroupHandler").Error("创建系统消息失败", "error", err)
		}

		systemMsg.Sender = *currentUser

		now := time.Now()
		convSvc.UpdateConversation(conv.ID, map[string]interface{}{
			"last_message_id": systemMsg.ID,
			"last_message_at": now,
		})

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
		groupMembers, _ := convSvc.GetMembersWithUser(uint(convIDUint))

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
			group, _ := groupSvc.GetGroupByConversationID(uint(convIDUint))
			groupName := ""
			groupAvatar := ""
			if group != nil {
				groupName = group.Name
				groupAvatar = group.Avatar
			}

			addedMsg := ws.WSMessage{
				Type: "added_to_group",
				Data: gin.H{
					"conversation_id": conv.ID,
					"group_name":      groupName,
					"group_avatar":    groupAvatar,
					"members":         members,
				},
			}
			jsonMsg, _ := json.Marshal(addedMsg)
			ws.GlobalHub.SendToUser(member.ID, jsonMsg)
		}

		ws.GlobalHub.UpdateConversationMembers(uint(convIDUint))
	}

	response.SuccessWithMessage(c, "添加成员成功", addedMembers)
}

func RemoveMemberFromGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	memberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	_, err = groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" {
		response.BadRequest(c, "只能移除群聊成员")
		return
	}

	currentMember, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以移除成员")
		return
	}

	targetMember, err := convSvc.GetMember(convIDUint, uint(memberID))
	if err != nil {
		response.BadRequest(c, "目标用户不是群成员")
		return
	}

	if targetMember.Role == "owner" {
		response.BadRequest(c, "群主不能被移除")
		return
	}

	if err := convSvc.RemoveMember(convIDUint, uint(memberID)); err != nil {
		response.InternalServerError(c, "移除成员失败")
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
		ws.GlobalHub.SendToConversation(convIDUint, 0, jsonMsg)
		ws.GlobalHub.UpdateConversationMembers(convIDUint)
	}

	response.SuccessWithMessage(c, "移除成员成功", nil)
}

func ExitGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	_, err = groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" {
		response.BadRequest(c, "只能退出群聊")
		return
	}

	_, err = convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if err := convSvc.RemoveMember(convIDUint, userID.(uint)); err != nil {
		response.InternalServerError(c, "退出群聊失败")
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
		ws.GlobalHub.SendToConversation(convIDUint, 0, jsonMsg)
		ws.GlobalHub.UpdateConversationMembers(convIDUint)
	}

	response.SuccessWithMessage(c, "退出群聊成功", nil)
}

func UpdateGroupInfo(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	var req struct {
		Name             string `json:"name"`
		Avatar           string `json:"avatar"`
		InvitePermission string `json:"invite_permission"`
		AIEnabled        *bool  `json:"ai_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能更新群聊或讨论组信息")
		return
	}

	member, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以更新群聊信息")
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
		aiConfig := group.GetAIConfig()
		aiConfig.Enabled = *req.AIEnabled
		group.SetAIConfig(aiConfig)
	}
	groupSvc.UpdateGroup(group)

	aiConfig := group.GetAIConfig()

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
				"ai_config": gin.H{
					"ai_enabled":            aiConfig.Enabled,
					"ai_assistant_name":     aiConfig.AssistantName,
					"ai_reply_mode":         aiConfig.ReplyMode,
					"ai_personality":        aiConfig.Personality,
					"ai_custom_prompt":      aiConfig.CustomPrompt,
					"ai_language":           aiConfig.Language,
					"ai_max_length":         aiConfig.MaxLength,
					"ai_mention_reply_mode": aiConfig.MentionReplyMode,
					"ai_anti_spam_interval": aiConfig.AntiSpamInterval,
					"ai_trigger_keywords":   aiConfig.TriggerKeywords,
					"ai_learn_enabled":      aiConfig.LearnEnabled,
				},
				"last_message_id": conv.LastMessageID,
				"last_message_at": conv.LastMessageAt,
				"created_at":      conv.CreatedAt,
				"updated_at":      conv.UpdatedAt,
			},
		}
		jsonMsg, _ := json.Marshal(updateMsg)
		ws.GlobalHub.SendToConversation(convIDUint, 0, jsonMsg)
	}

	response.SuccessWithMessage(c, "群聊信息更新成功", gin.H{
		"id":                conv.ID,
		"type":              conv.Type,
		"name":              group.Name,
		"avatar":            group.Avatar,
		"announcement":      group.Announcement,
		"invite_permission": group.InvitePermission,
		"ai_config": gin.H{
			"ai_enabled":            aiConfig.Enabled,
			"ai_assistant_name":     aiConfig.AssistantName,
			"ai_reply_mode":         aiConfig.ReplyMode,
			"ai_personality":        aiConfig.Personality,
			"ai_custom_prompt":      aiConfig.CustomPrompt,
			"ai_language":           aiConfig.Language,
			"ai_max_length":         aiConfig.MaxLength,
			"ai_mention_reply_mode": aiConfig.MentionReplyMode,
			"ai_anti_spam_interval": aiConfig.AntiSpamInterval,
			"ai_trigger_keywords":   aiConfig.TriggerKeywords,
			"ai_learn_enabled":      aiConfig.LearnEnabled,
		},
	})
}

func GetGroupAISettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能获取群聊或讨论组的 AI 设置")
		return
	}

	_, err = convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	aiConfig := group.GetAIConfig()

	approvalSvc := di.GlobalContainer.ApprovalService
	approval, _ := approvalSvc.GetApproval(model.ApprovalTypeGroupAI, group.ID)
	approvalStatus := ""
	rejectReason := ""
	if approval != nil {
		approvalStatus = approval.Status
		rejectReason = approval.RejectReason
	}

	response.Success(c, gin.H{
		"ai_enabled":            aiConfig.Enabled,
		"ai_assistant_name":     aiConfig.AssistantName,
		"ai_reply_mode":         aiConfig.ReplyMode,
		"ai_personality":        aiConfig.Personality,
		"ai_custom_prompt":      aiConfig.CustomPrompt,
		"ai_language":           aiConfig.Language,
		"ai_max_length":         aiConfig.MaxLength,
		"ai_mention_reply_mode": aiConfig.MentionReplyMode,
		"ai_anti_spam_interval": aiConfig.AntiSpamInterval,
		"ai_trigger_keywords":   aiConfig.TriggerKeywords,
		"ai_learn_enabled":      aiConfig.LearnEnabled,
		"approval_status":       approvalStatus,
		"reject_reason":         rejectReason,
	})
}

func UpdateGroupAISettings(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
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
		AIExtractTodos     *bool   `json:"ai_extract_todos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.AIAssistantName != nil {
		if err := validation.ValidateAliasName(*req.AIAssistantName); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	groupSvc := di.GlobalContainer.GroupService
	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能更新群聊或讨论组的 AI 设置")
		return
	}

	member, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以更新 AI 设置")
		return
	}

	aiConfig := group.GetAIConfig()
	oldEnabled := aiConfig.Enabled

	if req.AIEnabled != nil && *req.AIEnabled && !oldEnabled {
		approvalService := di.GlobalContainer.ApprovalService
		needsApproval := approvalService.IsApprovalEnabled(model.ApprovalTypeGroupAI)

		if needsApproval {
			if existingApproval, err := approvalService.GetApproval(model.ApprovalTypeGroupAI, group.ID); err == nil && existingApproval.Status == model.ApprovalStatusPending {
				response.BadRequest(c, "已有待审批的AI助手申请")
				return
			}

			aiConfig.Enabled = false
			if req.AIAssistantName != nil {
				aiConfig.AssistantName = *req.AIAssistantName
			}
			if req.AIReplyMode != nil {
				validModes := map[string]bool{"always": true, "mention_only": true, "smart": true, "off": true}
				if validModes[*req.AIReplyMode] {
					aiConfig.ReplyMode = *req.AIReplyMode
				}
			}
			if req.AIPersonality != nil {
				aiConfig.Personality = *req.AIPersonality
			}
			if req.AICustomPrompt != nil {
				aiConfig.CustomPrompt = *req.AICustomPrompt
			}
			if req.AILanguage != nil {
				aiConfig.Language = *req.AILanguage
			}
			if req.AIMaxLength != nil {
				aiConfig.MaxLength = *req.AIMaxLength
			}
			if req.AIMentionReplyMode != nil {
				aiConfig.MentionReplyMode = *req.AIMentionReplyMode
			}
			if req.AIAntiSpamInterval != nil {
				aiConfig.AntiSpamInterval = *req.AIAntiSpamInterval
			}
			if req.AITriggerKeywords != nil {
				aiConfig.TriggerKeywords = *req.AITriggerKeywords
			}
			if req.AILearnEnabled != nil {
				aiConfig.LearnEnabled = *req.AILearnEnabled
			}
			if req.AIExtractTodos != nil {
				aiConfig.ExtractTodos = *req.AIExtractTodos
			}

			group.SetAIConfig(aiConfig)
			groupSvc.UpdateGroup(group)

			now := time.Now()
			if err := approvalService.CreateApproval(model.ApprovalTypeGroupAI, group.ID, userID.(uint)); err != nil {
				response.InternalServerError(c, "提交申请失败")
				return
			}

			di.GlobalContainer.OperationLogService.LogUserOperation(c, "group_ai", "apply_approval")

			response.SuccessWithMessage(c, "AI助手申请已提交，等待系统管理员审批", gin.H{
				"approval_status": model.ApprovalStatusPending,
				"applied_at":      now,
			})
			return
		}
		aiConfig.Enabled = true
	}

	if req.AIEnabled != nil {
		aiConfig.Enabled = *req.AIEnabled
	}
	if req.AIReplyMode != nil {
		validModes := map[string]bool{"always": true, "mention_only": true, "smart": true, "off": true}
		if validModes[*req.AIReplyMode] {
			aiConfig.ReplyMode = *req.AIReplyMode
		}
	}
	if req.AIAssistantName != nil {
		aiConfig.AssistantName = *req.AIAssistantName
	}
	if req.AIPersonality != nil {
		aiConfig.Personality = *req.AIPersonality
	}
	if req.AICustomPrompt != nil {
		aiConfig.CustomPrompt = *req.AICustomPrompt
	}
	if req.AILanguage != nil {
		aiConfig.Language = *req.AILanguage
	}
	if req.AIMaxLength != nil {
		aiConfig.MaxLength = *req.AIMaxLength
	}
	if req.AIMentionReplyMode != nil {
		aiConfig.MentionReplyMode = *req.AIMentionReplyMode
	}
	if req.AIAntiSpamInterval != nil {
		aiConfig.AntiSpamInterval = *req.AIAntiSpamInterval
	}
	if req.AITriggerKeywords != nil {
		aiConfig.TriggerKeywords = *req.AITriggerKeywords
	}
	if req.AILearnEnabled != nil {
		aiConfig.LearnEnabled = *req.AILearnEnabled
	}

	if req.AIExtractTodos != nil {
		aiConfig.ExtractTodos = *req.AIExtractTodos
	}
	if err := group.SetAIConfig(aiConfig); err != nil {
		response.InternalServerError(c, "保存AI配置失败")
		return
	}
	groupSvc.UpdateGroup(group)
	service.GetAINameCache().InvalidateGroupAssistantName(group.ConversationID)

	di.GlobalContainer.OperationLogService.LogUserOperation(c, "group_ai", "update_settings")

	response.SuccessWithMessage(c, "AI 设置更新成功", gin.H{
		"ai_enabled":            aiConfig.Enabled,
		"ai_assistant_name":     aiConfig.AssistantName,
		"ai_reply_mode":         aiConfig.ReplyMode,
		"ai_personality":        aiConfig.Personality,
		"ai_custom_prompt":      aiConfig.CustomPrompt,
		"ai_language":           aiConfig.Language,
		"ai_max_length":         aiConfig.MaxLength,
		"ai_mention_reply_mode": aiConfig.MentionReplyMode,
		"ai_anti_spam_interval": aiConfig.AntiSpamInterval,
		"ai_trigger_keywords":   aiConfig.TriggerKeywords,
		"ai_learn_enabled":      aiConfig.LearnEnabled,
	})
}

func SetMemberRole(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin member"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	_, err = groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" {
		response.BadRequest(c, "只能为群聊设置管理员")
		return
	}

	currentMember, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if currentMember.Role != "owner" {
		response.Forbidden(c, "只有群主可以设置管理员")
		return
	}

	targetMember, err := convSvc.GetMember(convIDUint, uint(targetMemberID))
	if err != nil {
		response.BadRequest(c, "目标用户不是群成员")
		return
	}

	if targetMember.Role == "owner" {
		response.BadRequest(c, "不能修改群主角色")
		return
	}

	targetMember.Role = req.Role
	convSvc.UpdateMember(targetMember)

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
		ws.GlobalHub.SendToConversation(convIDUint, 0, jsonMsg)
	}

	response.SuccessWithMessage(c, "角色设置成功", targetMember)
}

func TransferOwner(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")
	memberIDStr := c.Param("user_id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	targetMemberID, err := strconv.ParseUint(memberIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	convSvc := di.GlobalContainer.ConversationService

	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" {
		response.BadRequest(c, "只能转让群主给群聊成员")
		return
	}

	currentMember, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if currentMember.Role != "owner" {
		response.Forbidden(c, "只有群主可以转让群主")
		return
	}

	targetMember, err := convSvc.GetMember(convIDUint, uint(targetMemberID))
	if err != nil {
		response.BadRequest(c, "目标用户不是群成员")
		return
	}

	if err := groupSvc.TransferOwner(&service.TransferOwnerParams{
		ConversationID: convIDUint,
		CurrentOwnerID: userID.(uint),
		NewOwnerID:     uint(targetMemberID),
		CurrentMember:  currentMember,
		TargetMember:   targetMember,
		Group:          group,
	}); err != nil {
		response.InternalServerError(c, "转让失败")
		return
	}

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
		ws.GlobalHub.SendToConversation(convIDUint, 0, jsonMsg)
	}

	response.SuccessWithMessage(c, "群主转让成功", targetMember)
}

func UpdateAnnouncement(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	var req struct {
		Announcement string `json:"announcement"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	groupSvc := di.GlobalContainer.GroupService
	userSvc := di.GlobalContainer.UserService
	msgSvc := di.GlobalContainer.MessageService

	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能为群聊或讨论组设置公告")
		return
	}

	member, err := convSvc.GetMember(convIDUint, userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if group.GroupType == "group" && member.Role != "owner" && member.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以设置群公告")
		return
	}

	group.Announcement = req.Announcement
	if err := groupSvc.UpdateGroup(group); err != nil {
		response.InternalServerError(c, "更新群公告失败")
		return
	}

	currentUser, err := userSvc.GetUser(userID.(uint))
	if err != nil {
		logger.WithModule("GroupHandler").Error("获取用户信息失败", "error", err)
		currentUser = &model.User{
			ID:       userSvc.GetSystemUserID(),
			Username: "system",
			Nickname: "系统",
		}
	}

	systemMessageContent := fmt.Sprintf("%s 修改群公告", currentUser.Nickname)
	if currentUser.Nickname == "" {
		systemMessageContent = fmt.Sprintf("%s 修改群公告", currentUser.Username)
	}

	systemMsg := &model.Message{
		ConversationID: convIDUint,
		SenderID:       currentUser.ID,
		Type:           "system",
		Content:        systemMessageContent,
		IsRead:         true,
	}
	if err := msgSvc.CreateMessage(systemMsg); err != nil {
		logger.WithModule("GroupHandler").Error("创建系统消息失败", "error", err)
	}

	systemMsg.Sender = *currentUser

	now := time.Now()
	convSvc.UpdateConversation(convIDUint, map[string]interface{}{
		"last_message_id": systemMsg.ID,
		"last_message_at": now,
	})

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

	response.SuccessWithMessage(c, "群公告更新成功", group)
}

func ApplyJoinGroup(c *gin.Context) {
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

	convSvc := di.GlobalContainer.ConversationService
	groupSvc := di.GlobalContainer.GroupService
	userSvc := di.GlobalContainer.UserService
	notifSvc := di.GlobalContainer.NotificationService

	conv, err := convSvc.GetConversation(uint(convID))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" && conv.Type != "discussion" {
		response.BadRequest(c, "只能申请加入群聊或讨论组")
		return
	}

	existingMember, _ := convSvc.GetMember(uint(convID), userID.(uint))
	if existingMember != nil {
		response.Conflict(c, "您已经是群成员")
		return
	}

	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊信息不存在")
		return
	}

	currentUser, err := userSvc.GetUser(userID.(uint))
	if err != nil {
		response.InternalServerError(c, "获取用户信息失败")
		return
	}

	ownersAndAdmins, _ := convSvc.GetMembersByRoles(uint(convID), []string{"owner", "admin"})

	for _, admin := range ownersAndAdmins {
		notifSvc.Create(&model.Notification{
			UserID:     admin.UserID,
			Type:       "group_join_request",
			Title:      "入群申请",
			Content:    fmt.Sprintf("%s 申请加入群聊 %s", currentUser.Nickname, group.Name),
			Priority:   "important",
			ActionType: "approve_reject",
			ActionPayload: func() string {
				b, _ := json.Marshal(map[string]interface{}{"conversation_id": uint(convID), "user_id": userID.(uint)})
				return string(b)
			}(),
		})
	}

	response.SuccessWithMessage(c, "申请已发送，请等待管理员审批", nil)
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
		response.BadRequest(c, "无效的会话ID")
		return
	}

	targetUserID, err := strconv.ParseUint(targetUserIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	convSvc := di.GlobalContainer.ConversationService
	groupSvc := di.GlobalContainer.GroupService
	userSvc := di.GlobalContainer.UserService
	notifSvc := di.GlobalContainer.NotificationService

	_, err = convSvc.GetConversation(uint(convID))
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	currentMember, err := convSvc.GetMember(uint(convID), userID.(uint))
	if err != nil {
		response.Forbidden(c, "您不是群成员")
		return
	}

	if currentMember.Role != "owner" && currentMember.Role != "admin" {
		response.Forbidden(c, "只有群主或管理员可以拒绝加入请求")
		return
	}

	_, err = userSvc.GetUser(uint(targetUserID))
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	group, err := groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊信息不存在")
		return
	}

	notifSvc.Create(&model.Notification{
		UserID:   uint(targetUserID),
		Type:     "group_join_rejected",
		Title:    "入群申请被拒绝",
		Content:  fmt.Sprintf("您加入群聊 %s 的申请已被拒绝", group.Name),
		Priority: "normal",
	})

	response.SuccessWithMessage(c, "已拒绝加入请求", nil)
}

func DissolveGroup(c *gin.Context) {
	userID, _ := c.Get("user_id")
	convIDStr := c.Param("id")

	convID, err := strconv.ParseUint(convIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的群ID")
		return
	}

	groupSvc := di.GlobalContainer.GroupService
	_, err = groupSvc.GetGroupByConversationID(uint(convID))
	if err != nil {
		response.NotFound(c, "群聊不存在")
		return
	}

	convIDUint := uint(convID)
	convSvc := di.GlobalContainer.ConversationService

	conv, err := convSvc.GetConversation(convIDUint)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}

	if conv.Type != "group" {
		response.BadRequest(c, "只能解散群聊")
		return
	}

	members, err := convSvc.GetMembersExcept(convIDUint, 0)
	if err != nil {
		response.InternalServerError(c, "获取群成员失败")
		return
	}

	isOwner := false
	for _, m := range members {
		if m.UserID == userID.(uint) && m.Role == "owner" {
			isOwner = true
			break
		}
	}

	if !isOwner {
		response.Forbidden(c, "只有群主可以解散群聊")
		return
	}

	if err := convSvc.DeleteConversation(convIDUint, userID.(uint)); err != nil {
		response.InternalServerError(c, "解散群聊失败")
		return
	}

	response.SuccessWithMessage(c, "群聊解散成功", nil)
}
