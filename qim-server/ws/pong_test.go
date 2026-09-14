package ws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// 回归：前端 useWebSocket 的 connectionMonitor 每 30s 发 {type:'ping'} 并等 {type:'pong'}
// 才 recordPong 复位 pendingPong；服务端若只吞掉 ping 不回 pong，监控在首个心跳后永久假死。
func TestPing_RepliesPong(t *testing.T) {
	client := &Client{hub: &Hub{}, send: make(chan []byte, 8)}

	client.handlePing()

	select {
	case raw := <-client.send:
		var msg WSMessage
		require.NoError(t, json.Unmarshal(raw, &msg))
		require.Equal(t, "pong", msg.Type)
	default:
		t.Fatal("收到 ping 后应回发 pong")
	}
}
