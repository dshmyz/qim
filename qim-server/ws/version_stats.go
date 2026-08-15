package ws

import (
	"strings"
	"sync/atomic"
)

func versionStatsKey(version, platform string) string {
	return version + "|" + platform
}

// incVersionStats 版本计数 +1

func (h *Hub) incVersionStats(version, platform string) {
	if version == "" {
		return
	}
	key := versionStatsKey(version, platform)
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
	if version == "" {
		return
	}
	key := versionStatsKey(version, platform)
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
