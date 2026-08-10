package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
)

// TestBotLoop_InboundReplyForwardedToWebhook 验证 inbound 半闭环：
// 用户在 external_webhook 模式的 bot 会话中发消息 ->
// handleBotMessage 把回复转发到 agent webhook（含 HMAC 签名 + thread_id + message_id）。
// 生产中 handleBotMessage 由 SendMessage 在 convType=="bot" 时 SafeGo 异步触发；
// 这里直接同步调用以断言。
func TestBotLoop_InboundReplyForwardedToWebhook(t *testing.T) {
	db := setupBotMessagingTestDB(t)

	vUser := &model.User{Username: "bot_loop", Nickname: "LoopBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "dave", Nickname: "Dave", Type: "user"}
	db.Create(human)

	secret := "loop_secret"
	type received struct {
		body  []byte
		sig   string
		event string
	}
	got := make(chan received, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- received{body: body, sig: r.Header.Get("X-QIM-Signature"), event: r.Header.Get("X-QIM-Event")}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgJSON := `{"mode":"external_webhook","webhook_url":"` + srv.URL + `","webhook_secret":"` + secret + `"}`
	bot := &model.Bot{Name: "LoopBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID, Config: cfgJSON}
	db.Create(bot)

	// 建 bot 会话
	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	// 用户先发一条消息（成为 lastMsg，供 webhook 取 message_id）
	userMsg := &model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "我看下日志", Origin: "user"}
	db.Create(userMsg)

	// MessageService：hub/ai 均可 nil，external_webhook 分支不依赖它们
	msgSvc := NewMessageService(db, nil, nil)

	// 触发 inbound 转发
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "我看下日志")

	select {
	case r := <-got:
		assert.Equal(t, "bot.message", r.event)
		// 校验 HMAC-SHA256 签名
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(r.body)
		assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), r.sig)

		var payload BotWebhookPayload
		assert.NoError(t, json.Unmarshal(r.body, &payload))
		assert.Equal(t, bot.ID, payload.BotID)
		assert.Equal(t, conv.ID, payload.ThreadID)
		assert.Equal(t, human.ID, payload.UserID)
		assert.Equal(t, "我看下日志", payload.Content)
		assert.Equal(t, userMsg.ID, payload.MessageID)
		assert.NotEmpty(t, payload.DeliveryID)
	case <-time.After(2 * time.Second):
		t.Fatal("webhook 未在超时内收到")
	}
}

// TestBotLoop_InternalAiModeDoesNotForward 验证默认 internal_ai 模式不触发 webhook 转发，
// 即既有行为不被破坏。
func TestBotLoop_InternalAiModeDoesNotForward(t *testing.T) {
	db := setupBotMessagingTestDB(t)

	vUser := &model.User{Username: "bot_ai", Nickname: "AIBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "eve", Nickname: "Eve", Type: "user"}
	db.Create(human)

	called := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// 默认 internal_ai 模式（webhook_url 仍配置，但 mode 不是 external_webhook）
	cfgJSON := `{"mode":"internal_ai","webhook_url":"` + srv.URL + `","webhook_secret":"x"}`
	bot := &model.Bot{Name: "AIBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID, Config: cfgJSON}
	db.Create(bot)

	bmSvc := NewBotMessagingService(db, nil)
	conv, _, err := bmSvc.EnsureBotConversation(bot.ID, human.ID)
	assert.NoError(t, err)

	db.Create(&model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi", Origin: "user"})

	msgSvc := NewMessageService(db, nil, nil) // aiService=nil，internal_ai 分支会走"AI 服务未配置"兜底但不转发
	msgSvc.handleBotMessage(human.ID, conv.ID, "text", "hi")

	select {
	case <-called:
		t.Fatal("internal_ai 模式不应转发到 webhook")
	case <-time.After(300 * time.Millisecond):
		// 预期：webhook 未被调用
	}
}
