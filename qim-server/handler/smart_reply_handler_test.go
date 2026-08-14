package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type avatarTriggerDeciderStub struct {
	shouldReply bool
	config      model.AvatarConfig
	convID      uint
	message     string
	senderName  string
	isGroupChat bool
	mentionIDs  []uint
}

func (s *avatarTriggerDeciderStub) DecideReply(config model.AvatarConfig, conversationID uint, message string, senderName string, isGroupChat bool, mentionUserIDs []uint) (bool, string, error) {
	s.config = config
	s.convID = conversationID
	s.message = message
	s.senderName = senderName
	s.isGroupChat = isGroupChat
	s.mentionIDs = mentionUserIDs
	return s.shouldReply, "test decision", nil
}

func TestAIMentionDetectsStructuredMentionTokenByAssistantName(t *testing.T) {
	engine := &SmartReplyEngine{}
	content := mention.Encode(42, "青雀一号") + " 帮我总结一下"

	assert.True(t, engine.isAIMention(content, "青雀一号"))
	assert.Equal(t, "帮我总结一下", extractAIQuestion(content, "青雀一号"))
}

func TestShouldTriggerAvatar_SmartDoesNotFallbackToAllWithoutDecider(t *testing.T) {
	db := setupHandlerTestDB(t)
	if err := db.Exec(`CREATE TABLE avatar_configs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		enabled BOOLEAN NOT NULL,
		trigger_rules_json TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create avatar_configs table: %v", err)
	}
	database.DB = db

	const avatarUserID = uint(1)
	if err := db.Exec(
		`INSERT INTO avatar_configs (user_id, enabled, trigger_rules_json) VALUES (?, ?, ?)`,
		avatarUserID,
		true,
		`{"mode":"smart"}`,
	).Error; err != nil {
		t.Fatalf("seed smart avatar config: %v", err)
	}

	engine := &SmartReplyEngine{}
	triggered := engine.shouldTriggerAvatar(
		&model.AvatarSession{ConversationID: 1, UserID: avatarUserID, AvatarEnabled: true},
		"今天进度如何？",
		true,
		nil,
		"",
	)

	assert.False(t, triggered, "smart mode must not behave as all when its intent decider is unavailable")
}

func TestShouldTriggerAvatar_SmartUsesIntentDecider(t *testing.T) {
	db := setupHandlerTestDB(t)
	if err := db.Exec(`CREATE TABLE avatar_configs (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL,
		enabled BOOLEAN NOT NULL,
		trigger_rules_json TEXT,
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create avatar_configs table: %v", err)
	}
	database.DB = db

	const avatarUserID = uint(1)
	const senderID = uint(2)
	db.Create(&model.User{ID: senderID, Username: "sender", PasswordHash: "hash", Nickname: "Sender"})
	db.Exec(`INSERT INTO avatar_configs (user_id, enabled, trigger_rules_json) VALUES (?, ?, ?)`, avatarUserID, true, `{"mode":"smart"}`)

	decider := &avatarTriggerDeciderStub{shouldReply: true}
	engine := &SmartReplyEngine{avatarTriggerSvc: decider}
	triggered := engine.shouldTriggerAvatar(
		&model.AvatarSession{ConversationID: 9, UserID: avatarUserID, AvatarEnabled: true},
		"请帮我判断这个方案",
		true,
		nil,
		"Sender",
	)

	assert.True(t, triggered)
	assert.Equal(t, avatarUserID, decider.config.UserID)
	assert.Equal(t, uint(9), decider.convID)
	assert.Equal(t, "请帮我判断这个方案", decider.message)
	assert.Equal(t, "Sender", decider.senderName)
}

func TestCheckAvatarTriggersDoesNotAutoSubscribe(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.AvatarConfig{},
		&model.AvatarSession{},
	))
	database.DB = db

	sender := model.User{Username: "sender", PasswordHash: "hash", Nickname: "发送者"}
	require.NoError(t, db.Create(&sender).Error)

	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: sender.ID}).Error)

	// optedIn: 在该会话显式开启过分身（有 session + 全局 config）
	optedIn := model.User{Username: "optedin", PasswordHash: "hash", Nickname: "已开启"}
	require.NoError(t, db.Create(&optedIn).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: optedIn.ID}).Error)
	require.NoError(t, db.Create(&model.AvatarConfig{
		UserID:           optedIn.ID,
		Enabled:          true,
		TriggerRulesJSON: `{"mode":"mention"}`,
	}).Error)
	require.NoError(t, db.Create(&model.AvatarSession{
		ConversationID: conv.ID,
		UserID:         optedIn.ID,
		AvatarEnabled:  true,
	}).Error)

	// notOptedIn: 全局分身已启用，但未在该会话开启（无 session）——不应被自动订阅
	notOptedIn := model.User{Username: "notoptedin", PasswordHash: "hash", Nickname: "未开启"}
	require.NoError(t, db.Create(&notOptedIn).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: notOptedIn.ID}).Error)
	require.NoError(t, db.Create(&model.AvatarConfig{
		UserID:           notOptedIn.ID,
		Enabled:          true,
		TriggerRulesJSON: `{"mode":"mention"}`,
	}).Error)

	engine := &SmartReplyEngine{}
	engine.checkAvatarTriggers(sender.ID, &conv, "普通群聊消息", "text", nil)

	var sessions []model.AvatarSession
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).Find(&sessions).Error)
	require.Len(t, sessions, 1, "全局已启用但无会话级 session 的成员不应被自动创建 session")
	assert.Equal(t, optedIn.ID, sessions[0].UserID)
}

func TestCheckAvatarTriggersDefaultOnActivatesWithoutSession(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{},
		&model.Conversation{},
		&model.ConversationMember{},
		&model.AvatarConfig{},
		&model.AvatarSession{},
	))
	database.DB = db

	sender := model.User{Username: "sender", PasswordHash: "hash", Nickname: "发送者"}
	require.NoError(t, db.Create(&sender).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: sender.ID}).Error)

	// 默认开成员：config 已启用 + ActivateByDefault=true，但无 session 行
	defUser := model.User{Username: "defon", PasswordHash: "hash", Nickname: "默认开"}
	require.NoError(t, db.Create(&defUser).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: defUser.ID}).Error)
	require.NoError(t, db.Create(&model.AvatarConfig{
		UserID:            defUser.ID,
		Enabled:           true,
		ActivateByDefault: true,
		TriggerRulesJSON:  `{"mode":"mention"}`,
	}).Error)

	// decider 返回 false 以避免走到 worker pool（engine 未注入 pool），
	// 但仍会被调用——被调用即证明该成员被选为候选
	decider := &avatarTriggerDeciderStub{shouldReply: false}
	engine := &SmartReplyEngine{avatarTriggerSvc: decider}
	engine.checkAvatarTriggers(sender.ID, &conv, "在吗", "text", nil)

	assert.Equal(t, defUser.ID, decider.config.UserID,
		"ActivateByDefault=true 且无 session 的成员应被选为候选并进入触发判断")

	// 默认开不应写库创建 session 行
	var sessions []model.AvatarSession
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).Find(&sessions).Error)
	assert.Empty(t, sessions, "默认开成员不应产生 session 行")
}

func TestLooksMemorable(t *testing.T) {
	// 短消息/问候不记
	assert.False(t, looksMemorable("好的"))
	assert.False(t, looksMemorable("在吗"))
	// 纯 @AI 指令不记
	assert.False(t, looksMemorable("@AI 帮我总结一下今天的会议纪要"))
	// 主人本人的实质表达应进入后续门控
	assert.True(t, looksMemorable("我下周三要去上海出差，项目A的评审改到周五上午"))
	assert.True(t, looksMemorable("I prefer replies in English, thanks"))
}
