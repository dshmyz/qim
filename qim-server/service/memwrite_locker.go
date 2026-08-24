package service

import "sync"

// keyedMutex 按 key 粒度的互斥锁：保护同一 key（同一个用户/群）的记忆写路径，
// 防止多条近义/冲突记忆并发对同一目标做读-改-写竞态（R1 去重在并发下的放大器）。
// 锁只在 saveConsolidated*/Remember 的短 DB 写段持有，不跨 Recall/LLM 调用，
// 避免长锁阻塞其他用户/群的写入。key 只增不减（仅在有写入时添加），写路径低频、成长有界，可接受。
type keyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newKeyedMutex() *keyedMutex {
	return &keyedMutex{locks: map[string]*sync.Mutex{}}
}

func (k *keyedMutex) Lock(key string) {
	k.mu.Lock()
	l, ok := k.locks[key]
	if !ok {
		l = &sync.Mutex{}
		k.locks[key] = l
	}
	k.mu.Unlock()
	l.Lock()
}

func (k *keyedMutex) Unlock(key string) {
	k.mu.Lock()
	l := k.locks[key]
	k.mu.Unlock()
	if l != nil {
		l.Unlock()
	}
}

// 分身与群记忆各自的写互斥：按 userID / groupID 隔离，互不阻塞。
var (
	avatarWriteMutex = newKeyedMutex()
	groupWriteMutex  = newKeyedMutex()
)