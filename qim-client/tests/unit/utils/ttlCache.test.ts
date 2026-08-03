import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createTTLCache } from '@/utils/ttlCache'

describe('createTTLCache', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('基本读写', () => {
    it('未命中时返回 undefined', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      expect(cache.get('missing')).toBeUndefined()
    })

    it('写入后可读取', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      expect(cache.get('k1')).toBe('v1')
    })

    it('has() 反映条目存在性', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      expect(cache.has('k1')).toBe(false)
      cache.set('k1', 'v1')
      expect(cache.has('k1')).toBe(true)
    })
  })

  describe('TTL 过期', () => {
    it('未过期时命中缓存', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(59999)
      expect(cache.get('k1')).toBe('v1')
    })

    it('过期后返回 undefined', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(60001)
      expect(cache.get('k1')).toBeUndefined()
    })

    it('过期后 has() 返回 false', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(60001)
      expect(cache.has('k1')).toBe(false)
    })

    it('不同条目可使用不同 TTL（set 时覆盖默认）', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('short', 'v1', 10000)
      cache.set('long', 'v2', 60000)
      vi.advanceTimersByTime(15000)
      expect(cache.get('short')).toBeUndefined()
      expect(cache.get('long')).toBe('v2')
    })
  })

  describe('容量上限', () => {
    it('超过 maxSize 时淘汰最旧条目', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 2 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(1)
      cache.set('k2', 'v2')
      vi.advanceTimersByTime(1)
      cache.set('k3', 'v3') // 触发淘汰 k1

      expect(cache.size()).toBe(2)
      expect(cache.get('k1')).toBeUndefined()
      expect(cache.get('k2')).toBe('v2')
      expect(cache.get('k3')).toBe('v3')
    })

    it('淘汰按写入时间顺序，而非访问顺序', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 2 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(1)
      cache.set('k2', 'v2')
      vi.advanceTimersByTime(1)
      // 读取 k1 不会重置其淘汰优先级
      cache.get('k1')
      vi.advanceTimersByTime(1)
      cache.set('k3', 'v3')

      expect(cache.get('k1')).toBeUndefined()
      expect(cache.get('k2')).toBe('v2')
    })
  })

  describe('删除与清空', () => {
    it('delete() 删除指定条目', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      expect(cache.delete('k1')).toBe(true)
      expect(cache.get('k1')).toBeUndefined()
    })

    it('delete() 删除不存在的 key 返回 false', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      expect(cache.delete('missing')).toBe(false)
    })

    it('clear() 清空所有条目', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      cache.set('k2', 'v2')
      cache.clear()
      expect(cache.size()).toBe(0)
    })

    it('clear(pattern) 按正则批量删除', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('task_1_conv2', 'v1')
      cache.set('task_2_conv2', 'v2')
      cache.set('task_3_conv3', 'v3')
      cache.set('other_key', 'v4')

      cache.clear(/task_\d+_conv2/)
      expect(cache.has('task_1_conv2')).toBe(false)
      expect(cache.has('task_2_conv2')).toBe(false)
      expect(cache.has('task_3_conv3')).toBe(true)
      expect(cache.has('other_key')).toBe(true)
    })
  })

  describe('size()', () => {
    it('返回当前有效条目数', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      expect(cache.size()).toBe(0)
      cache.set('k1', 'v1')
      cache.set('k2', 'v2')
      expect(cache.size()).toBe(2)
    })

    it('过期但未清理的条目不计入', () => {
      const cache = createTTLCache<string>({ ttl: 60000, maxSize: 10 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(60001)
      expect(cache.size()).toBe(0)
    })
  })

  describe('默认参数', () => {
    it('未传 ttl 时使用默认 5 分钟', () => {
      const cache = createTTLCache<string>({ maxSize: 10 })
      cache.set('k1', 'v1')
      vi.advanceTimersByTime(4 * 60 * 1000 + 59 * 1000)
      expect(cache.get('k1')).toBe('v1')
      vi.advanceTimersByTime(2000)
      expect(cache.get('k1')).toBeUndefined()
    })

    it('未传 maxSize 时使用默认 100', () => {
      const cache = createTTLCache<string>({ ttl: 60000 })
      for (let i = 0; i < 100; i++) {
        cache.set(`k${i}`, `v${i}`)
        vi.advanceTimersByTime(1)
      }
      expect(cache.size()).toBe(100)
      cache.set('k100', 'v100')
      expect(cache.size()).toBe(100)
      expect(cache.get('k0')).toBeUndefined()
    })
  })

  describe('泛型支持', () => {
    it('支持对象类型', () => {
      interface Task { id: number; title: string }
      const cache = createTTLCache<Task>({ ttl: 60000, maxSize: 10 })
      cache.set('task_1', { id: 1, title: '测试任务' })
      const result = cache.get('task_1')
      expect(result).toEqual({ id: 1, title: '测试任务' })
    })
  })
})
