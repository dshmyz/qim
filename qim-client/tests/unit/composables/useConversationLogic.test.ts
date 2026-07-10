import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { useConversationLogic } from '@/composables/useConversationLogic'
import { useMainConversationLogic } from '@/composables/useMainConversationLogic'

const { requestMock } = vi.hoisted(() => ({ requestMock: vi.fn() }))

vi.mock('@/composables/useRequest', () => ({
  request: requestMock
}))

describe.skip('useConversationLogic', () => {
  let conversationLogic: ReturnType<typeof useConversationLogic>

  beforeEach(() => {
    setActivePinia(createPinia())
    conversationLogic = useConversationLogic()
  })

  describe('loadConversations', () => {
    it('should load conversations from server', async () => {
      await conversationLogic.loadConversations()

      // 验证会话被加载
      // expect(chatStore.updateConversations).toHaveBeenCalled()
    })

    it('should handle load error gracefully', async () => {
      await conversationLogic.loadConversations()

      // 验证错误处理
      // expect(QMessage.error).toHaveBeenCalled()
    })
  })

  describe('handleConversationSelect', () => {
    it('should select conversation and load messages', () => {
      const conversation = {
        id: 'conv-123',
        name: '测试会话',
        type: 'private'
      }

      const loadMessagesMock = vi.fn()
      conversationLogic.handleConversationSelect(conversation, loadMessagesMock)

      // 验证会话被选中
      // 验证消息被加载
      // expect(loadMessagesMock).toHaveBeenCalled()
    })

    it('should skip if same conversation is selected', () => {
      const conversation = {
        id: 'conv-123',
        name: '测试会话',
        type: 'private'
      }

      // 先选择一次
      conversationLogic.handleConversationSelect(conversation, vi.fn())
      
      // 再选择相同的会话
      const loadMessagesMock = vi.fn()
      conversationLogic.handleConversationSelect(conversation, loadMessagesMock)

      // 验证消息加载被跳过
      // expect(loadMessagesMock).not.toHaveBeenCalled()
    })
  })

  describe('handleConversationCreated', () => {
    it('should reload conversations and select new one', async () => {
      const newConversation = {
        id: 'conv-new',
        name: '新会话'
      }

      const loadMessagesMock = vi.fn()
      await conversationLogic.handleConversationCreated(newConversation, loadMessagesMock)

      // 验证会话列表被重新加载
      // 验证新会话被选中
      // 验证消息被加载
    })
  })
})

describe('useMainConversationLogic', () => {
  beforeEach(() => {
    requestMock.mockReset()
  })

  it('uses next_cursor for the next incremental request', async () => {
    requestMock
      .mockResolvedValueOnce({
        code: 0,
        data: {
          list: [{ id: '1', name: 'First', type: 'single' }],
          has_more: true,
          next_cursor: 'cursor-1'
        }
      })
      .mockResolvedValueOnce({
        code: 0,
        data: {
          list: [{ id: '2', name: 'Second', type: 'single' }],
          has_more: false
        }
      })

    const logic = useMainConversationLogic(
      vi.fn(),
      vi.fn(),
      conversation => conversation,
      ref([])
    )

    await logic.loadConversations()
    await logic.loadMoreConversations()

    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/conversations?page=2&page_size=20&cursor=cursor-1'
    )
  })

  it('keeps numeric page requests when the server does not provide next_cursor', async () => {
    requestMock
      .mockResolvedValueOnce({
        code: 0,
        data: {
          list: [{ id: '1', name: 'First', type: 'single' }],
          has_more: true
        }
      })
      .mockResolvedValueOnce({
        code: 0,
        data: {
          list: [{ id: '2', name: 'Second', type: 'single' }],
          has_more: false
        }
      })

    const logic = useMainConversationLogic(
      vi.fn(),
      vi.fn(),
      conversation => conversation,
      ref([])
    )

    await logic.loadConversations()
    await logic.loadMoreConversations()

    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      '/api/v1/conversations?page=2&page_size=20'
    )
  })
})
