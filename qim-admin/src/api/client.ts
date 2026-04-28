import { request } from '@/utils/request'
import type { ClientVersion, VersionDistribution, CreateVersionParams, UpdateVersionParams, CrashLog, UserFeedback } from '@/types/client'
import type { ApiResponse } from '@/types'
import type { AxiosResponse } from 'axios'

export function getVersions(params?: { page?: number; pageSize?: number; platform?: string }): Promise<AxiosResponse<ApiResponse<{ list: ClientVersion[]; total: number; page: number; pageSize: number }>>> {
  return request({
    url: '/v1/admin/versions',
    method: 'get',
    params,
  })
}

export function createVersion(data: CreateVersionParams): Promise<AxiosResponse<ApiResponse<ClientVersion>>> {
  return request({
    url: '/v1/admin/versions',
    method: 'post',
    data,
  })
}

export function updateVersion(id: number, data: UpdateVersionParams): Promise<AxiosResponse<ApiResponse<ClientVersion>>> {
  return request({
    url: `/v1/admin/versions/${id}`,
    method: 'put',
    data,
  })
}

export function deleteVersion(id: number): Promise<AxiosResponse<ApiResponse<void>>> {
  return request({
    url: `/v1/admin/versions/${id}`,
    method: 'delete',
  })
}

export function getVersionDistribution(): Promise<AxiosResponse<ApiResponse<VersionDistribution[]>>> {
  return request({
    url: '/v1/admin/versions/distribution',
    method: 'get',
  })
}

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
