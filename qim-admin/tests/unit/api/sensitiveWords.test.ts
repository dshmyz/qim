import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getSensitiveWords, createSensitiveWord, updateSensitiveWord,
  deleteSensitiveWord, toggleSensitiveWordStatus, batchCreateSensitiveWords,
} from '@/api/sensitiveWords'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('sensitiveWords API', () => {
  const mockSensitiveWord = {
    id: 1,
    word: '敏感词',
    category: '政治',
    level: 'high' as const,
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getSensitiveWords', () => {
    it('应该正确获取敏感词列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockSensitiveWord], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getSensitiveWords({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/sensitive-words', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按分类过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getSensitiveWords({ page: 1, pageSize: 10, category: '政治' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ category: '政治' }) })
      )
    })
  })

  describe('createSensitiveWord', () => {
    it('应该正确创建敏感词', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockSensitiveWord } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { word: '新敏感词', category: '政治', level: 'high' as const }
      const response = await createSensitiveWord(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/sensitive-words', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockSensitiveWord)
    })
  })

  describe('updateSensitiveWord', () => {
    it('应该正确更新敏感词', async () => {
      const updatedWord = { ...mockSensitiveWord, category: '更新后分类' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedWord } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { category: '更新后分类' }
      const response = await updateSensitiveWord(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/sensitive-words/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedWord)
    })
  })

  describe('deleteSensitiveWord', () => {
    it('应该正确删除敏感词', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteSensitiveWord(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/sensitive-words/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('toggleSensitiveWordStatus', () => {
    it('应该正确切换敏感词状态', async () => {
      const updatedWord = { ...mockSensitiveWord, status: 'inactive' as const }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedWord } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await toggleSensitiveWordStatus(1, 'inactive')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/sensitive-words/1/toggle', method: 'patch', data: { status: 'inactive' },
      })
      expect(response.data.data).toEqual(updatedWord)
    })
  })

  describe('batchCreateSensitiveWords', () => {
    it('应该批量创建敏感词', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: { count: 3 } } }
      mockRequest.mockResolvedValue(mockResponse)

      const batchData = { words: ['词1', '词2', '词3'], category: '政治', level: 'medium' as const }
      const response = await batchCreateSensitiveWords(batchData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/sensitive-words/batch', method: 'post', data: batchData })
      expect(response.data.data.count).toBe(3)
    })
  })
})
