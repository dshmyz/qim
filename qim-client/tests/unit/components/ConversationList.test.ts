import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ConversationList from '@/components/conversation/ConversationList.vue'
import { DRAFT_CHANGED_EVENT } from '@/utils/drafts'

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

describe('ConversationList drafts', () => {
  it('refreshes a draft in the same page when it is cleared', async () => {
    const conversationId = 'group-1'
    storage.set(`qim_draft_${conversationId}`, JSON.stringify({ text: '@alice', quoted: null }))
    const wrapper = mount(ConversationList, {
      props: {
        conversations: [{
          id: conversationId,
          name: '项目组',
          type: 'group',
          lastMessage: { type: 'text', content: '最后一条消息' },
        }],
        currentConversationId: null,
        serverUrl: '',
      },
      global: {
        stubs: { Avatar: true },
      },
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('[草稿] @alice')

    storage.delete(`qim_draft_${conversationId}`)
    window.dispatchEvent(new CustomEvent(DRAFT_CHANGED_EVENT, { detail: { conversationId } }))
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).not.toContain('[草稿]')
    expect(wrapper.text()).toContain('最后一条消息')
  })
})
