import type { ApiResponse, Version, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getVersions = (params?: PaginationParams & { platform?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Version>>>> => {
  return request({
    url: '/v1/versions',
    method: 'get',
    params,
  })
}

export const createVersion = (data: { version: string; platform: 'windows' | 'macos' | 'linux'; releaseDate: string; updateNotes: string; forceUpdate: boolean; downloadUrl: string }): Promise<AxiosResponse<ApiResponse<Version>>> => {
  return request({
    url: '/v1/versions',
    method: 'post',
    data,
  })
}

export const updateVersion = (id: number, data: { updateNotes?: string; forceUpdate?: boolean; status?: 'active' | 'inactive' }): Promise<AxiosResponse<ApiResponse<Version>>> => {
  return request({
    url: `/v1/versions/${id}`,
    method: 'put',
    data,
  })
}

export const deleteVersion = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/versions/${id}`,
    method: 'delete',
  })
}

export const toggleVersionStatus = (id: number, status: 'active' | 'inactive'): Promise<AxiosResponse<ApiResponse<Version>>> => {
  return request({
    url: `/v1/versions/${id}/status`,
    method: 'patch',
    data: { status },
  })
}
