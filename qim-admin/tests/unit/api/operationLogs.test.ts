import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getOperationLogs, getOperationLogDetail, exportOperationLogs } from '@/api/operationLogs'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('operationLogs API', () => {
  const mockLog = {
    id: 1,
    operatorId: 1,
    operatorName: 'admin',
    action: '创建用户',
    targetType: 'user',
    targetId: 1,
    detail: '创建新用户',
    ip: '127.0.0.1',
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getOperationLogs', () => {
    it('应该正确获取操作日志列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockLog], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getOperationLogs({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/operation-logs', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按操作类型过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getOperationLogs({ page: 1, pageSize: 10, action: '创建用户' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ action: '创建用户' }) })
      )
    })

    it('应该支持按操作人过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getOperationLogs({ page: 1, pageSize: 10, operatorName: 'admin' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ operatorName: 'admin' }) })
      )
    })
  })

  describe('getOperationLogDetail', () => {
    it('应该获取操作日志详情', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockLog } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getOperationLogDetail(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/operation-logs/1', method: 'get' })
      expect(response.data.data).toEqual(mockLog)
    })
  })

  describe('exportOperationLogs', () => {
    it('应该导出操作日志', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: { url: 'https://example.com/export.csv' } } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await exportOperationLogs({ startDate: '2024-01-01', endDate: '2024-01-31' })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/operation-logs/export', method: 'get', params: { startDate: '2024-01-01', endDate: '2024-01-31' },
      })
      expect(response.data.data.url).toBe('https://example.com/export.csv')
    })
  })
})
