import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/utils/avatar', () => ({
  generateAvatar: (name: string) => `avatar:${name}`,
}))

import MiniAppManager from '@/components/apps/MiniAppManager.vue'
import { useChatStore } from '@/stores/chat'

const requestMock = vi.fn()
const warningMock = vi.fn()
const successMock = vi.fn()

vi.mock('@/composables/useRequest', () => ({
  useRequest: () => ({ request: requestMock }),
}))

vi.mock('@/composables/useChatState', () => ({
  useChatState: () => ({
    $message: {
      warning: warningMock,
      success: successMock,
      error: vi.fn(),
      info: vi.fn(),
    },
  }),
}))

vi.mock('@/composables/useCurrentUser', () => ({
  useCurrentUser: () => ({
    currentUser: { value: { id: 1, username: 'alice', nickname: 'Alice', avatar: '' } },
  }),
}))

describe('MiniAppManager', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    requestMock.mockReset()
    warningMock.mockReset()
    successMock.mockReset()
  })

  it('renders panel fallback icons from each mini app name', async () => {
    requestMock.mockResolvedValueOnce({
      code: 0,
      data: [
        { id: 1, name: '审批助手', icon: '', path: '/mini/approval', description: '处理审批' },
        { id: 2, name: '项目日报', icon: '', path: '/mini/daily', description: '同步项目' },
      ],
    })

    const wrapper = mount(MiniAppManager, {
      props: { showMiniAppList: false },
      global: {
        stubs: { MiniAppDrawer: true },
      },
    })

    await wrapper.setProps({ showMiniAppList: true })
    await Promise.resolve()
    await wrapper.vm.$nextTick()

    const icons = wrapper.findAll('.mini-app-item-icon img').map(img => img.attributes('src'))
    expect(icons).toEqual(['avatar:审批助手', 'avatar:项目日报'])
  })

  it('does not store the generated default icon in sent mini app payloads', async () => {
    requestMock
      .mockResolvedValueOnce({
        code: 0,
        data: [{ id: 1, name: '审批助手', icon: '', path: '/mini/approval', description: '处理审批' }],
      })
      .mockResolvedValueOnce({
        code: 0,
        data: {
          id: 101,
          content: JSON.stringify({ id: 1, name: '审批助手', path: '/mini/approval', description: '处理审批' }),
          sender: { id: 1, username: 'alice', nickname: 'Alice', avatar: '' },
          type: 'miniApp',
        },
      })

    const chatStore = useChatStore()
    chatStore.currentConversationId = 'conversation-1'

    const wrapper = mount(MiniAppManager, {
      props: { showMiniAppList: false },
      global: {
        stubs: { MiniAppDrawer: true },
      },
    })

    await wrapper.setProps({ showMiniAppList: true })
    await Promise.resolve()
    await wrapper.vm.$nextTick()
    await wrapper.get('.mini-app-action-btn').trigger('click')
    await Promise.resolve()

    const sendCall = requestMock.mock.calls[1]
    const sentBody = JSON.parse(sendCall[1].body)
    const sentMiniApp = JSON.parse(sentBody.content)

    expect(sentMiniApp.name).toBe('审批助手')
    expect(sentMiniApp.icon).toBe('')
  })
})
