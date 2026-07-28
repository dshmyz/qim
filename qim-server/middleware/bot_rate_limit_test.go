package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBotRateLimiter_AllowsUntilLimitThenBlocks(t *testing.T) {
	l := NewBotRateLimiter(3, time.Minute)
	assert.True(t, l.Allow(1))
	assert.True(t, l.Allow(1))
	assert.True(t, l.Allow(1))
	assert.False(t, l.Allow(1)) // 第 4 次，超限
}

func TestBotRateLimiter_IndependentPerBot(t *testing.T) {
	l := NewBotRateLimiter(2, time.Minute)
	assert.True(t, l.Allow(1))
	assert.True(t, l.Allow(1))
	assert.False(t, l.Allow(1))
	// 另一个 bot 独立计数
	assert.True(t, l.Allow(2))
	assert.True(t, l.Allow(2))
	assert.False(t, l.Allow(2))
}

// TestBotRateLimiter_WindowResetReleasesBlock 死亡螺旋回归：
// 超限后即便持续请求（如 agent 1s 轮询），窗口过期后应自动恢复。
// 旧实现拒绝时也 lastSeen=now，窗口永不重置 -> agent 被永久限流直到流量停歇 1min。
func TestBotRateLimiter_WindowResetReleasesBlock(t *testing.T) {
	l := NewBotRateLimiter(2, 50*time.Millisecond)
	assert.True(t, l.Allow(1))
	assert.True(t, l.Allow(1))
	assert.False(t, l.Allow(1)) // 超限
	// 窗口未过期前持续打，仍拒绝（且不刷新窗口）
	assert.False(t, l.Allow(1))
	// 窗口过期：恢复
	time.Sleep(60 * time.Millisecond)
	assert.True(t, l.Allow(1))
}
