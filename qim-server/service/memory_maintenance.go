package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/store"
	"github.com/dshmyz/gracedb/pkg/types"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
)

const (
	// memoryArchiveSweepInterval 后台归档扫描周期。弱记忆归档是卫生动作，不需要高频。
	memoryArchiveSweepInterval = 24 * time.Hour
	memoryBucketPrefix         = "memory:user:"
)

// StartMemoryArchiveSweeper 启动群记忆的后台归档扫描。
//
// 旧实现只在打开「记忆」面板时顺带归档（GetGroupMemories → lazyArchiveWeakMemories），
// 长期不看面板的群记忆桶会无限增长。这里按固定周期枚举所有群记忆桶，执行与面板
// 相同的保守策略（强度 < 0.4 且闲置 >= 7 天），使归档不再依赖用户打开面板。
//
// 返回 stop 以支持测试/优雅退出；生产调用方可忽略返回值。
func StartMemoryArchiveSweeper(db *gracedb.DB, interval time.Duration) (stop func()) {
	if db == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = memoryArchiveSweepInterval
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		timer := time.NewTimer(interval)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				archiveWeakGroupMemories(db)
				timer.Reset(interval)
			}
		}
	}()
	return cancel
}

// archiveWeakGroupMemories 遍历所有群记忆桶并归档弱记忆，返回归档条数。
// 只处理群助手命名空间，分身记忆（avatar）不在此处自动归档。
// 归档后顺带做主题堆积观测（纯日志），为"是否值得做主题级 consolidation"积累数据。
func archiveWeakGroupMemories(db *gracedb.DB) int {
	buckets, err := db.EnumMemoryBuckets()
	if err != nil {
		logger.WithModule("MemoryArchive").Warn("枚举记忆桶失败", "error", err)
		return 0
	}
	total := 0
	for _, bucket := range buckets {
		userID, namespace, ok := parseMemoryBucket(bucket)
		if !ok || namespace != groupMemoryNamespace {
			continue
		}
		n, err := db.ArchiveWeakMemoriesUser(userID, namespace, store.ArchiveOptions{
			StrengthThreshold: lazyArchiveStrengthThreshold,
			MaxIdle:           lazyArchiveMaxIdle,
		})
		if err != nil {
			logger.WithModule("MemoryArchive").Warn("定时归档失败",
				"bucket", bucket, "error", err)
			continue
		}
		total += n
		logGroupMemoryThemeProfile(db, userID)
	}
	if total > 0 {
		logger.WithModule("MemoryArchive").Info("定时归档弱群记忆", "count", total)
	}
	return total
}

// themeProfileSingleThemeThreshold 单主题被判定为"堆积"的条目数下限（占 groupGraph
// 无关；经验值：召回 topK=2 时同主题 ≥6 条已会互相挤占）。
const themeProfileSingleThemeThreshold = 6

// logGroupMemoryThemeProfile 统计某群记忆的主题分布并输出日志（纯观测，零写副作用）。
// 判定信号：单主题条目数 ≥ themeProfileSingleThemeThreshold，或无主题条目占比过高。
// 连续两轮数据都触发时，再评估是否值得上主题级 consolidation（有损合并，不轻上）。
func logGroupMemoryThemeProfile(db *gracedb.DB, groupID string) {
	resp, err := db.SearchMemory(types.MemorySearchRequest{
		UserID:    groupID,
		Scope:     "user",
		Namespace: groupMemoryNamespace,
		TopK:      groupMemoryCountEnumerateLimit,
	})
	if err != nil || len(resp.Results) == 0 {
		return
	}
	themeCount := map[string]int{}
	noTheme := 0
	for _, hit := range resp.Results {
		themes := toStringSlice(hit.Memory.Metadata["knowledge_memory_themes"])
		if len(themes) == 0 {
			noTheme++
			continue
		}
		for _, th := range themes {
			themeCount[th]++
		}
	}

	type themeStat struct {
		name  string
		count int
	}
	stats := make([]themeStat, 0, len(themeCount))
	for name, c := range themeCount {
		stats = append(stats, themeStat{name, c})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].count > stats[j].count })

	top := stats
	if len(top) > 5 {
		top = top[:5]
	}
	pairs := make([]string, 0, len(top))
	hot := 0
	for _, st := range top {
		pairs = append(pairs, fmt.Sprintf("%s=%d", st.name, st.count))
		if st.count >= themeProfileSingleThemeThreshold {
			hot++
		}
	}
	logger.WithModule("MemoryArchive").Info("群记忆主题分布观测",
		"groupID", groupID, "total", len(resp.Results),
		"themes", len(themeCount), "no_theme", noTheme,
		"hot_themes", hot, "top", strings.Join(pairs, ","))
	if hot > 0 {
		logger.WithModule("MemoryArchive").Warn("检测到主题堆积，可评估主题级 consolidation",
			"groupID", groupID, "hot_themes", hot)
	}
}

// parseMemoryBucket 解析 "memory:user:<id>:<namespace>"。
func parseMemoryBucket(bucket string) (userID, namespace string, ok bool) {
	if !strings.HasPrefix(bucket, memoryBucketPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(bucket, memoryBucketPrefix)
	idx := strings.LastIndex(rest, ":")
	if idx <= 0 || idx == len(rest)-1 {
		return "", "", false
	}
	return rest[:idx], rest[idx+1:], true
}
