import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getVersions, createVersion, updateVersion, deleteVersion, toggleVersionStatus } from '@/api/versions'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('versions API', () => {
  const mockVersion = {
    id: 1,
    version: '1.0.0',
    platform: 'windows' as const,
    releaseDate: '2024-01-01',
    updateNotes: '版本更新说明',
    forceUpdate: false,
    downloadUrl: 'https://example.com/download',
    status: 'active' as const,
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getVersions', () => {
    it('应该正确获取版本列表', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [mockVersion], total: 1, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getVersions({ page: 1, pageSize: 10 })

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/versions', method: 'get', params: { page: 1, pageSize: 10 } })
      expect(response.data.data.list).toHaveLength(1)
    })

    it('应该支持按平台过滤', async () => {
      const mockResponse = {
        data: { code: 0, message: 'success', data: { list: [], total: 0, page: 1, pageSize: 10 } },
      }
      mockRequest.mockResolvedValue(mockResponse)

      await getVersions({ page: 1, pageSize: 10, platform: 'macos' })

      expect(mockRequest).toHaveBeenCalledWith(
        expect.objectContaining({ params: expect.objectContaining({ platform: 'macos' }) })
      )
    })
  })

  describe('createVersion', () => {
    it('应该正确创建版本', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockVersion } }
      mockRequest.mockResolvedValue(mockResponse)

      const createData = {
        version: '1.0.0', platform: 'windows' as const, releaseDate: '2024-01-01',
        updateNotes: '版本更新说明', forceUpdate: false, downloadUrl: 'https://example.com/download',
      }
      const response = await createVersion(createData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/versions', method: 'post', data: createData })
      expect(response.data.data).toEqual(mockVersion)
    })
  })

  describe('updateVersion', () => {
    it('应该正确更新版本', async () => {
      const updatedVersion = { ...mockVersion, updateNotes: '更新后的说明' }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedVersion } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { updateNotes: '更新后的说明' }
      const response = await updateVersion(1, updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/versions/1', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedVersion)
    })
  })

  describe('deleteVersion', () => {
    it('应该正确删除版本', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteVersion(1)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/versions/1', method: 'delete' })
      expect(response.data.code).toBe(0)
    })
  })

  describe('toggleVersionStatus', () => {
    it('应该正确切换版本状态', async () => {
      const updatedVersion = { ...mockVersion, status: 'inactive' as const }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedVersion } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await toggleVersionStatus(1, 'inactive')

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/versions/1/status', method: 'patch', data: { status: 'inactive' },
      })
      expect(response.data.data).toEqual(updatedVersion)
    })
  })
})
