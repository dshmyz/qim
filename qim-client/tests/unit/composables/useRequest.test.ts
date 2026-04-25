import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useRequest, request } from '@/composables/useRequest'

describe('useRequest', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => 'test-token')
  })

  describe('getToken', () => {
    it('应该从 localStorage 获取 token', () => {
      const { getToken } = useRequest()
      const token = getToken()
      expect(token).toBe('test-token')
      expect(localStorage.getItem).toHaveBeenCalledWith('token')
    })

    it('当 token 不存在时应该返回 null', () => {
      localStorage.getItem = vi.fn(() => null)
      const { getToken } = useRequest()
      const token = getToken()
      expect(token).toBeNull()
    })
  })

  describe('serverUrl', () => {
    it('应该使用默认 API_BASE_URL', () => {
      localStorage.getItem = vi.fn(() => null)
      const { serverUrl } = useRequest()
      expect(serverUrl.value).toBeDefined()
    })
  })
})

describe('request 函数', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.getItem = vi.fn(() => 'test-token')
    global.fetch = vi.fn()
  })

  it('应该发送带 token 的 GET 请求', async () => {
    const mockData = { code: 0, data: { id: 1, name: 'test' } }
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockData),
    })

    const result = await request('/api/v1/users/me')

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/users/me'),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer test-token',
          'Content-Type': 'application/json',
        }),
      })
    )
    expect(result).toEqual(mockData)
  })

  it('应该发送带自定义 baseUrl 的请求', async () => {
    const mockData = { code: 0 }
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockData),
    })

    await request('/api/v1/test', { baseUrl: 'http://custom:8080' })

    expect(global.fetch).toHaveBeenCalledWith(
      'http://custom:8080/api/v1/test',
      expect.any(Object)
    )
  })

  it('应该在响应失败时抛出错误', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ message: '服务器错误' }),
    })

    await expect(request('/api/v1/test')).rejects.toThrow('服务器错误')
  })

  it('应该在 403 时抛出权限不足错误', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 403,
      json: () => Promise.resolve({ message: '权限不足' }),
    })

    await expect(request('/api/v1/admin')).rejects.toThrow('权限不足')
  })

  it('应该在 JSON 解析失败时抛出通用错误', async () => {
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error('JSON 解析失败')),
    })

    await expect(request('/api/v1/test')).rejects.toThrow('请求失败')
  })

  it('应该支持 POST 请求', async () => {
    const mockData = { code: 0, data: { id: 1 } }
    ;(global.fetch as any).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockData),
    })

    const result = await request('/api/v1/users', {
      method: 'POST',
      body: JSON.stringify({ name: 'test' }),
    })

    expect(global.fetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'test' }),
      })
    )
    expect(result).toEqual(mockData)
  })
})
