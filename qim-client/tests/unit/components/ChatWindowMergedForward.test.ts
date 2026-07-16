import { shallowMount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it } from 'vitest'
import ChatWindow from '@/components/chat/ChatWindow.vue'

const makeMessage = (id: string, content: string) => ({
  id,
  content,
  type: 'text' as const,
  sender: { id: 'sender', name: '发送者', avatar: '' },
  timestamp: Number(id),
  isSelf: false,
  conversationId: 'conversation-1',
})

const makeChatWindowProps = (overrides: Record<string, unknown> = {}) => ({
  conversation: {
    id: 'conversation-1', name: '测试会话', avatar: '', unread_count: 0, timestamp: 0, type: 'single' as const,
  },
  messages: [],
  currentUser: { id: 'current-user', name: '当前用户' },
  hasMoreMessages: false,
  ...overrides,
})

describe('ChatWindow merged forwarding', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('emits selected messages in list order when forwarding', async () => {
    const wrapper = shallowMount(ChatWindow, {
      props: makeChatWindowProps({
        messages: [makeMessage('1', '第一条'), makeMessage('2', '第二条')],
      }),
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
    const received: any[] = []
    window.addEventListener('forwardMessage', event => received.push((event as CustomEvent).detail), { once: true })

    await (wrapper.vm as any).startMessageSelection()
    await (wrapper.vm as any).toggleMessageSelection('2')
    await (wrapper.vm as any).toggleMessageSelection('1')
    await (wrapper.vm as any).forwardSelectedMessages()

    expect(received[0].messages.map((message: any) => message.id)).toEqual(['1', '2'])
  })

  it('excludes recalled selected messages when forwarding', async () => {
    const wrapper = shallowMount(ChatWindow, {
      props: makeChatWindowProps({
        messages: [
          makeMessage('1', '第一条'),
          { ...makeMessage('2', '已撤回'), isRecalled: true },
          makeMessage('3', '第三条'),
        ],
      }),
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
    const received: any[] = []
    window.addEventListener('forwardMessage', event => received.push((event as CustomEvent).detail), { once: true })

    await (wrapper.vm as any).startMessageSelection()
    await (wrapper.vm as any).toggleMessageSelection('1')
    await (wrapper.vm as any).toggleMessageSelection('2')
    await (wrapper.vm as any).toggleMessageSelection('3')
    await (wrapper.vm as any).forwardSelectedMessages()

    expect(received[0].messages.map((message: any) => message.id)).toEqual(['1', '3'])
  })

  it('keeps a labelled selection toolbar with primary and secondary actions', () => {
    const source = readFileSync(resolve(__dirname, '../../../src/components/chat/ChatWindow.vue'), 'utf8')

    expect(source).toContain('aria-label="多选消息操作"')
    expect(source).toContain('class="message-selection-cancel"')
    expect(source).toContain('class="message-selection-forward"')
    expect(source).toContain('@media (max-width: 640px)')
  })
})
