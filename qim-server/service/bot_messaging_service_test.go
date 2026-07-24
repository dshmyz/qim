package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupBotMessagingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Conversation{}, &model.ConversationMember{},
		&model.Message{}, &model.Bot{}, &model.BotConversation{},
		&model.BotWebhookDelivery{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestSendOutbound_CreatesBotMessageAndUnread(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil) // hub=nil，跳过 WS

	vUser := &model.User{Username: "bot_1", Nickname: "AgentBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "alice", Nickname: "Alice", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "AgentBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	msg, err := svc.SendOutbound(bot, human.ID, "构建失败，请确认", "text", nil)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "bot", msg.Origin)
	assert.Equal(t, vUser.ID, msg.SenderID)
	assert.Equal(t, "构建失败，请确认", msg.Content)

	// 会话已建且为 bot 类型，last_message_id 已更新
	var conv model.Conversation
	db.First(&conv, msg.ConversationID)
	assert.Equal(t, "bot", conv.Type)
	assert.NotNil(t, conv.LastMessageID)
	assert.Equal(t, msg.ID, *conv.LastMessageID)

	// 人类成员未读 +1，bot 虚拟用户未读为 0
	var humanMember, botMember model.ConversationMember
	db.Where("conversation_id = ? AND user_id = ?", conv.ID, human.ID).First(&humanMember)
	db.Where("conversation_id = ? AND user_id = ?", conv.ID, vUser.ID).First(&botMember)
	assert.Equal(t, 1, humanMember.UnreadCount)
	assert.Equal(t, 0, botMember.UnreadCount)

	// BotConversation 关联已建
	var botConv model.BotConversation
	db.Where("bot_id = ? AND user_id = ?", bot.ID, human.ID).First(&botConv)
	assert.Equal(t, conv.ID, botConv.ConversationID)
}

func TestSendOutbound_ReusesThreadID(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_2", Nickname: "Bot2", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "bob", Nickname: "Bob", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "Bot2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	// 第一条：建会话
	msg1, err := svc.SendOutbound(bot, human.ID, "first", "text", nil)
	assert.NoError(t, err)

	// 第二条：带 thread_id 复用同一会话
	threadID := msg1.ConversationID
	msg2, err := svc.SendOutbound(bot, human.ID, "second", "text", &threadID)
	assert.NoError(t, err)
	assert.Equal(t, threadID, msg2.ConversationID)
}

func TestSendOutbound_RejectsWrongThreadID(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_3", Nickname: "Bot3", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "carol", Nickname: "Carol", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "Bot3", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	wrongID := uint(9999)
	_, err := svc.SendOutbound(bot, human.ID, "x", "text", &wrongID)
	assert.Error(t, err)
}

func TestSendOutbound_RejectsBotToBot(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_4", Nickname: "Bot4", Type: "bot"}
	db.Create(vUser)
	otherBot := &model.User{Username: "bot_5", Nickname: "Bot5", Type: "bot"}
	db.Create(otherBot)
	bot := &model.Bot{Name: "Bot4", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	// 不允许 bot 给 bot 类型用户发消息
	_, err := svc.SendOutbound(bot, otherBot.ID, "hi", "text", nil)
	assert.Error(t, err)
}

func TestParseBotConfig_IsExternalWebhook(t *testing.T) {
	assert.False(t, ParseBotConfig("").IsExternalWebhook())
	assert.False(t, ParseBotConfig(`{"mode":"internal_ai"}`).IsExternalWebhook())
	assert.False(t, ParseBotConfig(`{"mode":"external_webhook"}`).IsExternalWebhook()) // 缺 url
	assert.True(t, ParseBotConfig(`{"mode":"external_webhook","webhook_url":"http://x"}`).IsExternalWebhook())
}

func TestValidateCardContent(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"合法卡片", `{"title":"确认?","buttons":[{"id":"confirm","text":"确认","value":"confirm"}]}`, false},
		{"仅有按钮", `{"buttons":[{"id":"ok","text":"OK"}]}`, false},
		{"非 JSON", `not-json`, true},
		{"无按钮", `{"title":"x"}`, true},
		{"空按钮数组", `{"buttons":[]}`, true},
		{"按钮缺 id", `{"buttons":[{"text":"x"}]}`, true},
		{"按钮缺 text", `{"buttons":[{"id":"x"}]}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCardContent(tc.content)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendOutbound_AcceptsValidCard(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_card1", Nickname: "CardBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "dave", Nickname: "Dave", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "CardBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	card := `{"title":"确认回滚?","buttons":[{"id":"confirm","text":"确认回滚","style":"primary","value":"confirm"},{"id":"cancel","text":"取消","value":"cancel"}]}`
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, "card", msg.Type)
	assert.Equal(t, "bot", msg.Origin)
	assert.Equal(t, card, msg.Content)
}

func TestSendOutbound_RejectsInvalidCard(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_card2", Nickname: "CardBot2", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "erin", Nickname: "Erin", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "CardBot2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	cases := []string{
		`not-json`,
		`{"title":"x"}`,                 // 无按钮
		`{"buttons":[{"text":"x"}]}`,    // 按钮缺 id
	}
	for i, content := range cases {
		_, err := svc.SendOutbound(bot, human.ID, content, "card", nil)
		assert.Error(t, err, "case %d 应被拒绝", i)
	}
}
