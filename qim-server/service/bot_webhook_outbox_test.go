package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dshmyz/qim/qim-server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockWebhook 起一个可控的 mock webhook 端点，statusCtrl 控制返回状态码。
// 返回 server + 已收到的 delivery 数计数。
func newMockWebhook(t *testing.T, statusCtrl *int32) (*httptest.Server, *int32) {
	t.Helper()
	var received int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&received, 1)
		code := int(atomic.LoadInt32(statusCtrl))
		w.WriteHeader(code)
		if code >= 200 && code < 300 {
			w.Write([]byte("ok"))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &received
}

func TestEnqueueWebhookDelivery_LandsPending(t *testing.T) {
	db := setupServiceTestDB(t)
	id, err := EnqueueWebhookDelivery(db, 1, "bot.message", `{"content":"hi"}`, "http://x", "secret")
	require.NoError(t, err)
	assert.NotZero(t, id)

	var d model.BotWebhookDelivery
	require.NoError(t, db.First(&d, id).Error)
	assert.Equal(t, "pending", d.Status)
	assert.Equal(t, "bot.message", d.Event)
	assert.Equal(t, `{"content":"hi"}`, d.Payload)
	assert.Zero(t, d.Attempts)
}

func TestDeliverOnce_Success(t *testing.T) {
	db := setupServiceTestDB(t)
	var status int32 = 200
	srv, received := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: 1, Content: "hi"})
	id, err := EnqueueWebhookDelivery(db, 1, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	err = DeliverOnce(db, id)
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(received))

	var d model.BotWebhookDelivery
	require.NoError(t, db.First(&d, id).Error)
	assert.Equal(t, "done", d.Status)
	assert.Equal(t, 1, d.Attempts)
	assert.NotNil(t, d.DeliveredAt)
	assert.Nil(t, d.NextRetryAt)
}

func TestDeliverOnce_FailureRetriesThenDead(t *testing.T) {
	db := setupServiceTestDB(t)
	var status int32 = 500 // 持续失败
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: 1, Content: "hi"})
	id, err := EnqueueWebhookDelivery(db, 1, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	// 连续投递 MaxAttempts 次，应全部失败；最后一次进死信
	for i := 1; i <= MaxAttempts; i++ {
		err = DeliverOnce(db, id)
		require.NoError(t, err, "attempt %d", i)
		var d model.BotWebhookDelivery
		require.NoError(t, db.First(&d, id).Error)
		assert.Equal(t, i, d.Attempts, "attempt %d attempts", i)
		if i < MaxAttempts {
			assert.Equal(t, "pending", d.Status, "attempt %d should remain pending", i)
			require.NotNil(t, d.NextRetryAt, "attempt %d should have next_retry_at", i)
			assert.Contains(t, d.LastError, "HTTP 500")
		} else {
			assert.Equal(t, "dead", d.Status, "attempt %d should be dead", i)
			assert.Nil(t, d.NextRetryAt)
			assert.Contains(t, d.LastError, "HTTP 500")
		}
	}
}

func TestDeliverOnce_FailureThenSuccess(t *testing.T) {
	db := setupServiceTestDB(t)
	var status int32 = 503
	srv, received := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: 1, Content: "hi"})
	id, err := EnqueueWebhookDelivery(db, 1, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, err)

	// 第一次失败
	require.NoError(t, DeliverOnce(db, id))
	var d model.BotWebhookDelivery
	require.NoError(t, db.First(&d, id).Error)
	assert.Equal(t, "pending", d.Status)
	assert.Equal(t, 1, d.Attempts)

	// 模拟 agent 恢复，第二次成功
	atomic.StoreInt32(&status, 200)
	require.NoError(t, DeliverOnce(db, id))
	require.NoError(t, db.First(&d, id).Error)
	assert.Equal(t, "done", d.Status)
	assert.Equal(t, 2, d.Attempts)
	assert.Equal(t, int32(2), atomic.LoadInt32(received))
}

func TestProcessPendingDeliveries_OnlyDueOnes(t *testing.T) {
	db := setupServiceTestDB(t)
	var status int32 = 200
	srv, _ := newMockWebhook(t, &status)

	payload, _ := json.Marshal(BotWebhookPayload{Event: "bot.message", BotID: 1, Content: "hi"})

	// due：next_retry_at 为 nil（刚入队）
	id1, _ := EnqueueWebhookDelivery(db, 1, "bot.message", string(payload), srv.URL, "")

	// not due：next_retry_at 在未来
	future := time.Now().Add(10 * time.Minute)
	id2, _ := EnqueueWebhookDelivery(db, 1, "bot.message", string(payload), srv.URL, "")
	require.NoError(t, db.Model(&model.BotWebhookDelivery{}).Where("id = ?", id2).
		Update("next_retry_at", future).Error)

	ProcessPendingDeliveries(db)

	var d1, d2 model.BotWebhookDelivery
	require.NoError(t, db.First(&d1, id1).Error)
	require.NoError(t, db.First(&d2, id2).Error)
	assert.Equal(t, "done", d1.Status, "due 的应被投递")
	assert.Equal(t, "pending", d2.Status, "未到期的应跳过")
	assert.Zero(t, d2.Attempts)
}

// 编译期保证 fmt 被引用（调试用）。
var _ = fmt.Sprintf
