package cache

import (
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const defaultTTL = 10 * time.Minute

// Cache 基于 hashicorp/golang-lru 的 expirable LRU 的薄封装：
// LRU 容量淘汰 + 统一过期时间（构造时指定），并发安全。
// 不做 per-key TTL：当前所有调用点均使用构造时统一 TTL。
type Cache struct {
	lru *expirable.LRU[string, interface{}]
}

func NewCache(maxSize int) *Cache {
	return NewCacheWithTTL(maxSize, defaultTTL)
}

func NewCacheWithTTL(maxSize int, ttl time.Duration) *Cache {
	return &Cache{lru: expirable.NewLRU[string, interface{}](maxSize, nil, ttl)}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	return c.lru.Get(key)
}

func (c *Cache) Put(key string, value interface{}) {
	c.lru.Add(key, value)
}

func (c *Cache) Delete(key string) {
	c.lru.Remove(key)
}

func (c *Cache) Clear() {
	c.lru.Purge()
}

func (c *Cache) Len() int {
	return c.lru.Len()
}

var (
	UserCache               = NewCache(1000)
	ConversationMemberCache = NewCache(500)
	// AvatarPauseCache 缓存用户的 SelfMessagePause 值（分钟）。
	// key: "pause:{userID}", value: int (0=无配置或未启用，>0=启用且分钟数)。
	// TTL 5 分钟；AvatarConfig 创建/删除/更新 SelfMessagePause 时主动失效。
	AvatarPauseCache = NewCacheWithTTL(1000, 5*time.Minute)
)

// InvalidateAvatarPauseCache 清除指定用户的分身暂停缓存。
// 在 AvatarConfig 创建、删除或 SelfMessagePause 字段变更后调用。
func InvalidateAvatarPauseCache(userID uint) {
	AvatarPauseCache.Delete(fmt.Sprintf("pause:%d", userID))
}

// InvalidateConversationMemberCache 清除指定会话的成员列表缓存。
// 成员变更（加群/退群/改角色/转移群主）后调用，防止 TTL 内读到陈旧成员列表。
func InvalidateConversationMemberCache(convID uint) {
	ConversationMemberCache.Delete(fmt.Sprintf("conv_members:%d", convID))
}
