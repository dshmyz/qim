import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getBlacklist, removeBlacklistEntry } from '@/api/blacklist'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('blacklist API', () => {
  const mockBlacklistEntry = {
    id: 1,
    userId: 1,
    username: 'baduser',
    reason: '违规操作',
    operatorId: 1,
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getBlacklist', () => {
    it('应该正确获取黑名单列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockBlacklistEntry], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getBlacklist({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users/blacklist', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持不传参数', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockBlacklistEntry], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getBlacklist()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users/blacklist', method: 'get', params: undefined })
    })
  })

  describe('removeBlacklistEntry', () => {
    it('应该正确移除黑名单条目', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await removeBlacklistEntry(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users/blacklist/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
