package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/utils"
	"github.com/dshmyz/qim/qim-server/ws"

	"gorm.io/gorm"
)

var ErrMessageNotFound = errors.New("message not found")
var ErrMessageForbidden = errors.New("access forbidden")
var ErrMessageAlreadyRecalled = errors.New("message already recalled")
var ErrMessageRecallTimeout = errors.New("message recall timeout")
var ErrSensitiveWordBlocked = errors.New("message contains sensitive words")
var ErrAtAllForbidden = errors.New("only owner or admin can @all")

type MessageService struct {
	db  *gorm.DB
	hub *ws.Hub

	aiService            *ai.AIService
	sensitiveWordCache   []model.SensitiveWord
	sensitiveWordCacheMu sync.RWMutex
	sensitiveWordLoaded  bool
}

func NewMessageService(db *gorm.DB, hub *ws.Hub, aiService *ai.AIService) *MessageService {
	return &MessageService{
		db:        db,
		hub:       hub,
		aiService: aiService,
	}
}

func (s *MessageService) SetAIService(aiService *ai.AIService) {
	s.aiService = aiService
}

func (s *MessageService) loadSensitiveWords() {
	var words []model.SensitiveWord
	if err := s.db.Where("enabled = ?", true).Find(&words).Error; err == nil {
		s.sensitiveWordCacheMu.Lock()
		s.sensitiveWordCache = words
		s.sensitiveWordLoaded = true
		s.sensitiveWordCacheMu.Unlock()
	}
}

func (s *MessageService) RefreshSensitiveWordCache() {
	s.loadSensitiveWords()
}

func (s *MessageService) CheckSensitiveContent(content string) (bool, []string) {
	s.sensitiveWordCacheMu.RLock()
	loaded := s.sensitiveWordLoaded
	cache := s.sensitiveWordCache
	s.sensitiveWordCacheMu.RUnlock()

	if !loaded {
		s.loadSensitiveWords()
		s.sensitiveWordCacheMu.RLock()
		cache = s.sensitiveWordCache
		s.sensitiveWordCacheMu.RUnlock()
	}

	found := []string{}
	for _, word := range cache {
		if strings.Contains(content, word.Word) {
			found = append(found, word.Word)
		}
	}
	return len(found) > 0, found
}

type MessageQuery struct {
	ConvID      uint
	UserID      uint
	BeforeMsgID uint
	AfterMsgID  uint
	Limit       int
	Offset      int
	MessageType string
	Keyword     string
	StartDate   string
	EndDate     string
}

type MessageResult struct {
	Messages    []model.Message
	Total       int64
	TotalPages  int
	CurrentPage int
	PageSize    int
}

func (s *MessageService) SendMessage(convID, senderID uint, msgType, content string, quotedMessageID *uint) (*model.Message, error) {
	db := s.db

	if msgType == "text" && content != "" {
		if contains, words := s.CheckSensitiveContent(content); contains {
			return nil, fmt.Errorf("%w: %v", ErrSensitiveWordBlocked, words)
		}
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, senderID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	// 解析 content 中的 @ mention token（content 是唯一事实源）
	mentions := mention.Parse(content)

	// @all 权限校验：仅群主/管理员可 @all
	if mention.IsAllMentioned(mentions) {
		if member.Role != "owner" && member.Role != "admin" {
			return nil, ErrAtAllForbidden
		}
	}

	msg := model.Message{
		ConversationID:  convID,
		SenderID:        senderID,
		Type:            msgType,
		Content:         content,
		QuotedMessageID: quotedMessageID,
		IsRead:          false,
	}
	if err := db.Create(&msg).Error; err != nil {
		return nil, err
	}

	// 优化：单次预加载而非 3 次 Preload 调用
	db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msg.ID)

	now := time.Now()
	// 优化：合并查会话 + 更新会话为单次 UPDATE
	result := db.Exec("UPDATE conversations SET last_message_id = ?, last_message_at = ? WHERE id = ?", msg.ID, now, convID)
	if result.Error != nil {
		return nil, result.Error
	}

	// 获取会话类型用于判断 bot/正常
	var convType string
	db.Model(&model.Conversation{}).Where("id = ?", convID).Select("type").First(&convType)

	if convType == "bot" {
		utils.SafeGo(func() { s.handleBotMessage(senderID, convID, content) })
	} else {
		// 恢复会话显示：新消息到来时，如果会话被隐藏则恢复显示
		db.Model(&model.ConversationSession{}).
			Where("conversation_id = ? AND is_hidden = ?", convID, true).
			Update("is_hidden", false)

		db.Model(&model.ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, senderID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		// 计算被提及的用户 ID（@all 展开为全体成员，排除发送者）
		var allMembers []model.ConversationMember
		db.Where("conversation_id = ?", convID).Find(&allMembers)
		allMemberIDs := make([]uint, 0, len(allMembers))
		for _, m := range allMembers {
			allMemberIDs = append(allMemberIDs, m.UserID)
		}
		mentionUserIDs := mention.ExtractUserIDs(mentions, allMemberIDs, senderID)

		// 更新被提及成员的未读 @ 计数
		if len(mentionUserIDs) > 0 {
			db.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id IN ?", convID, mentionUserIDs).
				UpdateColumn("unread_at_mention_count", gorm.Expr("unread_at_mention_count + 1"))
		}

		if s.hub != nil {
			// 广播（mention_user_ids 数组随消息发送，前端据此算 is_at_mention）
			s.broadcastMessage(&msg, mentionUserIDs, senderID)
		}

		// 触发 AI / 分身（@all 不触发 AI，避免噪音）
		if s.hub != nil && s.hub.OnMessageSent != nil && !mention.IsAllMentioned(mentions) {
			mentionIDsForAI := mentionUserIDs
			utils.SafeGo(func() { s.hub.OnMessageSent(senderID, convID, content, mentionIDsForAI) })
		}
	}

	return &msg, nil
}

func (s *MessageService) handleBotMessage(userID, convID uint, content string) {
	db := s.db

	var botConv model.BotConversation
	if err := db.Where("conversation_id = ?", convID).First(&botConv).Error; err != nil {
		return
	}

	var bot model.Bot
	if err := db.First(&bot, botConv.BotID).Error; err != nil {
		return
	}

	var messages []model.Message
	db.Where("conversation_id = ?", convID).Order("created_at ASC").Limit(20).Find(&messages)

	systemUserID := s.GetSystemUserID()

	aiMessages := make([]ai.Message, 0, len(messages))
	for _, msg := range messages {
		role := "user"
		if msg.SenderID == systemUserID {
			role = "assistant"
		}
		aiMessages = append(aiMessages, ai.Message{
			Role:    role,
			Content: msg.Content,
		})
	}

	utils.SafeGo(func() {
		var builder strings.Builder
		done := make(chan struct{})
		var streamErr error

		go func() {
			streamErr = s.aiService.GetCompletionStream(ai.TaskTypeChat, aiMessages, func(chunk ai.StreamChunk) error {
				builder.WriteString(chunk.Content)
				return nil
			})
			close(done)
		}()

		// 超时保护：AI 调用最长 60 秒
		select {
		case <-done:
			// 正常完成
		case <-time.After(60 * time.Second):
			streamErr = fmt.Errorf("AI 响应超时")
		}

		response := builder.String()
		if streamErr != nil {
			response = "抱歉，AI 服务暂时不可用，请稍后再试。"
		}

		senderID := s.GetSystemUserID()
		botReply := model.Message{
			ConversationID: convID,
			SenderID:       senderID,
			Type:           "markdown",
			Content:        response,
		}
		db.Create(&botReply)
	})
}

func (s *MessageService) GetMessages(query MessageQuery) (*MessageResult, error) {
	db := s.db

	if query.Limit <= 0 {
		query.Limit = 20
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		var count int64
		db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ?", query.ConvID, query.UserID).Count(&count)
		if count == 0 {
			return nil, ErrMessageForbidden
		}
	}

	var total int64
	db.Model(&model.Message{}).Where("conversation_id = ?", query.ConvID).Count(&total)

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	var messages []model.Message
	q := db.Where("conversation_id = ?", query.ConvID)

	if query.AfterMsgID > 0 {
		var afterMsg model.Message
		if err := db.First(&afterMsg, query.AfterMsgID).Error; err != nil {
			return &MessageResult{
				Messages:    []model.Message{},
				Total:       0,
				TotalPages:  0,
				CurrentPage: 1,
				PageSize:    query.Limit,
			}, nil
		}
		q = q.Where("created_at > ?", afterMsg.CreatedAt).Order("created_at ASC").Limit(query.Limit)
		q.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Find(&messages)
		return &MessageResult{
			Messages:    messages,
			Total:       int64(len(messages)),
			TotalPages:  1,
			CurrentPage: 1,
			PageSize:    query.Limit,
		}, nil
	}

	if query.BeforeMsgID > 0 {
		var beforeMsg model.Message
		if err := db.First(&beforeMsg, query.BeforeMsgID).Error; err == nil {
			q = q.Where("created_at < ?", beforeMsg.CreatedAt)
		}
	}

	// DESC 查最新的 N 条，再翻转为正序（BeforeMsgID 游标分页需要 DESC）
	q.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) GetMessagesByFilter(query MessageQuery) (*MessageResult, error) {
	db := s.db

	if query.Limit <= 0 {
		query.Limit = 10
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", query.ConvID, query.UserID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	dbQuery := db.Where("conversation_id = ?", query.ConvID)

	if query.MessageType != "" {
		dbQuery = dbQuery.Where("type = ?", query.MessageType)
	}

	// 优化：使用全文索引搜索
	if query.Keyword != "" {
		if database.D.SupportsFulltext() {
			dbQuery = dbQuery.Where("MATCH(content) AGAINST(? IN BOOLEAN MODE)", query.Keyword)
		} else {
			// SQLite / TiDB 降级：LIKE 搜索
			dbQuery = dbQuery.Where("content LIKE ?", "%"+query.Keyword+"%")
		}
	}

	if query.StartDate != "" {
		dbQuery = dbQuery.Where("created_at >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		dbQuery = dbQuery.Where("created_at <= ?", query.EndDate+" 23:59:59")
	}

	var total int64
	dbQuery.Model(&model.Message{}).Count(&total)

	var messages []model.Message
	// DESC 查最新的 N 条，再翻转为正序（游标分页需要 DESC）
	dbQuery.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").Order("created_at DESC").Limit(query.Limit).Offset(query.Offset).Find(&messages)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	totalPages := int(total) / query.Limit
	if int(total)%query.Limit > 0 {
		totalPages++
	}

	return &MessageResult{
		Messages:    messages,
		Total:       total,
		TotalPages:  totalPages,
		CurrentPage: query.Offset/query.Limit + 1,
		PageSize:    query.Limit,
	}, nil
}

func (s *MessageService) SearchMessages(userID uint, keyword string, convID *uint, limit, offset int) ([]model.Message, error) {
	db := s.db

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	query := db.Model(&model.Message{}).Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").Where("conversation_members.user_id = ?", userID)

	if keyword != "" {
		// 优化：使用全文索引搜索
		if database.D.SupportsFulltext() {
			query = query.Where("MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", keyword)
		} else {
			// SQLite / TiDB 降级：LIKE 搜索
			query = query.Where("messages.content LIKE ?", "%"+keyword+"%")
		}
	}

	if convID != nil {
		query = query.Where("messages.conversation_id = ?", *convID)
	}

	var messages []model.Message
	if err := query.Preload("Sender").Preload("Conversation").Preload("Conversation.Members").Preload("Conversation.Members.User").Order("messages.created_at DESC").Limit(limit).Offset(offset).Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *MessageService) RecallMessage(msgID, userID uint) (*model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	if msg.SenderID != userID {
		return nil, ErrMessageForbidden
	}

	if msg.IsRecalled {
		return nil, ErrMessageAlreadyRecalled
	}

	configSvc := NewSystemConfigService(db)
	publicConfigs, err := configSvc.GetPublicConfigs()
	if err == nil {
		recallTimeLimit := 120
		if v, ok := publicConfigs["messageRecallTime"]; ok {
			if iv, ok := v.(int); ok {
				recallTimeLimit = iv
			}
		}
		if recallTimeLimit == 0 {
			return nil, ErrMessageRecallTimeout
		}
		if time.Since(msg.CreatedAt) > time.Duration(recallTimeLimit)*time.Second {
			return nil, ErrMessageRecallTimeout
		}
	}

	msg.IsRecalled = true
	msg.Content = "[消息已撤回]"
	now := time.Now()
	msg.RecalledAt = &now
	if err := db.Save(&msg).Error; err != nil {
		return nil, err
	}

	db.Preload("Sender").First(&msg, msg.ID)

	if s.hub != nil {
		recallMsg := ws.WSMessage{
			Type: "message_recalled",
			Data: msg,
		}
		jsonMsg, _ := json.Marshal(recallMsg)
		s.hub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return &msg, nil
}

func (s *MessageService) DeleteMessage(msgID, userID uint) error {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMessageNotFound
		}
		return err
	}

	if msg.SenderID != userID {
		return ErrMessageForbidden
	}

	if err := db.Delete(&msg).Error; err != nil {
		return err
	}

	if s.hub != nil {
		deleteMsg := ws.WSMessage{
			Type: "message_deleted",
			Data: map[string]interface{}{
				"message_id":      msg.ID,
				"conversation_id": msg.ConversationID,
			},
		}
		jsonMsg, _ := json.Marshal(deleteMsg)
		s.hub.SendToConversation(msg.ConversationID, 0, jsonMsg)
	}

	return nil
}

func (s *MessageService) MarkAsRead(convID, userID uint) error {
	db := s.db

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&member).Error; err != nil {
		return ErrMessageForbidden
	}

	// 无未读消息则跳过，避免无效的 INSERT SELECT 和 UPDATE
	if member.UnreadCount == 0 {
		return nil
	}

	if database.D.Type() == "mysql" {
		db.Exec(`
			INSERT IGNORE INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ? AND is_read = false
		`, convID, userID, time.Now(), convID, userID)
	} else {
		db.Exec(`
			INSERT INTO message_read_receipts (message_id, conversation_id, user_id, created_at)
			SELECT id, ?, ?, ?
			FROM messages
			WHERE conversation_id = ? AND sender_id != ? AND is_read = false
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, convID, userID, time.Now(), convID, userID)
	}

	result := db.Model(&model.Message{}).
		Where("conversation_id = ? AND sender_id != ? AND is_read = false", convID, userID).
		UpdateColumn("is_read", true)

	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumn("unread_count", 0)

	now := time.Now()
	db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		UpdateColumn("last_read_at", now)

	if result.RowsAffected > 0 {
		s.notifyMessageRead(convID, userID)
	}

	return nil
}

func (s *MessageService) notifyMessageRead(convID, userID uint) {
	if s.hub == nil {
		return
	}

	db := s.db

	var conv model.Conversation
	if err := db.First(&conv, convID).Error; err != nil {
		return
	}

	readMsg := ws.WSMessage{
		Type: "message_read",
		Data: map[string]interface{}{
			"conversation_id": convID,
			"user_id":         userID,
			"timestamp":       time.Now().Unix(),
		},
	}
	jsonMsg, _ := json.Marshal(readMsg)

	if conv.Type == "single" {
		var otherMember model.ConversationMember
		db.Where("conversation_id = ? AND user_id != ?", convID, userID).First(&otherMember)
		s.hub.SendToUser(otherMember.UserID, jsonMsg)
	} else if conv.Type == "group" {
		s.hub.SendToConversation(convID, userID, jsonMsg)
	}
}

func (s *MessageService) GetMessageByID(msgID uint) (*model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return &msg, nil
}

func (s *MessageService) GetMessageQuoteChain(msgID, userID uint) ([]model.Message, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, ErrMessageForbidden
	}

	var quoteChain []model.Message
	currentMsg := msg

	for i := 0; i < 3 && currentMsg.QuotedMessageID != nil; i++ {
		var quotedMsg model.Message
		if err := db.Preload("Sender").Preload("QuotedMessage").Preload("QuotedMessage.Sender").First(&quotedMsg, *currentMsg.QuotedMessageID).Error; err == nil {
			quoteChain = append(quoteChain, quotedMsg)
			currentMsg = quotedMsg
		} else {
			break
		}
	}

	return quoteChain, nil
}

func (s *MessageService) GetMessageReadUsers(msgID, userID uint) ([]model.User, int64, error) {
	db := s.db

	var msg model.Message
	if err := db.First(&msg, msgID).Error; err != nil {
		return nil, 0, ErrMessageNotFound
	}

	var member model.ConversationMember
	if err := db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
		return nil, 0, ErrMessageForbidden
	}

	var readReceipts []model.MessageReadReceipt
	if err := db.Where("message_id = ?", msgID).Preload("User").Order("created_at DESC").Find(&readReceipts).Error; err != nil {
		return nil, 0, err
	}

	readUsers := make([]model.User, 0, len(readReceipts))
	for _, receipt := range readReceipts {
		if receipt.User != nil && receipt.User.ID != userID {
			readUsers = append(readUsers, *receipt.User)
		}
	}

	var totalMembers int64
	db.Model(&model.ConversationMember{}).Where("conversation_id = ?", msg.ConversationID).Count(&totalMembers)

	return readUsers, totalMembers, nil
}

func (s *MessageService) BatchGetMessageReadUsers(msgIDs []uint, userID uint) (map[uint]struct {
	ReadUsers    []model.User `json:"read_users"`
	TotalMembers int64        `json:"total_members"`
	ReadCount    int64        `json:"read_count"`
}, error) {
	if len(msgIDs) == 0 {
		return make(map[uint]struct {
			ReadUsers    []model.User `json:"read_users"`
			TotalMembers int64        `json:"total_members"`
			ReadCount    int64        `json:"read_count"`
		}), nil
	}

	db := s.db

	// 优化：一次性查询所有消息的已读回执
	var readReceipts []model.MessageReadReceipt
	db.Where("message_id IN ?", msgIDs).Preload("User").Find(&readReceipts)

	// 按消息 ID 分组
	receiptsByMsg := make(map[uint][]model.MessageReadReceipt)
	for _, r := range readReceipts {
		receiptsByMsg[r.MessageID] = append(receiptsByMsg[r.MessageID], r)
	}

	// 优化：一次性查询所有会话的成员数
	var convIDs []uint
	var messages []model.Message
	db.Where("id IN ?", msgIDs).Find(&messages)
	for _, m := range messages {
		convIDs = append(convIDs, m.ConversationID)
	}

	type convCount struct {
		ConversationID uint
		Count          int64
	}
	var convCounts []convCount
	db.Model(&model.ConversationMember{}).
		Select("conversation_id, COUNT(*) as count").
		Where("conversation_id IN ?", convIDs).
		Group("conversation_id").
		Scan(&convCounts)

	memberCountByConv := make(map[uint]int64)
	for _, cc := range convCounts {
		memberCountByConv[cc.ConversationID] = cc.Count
	}

	// 构建结果
	convIDByMsg := make(map[uint]uint)
	for _, m := range messages {
		convIDByMsg[m.ID] = m.ConversationID
	}

	result := make(map[uint]struct {
		ReadUsers    []model.User `json:"read_users"`
		TotalMembers int64        `json:"total_members"`
		ReadCount    int64        `json:"read_count"`
	}, len(msgIDs))

	for _, msgID := range msgIDs {
		receipts := receiptsByMsg[msgID]
		readUsers := make([]model.User, 0, len(receipts))
		for _, r := range receipts {
			if r.User != nil && r.User.ID != userID {
				readUsers = append(readUsers, *r.User)
			}
		}

		totalMembers := int64(0)
		if convID, ok := convIDByMsg[msgID]; ok {
			totalMembers = memberCountByConv[convID]
		}

		result[msgID] = struct {
			ReadUsers    []model.User `json:"read_users"`
			TotalMembers int64        `json:"total_members"`
			ReadCount    int64        `json:"read_count"`
		}{
			ReadUsers:    readUsers,
			TotalMembers: totalMembers,
			ReadCount:    int64(len(readUsers)),
		}
	}

	return result, nil
}

func (s *MessageService) SearchMessagesByFullText(userID uint, keyword string, convID *uint, limit, offset int) ([]model.Message, error) {
	if keyword == "" {
		return s.SearchMessages(userID, "", convID, limit, offset)
	}

	db := s.db

	if database.D.SupportsFulltext() {
		var messages []model.Message
		query := db.Model(&model.Message{}).
			Joins("JOIN conversation_members ON messages.conversation_id = conversation_members.conversation_id").
			Where("conversation_members.user_id = ?", userID).
			Where("MATCH(messages.content) AGAINST(? IN BOOLEAN MODE)", keyword)

		if convID != nil {
			query = query.Where("messages.conversation_id = ?", *convID)
		}

		if err := query.Preload("Sender").
			Preload("Conversation").
			Preload("Conversation.Members").
			Preload("Conversation.Members.User").
			Order("messages.created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&messages).Error; err != nil {
			return nil, err
		}
		return messages, nil
	}

	// SQLite：使用 FTS5 虚拟表
	// 其他不支持 FULLTEXT 的数据库：降级到 LIKE 搜索
	if database.D.Type() == "sqlite" {
		if limit <= 0 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		var messages []model.Message
		query := db.Raw(`
			SELECT m.* FROM messages m
			JOIN messages_fts5 fts ON m.id = fts.rowid
			JOIN conversation_members cm ON m.conversation_id = cm.conversation_id
			WHERE cm.user_id = ? AND fts.content MATCH ?
		`, userID, keyword)

		if convID != nil {
			query = db.Raw(`
				SELECT m.* FROM messages m
				JOIN messages_fts5 fts ON m.id = fts.rowid
				JOIN conversation_members cm ON m.conversation_id = cm.conversation_id
				WHERE cm.user_id = ? AND fts.content MATCH ? AND m.conversation_id = ?
			`, userID, keyword, *convID)
		}

		if err := query.Find(&messages).Error; err != nil {
			return nil, err
		}

		// 预加载关联数据
		if len(messages) > 0 {
			msgIDs := make([]uint, len(messages))
			for i, m := range messages {
				msgIDs[i] = m.ID
			}
			db.Where("id IN ?", msgIDs).
				Preload("Sender").
				Preload("Conversation").
				Preload("Conversation.Members").
				Preload("Conversation.Members.User").
				Find(&messages)
		}

		return messages, nil
	}

	// TiDB / 其他不支持 FULLTEXT 的数据库：降级到 LIKE 搜索
	return s.SearchMessages(userID, keyword, convID, limit, offset)
}

// buildMessageResponse 构建消息响应体。
// HTTP 响应、WS 广播、历史拉取三路共用。
// mentionUserIDs 为已展开（含 @all 展开）的被提及用户 ID 列表。
// is_at_mention 不在此处计算——per-recipient，由调用方按当前用户算。
func (s *MessageService) buildMessageResponse(msg model.Message, mentionUserIDs []uint) map[string]interface{} {
	if mentionUserIDs == nil {
		mentionUserIDs = []uint{}
	}
	isAvatarReply := msg.AIType == "avatar"
	isAIMessage := msg.AIType == "assistant" || msg.AIType == "avatar" ||
		msg.Sender.Type == "bot_assistant" || msg.Sender.Type == "bot_avatar" || msg.Sender.Type == "system"
	return map[string]interface{}{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"is_avatar_reply":   isAvatarReply,
		"is_ai_message":     isAIMessage,
		"ai_type":           msg.AIType,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"mention_user_ids":  mentionUserIDs,
	}
}

// broadcastMessage 广播消息到会话所有成员（排除发送者）。
// mention_user_ids 数组随消息一起广播，前端据此计算 is_at_mention。
// 无需 per-recipient 发送，效率与原方案一致。
func (s *MessageService) broadcastMessage(msg *model.Message, mentionUserIDs []uint, senderID uint) {
	if s.hub == nil {
		return
	}
	payload := s.buildMessageResponse(*msg, mentionUserIDs)
	wsMsg := ws.WSMessage{Type: "new_message", Data: payload}
	jsonMsg, _ := json.Marshal(wsMsg)
	s.hub.SendToConversation(msg.ConversationID, senderID, jsonMsg)
}

func (s *MessageService) CreateMessage(msg *model.Message) error {
	db := s.db
	return db.Create(msg).Error
}

func (s *MessageService) IsAIMessage(senderID uint) bool {
	systemUserID := s.GetSystemUserID()
	return systemUserID > 0 && senderID == systemUserID
}

func (s *MessageService) GetSystemUserID() uint {
	var systemUser model.User
	if err := s.db.Where("type = ?", "system").First(&systemUser).Error; err != nil {
		return 0
	}
	return systemUser.ID
}
