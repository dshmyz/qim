package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// nodeRelayRecorder 记录节点中继 HTTP 请求（路径 + body），用于断言"只转发一次、不回传"。
type nodeRelayRecorder struct {
	mu          sync.Mutex
	broadcasts  []map[string]interface{}
	sendToUsers []map[string]interface{}
}

func (r *nodeRelayRecorder) count() (broadcast, sendToUser int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.broadcasts), len(r.sendToUsers)
}

func (r *nodeRelayRecorder) origin(path string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if path == "/api/v1/node/broadcast" {
		if len(r.broadcasts) == 0 {
			return ""
		}
		o, _ := r.broadcasts[0]["origin"].(string)
		return o
	}
	if len(r.sendToUsers) == 0 {
		return ""
	}
	o, _ := r.sendToUsers[0]["origin"].(string)
	return o
}

// newNodeTestHub 构造带一个"对端节点"（httptest.Server）的 Hub，并注册一个本地测试客户端（userID=42）。
func newNodeTestHub(t *testing.T) (*Hub, *Client, *nodeRelayRecorder) {
	t.Helper()
	var rec nodeRelayRecorder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		rec.mu.Lock()
		switch r.URL.Path {
		case "/api/v1/node/broadcast":
			rec.broadcasts = append(rec.broadcasts, body)
		case "/api/v1/node/send-to-user":
			rec.sendToUsers = append(rec.sendToUsers, body)
		default:
			t.Errorf("意外路径: %s", r.URL.Path)
		}
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	hub := &Hub{
		nodeID:      "node-A",
		nodes:       []string{strings.TrimPrefix(srv.URL, "http://")},
		nodeScheme:  "http",
		sendSem:     make(chan struct{}, 50),
		clients:     sync.Map{},
		userClients: sync.Map{},
	}
	local := newSyncTestClient()
	local.hub = hub
	local.userID = 42
	hub.clients.Store(local, true)
	hub.userClients.Store(uint(42), []*Client{local})
	return hub, local, &rec
}

// recvOrFail 等待客户端 send 通道收到消息（带超时）。
func recvOrFail(t *testing.T, c *Client, what string) {
	t.Helper()
	select {
	case <-c.send:
	case <-time.After(2 * time.Second):
		t.Fatalf("应收到消息: %s", what)
	}
}

// noRecv 断言客户端 send 通道在窗口期内没有消息。
func noRecv(t *testing.T, c *Client) {
	t.Helper()
	select {
	case <-c.send:
		t.Fatal("不应收到消息（回环应被丢弃）")
	case <-time.After(150 * time.Millisecond):
	}
}

// TestNodeBroadcast_NoInfiniteLoop 广播路径防环：
//  1. 本地广播 asyncBroadcast 向对端只转发一次、且带 origin；
//  2. 来自他节点的广播中继 DeliverBroadcastFromNode 仅本地投递，绝不回传对端；
//  3. origin==本节点 的回环直接丢弃（本节点地址被误配进 nodes 时的自我保护）。
func TestNodeBroadcast_NoInfiniteLoop(t *testing.T) {
	hub, local, rec := newNodeTestHub(t)
	payload := []byte(`{"type":"new_message","data":{}}`)

	// 1. 本地广播：本地投递 + 对端转发恰好一次（带 origin）
	hub.asyncBroadcast(payload)
	recvOrFail(t, local, "本地广播应投递到本节点客户端")
	require.Eventually(t, func() bool {
		b, _ := rec.count()
		return b == 1
	}, 2*time.Second, 20*time.Millisecond, "本地广播应对每个对端节点转发一次")
	require.Equal(t, "node-A", rec.origin("/api/v1/node/broadcast"), "转发载荷应带发送方 origin")

	// 2. 来自他节点的广播中继：本节点客户端收到，但不再回传对端（环的根因）
	hub.DeliverBroadcastFromNode("node-B", payload)
	recvOrFail(t, local, "节点中继应投递到本节点客户端")
	time.Sleep(150 * time.Millisecond) // 留出中继窗口，若错误回传应能观察到
	b, _ := rec.count()
	require.Equal(t, 1, b, "来自他节点的广播不应再次转发对端（否则形成无限环）")

	// 3. 回环保护：origin==本节点 直接丢弃，本地不收、也不转发
	hub.DeliverBroadcastFromNode("node-A", payload)
	noRecv(t, local)
	b, _ = rec.count()
	require.Equal(t, 1, b, "回环广播不应产生任何新的转发")
}

// TestNodeSendToUser_NoInfiniteLoop 用户定向消息路径防环：
//  1. SendToUser 本地投递 + 对端转发恰好一次（带 origin）；
//  2. 来自他节点的定向中继仅本地投递，不回传发送方（此前双向无限循环）；
//  3. origin==本节点 的回环直接丢弃。
func TestNodeSendToUser_NoInfiniteLoop(t *testing.T) {
	hub, local, rec := newNodeTestHub(t)
	payload := []byte(`{"type":"new_message","data":{}}`)

	// 1. 本地定向发送：本地投递 + 对端转发一次（带 origin）
	hub.SendToUser(42, payload)
	recvOrFail(t, local, "SendToUser 应投递到本节点用户")
	require.Eventually(t, func() bool {
		_, s := rec.count()
		return s == 1
	}, 2*time.Second, 20*time.Millisecond, "SendToUser 应对每个对端节点转发一次")
	require.Equal(t, "node-A", rec.origin("/api/v1/node/send-to-user"), "转发载荷应带发送方 origin")

	// 2. 来自他节点的定向中继：本节点用户收到，但不再回传发送方
	hub.DeliverToUserFromNode("node-B", 42, payload)
	recvOrFail(t, local, "节点定向中继应投递到本节点用户")
	time.Sleep(150 * time.Millisecond)
	_, s := rec.count()
	require.Equal(t, 1, s, "来自他节点的定向消息不应回传发送方（否则双向无限循环）")

	// 3. 回环保护
	hub.DeliverToUserFromNode("node-A", 42, payload)
	noRecv(t, local)
	_, s = rec.count()
	require.Equal(t, 1, s, "回环定向消息不应产生任何新的转发")
}
