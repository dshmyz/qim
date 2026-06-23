import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DRAFT_CHANGED_EVENT, saveDraft } from '@/utils/drafts'

const storage = new Map<string, string>()

beforeEach(() => {
  vi.mocked(localStorage.getItem).mockImplementation((key) => storage.get(key) ?? null)
  vi.mocked(localStorage.setItem).mockImplementation((key, value) => {
    storage.set(key, value)
  })
  vi.mocked(localStorage.removeItem).mockImplementation((key) => {
    storage.delete(key)
  })
  vi.mocked(localStorage.clear).mockImplementation(() => {
    storage.clear()
  })
})

afterEach(() => {
  localStorage.clear()
  vi.clearAllMocks()
})

describe('saveDraft', () => {
  it('removes an empty draft and notifies same-page listeners', () => {
    const conversationId = 'group-1'
    localStorage.setItem(`qim_draft_${conversationId}`, JSON.stringify({ text: '@alice', quoted: null }))
    const changedIds: string[] = []
    const listener = (event: Event) => {
      changedIds.push((event as CustomEvent<{ conversationId: string }>).detail.conversationId)
    }
    window.addEventListener(DRAFT_CHANGED_EVENT, listener)

    saveDraft(conversationId, '', null)

    expect(localStorage.getItem(`qim_draft_${conversationId}`)).toBeNull()
    expect(changedIds).toEqual([conversationId])
    window.removeEventListener(DRAFT_CHANGED_EVENT, listener)
  })

  it('persists non-empty text', () => {
    saveDraft('group-1', '@alice 请看', { id: 'quoted-1' })

    expect(JSON.parse(localStorage.getItem('qim_draft_group-1') || '')).toEqual({
      text: '@alice 请看',
      quoted: { id: 'quoted-1' },
    })
  })
})
