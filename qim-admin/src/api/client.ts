import type { ApiResponse, PaginationParams, PaginatedResponse } from '@/types'
import type { ClientVersion, CrashLog, UserFeedback, VersionDistribution } from '@/types/client'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getVersions = (params?: PaginationParams & { platform?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<ClientVersion>>>> => {
  return request({
    url: '/v1/admin/versions',
    method: 'get',
    params,
  })
}

export const createVersion = (data: Partial<ClientVersion>): Promise<AxiosResponse<ApiResponse<ClientVersion>>> => {
  return request({
    url: '/v1/admin/versions',
    method: 'post',
    data,
  })
}

export const updateVersion = (id: number, data: Partial<ClientVersion>): Promise<AxiosResponse<ApiResponse<ClientVersion>>> => {
  return request({
    url: `/v1/admin/versions/${id}`,
    method: 'put',
    data,
  })
}

export const deleteVersion = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/versions/${id}`,
    method: 'delete',
  })
}

export const getVersionDistribution = (): Promise<AxiosResponse<ApiResponse<VersionDistribution[]>>> => {
  return request({
    url: '/v1/admin/versions/distribution',
    method: 'get',
  })
}

export const getCrashLogs = (params: PaginationParams & { platform?: string; appVersion?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<CrashLog>>>> => {
  return request({
    url: '/v1/admin/crashes',
    method: 'get',
    params,
  })
}

export const getCrashDetail = (id: number): Promise<AxiosResponse<ApiResponse<CrashLog>>> => {
  return request({
    url: `/v1/admin/crashes/${id}`,
    method: 'get',
  })
}

export const getFeedbacks = (params: PaginationParams & { status?: string; type?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<UserFeedback>>>> => {
  return request({
    url: '/v1/admin/feedbacks',
    method: 'get',
    params,
  })
}

export const updateFeedback = (id: number, data: { status?: UserFeedback['status']; reply?: string }): Promise<AxiosResponse<ApiResponse<UserFeedback>>> => {
  return request({
    url: `/v1/admin/feedbacks/${id}`,
    method: 'put',
    data,
  })
}
