import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getRoles, createRole, updateRole, deleteRole, getRoleUsers } from '@/api/roles'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('roles API', () => {
  const mockRole = {
    id: 1,
    name: '管理员',
    code: 'admin',
    description: '系统管理员',
    permissions: ['*'],
    userCount: 5,
    createdAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getRoles', () => {
    it('应该正确获取角色列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockRole], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getRoles({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/roles', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持不传参数', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockRole], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getRoles()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/roles', method: 'get', params: undefined })
    })
  })

  describe('createRole', () => {
    it('应该正确创建角色', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockRole } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = { name: '新角色', code: 'new_role', description: '新角色描述', permissions: ['read', 'write'] }
      const response = await createRole(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/roles', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockRole)
    })
  })

  describe('updateRole', () => {
    it('应该正确更新角色', async () => {
      const updatedRole = { ...mockRole, name: '更新后的角色' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedRole } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { name: '更新后的角色' }
      const response = await updateRole(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/roles/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedRole)
    })
  })

  describe('deleteRole', () => {
    it('应该正确删除角色', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteRole(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/roles/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('getRoleUsers', () => {
    it('应该获取角色下的用户列表', async () => {
      const mockUsers = [
        { id: 1, username: 'user1', nickname: '用户1', avatar: 'https://example.com/avatar.png' },
        { id: 2, username: 'user2', nickname: '用户2', avatar: 'https://example.com/avatar2.png' },
      ]
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: mockUsers, total: 2, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getRoleUsers(1, { page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/roles/1/users', method: 'get', params: { page: 1, pageSize: 10 },
      })
      expect(response.data.data.list).toHaveLength(2)
    })
  })
})
