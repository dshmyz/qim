import { describe, it, expect, beforeEach, vi } from 'vitest'
import { login, logout, getCurrentUser } from '@/api/auth'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('auth API', () => {
  const mockUser = {
    id: 1,
    username: 'admin',
    email: 'admin@example.com',
    avatar: 'https://example.com/avatar.png',
    role: 'admin',
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('login', () => {
    it('应该正确调用登录接口', async () => {
      const mockResponse = {
        data: {
          code: 0,
          message: 'success',
          data: {
            token: 'test-token-123',
            user: mockUser,
          },
        },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const loginData = { username: 'admin', password: 'password123' }
      const response = await login(loginData)

      expect(mockRequest).toHaveBeenCalledWith({
        baseURL: '/api/v1',
        url: '/auth/login',
        method: 'post',
        data: loginData,
      })
      expect(response.data.code).toBe(0)
      expect(response.data.data.token).toBe('test-token-123')
      expect(response.data.data.user).toEqual(mockUser)
    })
  })

  describe('logout', () => {
    it('应该正确调用登出接口', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: null },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await logout()

      expect(mockRequest).toHaveBeenCalledWith({
        baseURL: '/api/v1',
        url: '/auth/logout',
        method: 'post',
      })
      expect(response.data.code).toBe(0)
    })
  })

  describe('getCurrentUser', () => {
    it('应该正确获取当前用户信息', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: mockUser },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getCurrentUser()

      expect(mockRequest).toHaveBeenCalledWith({
        baseURL: '/api/v1',
        url: '/users/me',
        method: 'get',
      })
      expect(response.data.data).toEqual(mockUser)
    })
  })
})
