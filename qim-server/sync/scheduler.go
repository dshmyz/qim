package sync

import (
	"context"
	"sync"
	"time"

	"qim-server/database"
	"qim-server/model"
	"qim-server/pkg/logger"
)

type Scheduler struct {
	engine   *Engine
	timers   map[uint]*time.Ticker
	stopCh   chan struct{}
	mu       sync.RWMutex
	running  bool
}

func NewScheduler(engine *Engine) *Scheduler {
	return &Scheduler{
		engine: engine,
		timers: make(map[uint]*time.Ticker),
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	logger.WithModule("Scheduler").Info("定时同步调度器已启动")
	s.reload()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	for id, ticker := range s.timers {
		ticker.Stop()
		delete(s.timers, id)
	}

	logger.WithModule("Scheduler").Info("定时同步调度器已停止")
}

func (s *Scheduler) Reload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reload()
}

func (s *Scheduler) reload() {
	for id, ticker := range s.timers {
		ticker.Stop()
		delete(s.timers, id)
	}

	db := database.GetDB()
	var configs []model.OrgSyncConfig
	if err := db.Where("enabled = ? AND schedule IS NOT NULL AND schedule != ?", true, "").Find(&configs).Error; err != nil {
		logger.WithModule("Scheduler").Error("加载同步配置失败", "error", err)
		return
	}

	for i := range configs {
		s.scheduleConfig(&configs[i])
	}
}

func (s *Scheduler) scheduleConfig(config *model.OrgSyncConfig) {
	if config.Schedule == "" {
		return
	}

	cronResult, err := ParseCron(config.Schedule)
	if err != nil {
		logger.WithModule("Scheduler").Warn("解析调度表达式失败",
			"config_id", config.ID,
			"schedule", config.Schedule,
			"error", err,
		)
		return
	}

	go func(cfg *model.OrgSyncConfig) {
		time.Sleep(cronResult.NextRun)

		ticker := time.NewTicker(cronResult.Interval)
		s.mu.Lock()
		s.timers[cfg.ID] = ticker
		s.mu.Unlock()

		logger.WithModule("Scheduler").Info("已调度同步任务",
			"config_id", cfg.ID,
			"name", cfg.Name,
			"schedule", cfg.Schedule,
			"interval", cronResult.Interval,
		)

		for {
			select {
			case <-ticker.C:
				logger.WithModule("Scheduler").Info("执行定时同步",
					"config_id", cfg.ID,
					"name", cfg.Name,
				)
				s.engine.Sync(context.Background(), cfg)
			case <-s.stopCh:
				ticker.Stop()
				return
			}
		}
	}(config)
}
