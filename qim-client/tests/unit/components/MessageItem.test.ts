import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import MessageItem from '@/components/message/MessageItem.vue'
import MergedForwardMessage from '@/components/message/MergedForwardMessage.vue'

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('MessageItem mention emphasis', () => {
  it('keeps the @ mention state without rendering a duplicate sender badge', () => {
    const wrapper = mount(MessageItem, {
      props: {
        message: {
          id: 'message-1',
          type: 'text',
          content: '@{mention:42|Alice} 请看一下',
          isAtMention: true,
          sender: { id: '42', name: 'Alice', avatar: '' },
          timestamp: Date.now(),
        },
        isSelf: false,
        isRecalled: false,
        conversationType: 'group',
        readUsersMap: {},
        serverUrl: '',
      },
      global: {
        stubs: {
          Avatar: true,
          TextMessage: true,
          AtMentionBadge: { template: '<span class="at-mention-badge">@</span>' },
        },
      },
    })

    expect(wrapper.classes()).toContain('at-mention')
    expect(wrapper.find('.at-mention-badge').exists()).toBe(false)
  })

  it('passes parsed mini app content to the mini app message card', () => {
    const wrapper = mount(MessageItem, {
      props: {
        message: {
          id: 'message-mini-app',
          type: 'miniApp',
          content: JSON.stringify({ name: '审批助手', icon: '', description: '处理审批' }),
          miniAppData: { name: 'Default', icon: '', description: '默认' },
          sender: { id: '42', name: 'Alice', avatar: '' },
          timestamp: Date.now(),
        },
        isSelf: false,
        isRecalled: false,
        conversationType: 'group',
        readUsersMap: {},
        serverUrl: '',
      },
      global: {
        stubs: {
          Avatar: true,
          MiniAppMessage: {
            props: ['miniAppData'],
            template: '<div class="mini-app-stub">{{ miniAppData.name }}</div>',
          },
        },
      },
    })

    expect(wrapper.find('.mini-app-stub').text()).toBe('审批助手')
  })

  it('passes nested mini app payload data to the mini app message card', () => {
    const wrapper = mount(MessageItem, {
      props: {
        message: {
          id: 'message-mini-app-nested',
          type: 'miniApp',
          content: JSON.stringify({ type: 'miniApp', data: { name: '审批助手', icon: '', description: '处理审批' } }),
          miniAppData: { name: 'Default', icon: '', description: '默认' },
          sender: { id: '42', name: 'Alice', avatar: '' },
          timestamp: Date.now(),
        },
        isSelf: false,
        isRecalled: false,
        conversationType: 'group',
        readUsersMap: {},
        serverUrl: '',
      },
      global: {
        stubs: {
          Avatar: true,
          MiniAppMessage: {
            props: ['miniAppData'],
            template: '<div class="mini-app-stub">{{ miniAppData.name }}</div>',
          },
        },
      },
    })

    expect(wrapper.find('.mini-app-stub').text()).toBe('审批助手')
  })

  it('renders merged forward messages with their content payload', () => {
    const wrapper = mount(MessageItem, {
      props: {
        message: {
          id: 'merged-1',
          type: 'merged_forward',
          content: JSON.stringify({ version: 1, title: '聊天记录', messages: [] }),
          sender: { id: '42', name: 'Alice', avatar: '' },
          timestamp: Date.now(),
        },
        isSelf: false,
        isRecalled: false,
        conversationType: 'group',
        readUsersMap: {},
        serverUrl: '',
      },
      global: {
        stubs: {
          Avatar: true,
          MergedForwardMessage: {
            props: ['content'],
            template: '<div class="merged-forward-stub">{{ content }}</div>',
          },
        },
      },
    })

    expect(wrapper.find('.merged-forward-stub').text()).toContain('聊天记录')
  })

  it('shows the merged-forward fallback for malformed message items', () => {
    const wrapper = mount(MergedForwardMessage, {
      props: {
        content: JSON.stringify({
          version: 1,
          title: '聊天记录',
          messages: [null],
        }),
      },
    })

    expect(wrapper.text()).toContain('聊天记录无法加载')
  })

  it('renders formatted rich previews instead of serialized message content', async () => {
    const wrapper = mount(MergedForwardMessage, {
      props: {
        content: JSON.stringify({
          version: 1,
          title: '聊天记录',
          messages: [
            { id: 'file-1', senderName: 'Alice', timestamp: 1, type: 'file', content: JSON.stringify({ name: '方案.pdf', size: 1024 }) },
            { id: 'share-1', senderName: 'Bob', timestamp: 2, type: 'share', content: JSON.stringify({ name: '设计说明' }) },
          ],
        }),
      },
    })

    await wrapper.get('[data-testid="merged-forward-toggle"]').trigger('click')

    expect(wrapper.text()).toContain('方案.pdf · 1 KB')
    expect(wrapper.text()).toContain('分享：设计说明')
    expect(wrapper.text()).not.toContain('{"name"')
    expect(wrapper.find('.fa-file').exists()).toBe(true)
  })

  it('marks the merged-forward card as responsive and its toggle as accessible', () => {
    const source = readFileSync(resolve(__dirname, '../../../src/components/message/MergedForwardMessage.vue'), 'utf8')

    expect(source).toContain('class="merged-forward-icon"')
    expect(source).toContain('aria-expanded="expanded"')
    expect(source).toContain('@media (max-width: 640px)')
    expect(source).toContain(':focus-visible')
    expect(source).toContain('width: fit-content;')
    expect(source).toContain('max-width: 100%;')
  })

  it('uses a global high-contrast rule for links inside self message bubbles', () => {
    const source = readFileSync(resolve(__dirname, '../../../src/components/message/MessageItem.vue'), 'utf8')

    expect(source).toContain(':global(.message-item.self .message-link)')
    expect(source).toContain('color: #0f3e91 !important;')
  })
})
