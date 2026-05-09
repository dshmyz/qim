package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type IPRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     int
	window   time.Duration
}

type Visitor struct {
	count    int
	lastSeen time.Time
}

func NewIPRateLimiter(rate int, window time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     rate,
		window:   window,
	}
	
	go limiter.cleanupVisitors()
	
	return limiter
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	visitor, exists := l.visitors[ip]
	now := time.Now()
	
	if !exists || now.Sub(visitor.lastSeen) > l.window {
		l.visitors[ip] = &Visitor{count: 1, lastSeen: now}
		return true
	}
	
	visitor.count++
	visitor.lastSeen = now
	
	return visitor.count <= l.rate
}

func (l *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		
		l.mu.Lock()
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > l.window {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		if !limiter.Allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    -1,
				"message": "请求过于频繁，请稍后再试",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
