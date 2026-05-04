package service

import (
	"context"
	"fmt"
	"sync"
	"time"
	"qim-server/model"

	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// AvatarTask 分身任务
type AvatarTask struct {
	UserID         uint
	ConversationID uint
	TriggerMessage string
	TriggerUserID  uint
	IsGroupChat    bool
	GroupName      string
	TriggerName    string
}

// AvatarWorkerPool 分身工作池
type AvatarWorkerPool struct {
	queue        chan AvatarTask
	workers      int
	limiter      *rate.Limiter
	userLimiters sync.Map
	service      *AvatarService
	db           *gorm.DB
}

// NewAvatarWorkerPool 创建分身工作池
func NewAvatarWorkerPool(workers int, globalRPM int, service *AvatarService) *AvatarWorkerPool {
	pool := &AvatarWorkerPool{
		queue:   make(chan AvatarTask, 100),
		workers: workers,
		limiter: rate.NewLimiter(rate.Every(time.Minute/time.Duration(globalRPM)), globalRPM),
		service: service,
		db:      service.db,
	}

	for i := 0; i < workers; i++ {
		go pool.run()
	}

	return pool
}

// Submit 提交任务
func (p *AvatarWorkerPool) Submit(task AvatarTask) error {
	select {
	case p.queue <- task:
		return nil
	default:
		return fmt.Errorf("队列已满，请稍后重试")
	}
}

// run 运行工作协程
func (p *AvatarWorkerPool) run() {
	for task := range p.queue {
		p.process(task)
	}
}

// process 处理任务
func (p *AvatarWorkerPool) process(task AvatarTask) {
	ctx := context.Background()

	if err := p.limiter.Wait(ctx); err != nil {
		return
	}

	userLimiter := p.getUserLimiter(task.UserID)
	if err := userLimiter.Wait(ctx); err != nil {
		return
	}

	var session model.AvatarSession
	err := p.db.Where("user_id = ? AND conversation_id = ?", task.UserID, task.ConversationID).First(&session).Error
	if err == nil && session.TakeoverUntil != nil && session.TakeoverUntil.After(time.Now()) {
		return
	}

	reply, err := p.service.GenerateReply(task.UserID, task.ConversationID, task.TriggerMessage)
	if err != nil {
		return
	}

	if task.IsGroupChat {
		p.sendPrivateReply(task, reply)
	} else {
		p.sendDirectReply(task, reply)
	}

	now := time.Now()
	p.db.Model(&session).Update("last_reply_at", now)
}

// getUserLimiter 获取用户级别的限流器
func (p *AvatarWorkerPool) getUserLimiter(userID uint) *rate.Limiter {
	limiterAny, _ := p.userLimiters.LoadOrStore(userID, rate.NewLimiter(rate.Every(time.Minute/10), 10))
	return limiterAny.(*rate.Limiter)
}

// sendPrivateReply 发送私聊回复（群聊场景）
func (p *AvatarWorkerPool) sendPrivateReply(task AvatarTask, reply string) {
	// TODO: 实现私聊回复逻辑
	// 1. 找到或创建分身用户与触发者的私聊会话
	// 2. 发送消息，标记 is_avatar_reply: true
	// 3. 通知分身用户
}

// sendDirectReply 发送直接回复（私聊场景）
func (p *AvatarWorkerPool) sendDirectReply(task AvatarTask, reply string) {
	// TODO: 实现直接回复逻辑
	// 1. 在当前会话中发送消息
	// 2. 标记 is_avatar_reply: true
}
