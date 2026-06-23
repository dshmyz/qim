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

  it('uses a global high-contrast rule for links inside self message bubbles', () => {
    const source = readFileSync(resolve(__dirname, '../../../src/components/message/MessageItem.vue'), 'utf8')

    expect(source).toContain(':global(.message-item.self .message-link)')
    expect(source).toContain('color: #0f3e91 !important;')
  })
})
