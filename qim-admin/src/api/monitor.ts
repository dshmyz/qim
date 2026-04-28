import type { ApiResponse, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ServerMetrics, ServiceStatus, AlertRule, AlertHistory } from '@/types/monitor'

export const getServerMetrics = (): Promise<AxiosResponse<ApiResponse<ServerMetrics>>> => {
  return request({
    url: '/v1/admin/monitor/server',
    method: 'get',
  })
}

export const getServerMetricsHistory = (params: {
  startTime: string
  endTime: string
  interval: number
}): Promise<AxiosResponse<ApiResponse<ServerMetrics[]>>> => {
  return request({
    url: '/v1/admin/monitor/server/history',
    method: 'get',
    params,
  })
}

export const getServiceStatus = (): Promise<AxiosResponse<ApiResponse<ServiceStatus[]>>> => {
  return request({
    url: '/v1/admin/monitor/services',
    method: 'get',
  })
}

export const healthCheck = (): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/admin/monitor/services/health-check',
    method: 'post',
  })
}

export const getAlertRules = (): Promise<AxiosResponse<ApiResponse<AlertRule[]>>> => {
  return request({
    url: '/v1/admin/monitor/alerts',
    method: 'get',
  })
}

export const createAlertRule = (rule: Omit<AlertRule, 'id' | 'createdAt'>): Promise<AxiosResponse<ApiResponse<AlertRule>>> => {
  return request({
    url: '/v1/admin/monitor/alerts',
    method: 'post',
    data: rule,
  })
}

export const updateAlertRule = (id: number, rule: Partial<AlertRule>): Promise<AxiosResponse<ApiResponse<AlertRule>>> => {
  return request({
    url: `/v1/admin/monitor/alerts/${id}`,
    method: 'put',
    data: rule,
  })
}

export const deleteAlertRule = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/monitor/alerts/${id}`,
    method: 'delete',
  })
}

export const getAlertHistory = (params: {
  ruleId?: number
  status?: string
  startTime?: string
  endTime?: string
  page: number
  pageSize: number
}): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AlertHistory>>>> => {
  return request({
    url: '/v1/admin/monitor/alerts/history',
    method: 'get',
    params,
  })
}
