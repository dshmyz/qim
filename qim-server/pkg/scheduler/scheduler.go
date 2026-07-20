// Package scheduler 提供统一的定时任务调度能力，
// 基于 robfig/cron/v3 实现，替代项目中分散的手写调度逻辑。
//
// 支持三种 Job 类型：
//   - AddCronJob：标准 5 段式 cron 表达式（如 "0 2 * * *"）
//   - AddDailyJob：每日定时，格式 "HH:MM"
//   - AddIntervalJob：固定间隔，robfig/cron 的 @every 语法
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/logger"

	"github.com/robfig/cron/v3"
)

// Scheduler 统一管理所有 Job 的生命周期，内部用 robfig/cron/v3 调度
type Scheduler struct {
	cron    *cron.Cron
	mu      sync.Mutex
	started bool
	stop    context.CancelFunc
	done    chan struct{}
	names   map[string]cron.EntryID // Job 名称 → cron EntryID，便于按名移除
}

// New 创建空 Scheduler，需通过 AddXxxJob 注册 Job 后再 Start
func New() *Scheduler {
	return &Scheduler{
		cron:  cron.New(cron.WithLocation(time.Local), cron.WithLogger(cron.PrintfLogger(&nopLogger{}))),
		names: make(map[string]cron.EntryID),
	}
}

// AddCronJob 注册标准 5 段式 cron 表达式任务
// 表达式非法时返回 error，不注册到调度器
// 同名 Job 已存在时返回 error
// 允许在 Start 后调用（robfig/cron v3 支持运行期增删 Job）
func (s *Scheduler) AddCronJob(name, expr string, fn func(ctx context.Context)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.names[name]; exists {
		return fmt.Errorf("job with name %q already exists", name)
	}

	// 提前校验表达式（避免 AddFunc 失败 panic）
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(expr); err != nil {
		return fmt.Errorf("invalid cron expr %q: %w", expr, err)
	}

	id, err := s.cron.AddFunc(expr, func() {
		fn(context.Background())
	})
	if err != nil {
		return fmt.Errorf("add cron job %q: %w", name, err)
	}
	s.names[name] = id
	logger.WithModule("Scheduler").Info("注册 cron job", "name", name, "expr", expr)
	return nil
}

// RemoveByName 移除已注册的 Job（按名称）
// Job 不存在时返回 error
func (s *Scheduler) RemoveByName(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, exists := s.names[name]
	if !exists {
		return fmt.Errorf("job %q not found", name)
	}
	s.cron.Remove(id)
	delete(s.names, name)
	logger.WithModule("Scheduler").Info("移除 cron job", "name", name)
	return nil
}

// ListNames 返回所有已注册 Job 名称（快照）
func (s *Scheduler) ListNames() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(s.names))
	for n := range s.names {
		names = append(names, n)
	}
	return names
}

// AddDailyJob 注册每日定时任务，time 格式 "HH:MM"
// 转译为 robfig/cron 表达式 "M HH * * *"
func (s *Scheduler) AddDailyJob(name, timeStr string, fn func(ctx context.Context)) error {
	t, err := time.ParseInLocation("15:04", timeStr, time.Local)
	if err != nil {
		return fmt.Errorf("invalid daily time %q: %w", timeStr, err)
	}
	expr := fmt.Sprintf("%d %d * * *", t.Minute(), t.Hour())
	return s.AddCronJob(name, expr, fn)
}

// AddIntervalJob 注册固定间隔任务
// interval 持续时长，由 robfig/cron 的 @every 语法处理
func (s *Scheduler) AddIntervalJob(name string, interval time.Duration, fn func(ctx context.Context)) error {
	expr := fmt.Sprintf("@every %s", interval.String())
	return s.AddCronJob(name, expr, fn)
}

// Start 启动所有已注册的 Job；可重复调用，重复调用为 no-op
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	ctx, cancel := context.WithCancel(ctx)
	s.stop = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()

	s.cron.Start()

	// 等待 ctx 取消后停止
	go func() {
		<-ctx.Done()
		stopCtx := s.cron.Stop()
		<-stopCtx.Done()
		close(s.done)
	}()
}

// Stop 取消所有 Job 并等待退出
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	stop := s.stop
	done := s.done
	s.started = false
	s.mu.Unlock()

	stop()
	<-done
}

// nopLogger 不输出 cron 内部日志（避免污染应用日志）
type nopLogger struct{}

func (l *nopLogger) Printf(format string, v ...interface{}) {
	// cron 内部日志默认不输出；如需调试可在此写 logger
	_ = format
	_ = v
}
