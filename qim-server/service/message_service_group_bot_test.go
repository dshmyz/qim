package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// assertOutboxCount 断言 BotWebhookDelivery 出队行数。
func assertOutboxCount(t *testing.T, db *gorm.DB, want int64, msg string) {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.BotWebhookDelivery{}).Count(&count).Error)
	assert.Equal(t, want, count, msg)
}

// TestHandleGroupBotMention_ForwardsToWebhook 群聊外部 agent：群成员 @ bot 虚拟用户时，
// 转发 bot.message 到 bot webhook（thread_id = 群会话 id、user_id = 群成员）。
func TestHandleGroupBotMention_ForwardsToWebhook(t *testing.T) {
	var mu sync.Mutex
	var received BotWebhookPayload
	var gotEvent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		_ = json.Unmarshal(body, &received)
		gotEvent = r.Header.Get("X-QIM-Event")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "gbot", Nickname: "群机器人", Type: "bot"}
	human := &model.User{Username: "ghuman", Nickname: "群成员", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	// 外部 webhook agent。BotConversation 关联到群会话 = 「bot 已入群」。
	bot := &model.Bot{
		Name: "WebhookAgent", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &botUser.ID,
		Config:        `{"mode":"external_webhook","webhook_url":"` + srv.URL + `","webhook_secret":"secret"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	group := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: human.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: botUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: group.ID}).Error)

	// 群成员先发一条消息，供 forwardBotMessageToWebhook 取 msg_type/message_id
	msg := &model.Message{ConversationID: group.ID, SenderID: human.ID, Type: "text", Content: "@群机器人 你好"}
	require.NoError(t, db.Create(msg).Error)

	svc.HandleGroupBotMention(group.ID, human.ID, []uint{botUser.ID}, "@群机器人 你好")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "bot.message", gotEvent, "应转发 bot.message 事件")
	assert.Equal(t, bot.ID, received.BotID)
	assert.Equal(t, group.ID, received.ThreadID, "thread_id 应为群会话 id")
	assert.Equal(t, human.ID, received.UserID, "user_id 应为 @ 触发的群成员")
	assert.Equal(t, "@群机器人 你好", received.Content)
}

// TestHandleGroupBotMention_IgnoresNoMentionOrNonGroup 未被 @ / 非群会话 / 内部 AI bot 时不转发。
func TestHandleGroupBotMention_IgnoresNoMentionOrNonGroup(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "gbot2", Nickname: "群机器人2", Type: "bot"}
	human := &model.User{Username: "gh2", Nickname: "成员2", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "Agent2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &botUser.ID,
		Config: `{"mode":"external_webhook","webhook_url":"http://127.0.0.1:1/x"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	group := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: human.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: botUser.ID}).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: group.ID}).Error)

	// 1) 未被 @ 到 bot → 不转发
	svc.HandleGroupBotMention(group.ID, human.ID, []uint{human.ID}, "普通消息")
	assertOutboxCount(t, db, 0, "未 @ 到 bot 不应入队")

	// 2) 非群会话（单聊 bot 会话）→ 不转发
	single := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(single).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: single.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: single.ID, UserID: human.ID}).Error)
	svc.HandleGroupBotMention(single.ID, human.ID, []uint{botUser.ID}, "@群机器人2 你好")
	assertOutboxCount(t, db, 0, "非群会话不应触发群转发")

	// 3) 内部 AI bot（非外部 webhook）@ 到 → 不转发
	internalUser := &model.User{Username: "ib", Nickname: "内部", Type: "bot"}
	require.NoError(t, db.Create(internalUser).Error)
	internalBot := &model.Bot{Name: "InternalAI", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &internalUser.ID}
	require.NoError(t, db.Create(internalBot).Error)
	group2 := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(group2).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group2.ID, UserID: human.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group2.ID, UserID: internalUser.ID}).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: internalBot.ID, ConversationID: group2.ID}).Error)
	svc.HandleGroupBotMention(group2.ID, human.ID, []uint{internalUser.ID}, "@内部 你好")
	assertOutboxCount(t, db, 0, "内部 AI bot 不应走群转发")
}

// TestHandleGroupBotMention_PullModeNotices 外部 webhook bot 未配回调地址（pull 模式）时，
// @ 它不投递 webhook，但会在群内落一条系统提示，避免成员无感知。
func TestHandleGroupBotMention_PullModeNotices(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "pullbot", Nickname: "拉取机器人", Type: "bot"}
	human := &model.User{Username: "ph", Nickname: "成员", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	// 外部 agent 但 webhook_url 为空 → pull 模式
	bot := &model.Bot{
		Name: "PullAgent", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &botUser.ID,
		Config: `{"mode":"external_webhook","webhook_url":""}`,
	}
	require.NoError(t, db.Create(bot).Error)

	group := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: human.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: botUser.ID}).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: group.ID}).Error)

	svc.HandleGroupBotMention(group.ID, human.ID, []uint{botUser.ID}, "@拉取机器人 在吗")

	// 不应投递 webhook（空 URL）
	assertOutboxCount(t, db, 0, "pull 模式不应入队 webhook 投递")

	// 应落一条系统提示消息
	var notice model.Message
	err := db.Where("conversation_id = ? AND type = ?", group.ID, "system").Order("id DESC").First(&notice).Error
	require.NoError(t, err, "应创建 pull 模式提示消息")
	assert.Contains(t, notice.Content, "PullAgent")
	assert.Contains(t, notice.Content, "pull 模式")
}

// TestHandleBotMessage_ExternalWebhook_ForwardsToWebhook 单聊外部 agent：用户在 1:1 bot 会话
// 中发消息时，转发 bot.message 到 bot webhook（thread_id = 会话 id）。
func TestHandleBotMessage_ExternalWebhook_ForwardsToWebhook(t *testing.T) {
	var mu sync.Mutex
	var received BotWebhookPayload
	var gotEvent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		_ = json.Unmarshal(body, &received)
		gotEvent = r.Header.Get("X-QIM-Event")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "sbot", Nickname: "单聊机器人", Type: "bot"}
	human := &model.User{Username: "shuman", Nickname: "用户", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "SingleAgent", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &botUser.ID,
		Config:        `{"mode":"external_webhook","webhook_url":"` + srv.URL + `","webhook_secret":"s3cret"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID}).Error)

	// 用户先发一条消息
	msg := &model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "你好"}
	require.NoError(t, db.Create(msg).Error)

	svc.handleBotMessage(human.ID, conv.ID, "text", "你好")

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "bot.message", gotEvent, "应转发 bot.message 事件")
	assert.Equal(t, bot.ID, received.BotID)
	assert.Equal(t, conv.ID, received.ThreadID, "thread_id 应为 bot 会话 id")
	assert.Equal(t, human.ID, received.UserID)
	assert.Equal(t, "你好", received.Content)
	assert.NotEmpty(t, received.DeliveryID, "应生成 delivery_id")
}

// TestHandleBotMessage_ExternalWebhook_PullModeSkips 单聊外部 agent pull 模式（webhook_url 空）时
// 不入 outbox。
func TestHandleBotMessage_ExternalWebhook_PullModeSkips(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "pull1", Nickname: "拉取机器人1", Type: "bot"}
	human := &model.User{Username: "pullh", Nickname: "用户", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "PullSingle", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &botUser.ID,
		Config: `{"mode":"external_webhook","webhook_url":""}`,
	}
	require.NoError(t, db.Create(bot).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID}).Error)

	msg := &model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi"}
	require.NoError(t, db.Create(msg).Error)

	svc.handleBotMessage(human.ID, conv.ID, "text", "hi")

	assertOutboxCount(t, db, 0, "pull 模式不应入队")
}

// TestHandleBotMessage_InternalAI_SkipsWebhook 内部 AI bot 不走 webhook 路径。
func TestHandleBotMessage_InternalAI_SkipsWebhook(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "ibot", Nickname: "内部AI", Type: "bot"}
	human := &model.User{Username: "ih", Nickname: "用户", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "InternalBot", Type: model.BotTypeAssistant, IsActive: true, VirtualUserID: &botUser.ID,
		Config: `{}`, // mode 为空 = internal_ai
	}
	require.NoError(t, db.Create(bot).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID}).Error)

	msg := &model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "hi"}
	require.NoError(t, db.Create(msg).Error)

	svc.handleBotMessage(human.ID, conv.ID, "text", "hi")

	assertOutboxCount(t, db, 0, "内部 AI bot 不应入队 webhook")
}

// TestE2E_1BotMessage_WebhookDelivery 端到端：模拟用户在 1:1 bot 会话发消息，
// 直接调 handleBotMessage（与 submitBotReply 内部调用等价），验证完整链路：
// handleBotMessage → forwardBotMessageToWebhook → outbox → mock server 收到 webhook。
// 注：submitBotReply 通过 goroutine 异步调 handleBotMessage，此处同步调用覆盖相同逻辑。
func TestE2E_1BotMessage_WebhookDelivery(t *testing.T) {
	webhookCh := make(chan BotWebhookPayload, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p BotWebhookPayload
		if err := json.Unmarshal(body, &p); err == nil {
			webhookCh <- p
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "e2e_bot", Nickname: "E2E机器人", Type: "bot"}
	human := &model.User{Username: "e2e_user", Nickname: "E2E用户", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "E2EAgent", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &botUser.ID,
		Config:        `{"mode":"external_webhook","webhook_url":"` + srv.URL + `","webhook_secret":"s3cret"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID}).Error)

	// 用户消息落库（forwardBotMessageToWebhook 要取 lastMsg）
	msg := &model.Message{ConversationID: conv.ID, SenderID: human.ID, Type: "text", Content: "E2E测试消息"}
	require.NoError(t, db.Create(msg).Error)

	// 直接调 handleBotMessage（等价于 submitBotReply 内部调用）
	svc.handleBotMessage(human.ID, conv.ID, "text", "E2E测试消息")

	select {
	case received := <-webhookCh:
		assert.Equal(t, "bot.message", received.Event)
		assert.Equal(t, bot.ID, received.BotID)
		assert.Equal(t, conv.ID, received.ThreadID)
		assert.Equal(t, human.ID, received.UserID)
		assert.Equal(t, "E2E测试消息", received.Content)
		assert.Equal(t, "text", received.MsgType)
		assert.NotEmpty(t, received.DeliveryID)
	case <-time.After(3 * time.Second):
		t.Fatal("超时：mock server 未收到 webhook")
	}

	var delivery model.BotWebhookDelivery
	require.NoError(t, db.Where("bot_id = ?", bot.ID).First(&delivery).Error)
	assert.Equal(t, "done", delivery.Status, "outbox 记录应为 done")
}

// TestE2E_GroupAtBot_WebhookDelivery 端到端：通过 SendMessage 模拟群聊 @bot，验证完整链路。
func TestE2E_GroupAtBot_WebhookDelivery(t *testing.T) {
	webhookCh := make(chan BotWebhookPayload, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var p BotWebhookPayload
		if err := json.Unmarshal(body, &p); err == nil {
			webhookCh <- p
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	botUser := &model.User{Username: "g_e2e_bot", Nickname: "群E2E机器人", Type: "bot"}
	human := &model.User{Username: "g_e2e_user", Nickname: "群E2E用户", Type: "user"}
	require.NoError(t, db.Create(botUser).Error)
	require.NoError(t, db.Create(human).Error)

	bot := &model.Bot{
		Name: "GroupE2EAgent", Type: model.BotTypeCustom, IsActive: true,
		VirtualUserID: &botUser.ID,
		Config:        `{"mode":"external_webhook","webhook_url":"` + srv.URL + `","webhook_secret":"sec"}`,
	}
	require.NoError(t, db.Create(bot).Error)

	group := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: human.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: botUser.ID, Role: "member"}).Error)
	require.NoError(t, db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: group.ID}).Error)

	// 发一条 @bot 的群消息
	botMention := mention.Encode(botUser.ID, "群E2E机器人")
	content := botMention + " 你好"
	msg, err := svc.SendMessage(group.ID, human.ID, "text", content, nil)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// HandleGroupBotMention 需要从消息内容解析 mention token
	mentionUserIDs := svc.MentionUserIDsForAI(group.ID, content)
	require.NotEmpty(t, mentionUserIDs, "应解析出 bot 的 virtual user ID")

	// 手动触发群 @bot 路径（生产中由 OnMessageSent 回调触发）
	svc.HandleGroupBotMention(group.ID, human.ID, mentionUserIDs, content)

	select {
	case received := <-webhookCh:
		assert.Equal(t, "bot.message", received.Event)
		assert.Equal(t, bot.ID, received.BotID)
		assert.Equal(t, group.ID, received.ThreadID, "群聊 thread_id 应为群会话 id")
		assert.Equal(t, human.ID, received.UserID)
		assert.Equal(t, content, received.Content)
	case <-time.After(3 * time.Second):
		t.Fatal("超时：mock server 未收到群 @bot webhook")
	}

	var delivery model.BotWebhookDelivery
	require.NoError(t, db.Where("bot_id = ?", bot.ID).First(&delivery).Error)
	assert.Equal(t, "done", delivery.Status)
}

// TestE2E_DiagnoseAllSilentFailures 诊断测试：逐一检查所有可能导致零投递记录的条件。
// 每个子测试模拟一种故障场景，确认代码能正确拦截并日志记录（不进 outbox）。
func TestE2E_DiagnoseAllSilentFailures(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewMessageService(db, nil, nil)

	cases := []struct {
		name   string
		setup  func() (convID, userID uint)
		expect string
	}{
		{
			name: "bot_conversations 记录缺失",
			setup: func() (uint, uint) {
				botUser := &model.User{Username: "d1", Type: "bot"}
				human := &model.User{Username: "d1h", Type: "user"}
				db.Create(botUser); db.Create(human)
				bot := &model.Bot{Name: "D1", IsActive: true, VirtualUserID: &botUser.ID,
					Config: `{"mode":"external_webhook","webhook_url":"http://127.0.0.1:1"}`}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				// 故意不创建 bot_conversations 记录
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				return conv.ID, human.ID
			},
			expect: "bot_conversations 缺失 → 静默跳过",
		},
		{
			name: "bot 已软删除",
			setup: func() (uint, uint) {
				botUser := &model.User{Username: "d2", Type: "bot"}
				human := &model.User{Username: "d2h", Type: "user"}
				db.Create(botUser); db.Create(human)
				bot := &model.Bot{Name: "D2", IsActive: true, VirtualUserID: &botUser.ID,
					Config: `{"mode":"external_webhook","webhook_url":"http://127.0.0.1:1"}`}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID})
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				// 软删除 bot
				db.Delete(bot)
				return conv.ID, human.ID
			},
			expect: "bot 已删除 → 静默跳过",
		},
		{
			name: "bot 缺少 virtual_user_id",
			setup: func() (uint, uint) {
				human := &model.User{Username: "d3h", Type: "user"}
				db.Create(human)
				bot := &model.Bot{Name: "D3", IsActive: true, VirtualUserID: nil,
					Config: `{"mode":"external_webhook","webhook_url":"http://127.0.0.1:1"}`}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID})
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				return conv.ID, human.ID
			},
			expect: "virtual_user_id=nil → 静默跳过",
		},
		{
			name: "config 为空（非 external_webhook）",
			setup: func() (uint, uint) {
				botUser := &model.User{Username: "d4", Type: "bot"}
				human := &model.User{Username: "d4h", Type: "user"}
				db.Create(botUser); db.Create(human)
				bot := &model.Bot{Name: "D4", IsActive: true, VirtualUserID: &botUser.ID, Config: ""}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID})
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				return conv.ID, human.ID
			},
			expect: "config 空 → 走内部 AI，不投 webhook",
		},
		{
			name: "mode 写错（拼写错误）",
			setup: func() (uint, uint) {
				botUser := &model.User{Username: "d5", Type: "bot"}
				human := &model.User{Username: "d5h", Type: "user"}
				db.Create(botUser); db.Create(human)
				bot := &model.Bot{Name: "D5", IsActive: true, VirtualUserID: &botUser.ID,
					Config: `{"mode":"external_webhookk","webhook_url":"http://127.0.0.1:1"}`}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID})
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				return conv.ID, human.ID
			},
			expect: "mode 拼错 → 不匹配 IsExternalWebhook，走内部 AI",
		},
		{
			name: "bot is_active=false（1:1 不检查，群聊检查）",
			setup: func() (uint, uint) {
				botUser := &model.User{Username: "d6", Type: "bot"}
				human := &model.User{Username: "d6h", Type: "user"}
				db.Create(botUser); db.Create(human)
				bot := &model.Bot{Name: "D6", IsActive: false, VirtualUserID: &botUser.ID,
					Config: `{"mode":"external_webhook","webhook_url":"http://127.0.0.1:1"}`}
				db.Create(bot)
				conv := &model.Conversation{Type: "bot"}
				db.Create(conv)
				db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: conv.ID})
				db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: human.ID})
				return conv.ID, human.ID
			},
			expect: "1:1 不检查 is_active，仍会尝试投递（失败是网络问题）",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			convID, userID := tc.setup()
			svc.handleBotMessage(userID, convID, "text", "诊断消息")
			// is_active=false 在 1:1 路径不被拦截，会尝试投递（URL 不可达会进重试队列）
			if tc.name == "bot is_active=false（1:1 不检查，群聊检查）" {
				var count int64
				db.Model(&model.BotWebhookDelivery{}).Count(&count)
				assert.Equal(t, int64(1), count, "1:1 不检查 is_active，仍会入队")
			} else {
				assertOutboxCount(t, db, 0, tc.expect)
			}
		})
	}
}

