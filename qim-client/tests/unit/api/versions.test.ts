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

import { versionsApi } from '@/api/versions'
import { http } from '@/api/core'

describe('VersionsAPI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('getLatest', () => {
    it('应调用 GET /api/v1/client/versions 并传入 platform 和 pageSize=1', async () => {
      const mockVersion = {
        id: 1,
        version: '1.2.0',
        releaseDate: '2026-07-01',
        updateNotes: '修复若干问题',
        downloadUrl: 'https://example.com/download',
      }
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [mockVersion] },
      })

      const result = await versionsApi.getLatest('macos')

      expect(http.get).toHaveBeenCalledWith('/api/v1/client/versions', {
        params: { platform: 'macos', pageSize: 1 },
        skipAuth: true,
        timeout: 8000,
      })
      expect(result).toEqual(mockVersion)
    })

    it('列表为空时应返回 null', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [] },
      })

      const result = await versionsApi.getLatest('windows')

      expect(result).toBeNull()
    })

    it('data 为 null 时应返回 null', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: null,
      })

      const result = await versionsApi.getLatest('linux')

      expect(result).toBeNull()
    })

    it('应使用 skipAuth=true（公开接口）', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [] },
      })

      await versionsApi.getLatest('macos')

      const callArgs = (http.get as any).mock.calls[0][1]
      expect(callArgs.skipAuth).toBe(true)
    })

    it('应使用 8 秒超时', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [] },
      })

      await versionsApi.getLatest('macos')

      const callArgs = (http.get as any).mock.calls[0][1]
      expect(callArgs.timeout).toBe(8000)
    })
  })
})
