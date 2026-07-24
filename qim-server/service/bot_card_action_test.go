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
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil)
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

func TestForwardCardAction_RejectsNonCardMessage(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)
	srv, cap := newCaptureServer(t)

	bot, _, human := setupCardBot(t, db, srv.URL, "testsecret")

	// 发一条文本消息（非卡片）
	msg, err := svc.SendOutbound(bot, human.ID, "plain text", "text", nil)
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
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil)
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
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil)
	assert.NoError(t, err)

	// 另一个用户尝试操作他人的卡片
	intruder := &model.User{Username: "intruder", Nickname: "Intruder", Type: "user"}
	db.Create(intruder)

	err = svc.ForwardCardAction(msg.ID, intruder.ID, "confirm", "confirm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无权")
	assert.False(t, cap.got, "非会话成员不应触发 webhook")
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
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil)
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
