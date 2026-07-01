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
  serverUrl: 'http://localhost:8080',
}

beforeEach(() => {
  viewerUpdate.mockClear()
  viewerDestroy.mockClear()
})

describe('MessageListView image viewer', () => {
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
