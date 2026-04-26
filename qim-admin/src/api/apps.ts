import type { ApiResponse, App, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateAppParams {
  name: string
  icon?: string
  category: string
  url: string
  openType: 'in-app' | 'external'
}

export interface UpdateAppParams {
  name?: string
  icon?: string
  category?: string
  url?: string
  openType?: 'in-app' | 'external'
  status?: 'active' | 'inactive'
}

export const getApps = (params: PaginationParams & { name?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<App>>>> => {
  return request({
    url: '/v1/apps',
    method: 'get',
    params,
  })
}

export const createApp = (data: CreateAppParams): Promise<AxiosResponse<ApiResponse<App>>> => {
  return request({
    url: '/v1/apps',
    method: 'post',
    data,
  })
}

export const updateApp = (id: number, data: UpdateAppParams): Promise<AxiosResponse<ApiResponse<App>>> => {
  return request({
    url: `/v1/apps/${id}`,
    method: 'put',
    data,
  })
}

export const deleteApp = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/apps/${id}`,
    method: 'delete',
  })
}
