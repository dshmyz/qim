import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getApps, createApp, updateApp, deleteApp } from '@/api/apps'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('apps API', () => {
  const mockApp = {
    id: 1,
    name: '测试应用',
    icon: 'https://example.com/app.png',
    category: '工具',
    url: 'https://example.com',
    openType: 'in-app' as const,
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getApps', () => {
    it('应该正确获取应用列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockApp], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getApps({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/apps/all', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持名称搜索', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getApps({ page: 1, pageSize: 10, name: '测试' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ name: '测试' }) })
      )
    })
  })

  describe('createApp', () => {
    it('应该正确创建应用', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockApp } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { name: '新应用', category: '工具', url: 'https://example.com', openType: 'in-app' as const }
      const response = await createApp(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/apps', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockApp)
    })
  })

  describe('updateApp', () => {
    it('应该正确更新应用', async () => {
      const updatedApp = { ...mockApp, name: '更新后的应用' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedApp } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的应用' }
      const response = await updateApp(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/apps/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedApp)
    })
  })

  describe('deleteApp', () => {
    it('应该正确删除应用', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteApp(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/apps/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
