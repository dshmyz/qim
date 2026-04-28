import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getConversations, deleteConversation } from '@/api/conversations'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('conversations API', () => {
  const mockConversation = {
    id: 1,
    type: 'single' as const,
    name: '会话1',
    creatorId: 1,
    creatorName: 'admin',
    memberCount: 2,
    isPinned: false,
    lastMessageAt: '2024-01-01T00:00:00Z',
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getConversations', () => {
    it('应该正确获取会话列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockConversation], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getConversations({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/conversations', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按类型过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getConversations({ page: 1, pageSize: 10, type: 'group' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ type: 'group' }) })
      )
    })

    it('应该支持关键词搜索', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getConversations({ page: 1, pageSize: 10, keyword: '测试' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ keyword: '测试' }) })
      )
    })
  })

  describe('deleteConversation', () => {
    it('应该正确删除会话', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteConversation(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/conversations/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
