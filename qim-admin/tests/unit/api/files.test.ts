import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getFileStatistics,
  getLargeFiles,
} from '@/api/files'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('Files API', () => {
  beforeEach(() => { vi.clearAllMocks() })

  describe('getFileStatistics', () => {
    it('should get file statistics', async () => {
      const mockResponse = {
        totalSize: 107374182400,
        usedSize: 53687091200,
        fileCount: 1000,
        sizeByType: [
          { type: 'image', size: 10737418240, count: 500 },
          { type: 'document', size: 21474836480, count: 300 }
        ]
      }

      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })

      const result = await getFileStatistics()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/files/statistics', method: 'get' })
      expect(result.data.data).toEqual(mockResponse)
    })
  })

  describe('getLargeFiles', () => {
    it('should get large files with default limit', async () => {
      const mockResponse = [
        {
          id: 1,
          fileName: 'large-file.pdf',
          fileSize: 104857600,
          fileType: 'application/pdf',
          uploaderId: 1,
          uploaderName: 'user1',
          createdAt: '2026-04-28T10:00:00Z'
        }
      ]

      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })

      const result = await getLargeFiles()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/files/large', method: 'get', params: { limit: 10 } })
      expect(result.data.data).toEqual(mockResponse)
    })

    it('should get large files with custom limit', async () => {
      const mockResponse = []

      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })

      await getLargeFiles(20)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/files/large', method: 'get', params: { limit: 20 } })
    })
  })
})
