package cache

import (
	"container/list"
	"sync"
)

type Cache struct {
	maxSize int
	mu      sync.Mutex
	items   map[string]*list.Element
	lru     *list.List
}

type entry struct {
	key   string
	value interface{}
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		maxSize: maxSize,
		items:   make(map[string]*list.Element),
		lru:     list.New(),
	}
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

func (c *Cache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.lru.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	if c.lru.Len() >= c.maxSize {
		oldest := c.lru.Back()
		if oldest != nil {
			c.lru.Remove(oldest)
			delete(c.items, oldest.Value.(*entry).key)
		}
	}

	newElem := c.lru.PushFront(&entry{key: key, value: value})
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
