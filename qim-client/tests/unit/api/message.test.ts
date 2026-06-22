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

import { messageApi } from '@/api/message'
import { http } from '@/api/core'

describe('MessageAPI - markAsRead 方法修正', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('markAsRead 应调用 POST /api/v1/conversations/:id/read（而非 PUT）', async () => {
    ;(http.post as any).mockResolvedValue({ data: { code: 0 } })

    await messageApi.markAsRead('123')

    expect(http.post).toHaveBeenCalledWith('/api/v1/conversations/123/read')
    expect(http.put).not.toHaveBeenCalled()
  })
})
