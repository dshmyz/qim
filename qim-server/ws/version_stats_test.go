package ws

import (
	"testing"
)

// versionStatCount 读取指定 version+platform 的当前版本统计计数；不存在则返回 0。
func versionStatCount(h *Hub, version, platform string) int64 {
	for _, s := range h.GetVersionStats() {
		if s.Version == version && s.Platform == platform {
			return s.Count
		}
	}
	return 0
}

// TestAsyncBroadcast_DecrementsVersionStatsOnFailedClient 验证 asyncBroadcast 在清理
// send channel 已满的失败客户端时，同步调用 decVersionStats，避免版本分布统计泄漏。
//
// 复现审查发现的偏离计划 T6 的 bug：弱网客户端被清理后版本计数永久残留，
// 导致 GetVersionDistribution 持续虚高。
func TestAsyncBroadcast_DecrementsVersionStatsOnFailedClient(t *testing.T) {
	// db 传 nil：asyncBroadcast 与本场景的清理路径不触达 DB；
	// 故意不把 client 写入 userClients，避免清理时 UpdateUserStatus 访问 nil db
	h := NewHub(nil, "")

	client := &Client{
		hub:      h,
		send:     make(chan []byte, 1),
		userID:   1,
		version:  "2.1.0",
		platform: "windows",
	}
	// 预填满 send channel，使 asyncBroadcast 的 select 走 default 分支 → 进入 failedChan
	client.send <- []byte("pending")

	h.clients.Store(client, true)
	h.incVersionStats(client.version, client.platform)

	if got := versionStatCount(h, "2.1.0", "windows"); got != 1 {
		t.Fatalf("广播前版本计数应为 1，实际 %d", got)
	}

	h.asyncBroadcast([]byte("broadcast-msg"))

	if got := versionStatCount(h, "2.1.0", "windows"); got != 0 {
		t.Fatalf("广播清理失败客户端后版本计数应为 0（已 decVersionStats），实际 %d（计数泄漏）", got)
	}
}
