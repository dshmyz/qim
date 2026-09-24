package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 质量核验被拒 → 草稿落库 → 主人批准 → 回复发出（消息落库 + 草稿 approved + 通知已处理）
func TestAvatarDraftApproval_ApproveSends(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "owner", Nickname: "Owner", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "peer", Nickname: "Peer", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Conversation{ID: 10, Type: "single"}).Error)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.AvatarConfig{UserID: 1, Name: "小助手"}).Error)

	draftSvc := NewAvatarReplyDraftService(db)
	draft := draftSvc.Create(AvatarDraftInput{
		OwnerID: 1, ConversationID: 10, TriggerMessageID: 100, TriggerUserID: 2,
		Content: "好的，明天见", Reason: "包含未证实事实",
		Sources: []KnowledgeSource{{Title: "s", Score: 0.8}},
	})
	require.NotNil(t, draft)
	require.Len(t, draftSvc.ListPending(1, 10), 1)
	require.NotNil(t, draft.NotificationID)

	pool := &AvatarWorkerPool{db: db, draftSvc: draftSvc}
	require.NoError(t, pool.ApproveDraft(1, draft.ID))

	var msgCount int64
	db.Model(&model.Message{}).Where("conversation_id = 10 AND origin = 'avatar'").Count(&msgCount)
	assert.Equal(t, int64(1), msgCount, "批准后回复应作为分身消息发出")

	approved, err := draftSvc.GetForOwner(1, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, "approved", approved.Status)
	var notif model.Notification
	require.NoError(t, db.First(&notif, *draft.NotificationID).Error)
	assert.True(t, notif.Handled, "批准后通知红点应消除")

	require.Error(t, pool.ApproveDraft(1, draft.ID), "已处理的草稿不可重复批准")
}

// 草稿创建后主人已亲自发言：让位门复验拦下批准，消息不外发
func TestAvatarDraftApproval_StandDownBlocksApprove(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "owner", Nickname: "Owner", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "peer", Nickname: "Peer", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Conversation{ID: 10, Type: "single"}).Error)
	require.NoError(t, db.Migrator().CreateTable(&model.AvatarConfig{}))
	require.NoError(t, db.Create(&model.AvatarConfig{UserID: 1, Name: "小助手"}).Error)

	draftSvc := NewAvatarReplyDraftService(db)
	draft := draftSvc.Create(AvatarDraftInput{
		OwnerID: 1, ConversationID: 10, TriggerUserID: 2,
		Content: "好的", Reason: "事实依据不足",
	})
	require.NotNil(t, draft)

	// 主人在草稿落库后亲自回复了同一会话
	require.NoError(t, db.Create(&model.Message{
		ConversationID: 10, SenderID: 1, Type: "text", Content: "我自己来回",
	}).Error)

	pool := &AvatarWorkerPool{db: db, draftSvc: draftSvc}
	err := pool.ApproveDraft(1, draft.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "让位")

	var msgCount int64
	db.Model(&model.Message{}).Where("origin = 'avatar'").Count(&msgCount)
	assert.Equal(t, int64(0), msgCount, "让位时草稿消息不得外发")
}

// 丢弃 + 跨用户归属校验
func TestAvatarDraftApproval_DiscardAndOwnerGuard(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.Create(&model.User{ID: 1, Username: "owner", Nickname: "Owner", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.User{ID: 2, Username: "other", Nickname: "Other", PasswordHash: "h"}).Error)
	require.NoError(t, db.Create(&model.Conversation{ID: 10, Type: "single"}).Error)

	draftSvc := NewAvatarReplyDraftService(db)
	draft := draftSvc.Create(AvatarDraftInput{
		OwnerID: 1, ConversationID: 10, Content: "草稿",
	})
	require.NotNil(t, draft)

	pool := &AvatarWorkerPool{db: db, draftSvc: draftSvc}
	require.NoError(t, pool.DiscardDraft(1, draft.ID))
	discarded, err := draftSvc.GetForOwner(1, draft.ID)
	require.NoError(t, err)
	assert.Equal(t, "discarded", discarded.Status)
	require.Error(t, pool.DiscardDraft(1, draft.ID), "已处理的草稿不可重复丢弃")

	_, err = draftSvc.GetForOwner(2, draft.ID)
	require.Error(t, err, "跨用户读取草稿必须被拒")
}

// QualityRejectedError 支持被 wrap 后 errors.As 识别（worker 不依赖字符串匹配）
func TestQualityRejectedError_ErrorsAs(t *testing.T) {
	err := fmt.Errorf("wrap: %w", &QualityRejectedError{Draft: "草稿", Reason: "原因"})
	var rejected *QualityRejectedError
	require.True(t, errors.As(err, &rejected))
	assert.Equal(t, "草稿", rejected.Draft)
	assert.Equal(t, "原因", rejected.Reason)
	assert.Contains(t, rejected.Error(), "reply quality rejected")
}
