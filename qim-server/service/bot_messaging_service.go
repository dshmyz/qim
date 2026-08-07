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

// ErrCardActionAlreadyHandled 该卡片已被该用户处理过（幂等拦截，防重复触发 webhook）。
// 调用方应返回 200 + "已处理"，而非 4xx，因为这是正常幂等响应而非错误。
var ErrCardActionAlreadyHandled = errors.New("该卡片已处理")

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

	// 查找已有 bot 会话：用户 1:1 bot 会话 = Type=bot 会话且成员含该 user。
	// 原来按 user_id 反查 BotConversation，现改用 ConversationMember + Type=bot join 等价替代。
	var botConv model.BotConversation
	if err := s.db.
		Joins("JOIN conversations c ON c.id = bot_conversations.conversation_id").
		Joins("JOIN conversation_members cm ON cm.conversation_id = c.id").
		Where("bot_conversations.bot_id = ? AND c.type = ? AND cm.user_id = ?", botID, "bot", userID).
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
	botConv = model.BotConversation{BotID: botID, ConversationID: conv.ID}
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
// replyToID 非空时要求引用消息存在且属于当前会话。
// 返回创建的消息（已预加载 Sender）。
func (s *BotMessagingService) SendOutbound(bot *model.Bot, toUserID uint, content, msgType string, threadID *uint, replyToID *uint) (*model.Message, error) {
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
		// 校验会话归属该 bot（bot_conversations 关联）+ 目标用户为其成员。
		// 原实现按 bot_conversations.user_id 判定，去掉该列后改为成员校验（单聊/群通吃）。
		var botConv model.BotConversation
		if err := s.db.Where("conversation_id = ? AND bot_id = ?", *threadID, bot.ID).
			First(&botConv).Error; err != nil {
			return nil, errors.New("会话不属于该 bot")
		}
		var member model.ConversationMember
		if err := s.db.Where("conversation_id = ? AND user_id = ?", *threadID, toUserID).
			First(&member).Error; err != nil {
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

	if replyToID != nil {
		if *replyToID == 0 {
			return nil, errors.New("引用消息 ID 无效")
		}
		var quoted model.Message
		if err := s.db.First(&quoted, *replyToID).Error; err != nil {
			return nil, errors.New("引用消息不存在")
		}
		if quoted.ConversationID != convID {
			return nil, errors.New("引用消息不属于当前会话")
		}
	}

	return s.sendToConversation(bot, convID, content, msgType, replyToID)
}

// sendToConversation 以 bot 身份向指定会话（单聊 type=bot 或群聊 type=group）发一条消息：
// 创建消息（bot 自身已读）→ 更新会话 last_message → 给非 bot 成员未读 +1 → WS 广播到会话。
// 供单聊 SendOutbound 与群聊 SendOutboundByConversation 共用。
func (s *BotMessagingService) sendToConversation(bot *model.Bot, convID uint, content, msgType string, replyToID *uint) (*model.Message, error) {
	// 创建消息（bot 自身已读）
	msg := model.Message{
		ConversationID:  convID,
		SenderID:        *bot.VirtualUserID,
		QuotedMessageID: replyToID,
		Type:            msgType,
		Content:         content,
		IsRead:          true,
		Origin:          "bot",
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

// SendOutboundByConversation 以 bot 身份向指定会话发送消息（单聊 type=bot 或群聊 type=group）。
// 校验 bot 拥有该会话（BotConversation{bot_id, conversation_id} 存在），供 agent 按 conversation_id
// 群发（MCP/CLI 及 Phase 3 群内出站）。replyToID 非空时要求引用消息属于当前会话。
func (s *BotMessagingService) SendOutboundByConversation(bot *model.Bot, convID uint, content, msgType string, replyToID *uint) (*model.Message, error) {
	if bot == nil || bot.VirtualUserID == nil {
		return nil, errors.New("bot 未配置虚拟用户")
	}
	if msgType == "" {
		msgType = "text"
	}
	if msgType == "card" {
		if err := validateCardContent(content); err != nil {
			return nil, err
		}
	}

	// 校验 bot 拥有该会话（单聊/群通用：bot 通过 BotConversation 关联了该会话即为已入群/已建会话）
	var botConv model.BotConversation
	if err := s.db.Where("conversation_id = ? AND bot_id = ?", convID, bot.ID).
		First(&botConv).Error; err != nil {
		return nil, errors.New("会话不属于该 bot")
	}

	if replyToID != nil {
		if *replyToID == 0 {
			return nil, errors.New("引用消息 ID 无效")
		}
		var quoted model.Message
		if err := s.db.First(&quoted, *replyToID).Error; err != nil {
			return nil, errors.New("引用消息不存在")
		}
		if quoted.ConversationID != convID {
			return nil, errors.New("引用消息不属于当前会话")
		}
	}

	return s.sendToConversation(bot, convID, content, msgType, replyToID)
}

// EnsureBotGroupConversation 把 bot 拉进一个群会话（Phase 3「拉 bot 进群」的服务端入口）：
// 校验会话为群；确保 bot 虚拟用户是 ConversationMember；幂等建 BotConversation{bot_id, conversation_id}。
// 返回该 bot 会话关联。
func (s *BotMessagingService) EnsureBotGroupConversation(botID, conversationID uint) (*model.BotConversation, error) {
	var bot model.Bot
	if err := s.db.First(&bot, botID).Error; err != nil {
		return nil, errors.New("bot 不存在")
	}
	if bot.VirtualUserID == nil {
		return nil, errors.New("bot 未配置虚拟用户")
	}
	var conv model.Conversation
	if err := s.db.First(&conv, conversationID).Error; err != nil {
		return nil, errors.New("会话不存在")
	}
	if conv.Type != "group" {
		return nil, errors.New("仅支持把 bot 拉进群会话")
	}

	var botConv model.BotConversation
	if err := s.db.Where("bot_id = ? AND conversation_id = ?", botID, conversationID).
		First(&botConv).Error; err == nil && botConv.ID > 0 {
		// 已关联：幂等补齐 bot 虚拟用户成员关系
		s.ensureMember(conversationID, *bot.VirtualUserID)
		return &botConv, nil
	}

	tx := s.db.Begin()
	if err := tx.Create(&model.ConversationMember{ConversationID: conversationID, UserID: *bot.VirtualUserID, Role: "member"}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	botConv = model.BotConversation{BotID: botID, ConversationID: conversationID}
	if err := tx.Create(&botConv).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	return &botConv, nil
}

// BotGroupConversation 是 bot 已入群的群会话摘要，供 agent 发现群 conversation_id。
type BotGroupConversation struct {
	ConversationID uint   `json:"conversation_id"`
	GroupName      string `json:"group_name"`
}

// ListBotGroupConversations 列出 bot 已入群的群会话（BotConversation 关联且会话类型为 group）。
// 供 agent 主动群发前发现可用的群 conversation_id。
func (s *BotMessagingService) ListBotGroupConversations(botID uint) ([]BotGroupConversation, error) {
	var rows []BotGroupConversation
	err := s.db.
		Table("bot_conversations").
		Select("bot_conversations.conversation_id AS conversation_id, COALESCE(g.name, '') AS group_name").
		Joins("JOIN conversations conv ON conv.id = bot_conversations.conversation_id AND conv.type = 'group'").
		Joins("LEFT JOIN groups g ON g.conversation_id = bot_conversations.conversation_id").
		Where("bot_conversations.bot_id = ?", botID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
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
	Title   string       `json:"title,omitempty"`
	Text    string       `json:"text,omitempty"`
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
// 然后做两件事：
//  1. 在会话内建一条 type=card_action 的消息（用户端显示「✓ 已选择:xxx」气泡，
//     同时让 pull-mode agent 经 GetBotMessages 能拉到点击事件）。
//  2. 以 event="bot.card_action" 转发到 agent webhook（webhook-mode agent）。
//
// card_action 之前只走 webhook，pull-mode agent（CLI/MCP）是黑洞--两条路现在都通。
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
	// 鉴权：仅该会话的人类成员可操作卡片（单聊 user / 群聊任意群成员都算，统一走成员校验）
	var member model.ConversationMember
	if err := s.db.Where("conversation_id = ? AND user_id = ?", msg.ConversationID, userID).First(&member).Error; err != nil {
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
	actionText, err := cardActionText(msg.Content, actionID)
	if err != nil {
		return err
	}

	var user model.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 幂等：同一卡片(message_id)+同一用户(user_id)只处理一次。
	// agent 调用 UpdateMessageContent 改写卡片时会删除该记录，从而允许新一轮点击。
	var existing model.CardActionRecord
	if err := s.db.Where("message_id = ? AND user_id = ?", msg.ID, userID).First(&existing).Error; err == nil {
		return ErrCardActionAlreadyHandled
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		logger.WithModule("BotMessaging").Error("卡片 action 幂等查询失败",
			"botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "error", err)
		return errors.New("卡片动作处理失败")
	}
	// 插入；并发下另一请求可能抢先插入 -> 唯一索引冲突，复查确认后视为已处理
	if err := s.db.Create(&model.CardActionRecord{
		MessageID: msg.ID,
		UserID:    userID,
		ActionID:  actionID,
		BotID:     bot.ID,
	}).Error; err != nil {
		var race model.CardActionRecord
		if err := s.db.Where("message_id = ? AND user_id = ?", msg.ID, userID).First(&race).Error; err == nil {
			return ErrCardActionAlreadyHandled
		}
		logger.WithModule("BotMessaging").Error("卡片 action 幂等记录插入失败",
			"botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "error", err)
		return errors.New("卡片动作处理失败")
	}

	// ① 在会话内建一条 card_action 消息：用户端显示气泡 + pull-mode agent 可拉到。
	// sender=人类用户（点击动作由用户发起）。content 为 JSON，客户端按 type=card_action 渲染。
	actionContent, _ := json.Marshal(map[string]interface{}{
		"action_id":       actionID,
		"action_text":     actionText,
		"value":           value,
		"card_message_id": msg.ID,
	})
	actionMsg := model.Message{
		ConversationID: msg.ConversationID,
		SenderID:       userID,
		Type:           "card_action",
		Content:        string(actionContent),
		Origin:         "user",
	}
	if err := s.db.Create(&actionMsg).Error; err != nil {
		logger.WithModule("BotMessaging").Error("卡片 action 消息创建失败",
			"botID", bot.ID, "messageID", msg.ID, "actionID", actionID, "error", err)
		// 不 return：消息气泡失败不阻断 webhook 转发，agent 仍能收事件
	} else {
		if err := s.db.Preload("Sender").First(&actionMsg, actionMsg.ID).Error; err != nil {
			logger.WithModule("BotMessaging").Error("卡片 action 消息预加载失败", "error", err)
		}
		// 更新会话最后消息 + 未读（bot 虚拟用户未读 +1）
		now := time.Now()
		if bot.VirtualUserID != nil {
			s.db.Model(&model.Conversation{}).Where("id = ?", msg.ConversationID).Updates(map[string]interface{}{
				"last_message_id": actionMsg.ID,
				"last_message_at": now,
			})
			s.db.Model(&model.ConversationMember{}).
				Where("conversation_id = ? AND user_id = ?", msg.ConversationID, *bot.VirtualUserID).
				UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))
		}
		// 广播新消息气泡到会话（排除点击者，客户端已乐观渲染）
		if s.hub != nil {
			resp := buildBotMessageResponse(actionMsg)
			wsMsg := ws.WSMessage{Type: "new_message", Data: resp}
			jsonMsg, _ := json.Marshal(wsMsg)
			s.hub.SendToConversation(msg.ConversationID, userID, jsonMsg)
		}
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

	// 纯 pull 模式（webhook_url 空）：不投 webhook。card_action 消息已在会话内（①），
	// pull-mode agent 靠 GET /bot/messages 拉到点击事件，无需 outbox 投递，也不产生死信。
	if !cfg.HasWebhook() {
		logger.WithModule("BotMessaging").Info("外部 bot 未配 webhook_url，card_action 走纯 pull 模式（不投递）",
			"botID", bot.ID, "messageID", msg.ID, "actionID", actionID)
		return nil
	}

	// 经 outbox：先落表再立即 best-effort 投递一次。
	// 鉴权/校验已在上面同步完成（返回 error 给前端）；此处只兜底"投递失败"。
	// 投递失败不再阻塞用户（原返回 400），改为入重试队列，调用方据返回值提示"已受理将重试"。
	deliveryID, err := EnqueueWebhookDelivery(s.db, bot.ID, "bot.card_action", string(payloadJSON), cfg.WebhookURL, cfg.WebhookSecret)
	if err != nil {
		// 入队失败须回滚幂等记录，否则会锁死该用户后续重试（点击永远返回"已处理"）
		s.db.Where("message_id = ? AND user_id = ?", msg.ID, userID).Delete(&model.CardActionRecord{})
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

func cardActionText(content, actionID string) (string, error) {
	if actionID == "" {
		return "", errors.New("卡片动作无效")
	}
	var card cardPayload
	if err := json.Unmarshal([]byte(content), &card); err != nil {
		return "", errors.New("卡片内容不是合法 JSON")
	}
	for _, b := range card.Buttons {
		if b.ID == actionID {
			return b.Text, nil
		}
	}
	return "", errors.New("无效的卡片动作")
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

// CleanupStaleStreamingMessages 把超时仍处 streaming 状态的消息收尾为 markdown。
//
// 背景：外部 agent（CLI/MCP）用 start_streaming_message 建一条 type=streaming 消息，
// 仅在调 finish 时转 markdown。若 agent 崩溃/断网/漏调 finish，消息永久卡在 streaming，
// 刷新后客户端渲染成无 typing 动画的空/半截气泡（REST 历史拉取 is_streaming=true 但无人再推分段）。
//
// 以 updated_at 判活：流式每段都更新 content（GORM 自动刷 updated_at），活跃流不会被误杀；
// updated_at 早于 maxAge 的视为已断，转 markdown（保留已累积 content）。空 content 转后为空 markdown 气泡，
// 罕见（agent 在首段前就崩），可接受。由 main.go 注册的 cron 每 2 分钟 + 启动时各扫一次。
func CleanupStaleStreamingMessages(db *gorm.DB, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge)
	res := db.Model(&model.Message{}).
		Where("type = ? AND updated_at < ?", "streaming", cutoff).
		Updates(map[string]interface{}{"type": "markdown"})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected > 0 {
		logger.WithModule("BotMessaging").Info("清理僵尸流式消息",
			"count", res.RowsAffected, "cutoff", cutoff.Format(time.RFC3339))
	}
	return res.RowsAffected, nil
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

	// 卡片改写即"新一轮"：删除该消息的点击幂等记录，释放锁定，允许用户再次点击。
	// agent 改写通常代表新状态/新按钮，旧 action 不应再阻塞交互。
	if err := s.db.Where("message_id = ?", messageID).
		Delete(&model.CardActionRecord{}).Error; err != nil {
		logger.WithModule("BotMessaging").Warn("删除卡片 action 幂等记录失败",
			"messageID", messageID, "error", err)
		// 非致命：仅影响新一轮点击的幂等，不阻塞改写本身
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

// ResolveBotThread 按用户名/昵称解析 bot 会话 ID。
// 用 bot_conversations 表查找 bot 与目标用户的会话。
func (s *BotMessagingService) ResolveBotThread(bot *model.Bot, nameOrID string) (uint, error) {
	if bot == nil {
		return 0, errors.New("bot 无效")
	}

	// 先试数字 ID
	if n, err := strconv.ParseUint(nameOrID, 10, 64); err == nil {
		// 是数字，直接走归属校验
		var bc model.BotConversation
		if err := s.db.Where("conversation_id = ? AND bot_id = ?", uint(n), bot.ID).First(&bc).Error; err != nil {
			return 0, errors.New("会话不属于该 bot")
		}
		return uint(n), nil
	}

	user, err := s.resolveUniqueUser(nameOrID)
	if err != nil {
		return 0, err
	}

	// 查该用户的 1:1 bot 会话（Type=bot 会话且成员含该 user）。
	// 原来按 user_id 反查 BotConversation，现改用 ConversationMember + Type=bot join 等价替代。
	var bc model.BotConversation
	if err := s.db.
		Joins("JOIN conversations c ON c.id = bot_conversations.conversation_id").
		Joins("JOIN conversation_members cm ON cm.conversation_id = c.id").
		Where("bot_conversations.bot_id = ? AND c.type = ? AND cm.user_id = ?", bot.ID, "bot", user.ID).
		First(&bc).Error; err != nil {
		return 0, errors.New("未找到与 " + nameOrID + " 的会话（可能尚未对话）")
	}
	return bc.ConversationID, nil
}

// ResolveUserID 按用户名或昵称查找用户 ID。
func (s *BotMessagingService) ResolveUserID(name string) (uint, error) {
	user, err := s.resolveUniqueUser(name)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (s *BotMessagingService) resolveUniqueUser(name string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", name).First(&user).Error; err == nil {
		return &user, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("查询用户失败")
	}

	var users []model.User
	if err := s.db.Where("nickname = ?", name).Find(&users).Error; err != nil {
		return nil, errors.New("查询用户失败")
	}
	switch len(users) {
	case 0:
		return nil, errors.New("未找到用户: " + name)
	case 1:
		return &users[0], nil
	default:
		return nil, errors.New("昵称不唯一，请使用用户名或 ID: " + name)
	}
}
