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

import { appsApi } from '@/api/apps'
import { http } from '@/api/core'

describe('AppsAPI', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('应调用 GET /api/v1/apps', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [{ id: 1, name: 'app1' }] },
      })

      const result = await appsApi.list()

      expect(http.get).toHaveBeenCalledWith('/api/v1/apps')
      expect(result).toEqual([{ id: 1, name: 'app1' }])
    })

    it('后端返回数组时也应正常解析', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: [{ id: 2, name: 'app2' }],
      })

      const result = await appsApi.list()

      expect(result).toEqual([{ id: 2, name: 'app2' }])
    })

    it('后端返回空 list 时应返回空数组', async () => {
      ;(http.get as any).mockResolvedValue({
        code: 0, data: { list: [] },
      })

      const result = await appsApi.list()

      expect(result).toEqual([])
    })
  })

  describe('create', () => {
    it('应调用 POST /api/v1/apps 并传入 payload', async () => {
      ;(http.post as any).mockResolvedValue({
        code: 0, data: { id: 10, name: 'newapp' },
      })

      const payload = { name: 'newapp', icon: 'fas fa-star', category: 'productivity', url: '', status: 'active', open_type: 'in-app' }
      const result = await appsApi.create(payload)

      expect(http.post).toHaveBeenCalledWith('/api/v1/apps', payload)
      expect(result).toEqual({ id: 10, name: 'newapp' })
    })
  })

  describe('update', () => {
    it('应调用 PUT /api/v1/apps/:id 并传入 payload', async () => {
      ;(http.put as any).mockResolvedValue({
        code: 0, data: { id: 5, name: 'updated' },
      })

      const payload = { name: 'updated' }
      const result = await appsApi.update(5, payload)

      expect(http.put).toHaveBeenCalledWith('/api/v1/apps/5', payload)
      expect(result).toEqual({ id: 5, name: 'updated' })
    })
  })

  describe('remove', () => {
    it('应调用 DELETE /api/v1/apps/:id', async () => {
      ;(http.delete as any).mockResolvedValue({
        code: 0, data: null,
      })

      await appsApi.remove(7)

      expect(http.delete).toHaveBeenCalledWith('/api/v1/apps/7')
    })
  })
})
