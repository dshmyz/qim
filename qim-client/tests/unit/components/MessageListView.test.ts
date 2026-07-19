import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Viewer from 'viewerjs'
import MessageListView from '@/components/chat/MessageListView.vue'

const viewerUpdate = vi.fn()
const viewerDestroy = vi.fn()

vi.mock('viewerjs', () => ({
  default: vi.fn().mockImplementation(function () {
    this.update = viewerUpdate
    this.destroy = viewerDestroy
  }),
}))

const baseProps = {
  messages: [],
  hasMoreMessages: false,
  conversationType: 'single',
  readUsersMap: {},
  showReadReceipt: true,
  serverUrl: 'http://localhost:8080',
}

beforeEach(() => {
  viewerUpdate.mockClear()
  viewerDestroy.mockClear()
})

describe('MessageListView image viewer', () => {
  it('does not show a selection control for system messages', () => {
    const wrapper = mount(MessageListView, {
      props: {
        ...baseProps,
        selectionMode: true,
        selectedMessageIds: new Set<string>(),
        messages: [
          { id: 'system-1', type: 'system', content: '用户加入群聊', isSelf: false, isRecalled: false, timestamp: 1 },
          { id: 'text-1', type: 'text', content: '普通消息', isSelf: false, isRecalled: false, timestamp: 2 },
        ],
      },
      global: {
        stubs: {
          MessageItem: {
            template: '<article class="message-item-stub"><slot name="selection-control" /></article>',
          },
        },
      },
    })

    expect(wrapper.find('[data-testid="message-select-system-1"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="message-select-text-1"]').exists()).toBe(true)
  })

  it('places the selection control in the message item and highlights selected messages', () => {
    const wrapper = mount(MessageListView, {
      props: {
        ...baseProps,
        selectionMode: true,
        selectedMessageIds: new Set(['text-1']),
        messages: [
          { id: 'text-1', type: 'text', content: '普通消息', isSelf: false, isRecalled: false, timestamp: 1 },
        ],
      },
      global: {
        stubs: {
          MessageItem: {
            template: '<article class="message-item-stub"><slot name="selection-control" /></article>',
          },
        },
      },
    })

    const item = wrapper.get('.message-item-stub')
    expect(item.classes()).toContain('message-selection-active')
    expect(item.get('[data-testid="message-select-text-1"]').exists()).toBe(true)
  })

  it('marks the message list as the Viewer.js gallery root', () => {
    const wrapper = mount(MessageListView, {
      props: baseProps,
      global: {
        stubs: {
          MessageItem: true,
        },
      },
    })

    const list = wrapper.get('.message-list')
    expect(list.attributes('data-viewer-gallery')).toBe('')
  })

  it('initializes the viewer without transition animation', () => {
    mount(MessageListView, {
      props: baseProps,
      global: {
        stubs: {
          MessageItem: true,
        },
      },
    })

    expect(Viewer).toHaveBeenCalled()
    const options = vi.mocked(Viewer).mock.calls[0][1]
    expect(options?.transition).toBe(false)
  })

  it('refreshes the viewer gallery after a message image finishes loading', async () => {
    const wrapper = mount(MessageListView, {
      props: {
        ...baseProps,
        messages: [
          { id: 'm1', type: 'image', content: '/uploads/a.png', isSelf: false, isRecalled: false, timestamp: 1 },
        ],
      },
      global: {
        stubs: {
          MessageItem: {
            template: '<button class="message-item-stub" @click="$emit(\'image-loaded\')" />',
            emits: ['image-loaded'],
          },
        },
      },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 20))
    viewerUpdate.mockClear()
    await wrapper.get('.message-item-stub').trigger('click')
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 20))

    expect(viewerUpdate).toHaveBeenCalled()
  })
})

describe('MessageListView self message detection', () => {
  it('marks messages from the current user as self even when isSelf is missing', () => {
    const wrapper = mount(MessageListView, {
      props: {
        ...baseProps,
        currentUserId: 7,
        messages: [
          {
            id: 'm1',
            type: 'text',
            content: 'hello',
            timestamp: 1,
            sender: { id: '7', name: 'me' },
          },
        ],
      },
      global: {
        stubs: {
          MessageItem: true,
        },
      },
    })

    const item = wrapper.findComponent({ name: 'MessageItem' })
    expect(item.props('isSelf')).toBe(true)
  })

  it('falls back to sender_id for self message detection', () => {
    const wrapper = mount(MessageListView, {
      props: {
        ...baseProps,
        currentUserId: '9',
        messages: [
          {
            id: 'm2',
            type: 'text',
            content: 'hello',
            timestamp: 1,
            sender_id: 9,
            sender: { name: 'me' },
          },
        ],
      },
      global: {
        stubs: {
          MessageItem: true,
        },
      },
    })

    const item = wrapper.findComponent({ name: 'MessageItem' })
    expect(item.props('isSelf')).toBe(true)
  })
})
