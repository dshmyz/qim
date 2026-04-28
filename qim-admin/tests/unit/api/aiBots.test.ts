import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getAIBots, createAIBot, updateAIBot, deleteAIBot, toggleAIBotStatus } from '@/api/aiBots'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('aiBots API', () => {
  const mockAIBot = {
    id: 1,
    name: 'AI助手',
    avatar: 'https://example.com/ai.png',
    description: 'AI助手',
    systemPrompt: '你是一个AI助手',
    status: 'active' as const,
    conversationCount: 100,
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getAIBots', () => {
    it('应该正确获取 AI 助手列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockAIBot], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getAIBots({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/ai-bots', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })
  })

  describe('createAIBot', () => {
    it('应该正确创建 AI 助手', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockAIBot } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { name: '新AI助手', description: '新AI描述', systemPrompt: '新AI提示词' }
      const response = await createAIBot(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/ai-bots', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockAIBot)
    })
  })

  describe('updateAIBot', () => {
    it('应该正确更新 AI 助手', async () => {
      const updatedBot = { ...mockAIBot, name: '更新后的AI助手' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedBot } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的AI助手' }
      const response = await updateAIBot(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/ai-bots/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedBot)
    })
  })

  describe('deleteAIBot', () => {
    it('应该正确删除 AI 助手', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteAIBot(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/ai-bots/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('toggleAIBotStatus', () => {
    it('应该正确切换 AI 助手状态', async () => {
      const updatedBot = { ...mockAIBot, status: 'inactive' as const }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedBot } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await toggleAIBotStatus(1, 'inactive')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/ai-bots/1/status', method: 'patch', data: { status: 'inactive' },
      })
      expect(response.data.data).toEqual(updatedBot)
    })
  })
})
