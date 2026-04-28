import { describe, it, expect, beforeEach, vi } from 'vitest'
import {
  getServerMetrics,
  getServiceStatus,
  getServerMetricsHistory,
  healthCheck,
  getAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  getAlertHistory,
} from '@/api/monitor'

const mockRequest = vi.fn()

vi.mock('@/utils/request', () => ({
  request: (config: any) => mockRequest(config),
}))

describe('monitor API', () => {
  beforeEach(() => { vi.clearAllMocks() })

  describe('getServerMetrics', () => {
    it('应该获取服务器指标数据', async () => {
      const mockData = {
        cpu: 45.5,
        memory: 62.3,
        disk: 78.9,
        network: { in: 1024000, out: 512000 },
        timestamp: '2026-04-28T10:00:00Z',
      }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getServerMetrics()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/monitor/server', method: 'get' })
      expect(response.data.data).toEqual(mockData)
    })
  })

  describe('getServiceStatus', () => {
    it('应该获取服务状态列表', async () => {
      const mockData = [
        {
          name: 'message-service',
          status: 'healthy',
          message: 'Service is running',
          lastCheck: '2026-04-28T10:00:00Z',
        },
      ]
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getServiceStatus()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/monitor/services', method: 'get' })
      expect(response.data.data).toHaveLength(1)
      expect(response.data.data[0].status).toBe('healthy')
    })
  })

  describe('getServerMetricsHistory', () => {
    it('应该获取服务器指标历史数据', async () => {
      const mockData = [
        { cpu: 40, memory: 60, disk: 70, network: { in: 1000, out: 500 }, timestamp: '2026-04-28T10:00:00Z' },
      ]
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const params = { startTime: '2026-04-28T00:00:00Z', endTime: '2026-04-28T23:59:59Z', interval: 300 }
      const response = await getServerMetricsHistory(params)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/server/history',
        method: 'get',
        params,
      })
      expect(response.data.data).toEqual(mockData)
    })
  })

  describe('healthCheck', () => {
    it('应该执行服务健康检查', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await healthCheck()

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/services/health-check',
        method: 'post',
      })
    })
  })

  describe('getAlertRules', () => {
    it('应该获取告警规则列表', async () => {
      const mockData = [
        {
          id: 1,
          name: 'High CPU Usage',
          metric: 'cpu',
          condition: 'gt',
          threshold: 90,
          duration: 300,
          notifyMethods: ['email'],
          notifyTargets: ['admin@example.com'],
          enabled: true,
          createdAt: '2026-04-28T10:00:00Z',
        },
      ]
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await getAlertRules()

      expect(mockRequest).toHaveBeenCalledWith({ url: '/v1/admin/monitor/alerts', method: 'get' })
      expect(response.data.data).toHaveLength(1)
    })
  })

  describe('createAlertRule', () => {
    it('应该创建告警规则', async () => {
      const rule = {
        name: 'High Memory Usage',
        metric: 'memory',
        condition: 'gt',
        threshold: 85,
        duration: 600,
        notifyMethods: ['email', 'webhook'],
        notifyTargets: ['admin@example.com'],
        enabled: true,
      }
      const mockData = { id: 1, ...rule, createdAt: '2026-04-28T10:00:00Z' }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await createAlertRule(rule)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/alerts',
        method: 'post',
        data: rule,
      })
      expect(response.data.data.id).toBe(1)
    })
  })

  describe('updateAlertRule', () => {
    it('应该更新告警规则', async () => {
      const rule = { enabled: false }
      const mockData = {
        id: 1,
        name: 'High CPU Usage',
        metric: 'cpu',
        condition: 'gt',
        threshold: 90,
        duration: 300,
        notifyMethods: ['email'],
        notifyTargets: ['admin@example.com'],
        enabled: false,
        createdAt: '2026-04-28T10:00:00Z',
      }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await updateAlertRule(1, rule)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/alerts/1',
        method: 'put',
        data: rule,
      })
      expect(response.data.data.enabled).toBe(false)
    })
  })

  describe('deleteAlertRule', () => {
    it('应该删除告警规则', async () => {
      const mockResponse = { data: { code: 0, message: 'success', data: null } }
      mockRequest.mockResolvedValue(mockResponse)

      const response = await deleteAlertRule(1)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/alerts/1',
        method: 'delete',
      })
    })
  })

  describe('getAlertHistory', () => {
    it('应该获取告警历史列表', async () => {
      const mockData = {
        list: [
          {
            id: 1,
            ruleId: 1,
            metric: 'cpu',
            value: 95,
            status: 'firing',
            createdAt: '2026-04-28T10:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      }
      const mockResponse = { data: { code: 0, message: 'success', data: mockData } }
      mockRequest.mockResolvedValue(mockResponse)

      const params = { page: 1, pageSize: 10 }
      const response = await getAlertHistory(params)

      expect(mockRequest).toHaveBeenCalledWith({
        url: '/v1/admin/monitor/alerts/history',
        method: 'get',
        params,
      })
      expect(response.data.data.list).toHaveLength(1)
      expect(response.data.data.total).toBe(1)
    })
  })
})
