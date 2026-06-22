import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  testProviderConnection,
} from '@/api/ai'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('ai API', () => {
  const mockProvider = {
    id: 1,
    name: 'openai',
    displayName: 'OpenAI',
    apiKey: 'sk-xxx',
    apiEndpoint: 'https://api.openai.com/v1',
    models: ['gpt-4', 'gpt-3.5-turbo'],
    config: {},
    enabled: true,
    status: 'connected' as const,
    lastTestAt: '2024-01-01T00:00:00Z',
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  // ============ Provider Tests ============

  describe('getProviders', () => {
    it('应该正确获取提供商列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockProvider], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getProviders({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/providers',
        method: 'get',
        params: { page: 1, pageSize: 10 },
      })
      expect(response.data.data.list).toHaveLength(1)
      expect(response.data.data.list[0].name).toBe('openai')
    })
  })

  describe('createProvider', () => {
    it('应该正确创建提供商', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockProvider } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = {
        name: 'openai',
        displayName: 'OpenAI',
        apiKey: 'sk-xxx',
        apiEndpoint: 'https://api.openai.com/v1',
        models: ['gpt-4'],
        config: {},
        enabled: true,
      }
      const response = await createProvider(createData)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/providers',
        method: 'post',
        data: createData,
      })
      expect(response.data.data).toEqual(mockProvider)
    })
  })

  describe('updateProvider', () => {
    it('应该正确更新提供商', async () => {
      const updatedProvider = { ...mockProvider, displayName: 'OpenAI 更新' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedProvider } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { displayName: 'OpenAI 更新' }
      const response = await updateProvider(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/providers/1',
        method: 'put',
        data: updateData,
      })
      expect(response.data.data.displayName).toBe('OpenAI 更新')
    })
  })

  describe('deleteProvider', () => {
    it('应该正确删除提供商', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteProvider(1)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/providers/1',
        method: 'delete',
      })
      expect(response.data.code).toBe(0)
    })
  })

  describe('testProviderConnection', () => {
    it('应该正确测试连接', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { success: true, message: '连接成功' } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await testProviderConnection(1)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/providers/1/test',
        method: 'post',
      })
      expect(response.data.data.success).toBe(true)
    })
  })
})
