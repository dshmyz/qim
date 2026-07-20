import { shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import ChatWindow from '@/components/chat/ChatWindow.vue'

const makeProps = (conversationId = 'group-1') => ({
  conversation: {
    id: conversationId, name: '测试群组', avatar: '', unread_count: 0, timestamp: 0, type: 'group' as const,
    members: [{ id: 'current-user', role: 'owner' }],
  },
  messages: [],
  currentUser: { id: 'current-user', name: '当前用户' },
  hasMoreMessages: false,
})

describe('ChatWindow group files', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('closes the group files panel when the conversation changes', async () => {
    const wrapper = shallowMount(ChatWindow, {
      props: makeProps(),
      global: {
        stubs: {
          ChatBody: {
            template: '<div />',
            methods: {
              scrollToBottom: () => undefined,
              scrollToBottomWithDelay: () => undefined,
            },
          },
          AISummaryPanel: true,
          AITranslatePanel: true,
          GroupModals: true,
        },
      },
    })

    ;(wrapper.vm as any).openGroupFiles(18, 24)
    await wrapper.vm.$nextTick()
    expect(wrapper.findComponent({ name: 'GroupFilesPanel' }).exists()).toBe(true)

    await wrapper.setProps(makeProps('group-2'))

    expect(wrapper.findComponent({ name: 'GroupFilesPanel' }).exists()).toBe(false)
  })
})
