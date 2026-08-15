package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCache_BasicPutGet(t *testing.T) {
	c := NewCache(10)
	c.Put("a", 1)
	v, ok := c.Get("a")
	require.True(t, ok)
	require.Equal(t, 1, v)
	require.Equal(t, 1, c.Len())

	_, ok = c.Get("missing")
	require.False(t, ok)
}

func TestCache_DeleteInvalidates(t *testing.T) {
	// 主动失效是 ConversationMemberCache 修复的核心语义：
	// 成员变更后必须立即可见，不等 TTL 过期。
	c := NewCache(10)
	c.Put("conv_members:1", []byte("old"))
	c.Delete("conv_members:1")
	_, ok := c.Get("conv_members:1")
	require.False(t, ok, "Delete 后不应再命中")
}

func TestCache_TTLExpiry(t *testing.T) {
	c := NewCacheWithTTL(10, 50*time.Millisecond)
	c.Put("a", 1)
	_, ok := c.Get("a")
	require.True(t, ok)

	time.Sleep(80 * time.Millisecond)
	_, ok = c.Get("a")
	require.False(t, ok, "超过 TTL 后不应命中")
}

func TestCache_LRUEviction(t *testing.T) {
	c := NewCache(2)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3) // 超出容量，应淘汰最久未用的 "a"

	_, ok := c.Get("a")
	require.False(t, ok, "最久未用的条目应被淘汰")
	_, ok = c.Get("b")
	require.True(t, ok)
	_, ok = c.Get("c")
	require.True(t, ok)
}

func TestCache_Clear(t *testing.T) {
	c := NewCache(10)
	c.Put("a", 1)
	c.Clear()
	require.Zero(t, c.Len())
	_, ok := c.Get("a")
	require.False(t, ok)
}
