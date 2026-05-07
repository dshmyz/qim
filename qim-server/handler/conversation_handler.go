package handler

import (
	"strconv"
	"strings"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/response"

	"github.com/gin-gonic/gin"
)

func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	db := database.GetDB()

	var convMembers []model.ConversationMember
	db.Where("user_id = ?", uid).
		Preload("Conversation").
		Preload("Conversation.LastMessage").
		Find(&convMembers)

	if len(convMembers) == 0 {
		response.Success(c, []interface{}{})
		return
	}

	type AIConfig struct {
		AIEnabled          bool   `json:"ai_enabled,omitempty"`
		AIAssistantName    string `json:"ai_assistant_name,omitempty"`
		AIReplyMode        string `json:"ai_reply_mode,omitempty"`
		AIPersonality      string `json:"ai_personality,omitempty"`
		AICustomPrompt     string `json:"ai_custom_prompt,omitempty"`
		AILanguage         string `json:"ai_language,omitempty"`
		AIMaxLength        string `json:"ai_max_length,omitempty"`
		AIMentionReplyMode string `json:"ai_mention_reply_mode,omitempty"`
		AIAntiSpamInterval int    `json:"ai_anti_spam_interval,omitempty"`
		AITriggerKeywords  string `json:"ai_trigger_keywords,omitempty"`
		AILearnEnabled     bool   `json:"ai_learn_enabled,omitempty"`
	}

	type ConversationWithPin struct {
		model.Conversation
		Name             string    `json:"name,omitempty"`
		Avatar           string    `json:"avatar,omitempty"`
		CreatorID        uint      `json:"creator_id,omitempty"`
		Announcement     string    `json:"announcement,omitempty"`
		InvitePermission string    `json:"invite_permission,omitempty"`
		AIConfig         *AIConfig `json:"ai_config,omitempty"`
		IsPinned         bool      `json:"is_pinned"`
		UnreadCount      int       `json:"unread_count"`
		Muted            bool      `json:"muted"`
		IP               string    `json:"ip,omitempty"`
		Status           string    `json:"status,omitempty"`
		Signature        string    `json:"signature,omitempty"`
		OtherMemberID    uint      `json:"other_member_id,omitempty"`
		OtherMemberName  string    `json:"other_member_name,omitempty"`
	}

	groupConvIDs := make([]uint, 0, len(convMembers))
	singleConvIDs := make([]uint, 0, len(convMembers))
	for _, cm := range convMembers {
		if cm.Conversation.Type == "group" || cm.Conversation.Type == "discussion" {
			groupConvIDs = append(groupConvIDs, cm.Conversation.ID)
		} else if cm.Conversation.Type == "single" {
			singleConvIDs = append(singleConvIDs, cm.Conversation.ID)
		}
	}

	groupMap := make(map[uint]model.Group, len(groupConvIDs))
	if len(groupConvIDs) > 0 {
		var groups []model.Group
		db.Where("conversation_id IN ?", groupConvIDs).Find(&groups)
		for _, g := range groups {
			groupMap[g.ConversationID] = g
		}
	}

	otherMemberMap := make(map[uint]uint, len(singleConvIDs))
	if len(singleConvIDs) > 0 {
		var otherMembers []model.ConversationMember
		db.Where("conversation_id IN ? AND user_id != ?", singleConvIDs, uid).Find(&otherMembers)
		for _, om := range otherMembers {
			otherMemberMap[om.ConversationID] = om.UserID
		}
	}

	uniqueUserIDs := make([]uint, 0, len(otherMemberMap))
	seenUsers := make(map[uint]struct{}, len(otherMemberMap))
	for _, uid := range otherMemberMap {
		if _, exists := seenUsers[uid]; !exists {
			uniqueUserIDs = append(uniqueUserIDs, uid)
			seenUsers[uid] = struct{}{}
		}
	}

	userMap := make(map[uint]model.User, len(uniqueUserIDs))
	if len(uniqueUserIDs) > 0 {
		var users []model.User
		db.Where("id IN ?", uniqueUserIDs).Find(&users)
		for _, u := range users {
			userMap[u.ID] = u
		}
	}

	allConvIDs := make([]uint, 0, len(convMembers))
	for _, cm := range convMembers {
		allConvIDs = append(allConvIDs, cm.Conversation.ID)
	}

	sessionMap := make(map[uint]model.ConversationSession, len(convMembers))
	if len(allConvIDs) > 0 {
		var sessions []model.ConversationSession
		db.Where("user_id = ? AND conversation_id IN ?", uid, allConvIDs).Find(&sessions)
		for _, s := range sessions {
			sessionMap[s.ConversationID] = s
		}
	}

	var conversations []ConversationWithPin
	var sessionsToCreate []model.ConversationSession
	now := time.Now()

	for _, cm := range convMembers {
		convID := cm.Conversation.ID

		session, exists := sessionMap[convID]
		if !exists {
			session = model.ConversationSession{
				UserID:         uid,
				ConversationID: convID,
				IsPinned:       false,
				LastVisitedAt:  now,
			}
			sessionsToCreate = append(sessionsToCreate, session)
		}

		convWithPin := ConversationWithPin{
			Conversation: cm.Conversation,
			IsPinned:     session.IsPinned,
			UnreadCount:  cm.UnreadCount,
			Muted:        cm.Muted,
		}

		if cm.Conversation.Type == "group" || cm.Conversation.Type == "discussion" {
			if group, ok := groupMap[convID]; ok {
				aiConfig := group.GetAIConfig()
				convWithPin.Name = group.Name
				convWithPin.Avatar = group.Avatar
				convWithPin.CreatorID = group.CreatorID
				convWithPin.Announcement = group.Announcement
				convWithPin.InvitePermission = group.InvitePermission
				convWithPin.AIConfig = &AIConfig{
					AIEnabled:          aiConfig.Enabled,
					AIAssistantName:    aiConfig.AssistantName,
					AIReplyMode:        aiConfig.ReplyMode,
					AIPersonality:      aiConfig.Personality,
					AICustomPrompt:     aiConfig.CustomPrompt,
					AILanguage:         aiConfig.Language,
					AIMaxLength:        aiConfig.MaxLength,
					AIMentionReplyMode: aiConfig.MentionReplyMode,
					AIAntiSpamInterval: aiConfig.AntiSpamInterval,
					AITriggerKeywords:  aiConfig.TriggerKeywords,
					AILearnEnabled:     aiConfig.LearnEnabled,
				}
			}
		}

		if cm.Conversation.Type == "single" {
			if otherUserID, ok := otherMemberMap[convID]; ok {
				if otherUser, ok := userMap[otherUserID]; ok {
					convWithPin.IP = otherUser.IP
					convWithPin.Status = otherUser.Status
					convWithPin.Signature = otherUser.Signature
					convWithPin.OtherMemberID = otherUser.ID
					convWithPin.OtherMemberName = otherUser.Nickname
					convWithPin.Name = otherUser.Nickname
					convWithPin.Avatar = otherUser.Avatar
				}
			}
		}

		conversations = append(conversations, convWithPin)
	}

	if len(sessionsToCreate) > 0 {
		db.CreateInBatches(sessionsToCreate, 50)
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

	// 对于群聊和讨论组，从Group表获取名称、头像等信息
	if conv.Type == "group" || conv.Type == "discussion" {
		var group model.Group
		if err := db.Where("conversation_id = ?", conv.ID).First(&group).Error; err == nil {
			aiConfig := group.GetAIConfig()
			name := group.Name

			// 构建包含群聊信息的响应
			responseData := gin.H{
				"id":                conv.ID,
				"type":              conv.Type,
				"name":              name,
				"avatar":            group.Avatar,
				"creator_id":        group.CreatorID,
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
				"is_deleted":      conv.IsDeleted,
				"last_message_id": conv.LastMessageID,
				"last_message_at": conv.LastMessageAt,
				"created_at":      conv.CreatedAt,
				"updated_at":      conv.UpdatedAt,
				"members":         conv.Members,
			}
			response.Success(c, responseData)
			return
		}
	}

	response.Success(c, conv)
}

func CreateSingleConversation(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req struct {
		UserID      uint  `json:"user_id"`
		RecipientID uint  `json:"recipient_id"`
		BotID       *uint `json:"bot_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	if req.BotID != nil && *req.BotID > 0 {
		var bot model.Bot
		if err := db.First(&bot, *req.BotID).Error; err != nil {
			response.NotFound(c, "机器人不存在")
			return
		}

		var botConv model.BotConversation
		db.Where("bot_id = ? AND user_id = ?", *req.BotID, userID.(uint)).
			Preload("Conversation").First(&botConv)

		if botConv.ID > 0 {
			db.Preload("Conversation.Members").Preload("Conversation.Members.User").
				First(&botConv, botConv.ID)
			response.Success(c, botConv.Conversation)
			return
		}

		conv := model.Conversation{Type: "single"}
		db.Create(&conv)

		db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})
		db.Create(&model.BotConversation{
			BotID:          *req.BotID,
			UserID:         userID.(uint),
			ConversationID: conv.ID,
		})

		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
		response.Success(c, conv)
		return
	}

	if req.UserID == 0 {
		response.BadRequest(c, "缺少必要参数")
		return
	}

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
			Type: "single",
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
		Type: "single",
	}
	db.Create(&conv)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: req.UserID, Role: "member"})

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)

	response.Success(c, conv)
}

// CreateBotConversation 创建或获取 Bot 会话
func CreateBotConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "未授权")
		return
	}

	var req struct {
		BotID uint `json:"bot_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	db := database.GetDB()

	// 检查 Bot 是否存在
	var bot model.Bot
	if err := db.First(&bot, req.BotID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}

	// 查找是否已有会话
	var botConv model.BotConversation
	db.Where("bot_id = ? AND user_id = ?", req.BotID, userID.(uint)).
		Preload("Conversation").First(&botConv)

	if botConv.ID > 0 {
		db.Preload("Conversation.Members").Preload("Conversation.Members.User").
			First(&botConv, botConv.ID)
		response.Success(c, botConv.Conversation)
		return
	}

	// 创建新会话
	conv := model.Conversation{Type: "bot"}
	db.Create(&conv)

	// 添加用户为会话成员
	db.Create(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         userID.(uint),
		Role:           "member",
	})

	// 创建 Bot 会话关联
	db.Create(&model.BotConversation{
		BotID:          bot.ID,
		UserID:         userID.(uint),
		ConversationID: conv.ID,
	})

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
		Type: "group",
	}
	db.Create(&conv)

	// 创建群聊记录
	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "group",
		Name:             req.Name,
		Avatar:           req.Avatar,
		CreatorID:        userID.(uint),
		InvitePermission: "owner_admin",
	}
	db.Create(&group)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner"})

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	// 构建包含群聊信息的响应
	aiConfig := group.GetAIConfig()
	responseData := gin.H{
		"id":                conv.ID,
		"type":              conv.Type,
		"name":              group.Name,
		"avatar":            group.Avatar,
		"creator_id":        group.CreatorID,
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
		"is_deleted":      conv.IsDeleted,
		"last_message_id": conv.LastMessageID,
		"last_message_at": conv.LastMessageAt,
		"created_at":      conv.CreatedAt,
		"updated_at":      conv.UpdatedAt,
	}

	response.Success(c, responseData)
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
		Type: "discussion",
	}
	db.Create(&conv)

	// 创建群聊记录
	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "discussion",
		Name:             req.Name,
		Avatar:           req.Avatar,
		CreatorID:        userID.(uint),
		InvitePermission: "owner_admin",
	}
	db.Create(&group)

	db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner"})

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member"})
		}
	}

	// 构建包含群聊信息的响应
	aiConfig := group.GetAIConfig()
	responseData := gin.H{
		"id":                conv.ID,
		"type":              conv.Type,
		"name":              group.Name,
		"avatar":            group.Avatar,
		"creator_id":        group.CreatorID,
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
		"is_deleted":      conv.IsDeleted,
		"last_message_id": conv.LastMessageID,
		"last_message_at": conv.LastMessageAt,
		"created_at":      conv.CreatedAt,
		"updated_at":      conv.UpdatedAt,
	}

	response.Success(c, responseData)
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
		Muted bool `json:"muted"`
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

	var session model.ConversationSession
	result := db.Where("user_id = ? AND conversation_id = ?", userID, uint(convIDUint)).First(&session)
	if result.Error != nil {
		session = model.ConversationSession{
			UserID:         userID.(uint),
			ConversationID: uint(convIDUint),
			LastVisitedAt:  time.Now(),
		}
		db.Create(&session)
	}

	now := time.Now()
	session.IsHidden = true
	session.HiddenAt = &now
	if err := db.Save(&session).Error; err != nil {
		response.InternalServerError(c, "移除会话失败")
		return
	}

	response.Success(c, gin.H{
		"message": "已移除会话",
	})
}
