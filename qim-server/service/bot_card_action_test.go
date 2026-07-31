package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// capturedWebhook 记录最近一次 webhook 请求的 body 与 headers。
type capturedWebhook struct {
	mu      sync.Mutex
	body    []byte
	headers http.Header
	got     bool
}

func newCaptureServer(t *testing.T) (*httptest.Server, *capturedWebhook) {
	t.Helper()
	cap := &capturedWebhook{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		cap.mu.Lock()
		cap.body = body
		cap.headers = r.Header.Clone()
		cap.got = true
		cap.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, cap
}

func (c *capturedWebhook) payload() BotWebhookPayload {
	c.mu.Lock()
	defer c.mu.Unlock()
	var p BotWebhookPayload
	_ = json.Unmarshal(c.body, &p)
	return p
}

func (c *capturedWebhook) hasSignature() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.headers.Get("X-QIM-Signature") != ""
}

// setupCardBot 构造一个启用外部 webhook 的 bot + 人类用户，并返回 bot 与用户。
func setupCardBot(t *testing.T, db *gorm.DB, webhookURL, secret string) (*model.Bot, *model.User, *model.User) {
	t.Helper()
	vUser := &model.User{Username: "bot_action", Nickname: "ActionBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "frank", Nickname: "Frank", Type: "user"}
	db.Create(human)
	cfg := BotConfig{Mode: "external_webhook", WebhookURL: webhookURL, WebhookSecret: secret}
	cfgJSON, _ := json.Marshal(cfg)
	bot := &model.Bot{Name: "ActionBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID, Config: string(cfgJSON)}
	db.Create(bot)
	return bot, vUser, human
}

func TestForwardCardAction_ForwardsWebhook(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"title":"确认回滚?","buttons":[{"id":"confirm","text":"确认回滚","value":"confirm"},{"id":"cancel","text":"取消","value":"cancel"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	assert.NoError(t, err)
	assert.True(t, cap.got)

	p := cap.payload()
	assert.Equal(t, "bot.card_action", p.Event)
	assert.Equal(t, bot.ID, p.BotID)
	assert.Equal(t, msg.ID, p.MessageID)
	assert.Equal(t, msg.ConversationID, p.ThreadID)
	assert.Equal(t, human.ID, p.UserID)
	assert.Equal(t, "confirm", p.ActionID)
	assert.Equal(t, "confirm", p.ActionValue)
	assert.Equal(t, "card_action", p.MsgType)
	assert.Equal(t, "Frank", p.UserNickname)
	assert.True(t, cap.hasSignature(), "webhook 应携带 HMAC 签名")
}

// TestForwardCardAction_CreatesPullableActionMessage 验证点击卡片后：
//  1. 会话内多了一条 type=card_action 的 Message（之前是黑洞，pull-mode agent 拉不到）。
//  2. content 含从原卡片反查的 button text（用户端显示「✓ 已选择:xxx」）。
//  3. ListBotMessages（pull）能拉到这条 card_action。
func TestForwardCardAction_CreatesPullableActionMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, _ := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"title":"确认回滚?","buttons":[{"id":"confirm","text":"确认回滚","value":"confirm"},{"id":"cancel","text":"取消","value":"cancel"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	require.NoError(t, err)

	// 点击前：会话只有 1 条卡片消息
	before, err := svc.ListBotMessages(bot, msg.ConversationID, 0, 100)
	require.NoError(t, err)
	require.Len(t, before, 1)

	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	require.NoError(t, err)

	// 点击后：pull 流多了一条 card_action，agent 现在能拉到点击事件
	after, err := svc.ListBotMessages(bot, msg.ConversationID, 0, 100)
	require.NoError(t, err)
	require.Len(t, after, 2, "点击后应多一条 card_action 消息")

	var actionMsg *model.Message
	for i := range after {
		if after[i].Type == "card_action" {
			actionMsg = &after[i]
			break
		}
	}
	require.NotNil(t, actionMsg, "应存在 card_action 消息")
	assert.Equal(t, human.ID, actionMsg.SenderID, "sender 应为点击的人类用户")

	// content 含反查的 button text，供用户端气泡显示
	var content map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(actionMsg.Content), &content))
	assert.Equal(t, "confirm", content["action_id"])
	assert.Equal(t, "确认回滚", content["action_text"], "应从原卡片反查 button text")
	assert.EqualValues(t, msg.ID, content["card_message_id"])
}

// TestForwardCardAction_PullModeSkipsOutbox 验证纯 pull 模式（mode=external_webhook 但 webhook_url 空）：
//  1. card_action Message 仍创建（agent 能拉到点击事件）。
//  2. 不入 outbox（不产生死信垃圾）。
//  3. ForwardCardAction 返回 nil（成功），而非 ErrCardActionPendingRetry。
func TestForwardCardAction_PullModeSkipsOutbox(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	_, cap := newCaptureServer(t) // 占位 webhook server，pull 模式不应被调用

	// webhook_url 空 = 纯 pull 模式
	bot, _, human := setupCardBot(t, db, "", "testsecret")

	card := `{"title":"确认回滚?","buttons":[{"id":"confirm","text":"确认回滚","value":"confirm"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	require.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	require.NoError(t, err, "pull 模式应直接成功，不报 pending-retry")

	// webhook server 不应被调用
	assert.False(t, cap.got, "纯 pull 模式不应投递 webhook")

	// 不应产生 outbox 记录
	var count int64
	db.Model(&model.BotWebhookDelivery{}).Where("bot_id = ?", bot.ID).Count(&count)
	assert.EqualValues(t, 0, count, "pull 模式不应入 outbox")

	// card_action 消息仍存在且可拉取
	after, err := svc.ListBotMessages(bot, msg.ConversationID, 0, 100)
	require.NoError(t, err)
	found := false
	for _, m := range after {
		if m.Type == "card_action" {
			found = true
			break
		}
	}
	assert.True(t, found, "card_action 消息应已创建并可拉取")
}

func TestForwardCardAction_RejectsNonCardMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	// 发一条文本消息（非卡片）
	msg, err := svc.SendOutbound(bot, human.ID, "plain text", "text", nil, nil)
	assert.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	assert.Error(t, err)
	assert.False(t, cap.got, "非卡片消息不应触发 webhook")
}

func TestForwardCardAction_RejectsNonWebhookBot(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	_, cap := newCaptureServer(t)

	// internal_ai 模式（默认，未配置 webhook）
	vUser := &model.User{Username: "bot_internal", Nickname: "InternalBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "grace", Nickname: "Grace", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "InternalBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	card := `{"buttons":[{"id":"ok","text":"OK"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "ok", "ok")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook")
	assert.False(t, cap.got)
}

func TestForwardCardAction_RejectsNonMember(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"buttons":[{"id":"confirm","text":"确认"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	// 另一个用户尝试操作他人的卡片
	intruder := &model.User{Username: "intruder", Nickname: "Intruder", Type: "user"}
	db.Create(intruder)

	err = svc.ForwardCardAction(msg.ID, intruder.ID, "confirm", "confirm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无权")
	assert.False(t, cap.got, "非会话成员不应触发 webhook")
}

func TestForwardCardAction_RejectsUnknownActionID(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"buttons":[{"id":"confirm","text":"确认"},{"id":"cancel","text":"取消"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	require.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "approve-admin", "approve-admin")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "无效")
	assert.False(t, cap.got, "未展示的按钮 action 不应触发 webhook")

	var count int64
	db.Model(&model.CardActionRecord{}).Where("message_id = ?", msg.ID).Count(&count)
	assert.EqualValues(t, 0, count, "无效 action 不应写入幂等记录")
}

func TestForwardCardAction_RejectsMissingMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, _ := newCaptureServer(t)
	_, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	err := svc.ForwardCardAction(999999, human.ID, "confirm", "confirm")
	assert.Error(t, err)
}

// TestForwardCardAction_WebhookFailEnqueuesRetry 验证：webhook 立即投递失败时，
// ForwardCardAction 不阻塞用户，返回 ErrCardActionPendingRetry 并把投递落表为 pending 等待重试。
func TestForwardCardAction_WebhookFailEnqueuesRetry(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	// webhook 始终返回 500
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"buttons":[{"id":"confirm","text":"确认"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	assert.ErrorIs(t, err, ErrCardActionPendingRetry)

	// 投递记录应处于 pending，且已计入一次失败尝试、安排了重试时间
	var delivery model.BotWebhookDelivery
	assert.NoError(t, db.First(&delivery, "bot_id = ? AND event = ?", bot.ID, "bot.card_action").Error)
	assert.Equal(t, "pending", delivery.Status)
	assert.Equal(t, 1, delivery.Attempts)
	assert.NotNil(t, delivery.NextRetryAt, "失败后应安排下次重试时间")
	assert.NotEmpty(t, delivery.LastError, "应记录失败原因")
}

// TestForwardCardAction_IdempotentSecondCall 验证：同一卡片+同一用户第二次点击被幂等拦截，
// 返回 ErrCardActionAlreadyHandled 且不重复触发 webhook。
func TestForwardCardAction_IdempotentSecondCall(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"buttons":[{"id":"confirm","text":"确认"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	// 第一次：成功转发
	assert.NoError(t, svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm"))
	assert.True(t, cap.got)

	// 第二次：幂等命中，不重复触发
	cap.got = false
	err = svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm")
	assert.ErrorIs(t, err, ErrCardActionAlreadyHandled)
	assert.False(t, cap.got, "幂等命中不应再次触发 webhook")

	// 幂等记录已落表
	var rec model.CardActionRecord
	assert.NoError(t, db.Where("message_id = ? AND user_id = ?", msg.ID, human.ID).First(&rec).Error)
	assert.Equal(t, "confirm", rec.ActionID)
}

// TestForwardCardAction_UpdateMessageReleasesLock 验证：agent 改写卡片（UpdateMessageContent）
// 会删除幂等记录，释放锁定，允许用户再次点击。
func TestForwardCardAction_UpdateMessageReleasesLock(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	card := `{"buttons":[{"id":"confirm","text":"确认"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
	assert.NoError(t, err)

	// 第一次点击锁定
	assert.NoError(t, svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm"))
	assert.ErrorIs(t, svc.ForwardCardAction(msg.ID, human.ID, "confirm", "confirm"), ErrCardActionAlreadyHandled)

	// agent 改写卡片（新按钮），应解除锁定
	newCard := `{"buttons":[{"id":"done","text":"完成"}]}`
	assert.NoError(t, svc.UpdateMessageContent(bot, msg.ID, newCard, "card"))

	// 改写后可再次点击（新 action），并落新记录
	cap.got = false
	assert.NoError(t, svc.ForwardCardAction(msg.ID, human.ID, "done", "done"))
	assert.True(t, cap.got, "卡片改写后应允许新一轮点击")
	assert.ErrorIs(t, svc.ForwardCardAction(msg.ID, human.ID, "done", "done"), ErrCardActionAlreadyHandled)

	// 仅留一条记录（改写删除旧的，新点击写新的）
	var count int64
	db.Model(&model.CardActionRecord{}).Where("message_id = ?", msg.ID).Count(&count)
	assert.Equal(t, int64(1), count)
}
