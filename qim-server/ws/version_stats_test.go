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

// TestUnknownVersionBucket 验证老客户端（未上报版本号）不再被静默丢弃，
// 而是归入"未知版本"桶，且 inc/dec 对称、断开后计数归零。
func TestUnknownVersionBucket(t *testing.T) {
	h := NewHub(nil, "", "http")

	h.incVersionStats("", "windows")
	h.incVersionStats("", "macos")
	h.incVersionStats("2.1.0", "windows")

	if got := versionStatCount(h, "未知版本", "windows"); got != 1 {
		t.Fatalf("空版本 windows 应计入\"未知版本\"桶=1，实际 %d", got)
	}
	if got := versionStatCount(h, "未知版本", "macos"); got != 1 {
		t.Fatalf("空版本 macos 应计入\"未知版本\"桶=1，实际 %d", got)
	}
	if got := versionStatCount(h, "2.1.0", "windows"); got != 1 {
		t.Fatalf("正常版本计数应为 1，实际 %d", got)
	}

	h.decVersionStats("", "windows")
	if got := versionStatCount(h, "未知版本", "windows"); got != 0 {
		t.Fatalf("空版本断开后\"未知版本\"桶应归零，实际 %d", got)
	}
}

// TestGetVersionUsers 验证在线用户枚举：按版本过滤、排除未认证连接、
// 老客户端版本号显示为"未知版本"。
func TestGetVersionUsers(t *testing.T) {
	h := NewHub(nil, "", "http")

	mk := func(userID uint, username, version, platform string) *Client {
		c := &Client{hub: h, userID: userID, username: username, version: version, platform: platform}
		h.clients.Store(c, true)
		return c
	}
	mk(1, "alice", "2.1.0", "windows")
	mk(2, "bob", "", "macos")   // 老客户端：未上报版本
	mk(0, "", "2.1.0", "linux") // 未认证连接，不应计入

	all := h.GetVersionUsers("")
	if len(all) != 2 {
		t.Fatalf("应返回 2 个已认证用户，实际 %d: %+v", len(all), all)
	}

	unknown := h.GetVersionUsers("未知版本")
	if len(unknown) != 1 || unknown[0].Username != "bob" {
		t.Fatalf("未知版本桶应只含 bob，实际 %+v", unknown)
	}
	if unknown[0].Version != "未知版本" {
		t.Fatalf("老客户端版本号应规范化为\"未知版本\"，实际 %q", unknown[0].Version)
	}

	v210 := h.GetVersionUsers("2.1.0")
	if len(v210) != 1 || v210[0].Username != "alice" {
		t.Fatalf("2.1.0 桶应只含 alice，实际 %+v", v210)
	}
}

// TestAsyncBroadcast_DecrementsVersionStatsOnFailedClient 验证 asyncBroadcast 在清理
// send channel 已满的失败客户端时，同步调用 decVersionStats，避免版本分布统计泄漏。
//
// 复现审查发现的偏离计划 T6 的 bug：弱网客户端被清理后版本计数永久残留，
// 导致 GetVersionDistribution 持续虚高。
func TestAsyncBroadcast_DecrementsVersionStatsOnFailedClient(t *testing.T) {
	// db 传 nil：asyncBroadcast 与本场景的清理路径不触达 DB；
	// 故意不把 client 写入 userClients，避免清理时 UpdateUserStatus 访问 nil db
	h := NewHub(nil, "", "http")

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
