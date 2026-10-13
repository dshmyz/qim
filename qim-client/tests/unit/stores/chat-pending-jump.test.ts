import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatStore } from '../../../src/stores/chat'

describe('chat store pendingJump', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('setPendingJump 后同会话 consumePendingJump 返回 messageId 并清空', () => {
    const store = useChatStore()
    store.setPendingJump(42, 1001)
    expect(store.pendingJump).toEqual({ conversationId: '42', messageId: '1001' })
    expect(store.consumePendingJump(42)).toBe('1001')
    expect(store.pendingJump).toBeNull()
  })

  it('会话不匹配时返回 null 且目标保留', () => {
    const store = useChatStore()
    store.setPendingJump(42, 1001)
    expect(store.consumePendingJump(99)).toBeNull()
    expect(store.pendingJump).toEqual({ conversationId: '42', messageId: '1001' })
  })

  it('消费后再 consume 返回 null（一次性）', () => {
    const store = useChatStore()
    store.setPendingJump(42, 1001)
    store.consumePendingJump(42)
    expect(store.consumePendingJump(42)).toBeNull()
  })

  it('数字与字符串 ID 等价匹配', () => {
    const store = useChatStore()
    store.setPendingJump('42', '1001')
    expect(store.consumePendingJump(42)).toBe('1001')
  })

  it('未设置时 consume 返回 null', () => {
    const store = useChatStore()
    expect(store.consumePendingJump(42)).toBeNull()
  })
})
