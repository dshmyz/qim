package service

import (
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/dshmyz/gracedb/pkg/gracedb"
	"github.com/dshmyz/gracedb/pkg/store"
)

// 懒归档（lazy archive）的默认参数。
//
// 在列表访问（GetUserMemories / GetGroupMemories）路径上顺带触发一次
// ArchiveWeakMemoriesUser，把"既弱又长期闲置"的记忆打上 Archived 软删标记，
// 使其自动从默认召回与空查询枚举中消失（软遗忘）。它不物理删除、可逆（清掉
// Archived 标记即恢复），因此即使误归档也能找回——适合做成渐进式卫生动作。
const (
	// lazyArchiveCooldown 同一桶两次归档的最小间隔，避免每次打开面板/构建图谱
	// 都去扫桶做一次归档（那会是热点写路径上的 O(桶大小) 扫描）。
	lazyArchiveCooldown = time.Hour

	// lazyArchiveMaxIdle 记忆至少闲置多久才算归档候选。用它垫一个下限，保证
	// 即便是低强度记忆，也不会在真正"久未使用"前被过早 soft-hide。
	//
	// 这里是时间下限（结合 StrengthThreshold 一起判）——两条都满足才归档。
	lazyArchiveMaxIdle = 7 * 24 * time.Hour // 7 天

	// lazyArchiveStrengthThreshold 记忆强度严格低于此值才算"弱"。默认 gracedb 侧
	// 阈值是 0.4，此处与默认对齐：按 Ebbinghaus 曲线（base=0.4+0.6*Imp、7 天半衰期），
	// 中等重要度且约 2 周没被触碰的记忆才会低于该阈值，属于保守档。
	lazyArchiveStrengthThreshold = 0.4
)

// archiveCooldown 记录每个桶上次执行归档的时间，用于懒触发的节流。
// key 形如 "<namespace>:<userID>"（分身 userID / 群 groupID 都是数字字符串）。
var (
	archiveCooldownMu sync.Mutex
	archiveLastRun    = map[string]time.Time{}
)

// lazyArchiveWeakMemories 对 userID 下 namespace 对应的记忆桶执行一次懒归档，但受
// cooldown 节流。返回本次是否实际执行了归档（未到冷却期或无可归档项都返回 0,nil）。
//
// bucketUserID 是桶键里的用户作用域：分身记忆传 userID，群记忆传 groupID（两者的
// Namespace 分别是 "avatar" / "group_assistant"）。
//
// 设计上故意走"列表访问时顺带做"而非独立协程：既不引入额外的定时调度面，又能利用
// 用户/群真正打开记忆面板或构建图谱的时机，天然按活跃度分发调度。
func lazyArchiveWeakMemories(db *gracedb.DB, bucketUserID, namespace string) (int, error) {
	return lazyArchiveWeakMemoriesOpts(db, bucketUserID, namespace, store.ArchiveOptions{
		StrengthThreshold: lazyArchiveStrengthThreshold,
		MaxIdle:           lazyArchiveMaxIdle,
	})
}

// lazyArchiveWeakMemoriesOpts 是 lazyArchiveWeakMemories 的可注入实现，便于测试用
// 更激进的阈值/无闲置下限来快速造出可归档项；生产路径通过 lazyArchiveWeakMemories
// 传入保守默认值。
func lazyArchiveWeakMemoriesOpts(db *gracedb.DB, bucketUserID, namespace string, opts store.ArchiveOptions) (int, error) {
	if db == nil {
		return 0, nil
	}
	key := namespace + ":" + bucketUserID

	archiveCooldownMu.Lock()
	last, ok := archiveLastRun[key]
	now := time.Now()
	if ok && now.Sub(last) < lazyArchiveCooldown {
		archiveCooldownMu.Unlock()
		return 0, nil
	}
	// 记录下次冷却点（即使本次扫描没有候选项，也防止反复热扫描）。
	archiveLastRun[key] = now
	archiveCooldownMu.Unlock()

	n, err := db.ArchiveWeakMemoriesUser(bucketUserID, namespace, opts)
	if err != nil {
		// 归档是尽力而为的卫生动作，失败不应阻断列表/图谱读取。
		logger.WithModule("MemoryArchive").Warn("懒归档失败",
			"bucket", key, "error", err)
		return 0, err
	}
	if n > 0 {
		logger.WithModule("MemoryArchive").Info("懒归档 soft-hide 弱记忆",
			"bucket", key, "count", n)
	}
	return n, nil
}

// resetArchiveCooldown 清空冷却记录，供测试在多次调用间重置节流状态。
func resetArchiveCooldown() {
	archiveCooldownMu.Lock()
	archiveLastRun = map[string]time.Time{}
	archiveCooldownMu.Unlock()
}
