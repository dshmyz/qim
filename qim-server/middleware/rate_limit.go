package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type IPRateLimiter struct {
	visitors map[string]*Visitor
	mu       sync.RWMutex
	rate     atomic.Int64
	window   atomic.Int64 // 存储为纳秒
}

type Visitor struct {
	count    int
	lastSeen time.Time
}

func NewIPRateLimiter(rate int, window time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		visitors: make(map[string]*Visitor),
	}
	limiter.rate.Store(int64(rate))
	limiter.window.Store(int64(window))

	go limiter.cleanupVisitors()

	return limiter
}

// UpdateConfig 动态更新速率限制参数
func (l *IPRateLimiter) UpdateConfig(rate int, window time.Duration) {
	l.rate.Store(int64(rate))
	l.window.Store(int64(window))
}

func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := time.Duration(l.window.Load())

	visitor, exists := l.visitors[ip]
	now := time.Now()

	if !exists || now.Sub(visitor.lastSeen) > window {
		l.visitors[ip] = &Visitor{count: 1, lastSeen: now}
		return true
	}

	visitor.count++
	visitor.lastSeen = now

	return visitor.count <= int(l.rate.Load())
}

func (l *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		l.mu.Lock()
		window := time.Duration(l.window.Load())
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > window {
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
	maxAttempts atomic.Int64
	window      atomic.Int64 // 纳秒
	banDuration atomic.Int64 // 纳秒
}

type LoginVisitor struct {
	attempts    int
	lastSeen    time.Time
	bannedUntil time.Time
}

func NewLoginLimiter(maxAttempts int, window time.Duration, banDuration time.Duration) *LoginLimiter {
	limiter := &LoginLimiter{
		visitors: make(map[string]*LoginVisitor),
	}
	limiter.maxAttempts.Store(int64(maxAttempts))
	limiter.window.Store(int64(window))
	limiter.banDuration.Store(int64(banDuration))
	go limiter.cleanup()
	return limiter
}

// UpdateConfig 动态更新登录限流参数
func (l *LoginLimiter) UpdateConfig(maxAttempts int, window time.Duration, banDuration time.Duration) {
	l.maxAttempts.Store(int64(maxAttempts))
	l.window.Store(int64(window))
	l.banDuration.Store(int64(banDuration))
}

func (l *LoginLimiter) Allow(ip string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	window := time.Duration(l.window.Load())
	banDuration := time.Duration(l.banDuration.Load())
	maxAttempts := int(l.maxAttempts.Load())

	visitor, exists := l.visitors[ip]

	if !exists || now.Sub(visitor.lastSeen) > window {
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

	if visitor.attempts >= maxAttempts {
		visitor.bannedUntil = now.Add(banDuration)
		return false, visitor.bannedUntil
	}

	return true, time.Time{}
}

func (l *LoginLimiter) RecordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	banDuration := time.Duration(l.banDuration.Load())
	maxAttempts := int(l.maxAttempts.Load())

	visitor, exists := l.visitors[ip]
	if !exists {
		l.visitors[ip] = &LoginVisitor{attempts: 1, lastSeen: now}
		return
	}
	visitor.attempts++
	if visitor.attempts >= maxAttempts {
		visitor.bannedUntil = now.Add(banDuration)
	}
}

func (l *LoginLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		l.mu.Lock()
		window := time.Duration(l.window.Load())
		for ip, visitor := range l.visitors {
			if time.Since(visitor.lastSeen) > window && visitor.bannedUntil.IsZero() {
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
var GlobalIPRateLimiter *IPRateLimiter

func SetGlobalLoginLimiter(l *LoginLimiter) {
	GlobalLoginLimiter = l
}

func SetGlobalIPRateLimiter(l *IPRateLimiter) {
	GlobalIPRateLimiter = l
}

func RecordLoginFailure(ip string) {
	if GlobalLoginLimiter != nil {
		GlobalLoginLimiter.RecordFailure(ip)
	}
}

// ReloadRateLimitFromDB 从数据库重新加载速率限制配置并更新限流器
func ReloadRateLimitFromDB(getConfig func(string) (string, error)) {
	if GlobalIPRateLimiter != nil {
		rate := 500
		window := time.Minute
		if v, err := getConfig("rate_limit:global_rate"); err == nil {
			fmt.Sscanf(v, "%d", &rate)
			if rate <= 0 {
				rate = 500
			}
		}
		if v, err := getConfig("rate_limit:global_window_seconds"); err == nil {
			sec := 60
			fmt.Sscanf(v, "%d", &sec)
			if sec > 0 {
				window = time.Duration(sec) * time.Second
			}
		}
		GlobalIPRateLimiter.UpdateConfig(rate, window)
	}

	if GlobalLoginLimiter != nil {
		maxAttempts := 5
		window := time.Minute
		banDuration := 15 * time.Minute
		if v, err := getConfig("rate_limit:login_max_attempts"); err == nil {
			fmt.Sscanf(v, "%d", &maxAttempts)
			if maxAttempts <= 0 {
				maxAttempts = 5
			}
		}
		if v, err := getConfig("rate_limit:login_window_seconds"); err == nil {
			sec := 60
			fmt.Sscanf(v, "%d", &sec)
			if sec > 0 {
				window = time.Duration(sec) * time.Second
			}
		}
		if v, err := getConfig("rate_limit:login_ban_seconds"); err == nil {
			sec := 900
			fmt.Sscanf(v, "%d", &sec)
			if sec > 0 {
				banDuration = time.Duration(sec) * time.Second
			}
		}
		GlobalLoginLimiter.UpdateConfig(maxAttempts, window, banDuration)
	}
}
