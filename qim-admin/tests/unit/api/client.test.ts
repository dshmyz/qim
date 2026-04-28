import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getVersions, createVersion, updateVersion, deleteVersion,
  getVersionDistribution, getCrashLogs, getCrashDetail,
  getFeedbacks, updateFeedback,
} from '@/api/client'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('client API', () => {
  const mockVersion = {
    id: 1,
    version: '1.0.0',
    platform: 'windows' as const,
    changelog: 'Initial release',
    downloadUrl: 'https://example.com/download',
    forceUpdate: false,
    grayRelease: false,
    grayRatio: 0,
    createdAt: '2026-04-28T10:00:00Z',
  }

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

  describe('getVersions', () => {
    it('应该正确获取版本列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockVersion], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getVersions({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/versions', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按平台过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getVersions({ page: 1, pageSize: 10, platform: 'mac' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ platform: 'mac' }) })
      )
    })
  })

  describe('createVersion', () => {
    it('应该正确创建版本', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockVersion } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = {
        version: '1.0.0',
        platform: 'windows' as const,
        changelog: 'Initial release',
        downloadUrl: 'https://example.com/download',
      }
      const response = await createVersion(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/versions', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockVersion)
    })
  })

  describe('updateVersion', () => {
    it('应该正确更新版本', async () => {
      const updatedVersion = { ...mockVersion, changelog: 'Updated changelog' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedVersion } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { changelog: 'Updated changelog' }
      const response = await updateVersion(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/versions/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedVersion)
    })
  })

  describe('deleteVersion', () => {
    it('应该正确删除版本', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteVersion(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/versions/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('getVersionDistribution', () => {
    it('应该正确获取版本分布', async () => {
      const mockDistribution = [
        { version: '1.0.0', count: 500, percentage: 50 },
        { version: '1.1.0', count: 300, percentage: 30 },
        { version: '1.2.0', count: 200, percentage: 20 },
      ]
      const mockResponse = { data: { code: 0, message: 'success', data: mockDistribution } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getVersionDistribution()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/versions/distribution', method: 'get' })
      expect(response.data.data).toHaveLength(3)
    })
  })

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
