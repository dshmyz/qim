package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newPendingActionFixture 造一个可用环境：db + MessageService + 两个用户 + 一个群会话。
func newPendingActionFixture(t *testing.T) (*AIPendingActionService, *model.User, *model.User, *model.Conversation) {
	t.Helper()
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIPendingAction{}))

	alice := &model.User{Username: "alice", Nickname: "爱丽丝"}
	bob := &model.User{Username: "bob", Nickname: "鲍勃"}
	require.NoError(t, db.Create(alice).Error)
	require.NoError(t, db.Create(bob).Error)

	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, Name: "项目讨论群", GroupType: "group", CreatorID: alice.ID}).Error)
	for _, u := range []*model.User{alice, bob} {
		require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: u.ID}).Error)
	}

	msgSvc := NewMessageService(db, nil, nil)
	return NewAIPendingActionService(msgSvc), alice, bob, conv
}

func TestCreatePendingSendResolvesTargetName(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "今晚八点同步进度")
	require.NoError(t, err)
	assert.Equal(t, model.AIPendingActionStatusPending, record.Status)
	assert.Equal(t, "项目讨论群", record.TargetName)
	assert.WithinDuration(t, time.Now().Add(pendingActionTTL), record.ExpiresAt, time.Minute)
}

func TestConfirmPendingSendSendsMessage(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "发版说明已更新")
	require.NoError(t, err)

	confirmed, handled, err := svc.ConfirmPendingSend(alice.ID, record.ID)
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Equal(t, model.AIPendingActionStatusConfirmed, confirmed.Status)
	assert.NotZero(t, confirmed.MessageID)

	// 消息真正落库
	var count int64
	svc.db.Model(&model.Message{}).Where("conversation_id = ? AND sender_id = ? AND content = ?", conv.ID, alice.ID, "发版说明已更新").Count(&count)
	assert.EqualValues(t, 1, count, "确认后应真正发出一条消息")

	// 幂等：重复确认返回已处理 + 终态，不再发第二条
	again, handled, err := svc.ConfirmPendingSend(alice.ID, record.ID)
	assert.ErrorIs(t, err, ErrPendingAlreadyHandled)
	assert.True(t, handled)
	assert.Equal(t, model.AIPendingActionStatusConfirmed, again.Status)
	svc.db.Model(&model.Message{}).Where("conversation_id = ?", conv.ID).Count(&count)
	assert.EqualValues(t, 1, count, "重复确认不应重复发送")
}

func TestCancelPendingSend(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "取消我")
	require.NoError(t, err)

	cancelled, handled, err := svc.CancelPendingSend(alice.ID, record.ID)
	require.NoError(t, err)
	assert.False(t, handled)
	assert.Equal(t, model.AIPendingActionStatusCancelled, cancelled.Status)

	var count int64
	svc.db.Model(&model.Message{}).Where("conversation_id = ?", conv.ID).Count(&count)
	assert.EqualValues(t, 0, count, "取消后不应发出消息")

	// 取消后再确认：已处理
	_, handled, err = svc.ConfirmPendingSend(alice.ID, record.ID)
	assert.ErrorIs(t, err, ErrPendingAlreadyHandled)
	assert.True(t, handled)
}

func TestPendingSendForbiddenForOtherUser(t *testing.T) {
	svc, alice, bob, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "只有爱丽丝能确认")
	require.NoError(t, err)

	_, _, err = svc.ConfirmPendingSend(bob.ID, record.ID)
	assert.ErrorIs(t, err, ErrPendingForbidden)

	_, _, err = svc.CancelPendingSend(bob.ID, record.ID)
	assert.ErrorIs(t, err, ErrPendingForbidden)
}

func TestPendingSendExpiry(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "这条会过期")
	require.NoError(t, err)
	// 手动拨快过期时间
	require.NoError(t, svc.db.Model(record).Update("expires_at", time.Now().Add(-time.Minute)).Error)

	got, handled, err := svc.ConfirmPendingSend(alice.ID, record.ID)
	assert.ErrorIs(t, err, ErrPendingExpired)
	assert.True(t, handled)
	assert.Equal(t, model.AIPendingActionStatusExpired, got.Status)

	var count int64
	svc.db.Model(&model.Message{}).Where("conversation_id = ?", conv.ID).Count(&count)
	assert.EqualValues(t, 0, count, "过期确认不应发出消息")
}

func TestCreatePendingSendCleansExpired(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	stale, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "旧请求")
	require.NoError(t, err)
	require.NoError(t, svc.db.Model(stale).Update("expires_at", time.Now().Add(-time.Hour)).Error)

	_, err = svc.CreatePendingSend(alice.ID, 0, conv.ID, "新请求")
	require.NoError(t, err)

	var staleCount, totalCount int64
	// 注意按内容断言而非按 ID：SQLite 无 AUTOINCREMENT，删除 id=1 后新插入会复用 id=1，
	// 按 stale.ID 计数会误命中新记录。
	svc.db.Model(&model.AIPendingAction{}).Where("content = ?", "旧请求").Count(&staleCount)
	assert.EqualValues(t, 0, staleCount, "过期记录应在新请求创建时被清理")
	svc.db.Model(&model.AIPendingAction{}).Count(&totalCount)
	assert.EqualValues(t, 1, totalCount, "只应剩新请求一条")
}

func TestSendMessageToolConfirmMode(t *testing.T) {
	db := setupServiceTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.AIPendingAction{}))

	user := &model.User{Username: "carol", Nickname: "卡罗尔"}
	require.NoError(t, db.Create(user).Error)
	conv := &model.Conversation{Type: "group"}
	require.NoError(t, db.Create(conv).Error)
	require.NoError(t, db.Create(&model.Group{ConversationID: conv.ID, Name: "工具群", GroupType: "group", CreatorID: user.ID}).Error)
	require.NoError(t, db.Create(&model.ConversationMember{ConversationID: conv.ID, UserID: user.ID}).Error)

	msgSvc := NewMessageService(db, nil, nil)
	pendings := NewAIPendingActionService(msgSvc)
	tool := NewSendMessageTool(msgSvc, pendings)

	// 确认制：不直接发送，生成待确认记录
	confirmCtx := &ai.CallerContext{UserID: user.ID, ConversationID: conv.ID, ConfirmTools: []string{"send_message"}}
	result, err := tool.Execute(map[string]interface{}{"content": "确认制消息", "conversation_id": fmt.Sprint(conv.ID)}, confirmCtx)
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "pending_confirmation", m["status"])

	var msgCount, pendCount int64
	db.Model(&model.Message{}).Count(&msgCount)
	assert.Zero(t, msgCount, "确认制下不应直接发消息")
	db.Model(&model.AIPendingAction{}).Count(&pendCount)
	assert.EqualValues(t, 1, pendCount)

	// 用户确认后消息真正发出
	info, ok := m["pending"].(ai.PendingSend)
	require.True(t, ok)
	assert.Equal(t, "工具群", info.TargetName)
	_, handled, err := pendings.ConfirmPendingSend(user.ID, info.ID)
	require.NoError(t, err)
	assert.False(t, handled)
	db.Model(&model.Message{}).Count(&msgCount)
	assert.EqualValues(t, 1, msgCount)

	// 非确认制入口（ConfirmTools 为空）：保持直接发送的旧语义
	result, err = tool.Execute(map[string]interface{}{"content": "直发消息"}, &ai.CallerContext{UserID: user.ID, ConversationID: conv.ID})
	require.NoError(t, err)
	m, ok = result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["sent"])
	db.Model(&model.Message{}).Count(&msgCount)
	assert.EqualValues(t, 2, msgCount)
}

func TestConfirmPendingSendConcurrentOnlySendsOnce(t *testing.T) {
	svc, alice, _, conv := newPendingActionFixture(t)

	record, err := svc.CreatePendingSend(alice.ID, 0, conv.ID, "并发只应发一条")
	require.NoError(t, err)

	const racers = 8
	var wg sync.WaitGroup
	results := make(chan int, racers) // 每个竞速者记录它是否执行了真实发送
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, _, err := svc.ConfirmPendingSend(alice.ID, record.ID)
			// 只有抢到原子占有的请求会走完 SendMessage 并拿到 confirmed 终态
			if err == nil && got != nil && got.Status == model.AIPendingActionStatusConfirmed {
				results <- 1
			} else {
				results <- 0
			}
		}()
	}
	wg.Wait()
	close(results)

	wins := 0
	for v := range results {
		wins += v
	}
	assert.EqualValues(t, 1, wins, "并发确认应有且仅有一个请求执行发送")

	var count int64
	svc.db.Model(&model.Message{}).Where("conversation_id = ?", conv.ID).Count(&count)
	assert.EqualValues(t, 1, count, "并发确认最终只应发出一条消息")

	var final model.AIPendingAction
	require.NoError(t, svc.db.First(&final, record.ID).Error)
	assert.Equal(t, model.AIPendingActionStatusConfirmed, final.Status)
}
