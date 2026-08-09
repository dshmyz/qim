package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// orchJob 编排任务：key 用于按 key 串行/限流，handle 为生成+发送闭包。
type orchJob struct {
	key    uint
	handle func()
}

// keyedLimiter 包装按 key 创建的限流器及其最后使用时间，用于 idle eviction。
// sync.Map 的 value 统一用此结构体，避免 limiter 与 lastUsed 分离导致的一致性问题。
type keyedLimiter struct {
	limiter  *rate.Limiter
	lastUsed atomic.Int64 // unix nano
}

// keyedLock 包装按 key 创建的串行锁及其最后使用时间，用于 idle eviction。
type keyedLock struct {
	mu       *sync.Mutex
	lastUsed atomic.Int64 // unix nano
}

// ReplyOrchestratorOpts 编排引擎的可选策略。
// 各字段可独立开关，把并发治理的语义收敛为一个组件声明：
//   - Workers     : 固定 worker 数（限并发/防洪峰），>=1（默认 1）
//   - GlobalRate  : 可选全局限流（nil=不限），已带突发上限，如分身全局 rate.NewLimiter(rpm, rpm)
//   - PerKeyRate  : 可选按 key（会话/用户）限流（rate.Limit(0)=不限），如分身每用户 10/min
//   - PerKeyBurst : 按 key 限流突发上限（默认 1）
//   - Serialize   : 是否按 key 严格串行（防同会话乱序/重复回复）
type ReplyOrchestratorOpts struct {
	Workers     int
	GlobalRate  *rate.Limiter
	PerKeyRate  rate.Limit
	PerKeyBurst int
	Serialize   bool
}

// ReplyOrchestrator 通用 AI 回复并发编排引擎，接管"限并发 + 速率治理 + 按会话串行"
// 这套通用并发语义，供专属机器人 / 群助手 / 分身等各回复入口共用。
//
// 各入口只做两件事：
//  1. 构造本引擎时声明自己的并发策略（Worker 数 / 是否限流 / 是否串行）；
//  2. Submit 时提供 (key, 生成+发送闭包)。
//
// 生成与发送逻辑本体不在此、由闭包承载，行为零变化；本引擎只保证并发边界。
type ReplyOrchestrator struct {
	queue       chan orchJob
	workers     int
	limiter     *rate.Limiter // 全局速率（nil=不限）
	keyLimiters sync.Map      // key -> *keyedLimiter：按 key 限流
	perKeyRate  rate.Limit    // 按 key 限流频率（0=不限）
	perKeyBurst int           // 按 key 限流突发上限
	convLocks   sync.Map      // key -> *keyedLock：按 key 串行
	serialize   bool
	submitCount atomic.Int64 // 提交计数，用于触发定期 idle key 清理
}

// NewReplyOrchestrator 创建并发编排引擎，启动 workers 个消费协程。
func NewReplyOrchestrator(opts ReplyOrchestratorOpts) *ReplyOrchestrator {
	if opts.Workers < 1 {
		opts.Workers = 1
	}
	if opts.PerKeyBurst < 1 {
		opts.PerKeyBurst = 1
	}
	o := &ReplyOrchestrator{
		queue:       make(chan orchJob, 100),
		workers:     opts.Workers,
		limiter:     opts.GlobalRate,
		perKeyRate:  opts.PerKeyRate,
		perKeyBurst: opts.PerKeyBurst,
		serialize:   opts.Serialize,
	}
	for i := 0; i < o.workers; i++ {
		go o.run()
	}
	return o
}

// Submit 提交一个会话/用户的回复任务。队列满时阻塞等待最多 2 秒，仍失败则返回错误、
// 由调用方记录（不静默丢失也不无限堆积 goroutine）。
func (o *ReplyOrchestrator) Submit(key uint, handle func()) error {
	// 每 128 次提交触发一次 idle key 清理，防止 sync.Map 无限增长
	if o.submitCount.Add(1)%128 == 0 {
		o.cleanupIdleKeys()
	}
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case o.queue <- orchJob{key: key, handle: handle}:
		return nil
	case <-timer.C:
		return fmt.Errorf("会话回复入队超时，已跳过")
	}
}

// run 消费队列任务：按需全局限流 + 按 key 限流 + 按 key 串行后执行闭包。
func (o *ReplyOrchestrator) run() {
	ctx := context.Background()
	for job := range o.queue {
		// 1) 全局限流（nil=不限）
		if o.limiter != nil {
			if err := o.limiter.Wait(ctx); err != nil {
				continue
			}
		}
		// 2) 按 key 限流（0=不限）
		if o.perKeyRate > 0 {
			if err := o.keyLimiter(job.key).Wait(ctx); err != nil {
				continue
			}
		}
		// 3) 按 key 严格串行（可选）
		if o.serialize {
			lock := o.keyLock(job.key)
			lock.Lock()
			job.handle()
			lock.Unlock()
		} else {
			job.handle()
		}
	}
}

// keyLimiter 返回指定 key 的限流器（按需创建）。
func (o *ReplyOrchestrator) keyLimiter(key uint) *rate.Limiter {
	v, _ := o.keyLimiters.LoadOrStore(key, &keyedLimiter{
		limiter: rate.NewLimiter(o.perKeyRate, o.perKeyBurst),
	})
	kl := v.(*keyedLimiter)
	kl.lastUsed.Store(time.Now().UnixNano())
	return kl.limiter
}

// keyLock 返回指定 key 的串行锁（按需创建）。
func (o *ReplyOrchestrator) keyLock(key uint) *sync.Mutex {
	v, _ := o.convLocks.LoadOrStore(key, &keyedLock{
		mu: &sync.Mutex{},
	})
	kl := v.(*keyedLock)
	kl.lastUsed.Store(time.Now().UnixNano())
	return kl.mu
}

// cleanupIdleKeys 清理超过 30 分钟未使用的限流器/锁，防止 sync.Map 无限增长。
// 删除后若同一 key 再次提交，LoadOrStore 会按需创建新实例，功能不受影响。
//
// 已知局限（刻意取舍）：淘汰仅依据 lastUsed，不检查锁当前是否持有/在队列等待。lastUsed
// 只在出队（keyLock/keyLimiter）时刷新，故该机制基于「30 分钟远超单次回复时长」的失效抢占
// 假设——只要 job.handle() 在 30 分钟内完成，锁就不会在持有时被剔除。仅当某次生成+发送
// 挂死超 30 分钟且期间无同 key 新任务出队时，锁才可能被提前剔除，下一任务新建互斥锁并发
// 执行、破坏 Serialize 的按会话串行。属低概率边界，接受此权衡以换取 sync.Map 不再无限增长。
func (o *ReplyOrchestrator) cleanupIdleKeys() {
	cutoff := time.Now().Add(-30 * time.Minute).UnixNano()
	o.keyLimiters.Range(func(key, val any) bool {
		kl := val.(*keyedLimiter)
		if kl.lastUsed.Load() < cutoff {
			o.keyLimiters.Delete(key)
		}
		return true
	})
	o.convLocks.Range(func(key, val any) bool {
		kl := val.(*keyedLock)
		if kl.lastUsed.Load() < cutoff {
			o.convLocks.Delete(key)
		}
		return true
	})
}

// Close 关闭编排引擎，释放 worker goroutine。关闭后 Submit 将 panic（向已关闭 channel 发送），
// 调用方应确保 Close 后不再 Submit。通常在服务关闭时调用。
func (o *ReplyOrchestrator) Close() {
	close(o.queue)
}
