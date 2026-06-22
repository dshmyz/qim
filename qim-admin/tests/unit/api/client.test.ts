import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getCrashLogs, getCrashDetail,
  getFeedbacks, updateFeedback,
} from '@/api/client'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('client API', () => {
  const mockCrashLog = {
    id: 1,
    deviceId: 'device-1',
    deviceModel: 'iPhone 14',
    osVersion: 'iOS 16.0',
    appVersion: '1.0.0',
    crashReason: 'NullPointerException',
    stackTrace: 'at com.example.App.main(App.java:10)',
    userId: 100,
    createdAt: '2026-04-28T10:00:00Z',
  }

  const mockFeedback = {
    id: 1,
    userId: 100,
    type: 'bug' as const,
    title: 'App 崩溃问题',
    content: '打开应用时立即崩溃',
    screenshots: ['https://example.com/screenshot1.png'],
    status: 'pending' as const,
    handlerId: undefined,
    reply: undefined,
    createdAt: '2026-04-28T10:00:00Z',
    updatedAt: '2026-04-28T10:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getCrashLogs', () => {
    it('应该正确获取崩溃日志列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockCrashLog], total: 1, page: 1, pageSize: 20 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getCrashLogs({ page: 1, pageSize: 20 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/crashes', method: 'get', params: { page: 1, pageSize: 20 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按平台和版本过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 20 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getCrashLogs({ page: 1, pageSize: 20, platform: 'ios', appVersion: '1.0.0' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({
          params: expect.objectContaining({ platform: 'ios', appVersion: '1.0.0' }),
        })
      )
    })
  })

  describe('getCrashDetail', () => {
    it('应该正确获取崩溃详情', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockCrashLog } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getCrashDetail(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/crashes/1', method: 'get' })
      expect(response.data.data).toEqual(mockCrashLog)
    })
  })

  describe('getFeedbacks', () => {
    it('应该正确获取用户反馈列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockFeedback], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getFeedbacks({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/feedbacks', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按状态和类型过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getFeedbacks({ page: 1, pageSize: 10, status: 'pending', type: 'bug' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({
          params: expect.objectContaining({ status: 'pending', type: 'bug' }),
        })
      )
    })
  })

  describe('updateFeedback', () => {
    it('应该正确更新反馈', async () => {
      const updatedFeedback = { ...mockFeedback, status: 'processing' as const, reply: '正在处理中' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedFeedback } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { status: 'processing' as const, reply: '正在处理中' }
      const response = await updateFeedback(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/feedbacks/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedFeedback)
    })
  })
})
