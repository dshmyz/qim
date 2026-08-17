package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// resetNoticeDedup 清空全局去重状态（通知去重 map 是包级单例，测试间需隔离）。
func resetNoticeDedup() {
	noticeMu.Lock()
	defer noticeMu.Unlock()
	noticeLastAt[webhookNoticeFirstFailure] = map[uint]time.Time{}
	noticeLastAt[webhookNoticeDead] = map[uint]time.Time{}
}

// setupNotifyScenario 建一个 bot + bot 会话，供投递失败通知测试使用。
func setupNotifyScenario(t *testing.T) (*gorm.DB, uint, uint) {
	t.Helper()
	db := setupServiceTestDB(t)
	resetNoticeDedup()

	bot := &model.Bot{Name: "测试机器人", Type: model.BotTypeAssistant, IsActive: true}
	require.NoError(t, db.Create(bot).Error)
	conv := &model.Conversation{Type: "bot"}
	require.NoError(t, db.Create(conv).Error)
	return db, bot.ID, conv.ID
}

// systemNoticeCount 统计会话内 system 消息条数（按 id 升序返回内容）。
func systemNotices(t *testing.T, db *gorm.DB, convID uint) []model.Message {
	t.Helper()
	var notices []model.Message
	require.NoError(t, db.Where("conversation_id = ? AND type = 'system'", convID).
		Order("id ASC").Find(&notices).Error)
	return notices
}

// TestDeliverOnce_FailureNotifiesThenDeadNotifies 首次失败提示一次「自动重试」，
// 重试不再重复，死信时提示一次「投递失败」，且会话 last_message 被更新。
func TestDeliverOnce_FailureNotifiesThenDeadNotifies(t *testing.T) {
	db, botID, convID := setupNotifyScenario(t)
	var status int32 = 500
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: botID, ThreadID: convID, Content: "hi"})
	id, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	// 首次失败：产生一条「自动重试」系统提示
	require.NoError(t, DeliverOnce(db, id))
	notices := systemNotices(t, db, convID)
	require.Len(t, notices, 1)
	assert.Contains(t, notices[0].Content, "测试机器人")
	assert.Contains(t, notices[0].Content, "自动重试")

	// 重试再次失败：不再重复提示（首次失败仅提示一次）
	require.NoError(t, DeliverOnce(db, id))
	require.Len(t, systemNotices(t, db, convID), 1)

	// 重试耗尽进死信：提示一次「投递失败」
	for i := 3; i <= MaxAttempts; i++ {
		require.NoError(t, DeliverOnce(db, id))
	}
	notices = systemNotices(t, db, convID)
	require.Len(t, notices, 2)
	assert.Contains(t, notices[1].Content, "多次投递失败")

	var d model.BotWebhookDelivery
	require.NoError(t, db.First(&d, id).Error)
	assert.Equal(t, "dead", d.Status)

	// 会话 last_message 指向最后一条系统提示
	var conv model.Conversation
	require.NoError(t, db.First(&conv, convID).Error)
	require.NotNil(t, conv.LastMessageID)
	assert.Equal(t, notices[1].ID, *conv.LastMessageID)
}

// TestDeliverOnce_FailureNoticeDedupedAcrossDeliveries 用户停摆期间连发多条消息，
// 各 delivery 首次失败时只提示一次（会话级去重防刷屏）。
func TestDeliverOnce_FailureNoticeDedupedAcrossDeliveries(t *testing.T) {
	db, botID, convID := setupNotifyScenario(t)
	var status int32 = 500
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: botID, ThreadID: convID, Content: "hi"})
	id1, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)
	id2, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	require.NoError(t, DeliverOnce(db, id1))
	require.NoError(t, DeliverOnce(db, id2))

	require.Len(t, systemNotices(t, db, convID), 1, "同一会话连发多条消息只提示一次")
}

// TestDeliverOnce_SuccessCreatesNoNotice 投递成功不产生任何提示。
func TestDeliverOnce_SuccessCreatesNoNotice(t *testing.T) {
	db, botID, convID := setupNotifyScenario(t)
	var status int32 = 200
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: botID, ThreadID: convID, Content: "hi"})
	id, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)
	require.NoError(t, DeliverOnce(db, id))

	assert.Empty(t, systemNotices(t, db, convID), "投递成功不应产生提示")
}

// TestDeliverOnce_DeadNoticeDedupedAcrossDeliveries 多条 delivery 相继死信时只提示一次。
func TestDeliverOnce_DeadNoticeDedupedAcrossDeliveries(t *testing.T) {
	db, botID, convID := setupNotifyScenario(t)
	var status int32 = 500
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: botID, ThreadID: convID, Content: "hi"})
	id1, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)
	id2, err := EnqueueWebhookDelivery(db, botID, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	// 两条都走到死信
	for i := 1; i <= MaxAttempts; i++ {
		require.NoError(t, DeliverOnce(db, id1))
		require.NoError(t, DeliverOnce(db, id2))
	}

	notices := systemNotices(t, db, convID)
	require.Len(t, notices, 2, "首次失败提示 1 条 + 死信提示 1 条")
	assert.Contains(t, notices[0].Content, "自动重试")
	assert.Contains(t, notices[1].Content, "多次投递失败")
}
