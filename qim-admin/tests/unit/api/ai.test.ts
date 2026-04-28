import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getProviders,
  createProvider,
  updateProvider,
  deleteProvider,
  testProviderConnection,
  getConfig,
  updateConfig,
  getQuota,
  updateQuota,
  getStatistics,
  getUsage,
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

  const mockConfig = {
    id: 1,
    defaultProvider: 'openai',
    defaultModel: 'gpt-4',
    temperature: 0.7,
    maxTokens: 4096,
    topP: 1,
    frequencyPenalty: 0,
    presencePenalty: 0,
    timeout: 30,
  }

  const mockQuota = {
    id: 1,
    targetType: 'user' as const,
    targetId: 1,
    dailyLimit: 100,
    tokenLimit: 50000,
    concurrentLimit: 5,
    overlimitStrategy: 'reject' as const,
  }

  const mockUsage = {
    id: 1,
    userId: 1,
    provider: 'openai',
    model: 'gpt-4',
    promptTokens: 100,
    completionTokens: 200,
    totalTokens: 300,
    cost: 0.01,
    requestTime: 1500,
    status: 'success' as const,
    createdAt: '2024-01-01T00:00:00Z',
  }

  const mockStatistics = {
    totalCalls: 1000,
    totalTokens: 500000,
    totalCost: 10.5,
    byUser: [
      { userId: 1, userName: 'user1', calls: 500, tokens: 250000, cost: 5.25 },
    ],
    byModel: [
      { model: 'gpt-4', calls: 500, tokens: 250000, cost: 5.25 },
    ],
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

  // ============ Config Tests ============

  describe('getConfig', () => {
    it('应该正确获取配置', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockConfig } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getConfig()

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/config',
        method: 'get',
      })
      expect(response.data.data.defaultModel).toBe('gpt-4')
    })
  })

  describe('updateConfig', () => {
    it('应该正确更新配置', async () => {
      const updatedConfig = { ...mockConfig, temperature: 0.9 }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedConfig } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { temperature: 0.9 }
      const response = await updateConfig(updateData)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/config',
        method: 'put',
        data: updateData,
      })
      expect(response.data.data.temperature).toBe(0.9)
    })
  })

  // ============ Quota Tests ============

  describe('getQuota', () => {
    it('应该正确获取配额列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockQuota], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getQuota()

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/quota',
        method: 'get',
        params: undefined,
      })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按 targetType 和 targetId 过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getQuota({ targetType: 'user', targetId: 1 })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({
          params: { targetType: 'user', targetId: 1 },
        })
      )
    })
  })

  describe('updateQuota', () => {
    it('应该正确更新配额', async () => {
      const updatedQuota = { ...mockQuota, dailyLimit: 200 }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedQuota } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { dailyLimit: 200 }
      const response = await updateQuota(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/quota/1',
        method: 'put',
        data: updateData,
      })
      expect(response.data.data.dailyLimit).toBe(200)
    })
  })

  // ============ Statistics & Usage Tests ============

  describe('getStatistics', () => {
    it('应该正确获取统计数据', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockStatistics } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getStatistics()

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/statistics',
        method: 'get',
        params: undefined,
      })
      expect(response.data.data.totalCalls).toBe(1000)
    })

    it('应该支持按日期范围过滤', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockStatistics } }
      mockRequest.mockResolvedValue(mockResponse)

      await getStatistics({ startDate: '2024-01-01', endDate: '2024-01-31' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({
          params: { startDate: '2024-01-01', endDate: '2024-01-31' },
        })
      )
    })
  })

  describe('getUsage', () => {
    it('应该正确获取使用记录', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockUsage], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getUsage({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/ai/usage',
        method: 'get',
        params: { page: 1, pageSize: 10 },
      })
      expect(response.data.data.list).toHaveLength(1)
      expect(response.data.data.list[0].totalTokens).toBe(300)
    })

    it('应该支持按 userId 和 provider 过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getUsage({ page: 1, pageSize: 10, userId: 1, provider: 'openai' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({
          params: { page: 1, pageSize: 10, userId: 1, provider: 'openai' },
        })
      )
    })
  })
})
