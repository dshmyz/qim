package cache

import (
	"container/list"
	"sync"
	"time"
)

const defaultTTL = 10 * time.Minute

type Cache struct {
	maxSize   int
	defaultTTL time.Duration
	mu        sync.Mutex
	items     map[string]*list.Element
	lru       *list.List
}

type entry struct {
	key       string
	value     interface{}
	expiredAt time.Time
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
		items:      make(map[string]*list.Element),
		lru:        list.New(),
	}
}

func NewCacheWithTTL(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		maxSize:    maxSize,
		defaultTTL: ttl,
		items:      make(map[string]*list.Element),
		lru:        list.New(),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		e := elem.Value.(*entry)
		// 检查是否过期
		if !e.expiredAt.IsZero() && time.Now().After(e.expiredAt) {
			// 过期，删除并返回 miss
			c.lru.Remove(elem)
			delete(c.items, key)
			return nil, false
		}
		c.lru.MoveToFront(elem)
		return e.value, true
	}
	return nil, false
}

func (c *Cache) Put(key string, value interface{}, ttl ...time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 计算过期时间：优先使用传入的 TTL，否则使用默认 TTL
	var expiredAt time.Time
	if len(ttl) > 0 && ttl[0] > 0 {
		expiredAt = time.Now().Add(ttl[0])
	} else {
		expiredAt = time.Now().Add(c.defaultTTL)
	}

	if elem, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*entry).value = value
		elem.Value.(*entry).expiredAt = expiredAt
		return
	}

	if c.lru.Len() >= c.maxSize {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.items, oldest.Value.(*entry).key)
		}
	}

	newElem := c.lru.PushFront(&entry{key: key, value: value, expiredAt: expiredAt})
	c.items[key] = newElem
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.lru.Remove(elem)
		delete(c.items, key)
	}
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lru.Init()
	c.items = make(map[string]*list.Element)
}

func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lru.Len()
}

var (
	UserCache                 = NewCache(1000)
	ConversationMemberCache    = NewCache(500)
)
