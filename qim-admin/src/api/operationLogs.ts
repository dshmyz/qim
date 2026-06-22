import type { ApiResponse, OperationLog, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import service from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getOperationLogs = (params?: PaginationParams & { action?: string; module?: string; status?: string; username?: string; startDate?: string; endDate?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<OperationLog>>>> => {
  return request({
    url: '/v1/logs/operation',
    method: 'get',
    params,
  })
}

export const getOperationLogStats = (params?: { startDate?: string; endDate?: string; trend?: boolean }): Promise<AxiosResponse<ApiResponse<{ total: number; success: number; failed: number; avgDuration: number; trend: Array<{ date: string; count: number }> }>>> => {
  return request({
    url: '/v1/logs/operation/stats',
    method: 'get',
    params,
  })
}

export const getOperationLogDetail = (id: number): Promise<AxiosResponse<ApiResponse<OperationLog>>> => {
  return request({
    url: `/v1/logs/operation/${id}`,
    method: 'get',
  })
}

export const exportOperationLogs = (params?: { startDate?: string; endDate?: string }): Promise<AxiosResponse<Blob>> => {
  return service({
    url: '/v1/logs/operation/export',
    method: 'get',
    params,
    responseType: 'blob',
  })
}
