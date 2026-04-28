import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getSystemMessages, createSystemMessage, updateSystemMessage, deleteSystemMessage } from '@/api/systemMessages'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('systemMessages API', () => {
  const mockMessage = {
    id: 1,
    title: '系统消息',
    content: '这是一条系统消息',
    type: 'info' as const,
    senderId: 1,
    readCount: 10,
    status: 'published' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getSystemMessages', () => {
    it('应该正确获取系统消息列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockMessage], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getSystemMessages({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-messages', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })
  })

  describe('createSystemMessage', () => {
    it('应该正确创建系统消息', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockMessage } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { title: '新消息', content: '新消息内容', type: 'notification' as const }
      const response = await createSystemMessage(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-messages', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockMessage)
    })
  })

  describe('updateSystemMessage', () => {
    it('应该正确更新系统消息', async () => {
      const updatedMessage = { ...mockMessage, title: '更新后的标题' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedMessage } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { title: '更新后的标题' }
      const response = await updateSystemMessage(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-messages/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedMessage)
    })
  })

  describe('deleteSystemMessage', () => {
    it('应该正确删除系统消息', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteSystemMessage(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-messages/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
