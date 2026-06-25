import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fetchUserProfile } from '@/composables/useUserProfileInfo'

vi.mock('@/utils/logger', () => ({
  logger: {
    error: vi.fn()
  }
}))

describe('fetchUserProfile', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    global.fetch = vi.fn()
  })

  it('does not request the users API for placeholder user id 0', async () => {
    const fallbackUser = { id: '0', name: 'AI助手', avatar: '' }

    const result = await fetchUserProfile('0', fallbackUser)

    expect(global.fetch).not.toHaveBeenCalled()
    expect(result).toEqual({
      success: false,
      profile: fallbackUser
    })
  })

  it('requests and maps a positive numeric user id', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({
        code: 0,
        data: {
          id: 2,
          username: 'test_user',
          nickname: '测试用户',
          email: 'test@example.com',
          phone: '13800000000',
          department: '研发部',
          ip: '127.0.0.1',
          avatar: '/avatar.png'
        }
      })
    })

    const result = await fetchUserProfile('2')

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/users/2'),
      expect.any(Object)
    )
    expect(result).toEqual({
      success: true,
      profile: {
        id: 2,
        name: '测试用户',
        username: 'test_user',
        email: 'test@example.com',
        mobile: '13800000000',
        department: '研发部',
        ip: '127.0.0.1',
        avatar: '/avatar.png'
      }
    })
  })
})
