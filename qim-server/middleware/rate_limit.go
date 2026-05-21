package middleware

import (
	"fmt"
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

type LoginLimiter struct {
	visitors    map[string]*LoginVisitor
	mu          sync.RWMutex
	maxAttempts int
	window      time.Duration
	banDuration time.Duration
}

type LoginVisitor struct {
	attempts    int
	lastSeen    time.Time
	bannedUntil time.Time
}

func NewLoginLimiter(maxAttempts int, window time.Duration, banDuration time.Duration) *LoginLimiter {
	limiter := &LoginLimiter{
		visitors:    make(map[string]*LoginVisitor),
		maxAttempts: maxAttempts,
		window:      window,
		banDuration: banDuration,
	}
	go limiter.cleanup()
	return limiter
}

func (l *LoginLimiter) Allow(ip string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	visitor, exists := l.visitors[ip]

	if !exists || now.Sub(visitor.lastSeen) > l.window {
		l.visitors[ip] = &LoginVisitor{attempts: 1, lastSeen: now}
		return true, time.Time{}
	}

	visitor.lastSeen = now

	if !visitor.bannedUntil.IsZero() && now.Before(visitor.bannedUntil) {
		return false, visitor.bannedUntil
	}

	if visitor.bannedUntil.Before(now) {
		visitor.bannedUntil = time.Time{}
		visitor.attempts = 0
	}

	visitor.attempts++
	if visitor.attempts > l.maxAttempts {
		visitor.bannedUntil = now.Add(l.banDuration)
		return false, visitor.bannedUntil
	}

	return true, time.Time{}
}

func (l *LoginLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	visitor, exists := l.visitors[ip]
	if !exists {
		l.visitors[ip] = &LoginVisitor{attempts: 1, lastSeen: now}
		return
	}
	visitor.attempts++
	if visitor.attempts > l.maxAttempts {
		visitor.bannedUntil = now.Add(l.banDuration)
	}
}

func (l *LoginLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > l.window && visitor.bannedUntil.IsZero() {
				delete(l.visitors, ip)
			}
		}
		l.mu.Unlock()
	}
}

func LoginRateLimitMiddleware(limiter *LoginLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed, bannedUntil := limiter.Allow(ip)
		if !allowed {
			retryAfter := time.Until(bannedUntil)
			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    -1,
				"message": "登录尝试次数过多，请稍后再试",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

var GlobalLoginLimiter *LoginLimiter

func SetGlobalLoginLimiter(l *LoginLimiter) {
	GlobalLoginLimiter = l
}

func RecordLoginFailure(ip string) {
	if GlobalLoginLimiter != nil {
		GlobalLoginLimiter.RecordFailure(ip)
	}
}
