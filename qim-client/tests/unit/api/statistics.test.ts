import { describe, it, expect, beforeEach, vi } from 'vitest'

vi.mock('@/api/core', () => ({
  http: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  ApiError: class {},
}))

import { statisticsApi } from '@/api/statistics'
import { http } from '@/api/core'

describe('StatisticsAPI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('get', () => {
    it('应调用 GET /api/v1/statistics 并传入 period 参数', async () => {
      const mockData = {
        totalMessages: 100,
        totalFiles: 20,
        messageTrend: [],
        fileTypes: [],
      }
      ;(http.get as any).mockResolvedValue({
        code: 0, data: mockData,
      })

      const result = await statisticsApi.get('week')

      expect(http.get).toHaveBeenCalledWith('/api/v1/statistics', { params: { period: 'week' } })
      expect(result).toEqual(mockData)
    })

    it('period 为 day 时也应正常调用', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { totalMessages: 5 },
      })

      const result = await statisticsApi.get('day')

      expect(http.get).toHaveBeenCalledWith('/api/v1/statistics', { params: { period: 'day' } })
      expect(result).toEqual({ totalMessages: 5 })
    })
  })
})
