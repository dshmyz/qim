import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getMiniApps, createMiniApp, updateMiniApp, deleteMiniApp } from '@/api/miniApps'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('miniApps API', () => {
  const mockMiniApp = {
    id: 1,
    appID: 'wx1234567890',
    name: '测试小程序',
    icon: 'https://example.com/icon.png',
    path: '/pages/index',
    description: '这是一个测试小程序',
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getMiniApps', () => {
    it('应该正确获取小程序列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockMiniApp], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const params = { page: 1, pageSize: 10 }
      const response = await getMiniApps(params)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/mini-apps', method: 'get', params })
      expect(response.data.data.list).toHaveLength(1)
      expect(response.data.data.total).toBe(1)
    })

    it('应该支持搜索参数', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getMiniApps({ page: 1, pageSize: 10, name: '测试', status: 'active' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ name: '测试', status: 'active' }) })
      )
    })
  })

  describe('createMiniApp', () => {
    it('应该正确创建小程序', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockMiniApp } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { appID: 'wx1234567890', name: '新小程序', path: '/pages/index', description: '新创建的小程序' }
      const response = await createMiniApp(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/mini-apps', method: 'post', data: createData })
      expect(response.data.code).toBe(0)
      expect(response.data.data).toEqual(mockMiniApp)
    })
  })

  describe('updateMiniApp', () => {
    it('应该正确更新小程序', async () => {
      const updatedMiniApp = { ...mockMiniApp, name: '更新后的小程序' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedMiniApp } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的小程序', description: '更新后的描述' }
      const response = await updateMiniApp(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/mini-apps/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedMiniApp)
    })
  })

  describe('deleteMiniApp', () => {
    it('应该正确删除小程序', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteMiniApp(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/mini-apps/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
