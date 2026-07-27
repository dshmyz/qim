package middleware

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIPRateLimiter_AllowsUntilLimitThenBlocks(t *testing.T) {
	l := NewIPRateLimiter(3, time.Minute)
	assert.True(t, l.Allow("1.1.1.1"))
	assert.True(t, l.Allow("1.1.1.1"))
	assert.True(t, l.Allow("1.1.1.1"))
	assert.False(t, l.Allow("1.1.1.1")) // 第 4 次超限
}

func TestIPRateLimiter_IndependentPerIP(t *testing.T) {
	l := NewIPRateLimiter(2, time.Minute)
	assert.True(t, l.Allow("1.1.1.1"))
	assert.True(t, l.Allow("1.1.1.1"))
	assert.False(t, l.Allow("1.1.1.1"))
	// 另一个 IP 独立计数
	assert.True(t, l.Allow("2.2.2.2"))
	assert.True(t, l.Allow("2.2.2.2"))
}

// TestIPRateLimiter_WindowResetReleasesBlock 死亡螺旋回归：
// 超限后即便持续请求，窗口过期后应自动恢复（拒绝不刷新窗口起点）。
// 旧实现拒绝时也 lastSeen=now，导致窗口永不重置 -> 永久黑名单。
func TestIPRateLimiter_WindowResetReleasesBlock(t *testing.T) {
	l := NewIPRateLimiter(2, 50*time.Millisecond)
	assert.True(t, l.Allow("1.1.1.1"))
	assert.True(t, l.Allow("1.1.1.1"))
	assert.False(t, l.Allow("1.1.1.1")) // 超限
	// 窗口未过期前持续打，仍拒绝（且不刷新窗口）
	assert.False(t, l.Allow("1.1.1.1"))
	// 窗口过期：恢复
	time.Sleep(60 * time.Millisecond)
	assert.True(t, l.Allow("1.1.1.1"))
}

func TestIPRateLimiter_UpdateConfig(t *testing.T) {
	l := NewIPRateLimiter(1, time.Minute)
	assert.True(t, l.Allow("1.1.1.1"))
	assert.False(t, l.Allow("1.1.1.1"))
	// 放宽到 5/min：仍受限于本窗口已计 1，但新窗口能到 5
	l.UpdateConfig(5, time.Minute)
	// 新 IP 验证新配置
	assert.True(t, l.Allow("9.9.9.9"))
	assert.True(t, l.Allow("9.9.9.9"))
	assert.True(t, l.Allow("9.9.9.9"))
}
