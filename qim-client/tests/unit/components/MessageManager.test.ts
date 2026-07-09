import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import MessageManager from '@/components/chat/MessageManager.vue'
import { messageApi } from '@/api/message'
import QMessage from '@/utils/qmessage'

vi.mock('@/api/message', () => ({
  messageApi: {
    getMessagesByFilter: vi.fn(),
  },
}))

vi.mock('@/utils/qmessage', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
  },
}))

vi.mock('@/composables/useServerUrl', () => ({
  getStoredServerUrl: () => 'http://localhost:3000',
}))

vi.mock('@/utils/mentions', () => ({
  parseContent: (content: string) => ({ text: content, mentions: [] }),
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

const mockMessages = (messages: any[], total = messages.length) => {
  ;(messageApi.getMessagesByFilter as any).mockResolvedValue({
    messages,
    total,
  })
}

describe('MessageManager', () => {
  describe('Markdown 消息渲染', () => {
    it('markdown 类型消息应显示渲染后的内容', async () => {
      mockMessages([
        {
          id: 1,
          type: 'markdown',
          content: '# 标题\n**粗体**文本',
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const content = wrapper.find('.message-manager-item-content')
      expect(content.exists()).toBe(true)
      expect(content.find('.markdown-renderer').exists()).toBe(true)
    })
  })

  describe('jumpToPage 同步', () => {
    it('翻页后跳转输入框应同步显示当前页码', async () => {
      const total = 50
      mockMessages(
        [
          {
            id: 21,
            type: 'text',
            content: 'page 2 msg 1',
            created_at: '2024-01-02T00:00:00Z',
            is_recalled: false,
            sender: { id: 1, name: 'Alice', avatar: '' },
          },
        ],
        total,
      )

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const pageInput = wrapper.find('.page-input')
      expect(pageInput.exists()).toBe(true)
      expect((pageInput.element as HTMLInputElement).value).toBe('1')

      const nextBtn = wrapper.findAll('.pagination-btn').at(-1)
      expect(nextBtn?.exists()).toBe(true)
      await nextBtn!.trigger('click')
      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalledTimes(2)
      })
      await wrapper.vm.$nextTick()

      expect((pageInput.element as HTMLInputElement).value).toBe('2')
    })
  })

  describe('错误处理', () => {
    it('加载失败时应显示错误提示并清空列表', async () => {
      const testError = new Error('Network error')
      ;(messageApi.getMessagesByFilter as any).mockRejectedValue(testError)

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      expect(QMessage.error).toHaveBeenCalledWith(
        expect.stringContaining('Network error')
      )
      expect(wrapper.find('.empty-message').exists()).toBe(true)
    })

    it('axios 风格错误（非 Error 实例）也应显示提示', async () => {
      const axiosError = {
        response: { data: { message: '服务器错误' } },
        message: 'Request failed',
      }
      ;(messageApi.getMessagesByFilter as any).mockRejectedValue(axiosError)

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      expect(QMessage.error).toHaveBeenCalled()
      const callArg = (QMessage.error as any).mock.calls[0][0]
      expect(typeof callArg).toBe('string')
      expect(callArg.length).toBeGreaterThan(0)
    })
  })

  describe('关闭后状态重置', () => {
    it('关闭再打开时列表应重新加载', async () => {
      mockMessages([
        {
          id: 1,
          type: 'text',
          content: 'msg 1',
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      expect(wrapper.findAll('.message-manager-item').length).toBe(1)

      await wrapper.setProps({ visible: false })
      await wrapper.vm.$nextTick()

      mockMessages([
        {
          id: 2,
          type: 'text',
          content: 'new msg',
          created_at: '2024-01-02T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      await wrapper.setProps({ visible: true })
      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalledTimes(2)
      })
      await wrapper.vm.$nextTick()

      const items = wrapper.findAll('.message-manager-item')
      expect(items.length).toBe(1)
      expect(items[0].text()).toContain('new msg')
    })

    it('关闭再打开时搜索和筛选条件应重置', async () => {
      mockMessages([
        {
          id: 1,
          type: 'text',
          content: 'hello',
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const firstCallParams = (messageApi.getMessagesByFilter as any).mock.calls[0][0]
      expect(firstCallParams.search).toBeUndefined()
      expect(firstCallParams.type).toBeUndefined()

      const searchInput = wrapper.find('.search-input')
      await searchInput.setValue('test keyword')
      await searchInput.trigger('keyup.enter')
      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalledTimes(2)
      })
      await wrapper.vm.$nextTick()

      const searchCallParams = (messageApi.getMessagesByFilter as any).mock.calls[1][0]
      expect(searchCallParams.search).toBe('test keyword')

      await wrapper.setProps({ visible: false })
      await wrapper.vm.$nextTick()

      await wrapper.setProps({ visible: true })
      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalledTimes(3)
      })
      await wrapper.vm.$nextTick()

      const reopenCallParams = (messageApi.getMessagesByFilter as any).mock.calls[2][0]
      expect(reopenCallParams.search).toBeUndefined()
      expect(reopenCallParams.type).toBeUndefined()
    })
  })

  describe('自定义日期范围校验', () => {
    it('只填一个日期时应提示用户填写完整', async () => {
      mockMessages([])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const dateSelect = wrapper.findAll('.filter-select').at(1)
      await dateSelect!.setValue('custom')
      await wrapper.vm.$nextTick()

      const startInput = wrapper.findAll('.date-input').at(0)
      expect(startInput?.exists()).toBe(true)
      await startInput!.setValue('2024-01-01')
      await startInput!.trigger('change')
      await wrapper.vm.$nextTick()

      expect(QMessage.warning).toHaveBeenCalledWith(
        expect.stringContaining('日期范围')
      )
    })
  })

  describe('重复加载', () => {
    it('组件挂载且 visible=true 时只加载一次', async () => {
      mockMessages([])

      mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await new Promise(resolve => setTimeout(resolve, 100))

      expect(messageApi.getMessagesByFilter).toHaveBeenCalledTimes(1)
    })
  })

  describe('消息排序', () => {
    it('消息列表应按时间倒序排列（最新的在最上面）', async () => {
      mockMessages([
        {
          id: 3,
          type: 'text',
          content: '最新的消息',
          created_at: '2024-01-03T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
        {
          id: 2,
          type: 'text',
          content: '中间的消息',
          created_at: '2024-01-02T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
        {
          id: 1,
          type: 'text',
          content: '最早的消息',
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: true,
          },
        },
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const items = wrapper.findAll('.message-manager-item')
      expect(items.length).toBe(3)
      expect(items[0].text()).toContain('最新的消息')
      expect(items[1].text()).toContain('中间的消息')
      expect(items[2].text()).toContain('最早的消息')
    })
  })

  describe('媒体菜单边界检测', () => {
    it('菜单靠近右边缘时应向左偏移避免溢出', async () => {
      mockMessages([
        {
          id: 1,
          type: 'image',
          content: JSON.stringify({ url: '/test.jpg', name: 'test.jpg' }),
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: { template: '<div><slot /></div>' },
          },
        },
        attachTo: document.body,
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const fileLink = wrapper.find('.message-file-link')
      expect(fileLink.exists()).toBe(true)

      Object.defineProperty(window, 'innerWidth', { value: 800, writable: true })
      Object.defineProperty(window, 'innerHeight', { value: 600, writable: true })

      await fileLink.trigger('click', {
        clientX: 780,
        clientY: 300,
      })
      await wrapper.vm.$nextTick()

      const menu = wrapper.find('.media-action-menu')
      expect(menu.exists()).toBe(true)
      const left = parseInt(menu.attributes('style')?.match(/left:\s*(\d+)px/)?.[1] || '0')
      expect(left).toBeLessThanOrEqual(652)
      expect(left).toBeGreaterThan(0)

      wrapper.unmount()
    })

    it('菜单靠近下边缘时应向上偏移避免溢出', async () => {
      mockMessages([
        {
          id: 1,
          type: 'file',
          content: JSON.stringify({ url: '/test.pdf', name: 'test.pdf' }),
          created_at: '2024-01-01T00:00:00Z',
          is_recalled: false,
          sender: { id: 1, name: 'Alice', avatar: '' },
        },
      ])

      const wrapper = mount(MessageManager, {
        props: {
          visible: true,
          conversationId: '1',
        },
        global: {
          stubs: {
            Teleport: { template: '<div><slot /></div>' },
          },
        },
        attachTo: document.body,
      })

      await vi.waitFor(() => {
        expect(messageApi.getMessagesByFilter).toHaveBeenCalled()
      })
      await wrapper.vm.$nextTick()

      const fileLink = wrapper.find('.message-file-link')
      expect(fileLink.exists()).toBe(true)

      Object.defineProperty(window, 'innerWidth', { value: 800, writable: true })
      Object.defineProperty(window, 'innerHeight', { value: 600, writable: true })

      await fileLink.trigger('click', {
        clientX: 100,
        clientY: 580,
      })
      await wrapper.vm.$nextTick()

      const menu = wrapper.find('.media-action-menu')
      expect(menu.exists()).toBe(true)
      const top = parseInt(menu.attributes('style')?.match(/top:\s*(\d+)px/)?.[1] || '0')
      expect(top).toBeLessThanOrEqual(512)
      expect(top).toBeGreaterThan(0)

      wrapper.unmount()
    })
  })
})
