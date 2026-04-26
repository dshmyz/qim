import type { ApiResponse, MiniApp, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateMiniAppParams {
  appID: string
  name: string
  icon?: string
  path: string
  description?: string
}

export interface UpdateMiniAppParams {
  name?: string
  icon?: string
  path?: string
  description?: string
  status?: 'active' | 'inactive'
}

export const getMiniApps = (params: PaginationParams & { name?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<MiniApp>>>> => {
  return request({
    url: '/v1/mini-apps',
    method: 'get',
    params,
  })
}

export const createMiniApp = (data: CreateMiniAppParams): Promise<AxiosResponse<ApiResponse<MiniApp>>> => {
  return request({
    url: '/v1/mini-apps',
    method: 'post',
    data,
  })
}

export const updateMiniApp = (id: number, data: UpdateMiniAppParams): Promise<AxiosResponse<ApiResponse<MiniApp>>> => {
  return request({
    url: `/v1/mini-apps/${id}`,
    method: 'put',
    data,
  })
}

export const deleteMiniApp = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/mini-apps/${id}`,
    method: 'delete',
  })
}
