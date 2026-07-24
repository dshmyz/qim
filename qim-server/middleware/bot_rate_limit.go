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
	count    int
	lastSeen time.Time
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
	if !exists || now.Sub(v.lastSeen) > l.window {
		l.visitors[botID] = &botVisitor{count: 1, lastSeen: now}
		return true
	}
	v.count++
	v.lastSeen = now
	return v.count <= l.rate
}

func (l *BotRateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for id, v := range l.visitors {
			if time.Since(v.lastSeen) > l.window {
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
