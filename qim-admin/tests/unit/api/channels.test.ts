import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getChannels, createChannel, updateChannel, deleteChannel } from '@/api/channels'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('channels API', () => {
  const mockChannel = {
    id: 1,
    name: '测试频道',
    description: '测试频道描述',
    icon: 'https://example.com/icon.png',
    memberCount: 5,
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getChannels', () => {
    it('应该正确获取频道列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockChannel], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getChannels({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/channels', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持不传参数', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockChannel], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getChannels()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/channels', method: 'get', params: undefined })
    })
  })

  describe('createChannel', () => {
    it('应该正确创建频道', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockChannel } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { name: '新频道', description: '新频道描述', type: 'text' as const }
      const response = await createChannel(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/channels', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockChannel)
    })
  })

  describe('updateChannel', () => {
    it('应该正确更新频道', async () => {
      const updatedChannel = { ...mockChannel, name: '更新后的频道' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedChannel } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的频道' }
      const response = await updateChannel(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/channels/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedChannel)
    })
  })

  describe('deleteChannel', () => {
    it('应该正确删除频道', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteChannel(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/channels/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
