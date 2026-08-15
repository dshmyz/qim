package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newSyncTestClient() *Client {
	return &Client{hub: &Hub{}, send: make(chan []byte, 8)}
}

// TestSendSyncHints_KeepsFlagUntilAck 发送成功不清除 needsSync——由客户端 ack 清除。
// 这是可靠性核心：客户端拉取失败时，后续轮询会重发 hint。
func TestSendSyncHints_KeepsFlagUntilAck(t *testing.T) {
	hub := &Hub{}
	client := newSyncTestClient()
	client.needsSync.Store(true)
	hub.clients.Store(client, true)

	hub.sendSyncHints()
	require.True(t, client.needsSync.Load(), "发送成功后不应清除标记（应等 ack）")

	select {
	case <-client.send:
	default:
		t.Fatal("应收到 sync_hint")
	}

	// 模拟客户端完成拉取后 ack
	client.needsSync.Store(false)
	hub.sendSyncHints()
	require.False(t, client.needsSync.Load())
	select {
	case <-client.send:
		t.Fatal("ack 后不应再发送 hint")
	default:
	}
}

// TestSendSyncHints_RetriesWithinWindow 首次尝试发送后 3~30 秒内未 ack，应继续重发。
// 窗口起点是 syncHintStartedAt（首次尝试），而非最近一次成功发送。
func TestSendSyncHints_RetriesWithinWindow(t *testing.T) {
	hub := &Hub{}
	client := newSyncTestClient()
	client.needsSync.Store(true)
	client.syncHintStartedAt.Store(time.Now().Add(-5 * time.Second).UnixNano())
	hub.clients.Store(client, true)

	hub.sendSyncHints()
	require.True(t, client.needsSync.Load(), "重试窗口内应保持标记")
	select {
	case <-client.send:
	default:
		t.Fatal("重试窗口内应再次发送 hint")
	}
}

// TestSendSyncHints_TimeoutStopsRetry 超过 syncHintTimeout 仍无 ack，放弃重发。
// 老客户端不支持 acknowledge_sync，若不兜底会每 3 秒无限推送。
func TestSendSyncHints_TimeoutStopsRetry(t *testing.T) {
	hub := &Hub{}
	client := newSyncTestClient()
	client.needsSync.Store(true)
	client.syncHintStartedAt.Store(time.Now().Add(-(syncHintTimeout + time.Second)).UnixNano())
	hub.clients.Store(client, true)

	hub.sendSyncHints()
	require.False(t, client.needsSync.Load(), "超过重试窗口应放弃")
	select {
	case <-client.send:
		t.Fatal("超时后不应再发送 hint")
	default:
	}
}

// TestSendSyncHints_TimeoutNeverDelivered 极端场景：channel 一直满、hint 从未发送成功。
// 旧实现窗口起点只在发送成功时记录（lastSyncHintAt=0），从未投递即永不超时，每 3s 无限重试；
// 现在 SendToUser 首次溢出时就通过 CompareAndSwap 预置 syncHintStartedAt，超时对从未投递同样生效。
func TestSendSyncHints_TimeoutNeverDelivered(t *testing.T) {
	hub := &Hub{}
	client := newSyncTestClient()
	client.needsSync.Store(true)
	// 模拟 channel 从溢出起就一直满（从未投递成功）
	for i := 0; i < cap(client.send); i++ {
		client.send <- []byte("full")
	}
	// 窗口起点已超时（对应 SendToUser 已预置起点、但 hint 一直没发出去的场景）
	client.syncHintStartedAt.Store(time.Now().Add(-(syncHintTimeout + time.Second)).UnixNano())
	hub.clients.Store(client, true)

	hub.sendSyncHints()
	require.False(t, client.needsSync.Load(), "窗口超时后应放弃，即使从未投递成功")
	require.Zero(t, client.syncHintStartedAt.Load(), "放弃后窗口起点应归零，供下次溢出重新计时")
}

// TestSendSyncHints_SkipNonFlagged 未溢出的客户端不受影响。
func TestSendSyncHints_SkipNonFlagged(t *testing.T) {
	hub := &Hub{}
	client := newSyncTestClient()
	hub.clients.Store(client, true)

	hub.sendSyncHints()
	select {
	case <-client.send:
		t.Fatal("未溢出的客户端不应收到 hint")
	default:
	}
}
