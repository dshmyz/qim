import { request } from '@/utils/request'
import type { CrashLog, UserFeedback } from '@/types/client'
import type { ApiResponse } from '@/types'
import type { AxiosResponse } from 'axios'

// 客户端版本管理 API 已迁移到 `@/api/versions`，此文件仅保留崩溃日志、用户反馈等真正属于客户端反馈类的接口

export function getCrashLogs(params?: { page?: number; pageSize?: number; platform?: string; appVersion?: string }): Promise<AxiosResponse<ApiResponse<{ list: CrashLog[]; total: number; page: number; pageSize: number }>>> {
  return request({
    url: '/v1/admin/crashes',
    method: 'get',
    params,
  })
}

export function getCrashDetail(id: number): Promise<AxiosResponse<ApiResponse<CrashLog>>> {
  return request({
    url: `/v1/admin/crashes/${id}`,
    method: 'get',
  })
}

export function getFeedbacks(params?: { page?: number; pageSize?: number; status?: string; type?: string }): Promise<AxiosResponse<ApiResponse<{ list: UserFeedback[]; total: number; page: number; pageSize: number }>>> {
  return request({
    url: '/v1/admin/feedbacks',
    method: 'get',
    params,
  })
}

export function updateFeedback(id: number, data: Partial<UserFeedback>): Promise<AxiosResponse<ApiResponse<UserFeedback>>> {
  return request({
    url: `/v1/admin/feedbacks/${id}`,
    method: 'put',
    data,
  })
}
