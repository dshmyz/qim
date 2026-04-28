import { describe, it, expect, beforeEach, vi } from 'vitest'
import { getSystemConfig, updateSystemConfig } from '@/api/systemConfig'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('systemConfig API', () => {
  const mockConfig = {
    messageRecallTime: 120,
    maxFileSize: 100,
    imageQuality: 80,
    enableRegistration: true,
    enable2FA: false,
    enableFileUpload: true,
  }

  beforeEach(() => { vi.clearAllMocks() })

  describe('getSystemConfig', () => {
    it('应该获取系统配置', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: mockConfig } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getSystemConfig()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-config', method: 'get' })
      expect(response.data.data).toEqual(mockConfig)
    })
  })

  describe('updateSystemConfig', () => {
    it('应该正确更新系统配置', async () => {
      const updatedConfig = { ...mockConfig, enableRegistration: false }
      const mockResponse = { data: { code: 0, message: 'success', data: updatedConfig } }
      mockRequest.mockResolvedValue(mockResponse)

      const updateData = { enableRegistration: false }
      const response = await updateSystemConfig(updateData)

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/system-config', method: 'put', data: updateData })
      expect(response.data.data).toEqual(updatedConfig)
    })
  })
})
