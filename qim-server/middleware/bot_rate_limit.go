package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/dshmyz/qim/qim-server/pkg/response"
	"github.com/gin-gonic/gin"
)

// BotRateLimiter 按 bot_id 维度的固定窗口限流，防止单个 agent 滥用出站 API。
// 结构对齐 IPRateLimiter，仅 key 由 IP 改为 bot_id。
type BotRateLimiter struct {
	visitors map[uint]*botVisitor
	mu       sync.Mutex
	rate     int
	window   time.Duration
}

type botVisitor struct {
	count       int
	windowStart time.Time // 当前固定窗口起点；窗口过期后重置
}

// NewBotRateLimiter rate=窗口内允许的请求数，window=窗口时长。
func NewBotRateLimiter(rate int, window time.Duration) *BotRateLimiter {
	l := &BotRateLimiter{
		visitors: make(map[uint]*botVisitor),
		rate:     rate,
		window:   window,
	}
	go l.cleanup()
	return l
}

func (l *BotRateLimiter) Allow(botID uint) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	v, exists := l.visitors[botID]
	// 新 bot 或上一窗口已过期：开新窗口，计数从 1 起。
	if !exists || now.Sub(v.windowStart) > l.window {
		l.visitors[botID] = &botVisitor{count: 1, windowStart: now}
		return true
	}
	// 窗口内已达上限：拒绝，不刷新 windowStart、不增计数，窗口到期自然重置。
	// （若拒绝也刷新 lastSeen，持续轮询会让窗口永不重置 -> 死亡螺旋永久黑名单。）
	if v.count >= l.rate {
		return false
	}
	v.count++
	return true
}

func (l *BotRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for id, v := range l.visitors {
			if time.Since(v.windowStart) > l.window {
				delete(l.visitors, id)
			}
		}
		l.mu.Unlock()
	}
}

// BotRateLimitMiddleware 按 context 中的 bot_id 限流。须在 BotAuthMiddleware 之后使用。
func BotRateLimitMiddleware(limiter *BotRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		botIDVal, _ := c.Get("bot_id")
		botID, ok := botIDVal.(uint)
		if !ok {
			response.Unauthorized(c, "Bot 身份无效")
			c.Abort()
			return
		}
		if !limiter.Allow(botID) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    -1,
				"message": "Bot 请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
