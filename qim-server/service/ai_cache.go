package service

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type AICache struct {
	store sync.Map
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func NewAICache() *AICache {
	return &AICache{}
}

func (c *AICache) GenerateKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *AICache) Get(key string) (string, bool) {
	if v, ok := c.store.Load(key); ok {
		entry := v.(*cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.value, true
		}
		c.store.Delete(key)
	}
	return "", false
}

func (c *AICache) Set(key, value string, ttl time.Duration) {
	c.store.Store(key, &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	})
}

func (c *AICache) GetOrCompute(key string, compute func() (string, error), ttl time.Duration) (string, error) {
	if v, ok := c.Get(key); ok {
		return v, nil
	}

	result, err := compute()
	if err != nil {
		return "", err
	}

	c.Set(key, result, ttl)
	return result, nil
}

func (c *AICache) Delete(key string) {
	c.store.Delete(key)
}

func (c *AICache) DeleteByPrefix(prefix string) {
	c.store.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok && len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			c.store.Delete(key)
		}
		return true
	})
}
