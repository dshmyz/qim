import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getDashboardStats, getRecentRegistrations, getStatistics, getUserStatistics, getGroupStatistics, getMessageStatistics } from '@/api/statistics'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('statistics API', () => {
  beforeEach(() => { vi.clearAllMocks() })

  describe('getDashboardStats', () => {
    it('应该获取仪表盘统计数据', async () => {
      const mockData = {
        totalUsers: 100,
        onlineUsers: 50,
        totalGroups: 20,
        totalMessages: 5000,
      }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getDashboardStats()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/statistics', method: 'get' })
      expect(response.data.data).toEqual(mockData)
    })
  })

  describe('getRecentRegistrations', () => {
    it('应该获取最近注册列表', async () => {
      const mockRegistrations = [
        { id: 1, username: 'user1', email: 'user1@example.com', avatar: 'https://example.com/avatar.png', createdAt: '2024-01-01T00:00:00Z' },
      ]
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: mockRegistrations, total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getRecentRegistrations({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/recent-registrations', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })
  })

  describe('getStatistics', () => {
    it('应该获取统计数据', async () => {
      const mockData = { totalUsers: 100, activeUsers: 80 }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getStatistics()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/statistics', method: 'get' })
      expect(response.data.data).toEqual(mockData)
    })
  })

  describe('getUserStatistics', () => {
    it('应该获取用户统计数据', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: { chartData: [] } } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getUserStatistics('2024-01-01', '2024-01-31')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/statistics/users', method: 'get', params: { startDate: '2024-01-01', endDate: '2024-01-31' },
      })
      expect(response.data.data).toEqual({ chartData: [] })
    })
  })

  describe('getGroupStatistics', () => {
    it('应该获取群组统计数据', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: { chartData: [] } } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getGroupStatistics('2024-01-01', '2024-01-31')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/statistics/groups', method: 'get', params: { startDate: '2024-01-01', endDate: '2024-01-31' },
      })
      expect(response.data.data).toEqual({ chartData: [] })
    })
  })

  describe('getMessageStatistics', () => {
    it('应该获取消息统计数据', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: { chartData: [] } } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getMessageStatistics('2024-01-01', '2024-01-31')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/statistics/messages', method: 'get', params: { startDate: '2024-01-01', endDate: '2024-01-31' },
      })
      expect(response.data.data).toEqual({ chartData: [] })
    })
  })
})
