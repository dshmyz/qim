package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/mention"
	"github.com/stretchr/testify/assert"
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
