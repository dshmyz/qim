package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/ws"
	"gorm.io/gorm"
)

// BotMessagingService 处理外部 agent 经 Bot API 的出站消息：
// 确保 bot↔用户会话、创建以 bot 虚拟用户为发送者的消息、推送 WS。
// 范式对齐 avatar_worker_pool.sendPrivateReply，但会话类型为 "bot"，
// 消息 Origin 为 "bot"，且不触发 smart-reply/avatar 回调（避免递归）。
type BotMessagingService struct {
	db  *gorm.DB
	hub *ws.Hub
}

// ErrCardActionPendingRetry 卡片 action 已入 webhook 重试队列（立即投递失败，非鉴权错误）。
// 调用方据此返回"已受理将重试"而非 400，避免因 agent 暂时不可用阻塞用户。
var ErrCardActionPendingRetry = errors.New("卡片动作已受理，正在重试投递")

func NewBotMessagingService(db *gorm.DB, hub *ws.Hub) *BotMessagingService {
	return &BotMessagingService{db: db, hub: hub}
}

// EnsureBotConversation 查找或创建 bot 与用户的 1:1 bot 会话。
// 逻辑抽取自 handler.CreateSingleConversation 的 bot 分支，供出站 API 复用。
func (s *BotMessagingService) EnsureBotConversation(botID, userID uint) (*model.Conversation, *model.BotConversation, error) {
	var bot model.Bot
	if err := s.db.First(&bot, botID).Error; err != nil {
		return nil, nil, err
	}
	if bot.VirtualUserID == nil {
		return nil, nil, errors.New("bot 未配置虚拟用户")
	}

	// 查找已有 bot 会话
	var botConv model.BotConversation
	if err := s.db.Where("bot_id = ? AND user_id = ?", botID, userID).
		Preload("Conversation").First(&botConv).Error; err == nil && botConv.ID > 0 {
		// 防御性补齐 bot 虚拟用户成员关系
		s.ensureMember(botConv.ConversationID, *bot.VirtualUserID)
		s.db.Preload("Conversation").First(&botConv, botConv.ID)
		return &botConv.Conversation, &botConv, nil
	}

	// 创建新会话
	tx := s.db.Begin()
	conv := model.Conversation{Type: "bot"}
	if err := tx.Create(&conv).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: userID, Role: "member"}).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: *bot.VirtualUserID, Role: "member"}).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	botConv = model.BotConversation{BotID: botID, UserID: userID, ConversationID: conv.ID}
	if err := tx.Create(&botConv).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}
	s.db.Preload("Conversation").First(&botConv, botConv.ID)
	return &conv, &botConv, nil
}

// ensureMember 确保用户是会话成员（不存在则补建）。
func (s *BotMessagingService) ensureMember(convID, userID uint) {
	var count int64
	s.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).Count(&count)
	if count == 0 {
		s.db.Create(&model.ConversationMember{ConversationID: convID, UserID: userID, Role: "member"})
	}
}

// SendOutbound 以 bot 身份向用户发送一条消息。
// threadID 非空时校验其归属该 bot+用户，否则取/建 bot↔用户会话。
// 返回创建的消息（已预加载 Sender）。
func (s *BotMessagingService) SendOutbound(bot *model.Bot, toUserID uint, content, msgType string, threadID *uint) (*model.Message, error) {
	if bot == nil || bot.VirtualUserID == nil {
		return nil, errors.New("bot 未配置虚拟用户")
	}
	if msgType == "" {
		msgType = "text"
	}
	// 卡片消息校验内容契约（buttons[].id/text 必填），非法内容拒绝出站
	if msgType == "card" {
		if err := validateCardContent(content); err != nil {
			return nil, err
		}
	}

	// 校验目标用户存在且为普通用户（不允许 bot 给 bot/系统账号发消息）
	var user model.User
	if err := s.db.First(&user, toUserID).Error; err != nil {
		return nil, errors.New("目标用户不存在")
	}
	if user.Type != "" && user.Type != "user" {
		return nil, errors.New("目标用户类型不支持")
	}

	var convID uint
	if threadID != nil && *threadID != 0 {
		// 校验会话归属该 bot+用户
		var botConv model.BotConversation
		if err := s.db.Where("conversation_id = ? AND bot_id = ? AND user_id = ?", *threadID, bot.ID, toUserID).
			First(&botConv).Error; err != nil {
			return nil, errors.New("会话不属于该 bot 与用户")
		}
		convID = *threadID
	} else {
		conv, _, err := s.EnsureBotConversation(bot.ID, toUserID)
		if err != nil {
			return nil, err
		}
		convID = conv.ID
	}

	// 创建消息（bot 自身已读）
	msg := model.Message{
		ConversationID: convID,
		SenderID:       *bot.VirtualUserID,
		Type:           msgType,
		Content:        content,
		IsRead:         true,
		Origin:         "bot",
	}
	if err := s.db.Create(&msg).Error; err != nil {
		return nil, err
	}
	if err := s.db.Preload("Sender").First(&msg, msg.ID).Error; err != nil {
		return nil, err
	}

	// 更新会话最后消息
	now := time.Now()
	if err := s.db.Model(&model.Conversation{}).Where("id = ?", convID).Updates(map[string]interface{}{
		"last_message_id": msg.ID,
		"last_message_at": now,
	}).Error; err != nil {
		logger.WithModule("BotMessaging").Error("更新会话最后消息失败", "conv", convID, "error", err)
	}

	// 给人类成员未读 +1（bot 自身已读）
	if err := s.db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, *bot.VirtualUserID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error; err != nil {
		logger.WithModule("BotMessaging").Error("更新未读数失败", "conv", convID, "error", err)
	}

	// 广播到会话（排除 bot 虚拟用户）
	if s.hub != nil {
		responseData := buildBotMessageResponse(msg)
		wsMsg := ws.WSMessage{Type: "new_message", Data: responseData}
		jsonMsg, _ := json.Marshal(wsMsg)
		s.hub.SendToConversation(convID, *bot.VirtualUserID, jsonMsg)
	}

	return &msg, nil
}

// cardButton 卡片按钮的最小契约字段。
type cardButton struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Style string `json:"style,omitempty"`
	Value string `json:"value,omitempty"`
}

// cardPayload 卡片消息 content JSON 的最小契约。
type cardPayload struct {
	Title   string      `json:"title,omitempty"`
	Text    string      `json:"text,omitempty"`
	Buttons []cardButton `json:"buttons"`
}

// validateCardContent 校验 msg_type=="card" 的 content 是否符合契约：
// 可解析为 JSON、buttons 为非空数组、每个按钮含 id 与 text。
// title/text/style/value 可选。宽松校验，未识别字段忽略。
func validateCardContent(content string) error {
	var p cardPayload
	if err := json.Unmarshal([]byte(content), &p); err != nil {
		return errors.New("卡片内容不是合法 JSON")
	}
	if len(p.Buttons) == 0 {
		return errors.New("卡片至少需要一个按钮")
	}
	for i, b := range p.Buttons {
		if b.ID == "" || b.Text == "" {
			return errors.New("卡片按钮缺少 id 或 text（第 " + strconv.Itoa(i+1) + " 个）")
		}
	}
	return nil
}

// ForwardCardAction 处理用户对 bot 卡片按钮的点击：
// 校验消息为卡片、归属 bot 会话、调用方为该会话人类成员、bot 启用外部 webhook，
// 随后以 event="bot.card_action" 转发到 agent webhook。不创建聊天消息。
func (s *BotMessagingService) ForwardCardAction(messageID, userID uint, actionID, value string) error {
	var msg model.Message
	if err := s.db.First(&msg, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}
	if msg.Type != "card" {
		return errors.New("非卡片消息，不支持按钮交互")
	}

	var botConv model.BotConversation
	if err := s.db.Where("conversation_id = ?", msg.ConversationID).First(&botConv).Error; err != nil {
		return errors.New("消息不属于 bot 会话")
	}
	// 鉴权：仅该 bot↔user 会话的人类成员可操作自己的卡片
	if botConv.UserID != userID {
		return errors.New("无权操作此卡片")
	}

	var bot model.Bot
	if err := s.db.First(&bot, botConv.BotID).Error; err != nil {
		return errors.New("bot 不存在")
	}
	if !bot.IsActive {
		return errors.New("bot 未启用")
	}
	cfg := ParseBotConfig(bot.Config)
	if !cfg.IsExternalWebhook() {
		return errors.New("该 bot 未启用外部 webhook，不支持卡片交互")
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	payload := BotWebhookPayload{
		Event:        "bot.card_action",
		BotID:        bot.ID,
		ThreadID:     msg.ConversationID,
		MessageID:    msg.ID,
		UserID:       userID,
		UserNickname: user.Nickname,
		UserAvatar:   user.Avatar,
		MsgType:      "card_action",
		ActionID:     actionID,
		ActionValue:  value,
	}
	payloadJSON, _ := json.Marshal(payload)

	// 经 outbox：先落表再立即 best-effort 投递一次。
	// 鉴权/校验已在上面同步完成（返回 error 给前端）；此处只兜底"投递失败"。
	// 投递失败不再阻塞用户（原返回 400），改为入重试队列，调用方据返回值提示"已受理将重试"。
	deliveryID, err := EnqueueWebhookDelivery(s.db, bot.ID, "bot.card_action", string(payloadJSON), cfg.WebhookURL, cfg.WebhookSecret)
	if err != nil {
		logger.WithModule("BotMessaging").Error("卡片 action outbox 入队失败",
			"botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "error", err)
		return errors.New("转发卡片动作失败")
	}
	if err := DeliverOnce(s.db, deliveryID); err != nil {
		// 本地错误（落表/乐观锁等）罕见，按"已入重试队列"兜底
		logger.WithModule("BotMessaging").Warn("卡片 action 立即投递异常，已入重试队列",
			"deliveryID", deliveryID, "botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "error", err)
		return ErrCardActionPendingRetry
	}
	// DeliverOnce 返回 nil 不代表投递成功（best-effort：失败也返回 nil）。
	// 据落表状态区分：done=已送达；否则=立即投递失败，已入重试队列。
	var delivery model.BotWebhookDelivery
	if err := s.db.First(&delivery, deliveryID).Error; err == nil && delivery.Status != "done" {
		logger.WithModule("BotMessaging").Warn("卡片 action 立即投递失败，已入重试队列",
			"deliveryID", deliveryID, "botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "status", delivery.Status)
		return ErrCardActionPendingRetry
	}
	logger.WithModule("BotMessaging").Info("卡片 action webhook 已转发",
		"botID", bot.ID, "messageID", msg.ID, "actionID", actionID)
	return nil
}

// buildBotMessageResponse 组装 WS 推送的消息响应，字段对齐 MessageService.buildMessageResponse。
func buildBotMessageResponse(msg model.Message) map[string]interface{} {
	isAIMessage := msg.Origin == "assistant" || msg.Origin == "avatar" ||
		msg.Origin == "bot" || msg.Sender.Type == "bot" || msg.Sender.Type == "system"
	return map[string]interface{}{
		"id":                msg.ID,
		"conversation_id":   msg.ConversationID,
		"sender_id":         msg.SenderID,
		"type":              msg.Type,
		"content":           msg.Content,
		"quoted_message_id": msg.QuotedMessageID,
		"is_recalled":       msg.IsRecalled,
		"is_read":           msg.IsRead,
		"is_avatar_reply":   msg.Origin == "avatar",
		"is_ai_message":     isAIMessage,
		"is_streaming":      msg.Type == "streaming",
		"origin":            msg.Origin,
		"recalled_at":       msg.RecalledAt,
		"created_at":        msg.CreatedAt,
		"sender":            msg.Sender,
		"quoted_message":    msg.QuotedMessage,
		"mention_user_ids":  []uint{},
	}
}

// StreamChunk 处理外部 agent 对流式消息的分段追加：
// 校验消息为 streaming 类型且归属该 bot，累加 contentDelta，推 message_updated 到用户。
// finish=true 时把 Type 改为 markdown（最终渲染）并更新会话最后消息。
// 客户端 handleMessageUpdated 是替换语义，故每次推送全量 Content。
func (s *BotMessagingService) StreamChunk(bot *model.Bot, messageID uint, contentDelta string, finish bool) error {
	if bot == nil || bot.VirtualUserID == nil {
		return errors.New("bot 未配置虚拟用户")
	}

	var msg model.Message
	if err := s.db.First(&msg, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}
	if msg.Type != "streaming" {
		return errors.New("非流式消息，不支持分段追加")
	}

	// 归属校验：只允许发起该流的 bot 追加（仿 SendOutbound thread_id 校验）
	var botConv model.BotConversation
	if err := s.db.Where("conversation_id = ? AND bot_id = ?", msg.ConversationID, bot.ID).
		First(&botConv).Error; err != nil {
		return errors.New("会话不属于该 bot")
	}

	// 事务内读-改-写，防并发 delta 互相覆盖
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var cur model.Message
		if err := tx.First(&cur, messageID).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{}
		if contentDelta != "" {
			updates["content"] = cur.Content + contentDelta
		}
		if finish {
			updates["type"] = "markdown"
		}
		if len(updates) > 0 {
			if err := tx.Model(&model.Message{}).Where("id = ?", messageID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 重新读取最新内容用于推送
	if err := s.db.First(&msg, messageID).Error; err != nil {
		return err
	}

	// finish 时更新会话最后消息
	if finish {
		now := time.Now()
		if err := s.db.Model(&model.Conversation{}).Where("id = ?", msg.ConversationID).Updates(map[string]interface{}{
			"last_message_id": msg.ID,
			"last_message_at": now,
		}).Error; err != nil {
			logger.WithModule("BotMessaging").Error("流式完成更新会话最后消息失败", "conv", msg.ConversationID, "error", err)
		}
	}

	s.pushMessageUpdated(msg, *bot.VirtualUserID, msg.Type == "streaming")

	return nil
}

// pushMessageUpdated 推 message_updated 事件到会话（排除 excludeUserID，即发送方 bot 的虚拟用户）。
// 供流式分段（StreamChunk）与卡片更新（UpdateMessageContent）共用，避免重复拼装。
// isStreaming 透传给前端用于区分流式中气泡与最终渲染。
func (s *BotMessagingService) pushMessageUpdated(msg model.Message, excludeUserID uint, isStreaming bool) {
	if s.hub == nil {
		return
	}
	updateData := map[string]interface{}{
		"id":              msg.ID,
		"conversation_id": msg.ConversationID,
		"content":         msg.Content,
		"type":            msg.Type,
		"is_streaming":    isStreaming,
	}
	wsMsg := ws.WSMessage{Type: "message_updated", Data: updateData}
	jsonMsg, _ := json.Marshal(wsMsg)
	s.hub.SendToConversation(msg.ConversationID, excludeUserID, jsonMsg)
}


// UpdateMessageContent 全量更新一条 bot 消息的 content（供 agent 回写卡片状态用）。
// 归属校验：消息 Origin=="bot" 且 SenderID 为该 bot 虚拟用户；会话归属该 bot。
// msgType=="card" 走 validateCardContent 保持出站契约一致；msgType 空则保持原 type。
// 推 message_updated，客户端 CardMessage 据 content 变化重置交互态。
func (s *BotMessagingService) UpdateMessageContent(bot *model.Bot, messageID uint, content string, msgType string) error {
	if bot == nil || bot.VirtualUserID == nil {
		return errors.New("bot 未配置虚拟用户")
	}
	if content == "" {
		return errors.New("content 不能为空")
	}

	var msg model.Message
	if err := s.db.First(&msg, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}
	if msg.Origin != "bot" || msg.SenderID != *bot.VirtualUserID {
		return errors.New("无权更新该消息")
	}

	// 归属校验：会话属于该 bot（与 StreamChunk 一致）
	var botConv model.BotConversation
	if err := s.db.Where("conversation_id = ? AND bot_id = ?", msg.ConversationID, bot.ID).
		First(&botConv).Error; err != nil {
		return errors.New("会话不属于该 bot")
	}

	finalType := msg.Type
	if msgType != "" {
		if msgType == "card" {
			if err := validateCardContent(content); err != nil {
				return err
			}
		}
		finalType = msgType
	} else if msg.Type == "card" {
		// 类型不变但仍是卡片，校验新 content 合法
		if err := validateCardContent(content); err != nil {
			return err
		}
	}

	if err := s.db.Model(&model.Message{}).Where("id = ?", messageID).
		Updates(map[string]interface{}{"content": content, "type": finalType}).Error; err != nil {
		return err
	}

	// 重新读出最新消息用于推送
	if err := s.db.First(&msg, messageID).Error; err != nil {
		return err
	}
	s.pushMessageUpdated(msg, *bot.VirtualUserID, false)
	return nil
}

// ListBotMessages 供外部 agent pull 读取自己会话的消息（增量）。
// 归属校验仿 StreamChunk：BotConversation{conversation_id=threadID, bot_id=bot.ID} 必须存在。
// afterID=0 返回该会话全部（受 limit），agent 据此轮询新消息。按 id 升序，preload Sender。
func (s *BotMessagingService) ListBotMessages(bot *model.Bot, threadID uint, afterID uint, limit int) ([]model.Message, error) {
	if bot == nil {
		return nil, errors.New("bot 无效")
	}
	// 归属校验：只允许该 bot 读自己的会话
	var botConv model.BotConversation
	if err := s.db.Where("conversation_id = ? AND bot_id = ?", threadID, bot.ID).
		First(&botConv).Error; err != nil {
		return nil, errors.New("会话不属于该 bot")
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := s.db.Where("conversation_id = ?", threadID)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	var msgs []model.Message
	if err := q.Order("id ASC").Limit(limit).Preload("Sender").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}
