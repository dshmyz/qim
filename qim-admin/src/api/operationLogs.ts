import type { ApiResponse, OperationLog, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getOperationLogs = (params?: PaginationParams & { action?: string; operatorName?: string; startDate?: string; endDate?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<OperationLog>>>> => {
  return request({
    url: '/v1/logs/operation',
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

export const exportOperationLogs = (params?: { startDate?: string; endDate?: string }): Promise<AxiosResponse<ApiResponse<{ url: string }>>> => {
  return request({
    url: '/v1/logs/operation/export',
    method: 'get',
    params,
  })
}
