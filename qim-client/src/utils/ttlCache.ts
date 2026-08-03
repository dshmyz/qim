/**
 * 轻量 TTL 缓存
 *
 * 用于模块级共享缓存的场景：
 * - 读写命中快（Map）
 * - 每个条目带过期时间戳，get 时惰性淘汰
 * - 容量超限时按写入时间淘汰最旧条目
 * - 支持 clear(pattern) 按正则批量清除
 *
 * 设计原则：
 * - 不依赖事件总线或 store，纯内存
 * - 淘汰策略简单稳定（按写入时间，非访问顺序），避免 LRU 的复杂度
 * - get 时不刷新 timestamp，保证 TTL 语义可预期
 */

interface CacheEntry<T> {
  data: T
  timestamp: number
  expiresIn: number
}

export interface TTLCacheOptions {
  /** 默认过期时间（毫秒），默认 5 分钟 */
  ttl?: number
  /** 最大条目数，默认 100，超过时淘汰最旧条目 */
  maxSize?: number
}

export interface TTLCache<T> {
  /** 读取，未命中或已过期返回 undefined */
  get: (key: string) => T | undefined
  /** 写入，可针对单条覆盖 ttl */
  set: (key: string, data: T, ttl?: number) => void
  /** 判断是否存在且未过期 */
  has: (key: string) => boolean
  /** 删除指定 key，返回是否删除成功 */
  delete: (key: string) => boolean
  /** 清空全部，或按正则匹配 key 批量清除 */
  clear: (pattern?: RegExp) => void
  /** 返回当前未过期条目数 */
  size: () => number
}

const DEFAULT_TTL = 5 * 60 * 1000
const DEFAULT_MAX_SIZE = 100

export function createTTLCache<T>(options?: TTLCacheOptions): TTLCache<T> {
  const ttl = options?.ttl ?? DEFAULT_TTL
  const maxSize = options?.maxSize ?? DEFAULT_MAX_SIZE
  const store = new Map<string, CacheEntry<T>>()

  const isExpired = (entry: CacheEntry<T>): boolean =>
    Date.now() - entry.timestamp > entry.expiresIn

  // 淘汰最旧条目，直到条目数 <= maxSize
  const evictIfNeeded = (): void => {
    if (store.size <= maxSize) return
    const entries = Array.from(store.entries()).sort(
      (a, b) => a[1].timestamp - b[1].timestamp
    )
    const toDelete = entries.slice(0, entries.length - maxSize)
    for (const [key] of toDelete) {
      store.delete(key)
    }
  }

  return {
    get(key: string): T | undefined {
      const entry = store.get(key)
      if (!entry) return undefined
      if (isExpired(entry)) {
        store.delete(key)
        return undefined
      }
      return entry.data
    },

    set(key: string, data: T, entryTtl?: number): void {
      store.set(key, {
        data,
        timestamp: Date.now(),
        expiresIn: entryTtl ?? ttl,
      })
      evictIfNeeded()
    },

    has(key: string): boolean {
      const entry = store.get(key)
      if (!entry) return false
      if (isExpired(entry)) {
        store.delete(key)
        return false
      }
      return true
    },

    delete(key: string): boolean {
      return store.delete(key)
    },

    clear(pattern?: RegExp): void {
      if (!pattern) {
        store.clear()
        return
      }
      for (const key of store.keys()) {
        if (pattern.test(key)) {
          store.delete(key)
        }
      }
    },

    size(): number {
      // 惰性清理已过期条目
      let count = 0
      for (const [, entry] of store) {
        if (!isExpired(entry)) count++
      }
      return count
    },
  }
}
