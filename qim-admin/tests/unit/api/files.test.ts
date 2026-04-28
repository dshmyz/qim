import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  getFileStatistics,
  getLargeFiles,
  getFileAccessLogs,
  getCleanupRules,
  createCleanupRule,
  updateCleanupRule,
  deleteCleanupRule,
  previewCleanup,
  executeCleanup,
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
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/statistics', method: 'get' })
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
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/large', method: 'get', params: { limit: 10 } })
      expect(result.data.data).toEqual(mockResponse)
    })

    it('should get large files with custom limit', async () => {
      const mockResponse = []
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      await getLargeFiles(20)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/large', method: 'get', params: { limit: 20 } })
    })
  })

  describe('getFileAccessLogs', () => {
    it('should get file access logs with filters', async () => {
      const mockResponse = {
        list: [
          {
            id: 1,
            fileId: 100,
            fileName: 'test.pdf',
            action: 'download' as const,
            userId: 1,
            userName: 'user1',
            ipAddress: '192.168.1.1',
            createdAt: '2026-04-28T10:00:00Z'
          }
        ],
        total: 1
      }
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      const params = {
        fileId: 100,
        action: 'download',
        page: 1,
        pageSize: 10
      }
      const result = await getFileAccessLogs(params)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/access-logs', method: 'get', params })
      expect(result.data.data.list).toHaveLength(1)
    })
  })

  describe('getCleanupRules', () => {
    it('should get cleanup rules', async () => {
      const mockResponse = [
        {
          id: 1,
          name: 'Old files cleanup',
          type: 'time' as const,
          value: '30d',
          enabled: true,
          createdAt: '2026-04-01T10:00:00Z'
        }
      ]
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      const result = await getCleanupRules()
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/rules', method: 'get' })
      expect(result.data.data).toEqual(mockResponse)
    })
  })

  describe('createCleanupRule', () => {
    it('should create a cleanup rule', async () => {
      const newRule = {
        name: 'Large files cleanup',
        type: 'size' as const,
        value: '100MB',
        enabled: true
      }
      const mockResponse = { id: 2, ...newRule, createdAt: '2026-04-28T10:00:00Z' }
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      const result = await createCleanupRule(newRule)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/rules', method: 'post', data: newRule })
      expect(result.data.data.name).toBe('Large files cleanup')
    })
  })

  describe('updateCleanupRule', () => {
    it('should update a cleanup rule', async () => {
      const updatedRule = { name: 'Updated rule name' }
      const mockResponse = { id: 1, name: 'Updated rule name', type: 'time', value: '30d', enabled: true, createdAt: '2026-04-01T10:00:00Z' }
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      const result = await updateCleanupRule(1, updatedRule)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/rules/1', method: 'put', data: updatedRule })
      expect(result.data.data.name).toBe('Updated rule name')
    })
  })

  describe('deleteCleanupRule', () => {
    it('should delete a cleanup rule', async () => {
      const mockResponse = { code: 0, message: 'success', data: null }
      
      mockRequest.mockResolvedValue({ data: mockResponse })
      
      const result = await deleteCleanupRule(1)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/rules/1', method: 'delete' })
      expect(result.data.code).toBe(0)
    })
  })

  describe('previewCleanup', () => {
    it('should preview cleanup results', async () => {
      const mockResponse = { count: 150, size: 5368709120 }
      
      mockRequest.mockResolvedValue({ data: { code: 0, message: 'success', data: mockResponse } })
      
      const result = await previewCleanup(1)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/preview/1', method: 'get' })
      expect(result.data.data.count).toBe(150)
    })
  })

  describe('executeCleanup', () => {
    it('should execute cleanup rule', async () => {
      const mockResponse = { code: 0, message: 'success', data: null }
      
      mockRequest.mockResolvedValue({ data: mockResponse })
      
      const result = await executeCleanup(1)
      
      expect(mockRequest).toHaveBeenCalledWith({ url: '/files/cleanup/execute/1', method: 'post' })
      expect(result.data.code).toBe(0)
    })
  })
})
