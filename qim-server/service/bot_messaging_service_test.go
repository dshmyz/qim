package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBotMessagingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// 用临时文件数据库代替内存库，避免 SQLite 内存库 + goroutine 并发竞态
	// （内存库每连接隔离，GORM 连接池给 goroutine 分配不同连接时看不到表）。
	dbPath := t.TempDir() + "/test.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.Conversation{}, &model.ConversationMember{},
		&model.Message{}, &model.Bot{}, &model.BotConversation{},
		&model.BotWebhookDelivery{}, &model.CardActionRecord{}, &model.Group{},
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

	msg, err := svc.SendOutbound(bot, human.ID, "构建失败，请确认", "text", nil, nil)
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

	// BotConversation 关联已建（按 conversation_id 反查，user_id 维度已移除）
	var botConv model.BotConversation
	db.Where("bot_id = ? AND conversation_id = ?", bot.ID, conv.ID).First(&botConv)
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
	msg1, err := svc.SendOutbound(bot, human.ID, "first", "text", nil, nil)
	assert.NoError(t, err)

	// 第二条：带 thread_id 复用同一会话
	threadID := msg1.ConversationID
	msg2, err := svc.SendOutbound(bot, human.ID, "second", "text", &threadID, nil)
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
	_, err := svc.SendOutbound(bot, human.ID, "x", "text", &wrongID, nil)
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
	_, err := svc.SendOutbound(bot, otherBot.ID, "hi", "text", nil, nil)
	assert.Error(t, err)
}

func TestSendOutbound_SetsQuotedMessageIDForReply(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_reply_1", Nickname: "ReplyBot", Type: "bot"}
	db.Create(vUser)
	human := &model.User{Username: "reply_user", Nickname: "Reply User", Type: "user"}
	db.Create(human)
	bot := &model.Bot{Name: "ReplyBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	quoted, err := svc.SendOutbound(bot, human.ID, "first", "text", nil, nil)
	assert.NoError(t, err)

	threadID := quoted.ConversationID
	replyToID := quoted.ID
	reply, err := svc.SendOutbound(bot, human.ID, "second", "text", &threadID, &replyToID)
	assert.NoError(t, err)
	if assert.NotNil(t, reply.QuotedMessageID) {
		assert.Equal(t, quoted.ID, *reply.QuotedMessageID)
	}
}

func TestSendOutbound_RejectsReplyToMessageFromAnotherConversation(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_reply_2", Nickname: "ReplyBot2", Type: "bot"}
	db.Create(vUser)
	humanA := &model.User{Username: "reply_user_a", Nickname: "Reply User A", Type: "user"}
	humanB := &model.User{Username: "reply_user_b", Nickname: "Reply User B", Type: "user"}
	db.Create(humanA)
	db.Create(humanB)
	bot := &model.Bot{Name: "ReplyBot2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	msgA, err := svc.SendOutbound(bot, humanA.ID, "conv-a", "text", nil, nil)
	assert.NoError(t, err)
	msgB, err := svc.SendOutbound(bot, humanB.ID, "conv-b", "text", nil, nil)
	assert.NoError(t, err)

	threadID := msgB.ConversationID
	replyToID := msgA.ID
	_, err = svc.SendOutbound(bot, humanB.ID, "reply", "text", &threadID, &replyToID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "引用消息不属于当前会话")
}

func TestResolveUserID_RejectsAmbiguousNickname(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	db.Create(&model.User{Username: "dup_1", Nickname: "同名", Type: "user"})
	db.Create(&model.User{Username: "dup_2", Nickname: "同名", Type: "user"})

	_, err := svc.ResolveUserID("同名")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "昵称不唯一")
}

func TestResolveBotThread_RejectsAmbiguousNickname(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "bot_resolve_1", Nickname: "ResolveBot", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "ResolveBot", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	userA := &model.User{Username: "resolve_a", Nickname: "同名会话", Type: "user"}
	userB := &model.User{Username: "resolve_b", Nickname: "同名会话", Type: "user"}
	db.Create(userA)
	db.Create(userB)

	convA := &model.Conversation{Type: "bot"}
	convB := &model.Conversation{Type: "bot"}
	db.Create(convA)
	db.Create(convB)
	db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: convA.ID})
	db.Create(&model.BotConversation{BotID: bot.ID, ConversationID: convB.ID})

	_, err := svc.ResolveBotThread(bot, "同名会话")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "昵称不唯一")
}

func TestParseBotConfig_IsExternalWebhook(t *testing.T) {
	// IsExternalWebhook 只判身份（mode）：url 空也是外部 bot，走纯 pull 模式。
	assert.False(t, ParseBotConfig("").IsExternalWebhook())
	assert.False(t, ParseBotConfig(`{"mode":"internal_ai"}`).IsExternalWebhook())
	assert.True(t, ParseBotConfig(`{"mode":"external_webhook"}`).IsExternalWebhook()) // 缺 url = 纯 pull，仍是外部 bot
	assert.True(t, ParseBotConfig(`{"mode":"external_webhook","webhook_url":"http://x"}`).IsExternalWebhook())

	// HasWebhook 判是否需要投 webhook：url 空就不投（纯 pull），url 非空才投。
	assert.False(t, ParseBotConfig(`{"mode":"external_webhook"}`).HasWebhook())
	assert.True(t, ParseBotConfig(`{"mode":"external_webhook","webhook_url":"http://x"}`).HasWebhook())
	assert.False(t, ParseBotConfig(`{"mode":"internal_ai"}`).HasWebhook())
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
	msg, err := svc.SendOutbound(bot, human.ID, card, "card", nil, nil)
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
		`{"title":"x"}`,              // 无按钮
		`{"buttons":[{"text":"x"}]}`, // 按钮缺 id
	}
	for i, content := range cases {
		_, err := svc.SendOutbound(bot, human.ID, content, "card", nil, nil)
		assert.Error(t, err, "case %d 应被拒绝", i)
	}
}

// TestEnsureBotGroupConversation 拉 bot 进群：建成员关系 + BotConversation 关联，幂等；非群会话拒绝。
func TestEnsureBotGroupConversation(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "gbot_v", Nickname: "群Agent", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "GroupAgent", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	group := &model.Conversation{Type: "group"}
	db.Create(group)
	// 预置一位群成员（人类）以模拟真实群
	human := &model.User{Username: "gm", Nickname: "GM", Type: "user"}
	db.Create(human)
	db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: human.ID, Role: "owner"})

	// 拉 bot 进群
	botConv, err := svc.EnsureBotGroupConversation(bot.ID, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, group.ID, botConv.ConversationID)

	// bot 虚拟用户已为成员
	var member model.ConversationMember
	err = db.Where("conversation_id = ? AND user_id = ?", group.ID, vUser.ID).First(&member).Error
	assert.NoError(t, err)
	assert.Equal(t, "member", member.Role)

	// BotConversation 关联已建
	var bc model.BotConversation
	assert.NoError(t, db.Where("bot_id = ? AND conversation_id = ?", bot.ID, group.ID).First(&bc).Error)

	// 幂等：再次调用不报错、不产生重复关联
	again, err := svc.EnsureBotGroupConversation(bot.ID, group.ID)
	assert.NoError(t, err)
	assert.Equal(t, group.ID, again.ConversationID)
	var cnt int64
	db.Model(&model.BotConversation{}).Where("bot_id = ? AND conversation_id = ?", bot.ID, group.ID).Count(&cnt)
	assert.Equal(t, int64(1), cnt)

	// 非群会话拒绝
	single := &model.Conversation{Type: "bot"}
	db.Create(single)
	_, err = svc.EnsureBotGroupConversation(bot.ID, single.ID)
	assert.Error(t, err)
}

// TestSendOutboundByConversation_Group 按群会话发送：创建消息、机器人自身已读、人类成员未读 +1。
func TestSendOutboundByConversation_Group(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil) // hub=nil，跳过 WS

	vUser := &model.User{Username: "gbot2_v", Nickname: "群Agent2", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "GroupAgent2", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	group := &model.Conversation{Type: "group"}
	db.Create(group)
	alice := &model.User{Username: "ga", Nickname: "GA", Type: "user"}
	bob := &model.User{Username: "gb", Nickname: "GB", Type: "user"}
	db.Create(alice)
	db.Create(bob)
	db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: alice.ID, Role: "member"})
	db.Create(&model.ConversationMember{ConversationID: group.ID, UserID: bob.ID, Role: "member"})

	// 先拉 bot 进群
	_, err := svc.EnsureBotGroupConversation(bot.ID, group.ID)
	assert.NoError(t, err)

	msg, err := svc.SendOutboundByConversation(bot, group.ID, "群内来自 agent 的发言", "text", nil)
	assert.NoError(t, err)
	assert.NotNil(t, msg)
	assert.Equal(t, group.ID, msg.ConversationID)
	assert.Equal(t, vUser.ID, msg.SenderID)
	assert.Equal(t, "bot", msg.Origin)

	// 人类成员未读 +1，bot 虚拟用户自身未读为 0
	for _, uid := range []uint{alice.ID, bob.ID} {
		var m model.ConversationMember
		db.Where("conversation_id = ? AND user_id = ?", group.ID, uid).First(&m)
		assert.Equal(t, 1, m.UnreadCount)
	}
	var botMem model.ConversationMember
	db.Where("conversation_id = ? AND user_id = ?", group.ID, vUser.ID).First(&botMem)
	assert.Equal(t, 0, botMem.UnreadCount)

	// 会话 last_message 已更新
	var conv model.Conversation
	db.First(&conv, group.ID)
	assert.NotNil(t, conv.LastMessageID)
	assert.Equal(t, msg.ID, *conv.LastMessageID)
}

// TestSendOutboundByConversation_NotOwned 未关联该会话的 bot 无法按会话发送。
func TestSendOutboundByConversation_NotOwned(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "gbot3_v", Nickname: "群Agent3", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "GroupAgent3", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	group := &model.Conversation{Type: "group"}
	db.Create(group)
	// 未把 bot 拉进群（无 BotConversation 关联）
	_, err := svc.SendOutboundByConversation(bot, group.ID, "hi", "text", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "会话不属于该 bot")
}

// TestListBotGroupConversations 只列出 bot 已入群的群会话（含群名），未入群/非群会话不出现。
func TestListBotGroupConversations(t *testing.T) {
	db := setupBotMessagingTestDB(t)
	svc := NewBotMessagingService(db, nil)

	vUser := &model.User{Username: "gbot4_v", Nickname: "群Agent4", Type: "bot"}
	db.Create(vUser)
	bot := &model.Bot{Name: "GroupAgent4", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &vUser.ID}
	db.Create(bot)

	// 群 A：bot 已入群 → 应列出
	groupA := &model.Conversation{Type: "group"}
	db.Create(groupA)
	db.Create(&model.Group{ConversationID: groupA.ID, GroupType: "group", Name: "项目组A", CreatorID: 1})
	_, err := svc.EnsureBotGroupConversation(bot.ID, groupA.ID)
	require.NoError(t, err)

	// 群 B：bot 未入群（无关联）→ 不应列出
	groupB := &model.Conversation{Type: "group"}
	db.Create(groupB)

	// 单聊 bot 会话：bot 关联但 type=bot → 不应列出
	single := &model.Conversation{Type: "bot"}
	db.Create(single)
	_, _, err = svc.EnsureBotConversation(bot.ID, 99) // 会新建一个 type=bot 会话
	require.NoError(t, err)
	_ = single

	groups, err := svc.ListBotGroupConversations(bot.ID)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	assert.Equal(t, groupA.ID, groups[0].ConversationID)
	assert.Equal(t, "项目组A", groups[0].GroupName)

	// 其它 bot：无任何群
	otherUser := &model.User{Username: "gbot5_v", Nickname: "Agent5", Type: "bot"}
	db.Create(otherUser)
	other := &model.Bot{Name: "Agent5", Type: model.BotTypeCustom, IsActive: true, VirtualUserID: &otherUser.ID}
	db.Create(other)
	empty, err := svc.ListBotGroupConversations(other.ID)
	require.NoError(t, err)
	assert.Len(t, empty, 0)
}
