package service

import "sync"

// keyedMutex 按 key 粒度的互斥锁：保护同一 key（同一个用户/群）的记忆写路径，
// 防止多条近义/冲突记忆并发对同一目标做读-改-写竞态（R1 去重在并发下的放大器）。
// 锁覆盖 saveConsolidated*/Remember 的完整写段（含 mergeCheck 的 LLM 判定 + DB 写），
// 确保判定→写入是原子的：否则并发下两路都判完同一旧记忆、各自写入会互相覆盖。
// 代价是同一 key 的写路径在 LLM 调用期间串行；写路径本身异步（goroutine）、低频
// （仅通过三层门控的消息才触发），不影响其他用户/群。key 只增不减（仅在有写入时
// 添加），写路径低频、成长有界，可接受。
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
