import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import MessageItem from '@/components/message/MessageItem.vue'

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

  it('uses a global high-contrast rule for links inside self message bubbles', () => {
    const source = readFileSync(resolve(__dirname, '../../../src/components/message/MessageItem.vue'), 'utf8')

    expect(source).toContain(':global(.message-item.self .message-link)')
    expect(source).toContain('color: #0f3e91 !important;')
  })
})
