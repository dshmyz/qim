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
