package handler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type avatarTriggerDeciderStub struct {
	shouldReply bool
	userID      uint
	convID      uint
	message     string
	senderName  string
}

func (s *avatarTriggerDeciderStub) ShouldReply(userID uint, conversationID uint, message string, senderName string) (bool, string, error) {
	s.userID = userID
	s.convID = conversationID
	s.message = message
	s.senderName = senderName
	return s.shouldReply, "test decision", nil
}

type avatarConfigQueryCounter struct {
	logger.Interface
	mu      sync.Mutex
	count   int
	inQuery bool
}

func newAvatarConfigQueryCounter() *avatarConfigQueryCounter {
	return &avatarConfigQueryCounter{Interface: logger.Default.LogMode(logger.Info)}
}

func (l *avatarConfigQueryCounter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	normalized := strings.ToLower(sql)
	if strings.Contains(normalized, "avatar_configs") && strings.Contains(normalized, "select") {
		l.mu.Lock()
		l.count++
		if strings.Contains(normalized, " in ") {
			l.inQuery = true
		}
		l.mu.Unlock()
	}
	l.Interface.Trace(ctx, begin, func() (string, int64) { return sql, rows }, err)
}

func (l *avatarConfigQueryCounter) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

func (l *avatarConfigQueryCounter) SawINQuery() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.inQuery
}

func (l *avatarConfigQueryCounter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.count = 0
	l.inQuery = false
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
		2,
		"今天进度如何？",
		true,
		nil,
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
		senderID,
		"请帮我判断这个方案",
		true,
		nil,
	)

	assert.True(t, triggered)
	assert.Equal(t, avatarUserID, decider.userID)
	assert.Equal(t, uint(9), decider.convID)
	assert.Equal(t, "请帮我判断这个方案", decider.message)
	assert.Equal(t, "Sender", decider.senderName)
}

func TestCheckAvatarTriggersBatchLoadsMissingAvatarConfigs(t *testing.T) {
	queryCounter := newAvatarConfigQueryCounter()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: queryCounter})
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

	enabledUserIDs := make([]uint, 0, 2)
	for i := 0; i < 7; i++ {
		user := model.User{Username: "member" + string(rune('a'+i)), PasswordHash: "hash", Nickname: "成员"}
		require.NoError(t, db.Create(&user).Error)
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)

		if i == 1 || i == 4 {
			enabledUserIDs = append(enabledUserIDs, user.ID)
			require.NoError(t, db.Create(&model.AvatarConfig{
				UserID:           user.ID,
				Enabled:          true,
				TriggerRulesJSON: `{"mode":"mention"}`,
			}).Error)
		}
	}

	engine := &SmartReplyEngine{}
	queryCounter.Reset()
	engine.checkAvatarTriggers(sender.ID, &conv, "普通群聊消息", nil)

	var sessions []model.AvatarSession
	require.NoError(t, db.Where("conversation_id = ?", conv.ID).Find(&sessions).Error)
	require.Len(t, sessions, 2)
	assert.ElementsMatch(t, enabledUserIDs, []uint{sessions[0].UserID, sessions[1].UserID})
	assert.True(t, queryCounter.SawINQuery(), "缺失 session 的 AvatarConfig 应使用 IN 批量查询")
	assert.LessOrEqual(t, queryCounter.Count(), 3, "不应按每个群成员逐个查询 avatar_configs")
}
