package sync

import (
	"context"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
	"github.com/dshmyz/qim/qim-server/pkg/logger"
	"github.com/dshmyz/qim/qim-server/pkg/scheduler"

	"gorm.io/gorm"
)

const syncTimeout = 10 * time.Minute

// JobNamePrefix 是所有 OrgSync Job 在统一调度器中的名称前缀
const JobNamePrefix = "orgsync-"

// syncFn 把 OrgSyncConfig 同步执行出去，便于测试时替换
type syncFn func(ctx context.Context, cfg *model.OrgSyncConfig)

// LoadOrgSyncJobsWithSyncFn 从 DB 加载启用的 OrgSyncConfig 并注册到统一调度器
// 非法 cron 表达式、空 Schedule、禁用的 config 会被跳过，不阻塞其他 config 注册
// syncFn 用于回调时实际执行同步（便于测试注入 mock）
func LoadOrgSyncJobsWithSyncFn(sched *scheduler.Scheduler, db *gorm.DB, fn syncFn) error {
	var configs []model.OrgSyncConfig
	// 注意：SQLite 把 bool 存为 0/1，GORM 默认查询用 true/false 在不同方言下行为不一致
	// 这里用 enabled = ? 配合 1，确保跨方言（SQLite/MySQL）一致
	if err := db.Where("enabled = ? AND schedule IS NOT NULL AND schedule != ?",
		1, "").Find(&configs).Error; err != nil {
		return err
	}

	for i := range configs {
		cfg := configs[i]
		if cfg.Schedule == "" {
			continue
		}

		// 转译为 robfig/cron v3 兼容表达式
		cronExpr, err := translateToCronV3(cfg.Schedule)
		if err != nil {
			logger.WithModule("Scheduler").Warn("转译调度表达式失败，跳过",
				"config_id", cfg.ID,
				"schedule", cfg.Schedule,
				"error", err,
			)
			continue
		}

		jobName := JobNamePrefix + cfg.Name
		if err := sched.AddCronJob(jobName, cronExpr, func(ctx context.Context) {
			logger.WithModule("Scheduler").Info("执行定时同步",
				"config_id", cfg.ID,
				"name", cfg.Name,
			)
			syncCtx, cancel := context.WithTimeout(ctx, syncTimeout)
			defer cancel()
			fn(syncCtx, &cfg)
		}); err != nil {
			logger.WithModule("Scheduler").Error("注册同步 cron job 失败",
				"config_id", cfg.ID,
				"name", cfg.Name,
				"expr", cronExpr,
				"error", err,
			)
			continue
		}

		logger.WithModule("Scheduler").Info("已调度同步任务",
			"config_id", cfg.ID,
			"name", cfg.Name,
			"schedule", cfg.Schedule,
			"expr", cronExpr,
		)
	}
	return nil
}

// LoadOrgSyncJobs 是 LoadOrgSyncJobsWithSyncFn 的便捷封装，
// 使用 engine.Sync 作为同步回调（生产路径）
func LoadOrgSyncJobs(sched *scheduler.Scheduler, engine *Engine, db *gorm.DB) error {
	return LoadOrgSyncJobsWithSyncFn(sched, db, func(ctx context.Context, cfg *model.OrgSyncConfig) {
		engine.Sync(ctx, cfg)
	})
}

// ReloadOrgSyncJobs 移除所有 OrgSync-* Job 后重新加载
// 用于管理员新增/修改/删除 OrgSyncConfig 时让变更立即生效
func ReloadOrgSyncJobs(sched *scheduler.Scheduler, engine *Engine, db *gorm.DB) error {
	// 移除所有 orgsync-* 前缀的 Job
	for _, name := range sched.ListNames() {
		if len(name) > len(JobNamePrefix) && name[:len(JobNamePrefix)] == JobNamePrefix {
			if err := sched.RemoveByName(name); err != nil {
				logger.WithModule("Scheduler").Warn("移除旧 OrgSync Job 失败",
					"name", name, "error", err)
			}
		}
	}
	// 重新加载
	return LoadOrgSyncJobs(sched, engine, db)
}

// --- 兼容性保留：旧 sync.Scheduler 作为薄壳，内部转发到辅助函数 ---

// Scheduler 是 OrgSync 专用调度器，内部基于 pkg/scheduler（robfig/cron/v3）
// 对外保留 Start/Stop/Reload API 以兼容现有调用方
type Scheduler struct {
	engine   *Engine
	mu       sync.Mutex
	started  bool
	sched    *scheduler.Scheduler
}

func NewScheduler(engine *Engine) *Scheduler {
	return &Scheduler{
		engine: engine,
		sched:  scheduler.New(),
	}
}

// Start 启动调度器；重复调用为 no-op
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	s.mu.Unlock()

	logger.WithModule("Scheduler").Info("定时同步调度器已启动")
	if err := LoadOrgSyncJobs(s.sched, s.engine, database.GetDB()); err != nil {
		logger.WithModule("Scheduler").Error("加载 OrgSync 配置失败", "error", err)
	}
	s.sched.Start(context.Background())
}

// Stop 停止调度器并等待所有 Job 退出
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	s.mu.Unlock()

	s.sched.Stop()
	logger.WithModule("Scheduler").Info("定时同步调度器已停止")
}

// Reload 重新加载所有 OrgSyncConfig 并重建 Job
func (s *Scheduler) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return
	}
	if err := ReloadOrgSyncJobs(s.sched, s.engine, database.GetDB()); err != nil {
		logger.WithModule("Scheduler").Error("重载 OrgSync 配置失败", "error", err)
	}
}
