package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AddCronJob：标准 cron 表达式
func Test_CronJob_FiresAtCronExpression(t *testing.T) {
	var fired int32
	s := New()
	err := s.AddCronJob("test-cron", "*/1 * * * *", func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	// 等最多 70 秒（跨分钟边界）
	deadline := time.Now().Add(70 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&fired) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	assert.Greater(t, atomic.LoadInt32(&fired), int32(0), "应在 cron 表达式触发时调用")
}

// AddDailyJob：每日定时，转译为 0 HH MM * * *
func Test_DailyJob_FiresAtScheduledTime(t *testing.T) {
	now := time.Now()
	fireAt := now.Add(time.Minute).Format("15:04")

	var fired int32
	s := New()
	err := s.AddDailyJob("test-daily", fireAt, func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	maxWait := 75 * time.Second
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&fired) > 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	assert.Greater(t, atomic.LoadInt32(&fired), int32(0), "DailyJob 应在指定时刻触发")
}

// AddIntervalJob：robfig/cron 的 @every 表达式
// 注：robfig/cron v3 最小粒度为 1 秒
func Test_IntervalJob_FiresRepeatedly(t *testing.T) {
	var fired int32
	s := New()
	err := s.AddIntervalJob("test-interval", 1*time.Second, func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)
	defer s.Stop()

	time.Sleep(3500 * time.Millisecond)
	assert.GreaterOrEqual(t, atomic.LoadInt32(&fired), int32(3), "IntervalJob 应至少触发 3 次")
}

// Stop 让所有 Job 立即退出
func Test_Stop_TerminatesAllJobs(t *testing.T) {
	var fired int32
	s := New()
	require.NoError(t, s.AddIntervalJob("test-stop", 1*time.Second, func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Start(ctx)

	time.Sleep(1500 * time.Millisecond)
	s.Stop()
	firedBeforeStop := atomic.LoadInt32(&fired)
	require.Greater(t, firedBeforeStop, int32(0), "停止前应至少触发一次")

	time.Sleep(1500 * time.Millisecond)
	assert.Equal(t, firedBeforeStop, atomic.LoadInt32(&fired), "Stop 后不应再触发")
}

// ctx 取消也应让所有 Job 退出
func Test_ContextCancel_TerminatesAllJobs(t *testing.T) {
	var fired int32
	s := New()
	require.NoError(t, s.AddIntervalJob("test-ctx", 1*time.Second, func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	go s.Start(ctx)

	time.Sleep(1500 * time.Millisecond)
	cancel()
	firedBeforeCancel := atomic.LoadInt32(&fired)
	require.Greater(t, firedBeforeCancel, int32(0))

	time.Sleep(1500 * time.Millisecond)
	assert.Equal(t, firedBeforeCancel, atomic.LoadInt32(&fired), "ctx 取消后不应再触发")
}

// 非法 cron 表达式应被拒绝
func Test_AddCronJob_RejectsInvalidExpression(t *testing.T) {
	s := New()
	err := s.AddCronJob("bad", "not-a-cron", func(ctx context.Context) {})
	assert.Error(t, err, "非法 cron 表达式应被拒绝")
}

// 非法 DailyJob 时间不应 panic，应返回 error
func Test_AddDailyJob_RejectsInvalidTime(t *testing.T) {
	s := New()
	err := s.AddDailyJob("bad", "not-a-time", func(ctx context.Context) {})
	assert.Error(t, err, "非法 time 应返回 error")
}

// RemoveByName 移除已注册的 Job，移除后不再触发
func Test_RemoveByName_StopsFiring(t *testing.T) {
	var fired int32
	s := New()
	require.NoError(t, s.AddIntervalJob("to-remove", 1*time.Second, func(ctx context.Context) {
		atomic.AddInt32(&fired, 1)
	}))
	require.NoError(t, s.AddIntervalJob("keep", 1*time.Second, func(ctx context.Context) {
		// 不重要，仅占位
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	time.Sleep(1500 * time.Millisecond)
	firedBeforeRemove := atomic.LoadInt32(&fired)
	require.Greater(t, firedBeforeRemove, int32(0), "移除前应至少触发一次")

	// 移除并等待确认不再触发
	require.NoError(t, s.RemoveByName("to-remove"))
	time.Sleep(1500 * time.Millisecond)
	firedAfterRemove := atomic.LoadInt32(&fired)
	assert.Equal(t, firedBeforeRemove, firedAfterRemove, "RemoveByName 后不再触发")

	s.Stop()
}

// RemoveByName 移除不存在的 Job 应返回 error
func Test_RemoveByName_NonExistentReturnsError(t *testing.T) {
	s := New()
	err := s.RemoveByName("does-not-exist")
	assert.Error(t, err, "移除不存在的 Job 应返回 error")
}

// ListNames 返回所有已注册 Job 名称
func Test_ListNames_ReturnsAllRegisteredJobs(t *testing.T) {
	s := New()
	require.NoError(t, s.AddIntervalJob("job-a", 1*time.Second, func(ctx context.Context) {}))
	require.NoError(t, s.AddIntervalJob("job-b", 1*time.Second, func(ctx context.Context) {}))

	names := s.ListNames()
	assert.Contains(t, names, "job-a")
	assert.Contains(t, names, "job-b")
	assert.Len(t, names, 2)
}
