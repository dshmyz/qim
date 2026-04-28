import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getNotifications, markAsRead, markAllAsRead, deleteNotification } from '@/api/notifications'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('notifications API', () => {
  const mockNotification = {
    id: 1,
    type: 'info' as const,
    title: '测试通知',
    content: '这是一个测试通知',
    isRead: false,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getNotifications', () => {
    it('应该正确获取通知列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockNotification], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getNotifications({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/notifications', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按类型过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getNotifications({ page: 1, pageSize: 10, type: 'warning' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ type: 'warning' }) })
      )
    })

    it('应该支持按已读状态过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getNotifications({ page: 1, pageSize: 10, isRead: false })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ isRead: false }) })
      )
    })
  })

  describe('markAsRead', () => {
    it('应该正确标记通知为已读', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await markAsRead(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/notifications/1/read', method: 'put' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('markAllAsRead', () => {
    it('应该正确标记所有通知为已读', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await markAllAsRead()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/notifications/read-all', method: 'put' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('deleteNotification', () => {
    it('应该正确删除通知', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteNotification(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/notifications/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
