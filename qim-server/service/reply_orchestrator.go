package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// orchJob 编排任务：key 用于按 key 串行/限流，handle 为生成+发送闭包。
type orchJob struct {
	key    uint
	handle func()
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
	keyLimiters sync.Map      // key -> *rate.Limiter：按 key 限流
	perKeyRate  rate.Limit    // 按 key 限流频率（0=不限）
	perKeyBurst int           // 按 key 限流突发上限
	convLocks   sync.Map      // key -> *sync.Mutex：按 key 串行
	serialize   bool
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
	select {
	case o.queue <- orchJob{key: key, handle: handle}:
		return nil
	case <-time.After(2 * time.Second):
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
	v, _ := o.keyLimiters.LoadOrStore(key, rate.NewLimiter(o.perKeyRate, o.perKeyBurst))
	return v.(*rate.Limiter)
}

// keyLock 返回指定 key 的串行锁（按需创建）。
func (o *ReplyOrchestrator) keyLock(key uint) *sync.Mutex {
	v, _ := o.convLocks.LoadOrStore(key, &sync.Mutex{})
	return v.(*sync.Mutex)
}
