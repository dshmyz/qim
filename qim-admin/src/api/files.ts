import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import type { FileStatistics, LargeFile, CleanupRule, FileAccessLog } from '@/types/file'

export function getFileStatistics(): Promise<AxiosResponse<ApiResponse<FileStatistics>>> {
  return request({
    url: '/v1/admin/files/statistics',
    method: 'get',
  })
}

export function getLargeFiles(limit: number = 10): Promise<AxiosResponse<ApiResponse<LargeFile[]>>> {
  return request({
    url: '/v1/admin/files/large',
    method: 'get',
    params: { limit },
  })
}

export function getFileAccessLogs(params: {
  fileId?: number
  userId?: number
  action?: string
  startTime?: string
  endTime?: string
  page: number
  pageSize: number
}): Promise<AxiosResponse<ApiResponse<{ list: FileAccessLog[]; total: number }>>> {
  return request({
    url: '/v1/files/access-logs',
    method: 'get',
    params,
  })
}

export function getCleanupRules(): Promise<AxiosResponse<ApiResponse<CleanupRule[]>>> {
  return request({
    url: '/v1/files/cleanup/rules',
    method: 'get',
  })
}

export function createCleanupRule(rule: Omit<CleanupRule, 'id' | 'createdAt'>): Promise<AxiosResponse<ApiResponse<CleanupRule>>> {
  return request({
    url: '/v1/files/cleanup/rules',
    method: 'post',
    data: rule,
  })
}

export function updateCleanupRule(id: number, rule: Partial<CleanupRule>): Promise<AxiosResponse<ApiResponse<CleanupRule>>> {
  return request({
    url: `/v1/files/cleanup/rules/${id}`,
    method: 'put',
    data: rule,
  })
}

export function deleteCleanupRule(id: number): Promise<AxiosResponse<ApiResponse<void>>> {
  return request({
    url: `/v1/files/cleanup/rules/${id}`,
    method: 'delete',
  })
}

export function previewCleanup(ruleId: number): Promise<AxiosResponse<ApiResponse<{ count: number; size: number }>>> {
  return request({
    url: `/v1/files/cleanup/preview/${ruleId}`,
    method: 'get',
  })
}

export function executeCleanup(ruleId: number): Promise<AxiosResponse<ApiResponse<void>>> {
  return request({
    url: `/v1/files/cleanup/execute/${ruleId}`,
    method: 'post',
  })
}
