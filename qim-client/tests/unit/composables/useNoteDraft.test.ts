import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useNoteDraft } from '@/composables/useNoteDraft'

// 使用真实内存 localStorage 替代全局 mock（全局 mock getItem 固定返回 null）
const store = new Map<string, string>()
const realLocalStorage: Storage = {
  getItem: vi.fn((key: string) => store.get(key) ?? null),
  setItem: vi.fn((key: string, value: string) => { store.set(key, value) }),
  removeItem: vi.fn((key: string) => { store.delete(key) }),
  clear: vi.fn(() => { store.clear() }),
  get length() { return store.size },
  key: vi.fn((index: number) => [...store.keys()][index] ?? null),
}

Object.defineProperty(globalThis, 'localStorage', { value: realLocalStorage, writable: true })

describe('useNoteDraft', () => {
  const draft = useNoteDraft()

  beforeEach(() => {
    store.clear()
    vi.clearAllMocks()
  })

  it('save 后 load 可读回数据（含自动写入的 savedAt 时间戳）', () => {
    draft.save(1, { title: '标题', content: '内容', tags: ['a'] })
    const result = draft.load(1)
    expect(result).toEqual(expect.objectContaining({ title: '标题', content: '内容', tags: ['a'] }))
    expect(typeof result?.savedAt).toBe('number')
    expect(result!.savedAt!).toBeLessThanOrEqual(Date.now())
  })

  it('load 不存在的 id 返回 null', () => {
    expect(draft.load(999)).toBeNull()
  })

  it('clear 删除指定笔记的草稿', () => {
    draft.save(1, { title: 't', content: 'c', tags: [] })
    draft.save(2, { title: 't2', content: 'c2', tags: [] })
    draft.clear(1)
    expect(draft.load(1)).toBeNull()
    expect(draft.load(2)).not.toBeNull()
  })

  it('不同笔记 id 互不干扰', () => {
    draft.save(10, { title: 'a', content: 'b', tags: [] })
    draft.save(20, { title: 'c', content: 'd', tags: ['x'] })
    expect(draft.load(10)).toEqual(expect.objectContaining({ title: 'a', content: 'b', tags: [] }))
    expect(draft.load(20)).toEqual(expect.objectContaining({ title: 'c', content: 'd', tags: ['x'] }))
  })

  it('保存顺序不同时 savedAt 单调递增', () => {
    draft.save(1, { title: '旧', content: 'c', tags: [] })
    const first = draft.load(1)!.savedAt!
    draft.save(1, { title: '新', content: 'c', tags: [] })
    const second = draft.load(1)!.savedAt!
    expect(second).toBeGreaterThanOrEqual(first)
  })

  it('旧格式草稿（无 savedAt）load 后不抛异常', () => {
    store.set('qim_note_draft_1', JSON.stringify({ title: 't', content: 'c', tags: [] }))
    const result = draft.load(1)
    expect(result).toEqual({ title: 't', content: 'c', tags: [] })
    expect(result?.savedAt).toBeUndefined()
  })

  it('localStorage 损坏时 load 返回 null 不抛异常', () => {
    store.set('qim_note_draft_1', '{invalid json')
    expect(draft.load(1)).toBeNull()
  })
})
