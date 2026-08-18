package ws

import (
	"strings"
	"sync/atomic"
)

func versionStatsKey(version, platform string) string {
	return version + "|" + platform
}

// effectiveVersion 规范化版本号：空版本（老客户端未上报，2026-07-03 之前打包）归入"未知版本"桶，
// 使分布统计反映真实在线设备数，而非静默丢弃老客户端。inc/dec 必须共用同一规范化保证加减一致。
func effectiveVersion(version string) string {
	if version == "" {
		return "未知版本"
	}
	return version
}

// incVersionStats 版本计数 +1

func (h *Hub) incVersionStats(version, platform string) {
	key := versionStatsKey(effectiveVersion(version), platform)
	for {
		if v, ok := h.versionStats.Load(key); ok {
			atomic.AddInt64(v.(*int64), 1)
			return
		}
		var count int64 = 1
		actual, loaded := h.versionStats.LoadOrStore(key, &count)
		if !loaded {
			return
		}
		atomic.AddInt64(actual.(*int64), 1)
		return
	}
}

// decVersionStats 版本计数 -1，不低于 0

func (h *Hub) decVersionStats(version, platform string) {
	key := versionStatsKey(effectiveVersion(version), platform)
	if v, ok := h.versionStats.Load(key); ok {
		atomic.AddInt64(v.(*int64), -1)
	}
}

// VersionStat 版本统计项

func (h *Hub) GetVersionStats() []VersionStat {
	var stats []VersionStat
	h.versionStats.Range(func(key, value interface{}) bool {
		count := atomic.LoadInt64(value.(*int64))
		if count <= 0 {
			return true
		}
		parts := strings.Split(key.(string), "|")
		if len(parts) != 2 {
			return true
		}
		stats = append(stats, VersionStat{
			Version:  parts[0],
			Platform: parts[1],
			Count:    count,
		})
		return true
	})
	return stats
}

// VersionUser 在线客户端用户（版本分布明细，管理后台"查看具体人"用）
type VersionUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Version  string `json:"version"`
	Platform string `json:"platform"`
}

// GetVersionUsers 返回指定版本的在线客户端用户列表；version 为空返回全部在线用户。
// 未完成认证的连接（userID==0）不计入。版本号经 effectiveVersion 规范化，
// 与 GetVersionStats 的"未知版本"桶保持一致，使老客户端也能被检索到。
func (h *Hub) GetVersionUsers(version string) []VersionUser {
	var users []VersionUser
	h.clients.Range(func(key, value interface{}) bool {
		client := key.(*Client)
		if client.userID == 0 {
			return true
		}
		if version != "" && effectiveVersion(client.version) != version {
			return true
		}
		users = append(users, VersionUser{
			UserID:   client.userID,
			Username: client.username,
			Version:  effectiveVersion(client.version),
			Platform: client.platform,
		})
		return true
	})
	return users
}
