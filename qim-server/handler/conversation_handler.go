package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type conversationCursor struct {
	Pinned   bool   `json:"pinned"`
	Activity string `json:"activity"`
	ID       uint   `json:"id"`
}

func encodeConversationCursor(pinned bool, activity time.Time, id uint) (string, error) {
	payload, err := json.Marshal(conversationCursor{
		Pinned:   pinned,
		Activity: activity.UTC().Format(time.RFC3339Nano),
		ID:       id,
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeConversationCursor(encoded string) (conversationCursor, time.Time, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return conversationCursor{}, time.Time{}, err
	}
	var cursor conversationCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return conversationCursor{}, time.Time{}, err
	}
	activity, err := time.Parse(time.RFC3339Nano, cursor.Activity)
	if err != nil || cursor.ID == 0 {
		return conversationCursor{}, time.Time{}, fmt.Errorf("invalid cursor")
	}
	return cursor, activity, nil
}

// botConvIdentity 是 bot 会话的稳定显示身份，取自 bots 表（与 virtual_user 虚拟用户无关）。
type botConvIdentity struct {
	Name   string
	Avatar string
	BotID  uint
}

// resolveBotConvIdentities 批量解析 bot 会话（type=bot）的显示身份。
//
// bot 会话的对端身份应以 bots 表为准（bot_conversations.bot_id -> bots.id），
// 而不是从"虚拟用户成员"散点解析——后者在 bot 未设置 virtual_user_id（未作为成员加入会话）时
// 解析不出任何身份，导致同一会话在创建时与刷新列表时显示不一致（如自聊/无名空行）。
// 此 helper 供创建/列表/详情/搜索四处统一收敛。
func resolveBotConvIdentities(db *gorm.DB, convIDs []uint) map[uint]botConvIdentity {
	out := make(map[uint]botConvIdentity, len(convIDs))
	if len(convIDs) == 0 {
		return out
	}
	type row struct {
		ConversationID uint
		Name           string
		Avatar         string
		BotID          uint
	}
	var rows []row
	if err := db.Table("bot_conversations").
		Select("bot_conversations.conversation_id, bots.name, bots.avatar, bots.id AS bot_id").
		Joins("JOIN bots ON bots.id = bot_conversations.bot_id").
		Where("bot_conversations.conversation_id IN ?", convIDs).
		Scan(&rows).Error; err != nil {
		return out
	}
	for _, r := range rows {
		out[r.ConversationID] = botConvIdentity{Name: r.Name, Avatar: r.Avatar, BotID: r.BotID}
	}
	return out
}

// botConversationResponse 把 bot 会话序列化为带稳定显示身份的响应（name/avatar 取自 bots 表）。
// 用于创建 bot 会话的两条路径（req.BotID 与 recipient.Type=="bot"）的返回，保证"发起时"前端能立即
// processConversation 出正确的 bot 名，而不被 self-chat / 成员解析覆盖（与刷新列表的 GetConversations 一致）。
func botConversationResponse(conv model.Conversation, bot model.Bot) gin.H {
	resp := gin.H{
		"id":                conv.ID,
		"type":              conv.Type,
		"name":              bot.Name,
		"avatar":            bot.Avatar,
		"other_member_id":   nil,
		"other_member_name": bot.Name,
		"other_member_type": "bot",
		"is_deleted":        conv.IsDeleted,
		"last_message_id":   conv.LastMessageID,
		"last_message_at":   conv.LastMessageAt,
		"created_at":        conv.CreatedAt,
		"updated_at":        conv.UpdatedAt,
		"members":           conv.Members,
	}
	if bot.VirtualUserID != nil {
		resp["other_member_id"] = *bot.VirtualUserID
	}
	if conv.LastMessage != nil {
		resp["last_message"] = conv.LastMessage
	}
	return resp
}

func GetConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	// 解析分页参数
	page, _ := strconv.Atoi(c.Query("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// 按类型过滤（可选）：single / group / bot / discussion
	typeFilter := c.Query("type")

	cursorValue := c.Query("cursor")
	var cursor conversationCursor
	var cursorActivity time.Time
	if cursorValue != "" {
		var err error
		cursor, cursorActivity, err = decodeConversationCursor(cursorValue)
		if err != nil {
			response.BadRequest(c, "无效的分页游标")
			return
		}
	}

	db := database.GetDB()

	// 使用 JOIN 查询会话成员、会话信息和会话会话信息
	// 过滤掉已隐藏的会话，按置顶和最后消息时间排序
	type ConversationMemberWithMeta struct {
		model.ConversationMember
		LastMessageAt *time.Time
		CreatedAt     time.Time
		IsPinned      bool
		IsHidden      bool
	}

	var convMembersWithMeta []ConversationMemberWithMeta
	query := `
		SELECT 
			cm.*,
			c.last_message_at,
			c.created_at,
			COALESCE(cs.is_pinned, false) as is_pinned,
			COALESCE(cs.is_hidden, false) as is_hidden
		FROM conversation_members cm
		LEFT JOIN conversations c ON c.id = cm.conversation_id
		LEFT JOIN conversation_sessions cs ON cs.conversation_id = cm.conversation_id AND cs.user_id = ?
		WHERE cm.user_id = ? AND COALESCE(cs.is_hidden, false) = false
	`
	args := []interface{}{uid, uid}
	if typeFilter != "" {
		query += ` AND c.type = ?`
		args = append(args, typeFilter)
	}
	if cursorValue != "" {
		query += `
			AND (
				COALESCE(cs.is_pinned, false) < ?
				OR (COALESCE(cs.is_pinned, false) = ? AND COALESCE(c.last_message_at, c.created_at) < ?)
				OR (COALESCE(cs.is_pinned, false) = ? AND COALESCE(c.last_message_at, c.created_at) = ? AND cm.conversation_id < ?)
			)
		`
		args = append(args, cursor.Pinned, cursor.Pinned, cursorActivity, cursor.Pinned, cursorActivity, cursor.ID)
	}
	query += `
		ORDER BY is_pinned DESC, COALESCE(c.last_message_at, c.created_at) DESC, cm.conversation_id DESC
		LIMIT ?
	`
	if cursorValue == "" {
		query += " OFFSET ?"
		args = append(args, pageSize, offset)
	} else {
		args = append(args, pageSize+1)
	}
	db.Raw(query, args...).Scan(&convMembersWithMeta)

	// 查询总数（排除已隐藏的会话）
	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM conversation_members cm
		LEFT JOIN conversations c ON c.id = cm.conversation_id
		LEFT JOIN conversation_sessions cs ON cs.conversation_id = cm.conversation_id AND cs.user_id = ?
		WHERE cm.user_id = ? AND COALESCE(cs.is_hidden, false) = false
	`
	countArgs := []interface{}{uid, uid}
	if typeFilter != "" {
		countQuery += ` AND c.type = ?`
		countArgs = append(countArgs, typeFilter)
	}
	db.Raw(countQuery, countArgs...).Scan(&total)

	if len(convMembersWithMeta) == 0 {
		response.Success(c, gin.H{
			"list":        []interface{}{},
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"has_more":    false,
			"next_cursor": "",
		})
		return
	}

	hasMore := false
	if cursorValue != "" {
		hasMore = len(convMembersWithMeta) > pageSize
		if hasMore {
			convMembersWithMeta = convMembersWithMeta[:pageSize]
		}
	} else {
		hasMore = offset+len(convMembersWithMeta) < int(total)
	}

	nextCursor := ""
	if hasMore {
		last := convMembersWithMeta[len(convMembersWithMeta)-1]
		var err error
		activity := last.CreatedAt
		if last.LastMessageAt != nil {
			activity = *last.LastMessageAt
		}
		nextCursor, err = encodeConversationCursor(last.IsPinned, activity, last.ConversationID)
		if err != nil {
			response.InternalServerError(c, "生成分页游标失败")
			return
		}
	}

	// 提取会话成员记录和会话 ID
	var convMembers []model.ConversationMember
	conversationIDs := make([]uint, 0, len(convMembersWithMeta))
	isPinnedMap := make(map[uint]bool, len(convMembersWithMeta))
	for _, cmwm := range convMembersWithMeta {
		convMembers = append(convMembers, cmwm.ConversationMember)
		conversationIDs = append(conversationIDs, cmwm.ConversationID)
		isPinnedMap[cmwm.ConversationID] = cmwm.IsPinned
	}

	// 2. 批量查询会话信息
	conversationMap := make(map[uint]model.Conversation, len(conversationIDs))
	var convList []model.Conversation
	db.Where("id IN ?", conversationIDs).Find(&convList)
	for _, conv := range convList {
		conversationMap[conv.ID] = conv
	}

	// 3. 批量查询最后一条消息
	lastMessageIDs := make([]uint, 0, len(convList))
	for _, conv := range convList {
		if conv.LastMessageID != nil {
			lastMessageIDs = append(lastMessageIDs, *conv.LastMessageID)
		}
	}
	lastMessageMap := make(map[uint]model.Message, len(lastMessageIDs))
	if len(lastMessageIDs) > 0 {
		var lastMessages []model.Message
		db.Where("id IN ?", lastMessageIDs).Find(&lastMessages)
		for _, msg := range lastMessages {
			lastMessageMap[msg.ID] = msg
		}
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
		OtherMemberType  string    `json:"other_member_type,omitempty"`
	}

	groupConvIDs := make([]uint, 0, len(convMembers))
	singleConvIDs := make([]uint, 0, len(convMembers))
	for _, cm := range convMembers {
		conv := conversationMap[cm.ConversationID]
		if conv.Type == "group" || conv.Type == "discussion" {
			groupConvIDs = append(groupConvIDs, cm.ConversationID)
		} else if conv.Type == "single" || conv.Type == "bot" {
			singleConvIDs = append(singleConvIDs, cm.ConversationID)
		}
	}

	groupMap := make(map[uint]model.Group, len(groupConvIDs))
	if len(groupConvIDs) > 0 {
		var groups []model.Group
		db.Where("conversation_id IN ?", groupConvIDs).Find(&groups)
		for _, g := range groups {
			groupMap[g.ConversationID] = g
		}

		// 批量查询群聊成员（避免 N+1）
		var groupMembers []model.ConversationMember
		db.Where("conversation_id IN ?", groupConvIDs).Find(&groupMembers)

		// 提取群成员中的用户 ID
		groupMemberUserIDs := make([]uint, 0, len(groupMembers))
		seenGroupMemberUsers := make(map[uint]struct{})
		for _, gm := range groupMembers {
			if _, exists := seenGroupMemberUsers[gm.UserID]; !exists {
				groupMemberUserIDs = append(groupMemberUserIDs, gm.UserID)
				seenGroupMemberUsers[gm.UserID] = struct{}{}
			}
		}

		// 批量查询群成员用户信息
		groupMemberUserMap := make(map[uint]model.User, len(groupMemberUserIDs))
		if len(groupMemberUserIDs) > 0 {
			var groupMemberUsers []model.User
			db.Where("id IN ?", groupMemberUserIDs).Find(&groupMemberUsers)
			for _, u := range groupMemberUsers {
				groupMemberUserMap[u.ID] = u
			}
		}

		// 将成员信息按会话 ID 分组并填充 User 信息
		// 跳过用户已被软删除（查询不到有效 User）的成员
		groupMembersByConv := make(map[uint][]model.ConversationMember)
		for _, gm := range groupMembers {
			user, ok := groupMemberUserMap[gm.UserID]
			if !ok {
				continue
			}
			gm.User = user
			groupMembersByConv[gm.ConversationID] = append(groupMembersByConv[gm.ConversationID], gm)
		}

		// 将成员信息设置到会话中
		for convID, members := range groupMembersByConv {
			if conv, ok := conversationMap[convID]; ok {
				conv.Members = members
				conversationMap[convID] = conv
			}
		}
	}

	otherMemberMap := make(map[uint]uint, len(singleConvIDs))
	if len(singleConvIDs) > 0 {
		var otherMembers []model.ConversationMember
		db.Where("conversation_id IN ? AND user_id != ?", singleConvIDs, uid).Find(&otherMembers)
		for _, om := range otherMembers {
			otherMemberMap[om.ConversationID] = om.UserID
		}

		for _, convID := range singleConvIDs {
			conv := conversationMap[convID]
			if conv.Type == "single" {
				if _, ok := otherMemberMap[convID]; !ok {
					otherMemberMap[convID] = uid
				}
			}
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
		allConvIDs = append(allConvIDs, cm.ConversationID)
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

	// 预解析本页 bot 会话的稳定显示身份（取自 bots 表，与 virtual_user 无关）。
	// 无 virtual_user 的 bot 未作为成员加入会话，成员解析无法得到对端身份，故统一走此来源，
	// 保证与创建时（botConversationResponse）返回的 name/avatar 一致。
	var botConvIDs []uint
	for _, cm := range convMembers {
		if conversationMap[cm.ConversationID].Type == "bot" {
			botConvIDs = append(botConvIDs, cm.ConversationID)
		}
	}
	botIdentityMap := resolveBotConvIdentities(db, botConvIDs)

	for _, cm := range convMembers {
		convID := cm.ConversationID
		conv := conversationMap[convID]

		// 设置最后一条消息
		if conv.LastMessageID != nil {
			if msg, ok := lastMessageMap[*conv.LastMessageID]; ok {
				conv.LastMessage = &msg
			}
		}

		// 确保 session 存在
		if _, exists := sessionMap[convID]; !exists {
			session := model.ConversationSession{
				UserID:         uid,
				ConversationID: convID,
				IsPinned:       isPinnedMap[convID],
				LastVisitedAt:  now,
			}
			sessionsToCreate = append(sessionsToCreate, session)
		}

		convWithPin := ConversationWithPin{
			Conversation: conv,
			IsPinned:     isPinnedMap[convID],
			UnreadCount:  cm.UnreadCount,
			Muted:        cm.Muted,
		}

		if conv.Type == "group" || conv.Type == "discussion" {
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

		if conv.Type == "single" || conv.Type == "bot" {
			if conv.Type == "bot" {
				// bot 会话身份固定取自 bots 表，不依赖虚拟用户成员，保证创建/列表一致。
				if bi, ok := botIdentityMap[convID]; ok {
					convWithPin.Name = bi.Name
					convWithPin.Avatar = bi.Avatar
					convWithPin.OtherMemberName = bi.Name
					convWithPin.OtherMemberType = "bot"
					convWithPin.OtherMemberID = 0
				}
			} else if otherUserID, ok := otherMemberMap[convID]; ok {
				if otherUser, ok := userMap[otherUserID]; ok {
					convWithPin.IP = otherUser.IP
					convWithPin.Status = otherUser.Status
					convWithPin.Signature = otherUser.Signature
					convWithPin.OtherMemberID = otherUser.ID
					convWithPin.OtherMemberName = otherUser.Nickname
					convWithPin.OtherMemberType = otherUser.Type
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

	// 返回分页数据
	response.Success(c, gin.H{
		"list":        conversations,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
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

	// 过滤掉用户已被软删除（关联 User 查询不到）的成员
	activeMembers := make([]model.ConversationMember, 0, len(conv.Members))
	for _, m := range conv.Members {
		if m.User.ID != 0 {
			activeMembers = append(activeMembers, m)
		}
	}
	conv.Members = activeMembers

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

	// 对于单聊和 Bot 会话，从对方成员（虚拟用户）获取名称、头像等信息
	if conv.Type == "single" || conv.Type == "bot" {
		if conv.Type == "bot" {
			// bot 会话身份固定取自 bots 表，不依赖虚拟用户成员，保证与列表/创建一致。
			if bi, ok := resolveBotConvIdentities(db, []uint{conv.ID})[conv.ID]; ok {
				response.Success(c, gin.H{
					"id":                conv.ID,
					"type":              conv.Type,
					"name":              bi.Name,
					"avatar":            bi.Avatar,
					"other_member_id":   0,
					"other_member_name": bi.Name,
					"other_member_type": "bot",
					"signature":         "",
					"status":            "",
					"is_deleted":        conv.IsDeleted,
					"last_message_id":   conv.LastMessageID,
					"last_message_at":   conv.LastMessageAt,
					"created_at":        conv.CreatedAt,
					"updated_at":        conv.UpdatedAt,
					"members":           conv.Members,
				})
				return
			}
		}
		for _, m := range conv.Members {
			if m.UserID != userID.(uint) && m.UserID > 0 {
				responseData := gin.H{
					"id":                conv.ID,
					"type":              conv.Type,
					"name":              m.User.Nickname,
					"avatar":            m.User.Avatar,
					"other_member_id":   m.User.ID,
					"other_member_name": m.User.Nickname,
					"signature":         m.User.Signature,
					"status":            m.User.Status,
					"is_deleted":        conv.IsDeleted,
					"last_message_id":   conv.LastMessageID,
					"last_message_at":   conv.LastMessageAt,
					"created_at":        conv.CreatedAt,
					"updated_at":        conv.UpdatedAt,
					"members":           conv.Members,
				}
				response.Success(c, responseData)
				return
			}
		}
	}

	response.Success(c, conv)
}

// SearchConversations 搜索当前用户参与的会话（按群名或单聊对方昵称匹配）
func SearchConversations(c *gin.Context) {
	userID, _ := c.Get("user_id")
	uid := userID.(uint)

	query := c.Query("query")
	if strings.TrimSpace(query) == "" {
		response.Success(c, []interface{}{})
		return
	}

	db := database.GetDB()
	keyword := "%" + query + "%"

	// 查询当前用户参与的会话
	var userConvMembers []model.ConversationMember
	db.Where("user_id = ?", uid).Find(&userConvMembers)
	if len(userConvMembers) == 0 {
		response.Success(c, []interface{}{})
		return
	}

	convIDs := make([]uint, 0, len(userConvMembers))
	for _, cm := range userConvMembers {
		convIDs = append(convIDs, cm.ConversationID)
	}

	var convs []model.Conversation
	db.Where("id IN ?", convIDs).Find(&convs)

	// 区分群聊、bot 会话和单聊
	groupConvIDs := make([]uint, 0)
	singleConvIDs := make([]uint, 0)
	botConvIDs := make([]uint, 0)
	for _, conv := range convs {
		if conv.Type == "group" || conv.Type == "discussion" {
			groupConvIDs = append(groupConvIDs, conv.ID)
		} else if conv.Type == "bot" {
			botConvIDs = append(botConvIDs, conv.ID)
		} else if conv.Type == "single" {
			singleConvIDs = append(singleConvIDs, conv.ID)
		}
	}

	results := make([]gin.H, 0)

	// 群聊：按 Group.Name 匹配
	if len(groupConvIDs) > 0 {
		var groups []model.Group
		db.Where("conversation_id IN ? AND name LIKE ?", groupConvIDs, keyword).Find(&groups)
		for _, g := range groups {
			results = append(results, gin.H{
				"id":     g.ConversationID,
				"type":   "group",
				"name":   g.Name,
				"avatar": g.Avatar,
			})
		}
	}

	// bot 会话：按 bots 表名称匹配（身份固定取自 bots 表，不依赖虚拟用户成员）
	if len(botConvIDs) > 0 {
		lowerQuery := strings.ToLower(query)
		for convID, bi := range resolveBotConvIdentities(db, botConvIDs) {
			if strings.Contains(strings.ToLower(bi.Name), lowerQuery) {
				results = append(results, gin.H{
					"id":     convID,
					"type":   "bot",
					"name":   bi.Name,
					"avatar": bi.Avatar,
				})
			}
		}
	}

	// 单聊：按对方用户昵称/用户名匹配
	if len(singleConvIDs) > 0 {
		var members []model.ConversationMember
		db.Where("conversation_id IN ? AND user_id != ?", singleConvIDs, uid).Find(&members)

		convToOtherUserID := make(map[uint]uint, len(members))
		otherUserIDs := make([]uint, 0, len(members))
		for _, m := range members {
			convToOtherUserID[m.ConversationID] = m.UserID
			otherUserIDs = append(otherUserIDs, m.UserID)
		}

		if len(otherUserIDs) > 0 {
			var users []model.User
			db.Where("id IN ? AND (nickname LIKE ? OR username LIKE ?)", otherUserIDs, keyword, keyword).Find(&users)

			matchedUsers := make(map[uint]model.User, len(users))
			for _, u := range users {
				matchedUsers[u.ID] = u
			}

			for convID, otherUID := range convToOtherUserID {
				if u, ok := matchedUsers[otherUID]; ok {
					results = append(results, gin.H{
						"id":     convID,
						"type":   "single",
						"name":   u.Nickname,
						"avatar": u.Avatar,
					})
				}
			}
		}
	}

	response.Success(c, results)
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
		if !ensureHumanConversationInitiator(c, db, userID.(uint)) {
			return
		}

		var bot model.Bot
		if err := db.First(&bot, *req.BotID).Error; err != nil {
			response.NotFound(c, "机器人不存在")
			return
		}

		var botConv model.BotConversation
		findBotSingleConversation(db, *req.BotID, userID.(uint), &botConv)

		if botConv.ID > 0 {
			ensureBotConversationMember(db, botConv.ConversationID, bot.VirtualUserID)
			db.Preload("Conversation.Members").Preload("Conversation.Members.User").
				First(&botConv, botConv.ID)
			response.Success(c, botConversationResponse(botConv.Conversation, bot))
			return
		}

		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		conv := model.Conversation{Type: "bot"}
		if err := tx.Create(&conv).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
		if bot.VirtualUserID != nil {
			if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: *bot.VirtualUserID, Role: "member"}).Error; err != nil {
				tx.Rollback()
				response.InternalServerError(c, "创建会话失败")
				return
			}
		}

		if err := tx.Create(&model.BotConversation{
			BotID:          *req.BotID,
			ConversationID: conv.ID,
		}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
		response.Success(c, botConversationResponse(conv, bot))
		return
	}

	if req.UserID == 0 {
		response.BadRequest(c, "缺少必要参数")
		return
	}

	// 检查接收方用户类型，bot 类型用户自动创建 bot 会话
	var recipient model.User
	if err := db.First(&recipient, req.UserID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}
	if recipient.Type == "bot" {
		if !ensureHumanConversationInitiator(c, db, userID.(uint)) {
			return
		}

		var bot model.Bot
		if err := db.Where("virtual_user_id = ?", req.UserID).First(&bot).Error; err != nil {
			response.NotFound(c, "机器人不存在")
			return
		}
		if !bot.IsActive {
			response.Forbidden(c, "机器人未启用")
			return
		}

		// 查找已有 bot 会话
		var botConv model.BotConversation
		findBotSingleConversation(db, bot.ID, userID.(uint), &botConv)

		if botConv.ID > 0 {
			// 恢复会话显示
			db.Model(&model.ConversationSession{}).
				Where("user_id = ? AND conversation_id = ? AND is_hidden = ?", userID.(uint), botConv.ConversationID, true).
				Update("is_hidden", false)
			ensureBotConversationMember(db, botConv.ConversationID, bot.VirtualUserID)
			db.Preload("Conversation.Members").Preload("Conversation.Members.User").
				First(&botConv, botConv.ID)
			response.Success(c, botConversationResponse(botConv.Conversation, bot))
			return
		}

		// 创建新的 bot 会话
		tx := db.Begin()
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		conv := model.Conversation{Type: "bot"}
		if err := tx.Create(&conv).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
		if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
		if bot.VirtualUserID != nil {
			if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: *bot.VirtualUserID, Role: "member"}).Error; err != nil {
				tx.Rollback()
				response.InternalServerError(c, "创建会话失败")
				return
			}
		}
		if err := tx.Create(&model.BotConversation{
			BotID:          bot.ID,
			ConversationID: conv.ID,
		}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
		response.Success(c, botConversationResponse(conv, bot))
		return
	}

	var existingConv model.Conversation
	existingQuery := db.Model(&model.Conversation{}).
		Select("conversations.*").
		Joins("JOIN conversation_members cm ON cm.conversation_id = conversations.id").
		Where("conversations.type = ?", "single").
		Group("conversations.id")

	if userID.(uint) == req.UserID {
		existingQuery = existingQuery.Having(
			"COUNT(cm.id) = 1 AND SUM(CASE WHEN cm.user_id = ? THEN 1 ELSE 0 END) = 1",
			userID.(uint),
		)
	} else {
		existingQuery = existingQuery.Having(
			"COUNT(cm.id) = 2 AND SUM(CASE WHEN cm.user_id = ? THEN 1 ELSE 0 END) = 1 AND SUM(CASE WHEN cm.user_id = ? THEN 1 ELSE 0 END) = 1",
			userID.(uint),
			req.UserID,
		)
	}

	if err := existingQuery.Scan(&existingConv).Error; err != nil {
		response.InternalServerError(c, "查询会话失败")
		return
	}

	if existingConv.ID > 0 {
		// 恢复会话显示：用户主动发起聊天时，如果会话被隐藏则恢复显示
		if err := db.Model(&model.ConversationSession{}).
			Where("user_id = ? AND conversation_id = ? AND is_hidden = ?", userID.(uint), existingConv.ID, true).
			Update("is_hidden", false).Error; err != nil {
			response.InternalServerError(c, "恢复会话失败")
			return
		}
		if err := db.Preload("Members").Preload("Members.User").First(&existingConv, existingConv.ID).Error; err != nil {
			response.InternalServerError(c, "加载会话信息失败")
			return
		}

		response.Success(c, existingConv)
		return
	}

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if userID.(uint) == req.UserID {
		conv := model.Conversation{
			Type: "single",
		}
		if err := tx.Create(&conv).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}

		db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
		response.Success(c, conv)
		return
	}

	conv := model.Conversation{
		Type: "single",
	}
	if err := tx.Create(&conv).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "member"}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: req.UserID, Role: "member"}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	// 默认对接收方隐藏会话，直到发起方发送第一条消息后才恢复显示
	// 这样接收方不会在未收到任何消息前就看到空会话
	if err := tx.Create(&model.ConversationSession{
		UserID:         req.UserID,
		ConversationID: conv.ID,
		IsHidden:       true,
		LastVisitedAt:  time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

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

	if !ensureHumanConversationInitiator(c, db, userID.(uint)) {
		return
	}

	// 检查 Bot 是否存在
	var bot model.Bot
	if err := db.First(&bot, req.BotID).Error; err != nil {
		response.NotFound(c, "机器人不存在")
		return
	}

	// 检查机器人是否已启用（审批通过）
	if !bot.IsActive {
		response.Forbidden(c, "机器人未启用，请等待管理员审批")
		return
	}

	// 查找是否已有会话
	var botConv model.BotConversation
	findBotSingleConversation(db, req.BotID, userID.(uint), &botConv)

	if botConv.ID > 0 {
		ensureBotConversationMember(db, botConv.ConversationID, bot.VirtualUserID)
		db.Preload("Conversation.Members").Preload("Conversation.Members.User").
			First(&botConv, botConv.ID)
		response.Success(c, botConversationResponse(botConv.Conversation, bot))
		return
	}

	// 创建新会话
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	conv := model.Conversation{Type: "bot"}
	if err := tx.Create(&conv).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	if err := tx.Create(&model.ConversationMember{
		ConversationID: conv.ID,
		UserID:         userID.(uint),
		Role:           "member",
	}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}
	if bot.VirtualUserID != nil {
		if err := tx.Create(&model.ConversationMember{
			ConversationID: conv.ID,
			UserID:         *bot.VirtualUserID,
			Role:           "member",
		}).Error; err != nil {
			tx.Rollback()
			response.InternalServerError(c, "创建会话失败")
			return
		}
	}

	if err := tx.Create(&model.BotConversation{
		BotID:          bot.ID,
		ConversationID: conv.ID,
	}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	db.Preload("Members").Preload("Members.User").First(&conv, conv.ID)
	response.Success(c, botConversationResponse(conv, bot))
}

func ensureHumanConversationInitiator(c *gin.Context, db *gorm.DB, userID uint) bool {
	var currentUser model.User
	if err := db.First(&currentUser, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return false
	}

	// 正向判断：只拦截明确的非人类类型，新增用户类型时无需修改此处
	switch currentUser.Type {
	case "bot", "system", "api":
		response.Forbidden(c, "机器人或系统用户不能创建机器人会话")
		return false
	}

	return true
}

func ensureBotConversationMember(db *gorm.DB, conversationID uint, virtualUserID *uint) {
	if db == nil || virtualUserID == nil || *virtualUserID == 0 {
		return
	}

	db.FirstOrCreate(&model.ConversationMember{}, model.ConversationMember{
		ConversationID: conversationID,
		UserID:         *virtualUserID,
		Role:           "member",
	})
}

// findBotSingleConversation 反查某用户对该 bot 的 1:1 bot 会话。
// 原实现按 BotConversation.user_id 反查，去掉该列后改用
// ConversationMember(含该 user) + Conversation.Type=bot join 等价替代。
func findBotSingleConversation(db *gorm.DB, botID, userID uint, out *model.BotConversation) {
	db.
		Joins("JOIN conversations c ON c.id = bot_conversations.conversation_id").
		Joins("JOIN conversation_members cm ON cm.conversation_id = c.id").
		Where("bot_conversations.bot_id = ? AND c.type = ? AND cm.user_id = ?", botID, "bot", userID).
		Preload("Conversation").
		First(out)
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

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	conv := model.Conversation{
		Type: "group",
	}
	if err := tx.Create(&conv).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	// 创建群聊记录
	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "group",
		Name:             req.Name,
		Avatar:           req.Avatar,
		CreatorID:        userID.(uint),
		InvitePermission: "owner_admin",
	}
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建群聊失败")
		return
	}

	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner", JoinedAt: time.Now()}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "添加成员失败")
		return
	}

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member", JoinedAt: time.Now()}).Error; err != nil {
				tx.Rollback()
				response.InternalServerError(c, "添加成员失败")
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建群聊失败")
		return
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

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	conv := model.Conversation{
		Type: "discussion",
	}
	if err := tx.Create(&conv).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建会话失败")
		return
	}

	group := model.Group{
		ConversationID:   conv.ID,
		GroupType:        "discussion",
		Name:             req.Name,
		Avatar:           req.Avatar,
		CreatorID:        userID.(uint),
		InvitePermission: "owner_admin",
	}
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建讨论组失败")
		return
	}

	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID.(uint), Role: "owner", JoinedAt: time.Now()}).Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "添加成员失败")
		return
	}

	for _, mid := range req.MemberIDs {
		if mid != userID.(uint) {
			if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: mid, Role: "member", JoinedAt: time.Now()}).Error; err != nil {
				tx.Rollback()
				response.InternalServerError(c, "添加成员失败")
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		response.InternalServerError(c, "创建讨论组失败")
		return
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

	// 校验当前用户是会话成员，防止对未参与的会话写入 session 记录或探测会话是否存在
	var memberCount int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ? AND user_id = ?", uint(convIDUint), userID).Count(&memberCount)
	if memberCount == 0 {
		response.Forbidden(c, "您不是该会话的成员")
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

// CreateConversation 统一会话创建入口
// 请求体：
//
//	{
//	  "type": "single|bot|group|discussion",
//	  ...具体类型对应的字段
//	}
//
// 该 handler 仅作为分发，复用原有具体类型的 handler 逻辑。
func CreateConversation(c *gin.Context) {
	// 预读取 type
	body, err := c.GetRawData()
	if err != nil {
		response.BadRequest(c, "读取请求体失败")
		return
	}

	// 回写 body 供后续 ShouldBindJSON 使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &head); err != nil || head.Type == "" {
		response.BadRequest(c, "缺少 type 字段")
		return
	}

	switch head.Type {
	case "single":
		CreateSingleConversation(c)
	case "bot":
		CreateBotConversation(c)
	case "group":
		CreateGroupConversation(c)
	case "discussion":
		CreateDiscussionConversation(c)
	default:
		response.BadRequest(c, "不支持的会话类型: "+head.Type)
	}
}
