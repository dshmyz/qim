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

