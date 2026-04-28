import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getUsers, getUserById, createUser, updateUser, deleteUser,
  assignRoles, removeRole, banUser, unbanUser,
} from '@/api/users'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('users API', () => {
  const mockUser = {
    id: 1,
    username: 'testuser',
    nickname: '测试用户',
    email: 'test@example.com',
    phone: '13800138000',
    avatar: 'https://example.com/avatar.png',
    status: 'active' as const,
    roles: ['user'],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getUsers', () => {
    it('应该正确获取用户列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockUser], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const params = { page: 1, pageSize: 10 }
      const response = await getUsers(params)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/users', method: 'get', params })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持关键词搜索', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getUsers({ page: 1, pageSize: 10, keyword: 'test' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ keyword: 'test' }) })
      )
    })
  })

  describe('getUserById', () => {
    it('应该根据 ID 获取用户', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockUser } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getUserById(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/users/1', method: 'get' })
      expect(response.data.data).toEqual(mockUser)
    })

    it('应该使用正确的 ID 构建 URL', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockUser } }
      mockRequest.mockResolvedValue(mockResponse)

      await getUserById(42)

      expect(mockRequest).toHaveBeenCalledWith(expect.objectContaining({ url: '/users/42' }))
    })
  })

  describe('createUser', () => {
    it('应该正确创建用户', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockUser } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { username: 'newuser', password: 'password123', email: 'new@example.com' }
      const response = await createUser(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockUser)
    })
  })

  describe('updateUser', () => {
    it('应该正确更新用户', async () => {
      const updatedUser = { ...mockUser, nickname: '更新后的昵称' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedUser } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { nickname: '更新后的昵称' }
      const response = await updateUser(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedUser)
    })
  })

  describe('deleteUser', () => {
    it('应该正确删除用户', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteUser(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/users/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('assignRoles', () => {
    it('应该正确分配角色', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      await assignRoles(1, ['admin', 'user'])

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/users/1/roles/batch', method: 'post', data: { roles: ['admin', 'user'] },
      })
    })
  })

  describe('removeRole', () => {
    it('应该正确移除角色', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      await removeRole(1, 'admin')

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/users/1/roles/admin', method: 'delete' })
    })
  })

  describe('banUser', () => {
    it('应该正确封禁用户', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      await banUser(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/users/1/ban', method: 'post' })
    })
  })

  describe('unbanUser', () => {
    it('应该正确解封用户', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      await unbanUser(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/users/1/unban', method: 'post' })
    })
  })
})
