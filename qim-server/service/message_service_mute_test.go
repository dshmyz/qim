package service

import (
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupMuteTestDB(t *testing.T) (*gorm.DB, *model.User, *model.User, uint) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Conversation{}, &model.ConversationMember{},
		&model.Group{}, &model.Message{}, &model.SensitiveWord{},
	))
	database.DB = db

	owner := model.User{Username: "owner", Nickname: "群主"}
	muted := model.User{Username: "muted", Nickname: "被禁言"}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&muted).Error)
	conv := model.Conversation{Type: "group"}
	require.NoError(t, db.Create(&conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, GroupType: "group", Name: "测试群", CreatorID: owner.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: owner.ID, Role: "owner", JoinedAt: time.Now()}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: muted.ID, Role: "member", JoinedAt: time.Now()}).Error)
	return db, &owner, &muted, conv.ID
}

// TestSendMessage_MutedMemberRejected 被禁言成员发言被拒（ErrMuted）。
func TestSendMessage_MutedMemberRejected(t *testing.T) {
	db, _, muted, convID := setupMuteTestDB(t)
	until := time.Now().Add(1 * time.Hour)
	require.NoError(t, db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, muted.ID).
		Update("muted_until", until).Error)

	svc := NewMessageService(db, nil, nil)
	_, err := svc.SendMessage(convID, muted.ID, "text", "hello", nil)
	assert.ErrorIs(t, err, ErrMuted)
}

// TestSendMessage_AfterUnmute 解除禁言后可正常发言。
func TestSendMessage_AfterUnmute(t *testing.T) {
	db, _, muted, convID := setupMuteTestDB(t)
	require.NoError(t, db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, muted.ID).
		Update("muted_until", nil).Error)

	svc := NewMessageService(db, nil, nil)
	msg, err := svc.SendMessage(convID, muted.ID, "text", "hello", nil)
	require.NoError(t, err)
	assert.NotNil(t, msg)
}

// TestSendMessage_OwnerNotMutedEvenIfMutedUntilSet 群主即使被设了 MutedUntil 也能发言（豁免）。
func TestSendMessage_OwnerNotMutedEvenIfMutedUntilSet(t *testing.T) {
	db, owner, _, convID := setupMuteTestDB(t)
	until := time.Now().Add(1 * time.Hour)
	require.NoError(t, db.Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, owner.ID).
		Update("muted_until", until).Error)

	svc := NewMessageService(db, nil, nil)
	msg, err := svc.SendMessage(convID, owner.ID, "text", "hello", nil)
	require.NoError(t, err)
	assert.NotNil(t, msg)
}
