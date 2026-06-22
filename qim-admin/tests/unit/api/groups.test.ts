import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getGroups, getGroupById, getGroupMembers, removeGroupMember, deleteGroup } from '@/api/groups'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('groups API', () => {
  const mockGroup = {
    id: 1,
    name: '测试群组',
    description: '这是一个测试群组',
    ownerId: 1,
    avatar: 'https://example.com/group.png',
    memberCount: 10,
    status: 'active' as const,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getGroups', () => {
    it('应该正确获取群组列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockGroup], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getGroups({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/groups', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持关键词搜索', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getGroups({ page: 1, pageSize: 10, keyword: '测试' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ keyword: '测试' }) })
      )
    })
  })

  describe('getGroupById', () => {
    it('应该根据 ID 获取群组详情', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockGroup } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getGroupById(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/groups/1', method: 'get' })
      expect(response.data.data).toEqual(mockGroup)
    })
  })

  describe('getGroupMembers', () => {
    it('应该获取群组成员列表', async () => {
      const mockMembers = [
        { id: 1, userId: 1, username: 'user1', nickname: '用户1', avatar: 'https://example.com/avatar.png', role: 'owner' as const, joinedAt: '2024-01-01T00:00:00Z' },
      ]
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: mockMembers, total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getGroupMembers(1, { page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/groups/1/members', method: 'get', params: { page: 1, pageSize: 10 },
      })
      expect(response.data.data.list).toHaveLength(1)
    })
  })

  describe('removeGroupMember', () => {
    it('应该移除群组成员', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await removeGroupMember(1, 2)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/groups/1/members/2', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('deleteGroup', () => {
    it('应该正确解散群组', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteGroup(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/groups/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })
})
