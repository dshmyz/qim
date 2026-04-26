import type { ApiResponse, Role, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getRoles = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Role>>>> => {
  return request({
    url: '/v1/roles',
    method: 'get',
    params,
  })
}

export const createRole = (data: { name: string; code: string; description: string; permissions: string[] }): Promise<AxiosResponse<ApiResponse<Role>>> => {
  return request({
    url: '/v1/roles',
    method: 'post',
    data,
  })
}

export const updateRole = (id: number, data: { name?: string; description?: string; permissions?: string[] }): Promise<AxiosResponse<ApiResponse<Role>>> => {
  return request({
    url: `/v1/roles/${id}`,
    method: 'put',
    data,
  })
}

export const deleteRole = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/roles/${id}`,
    method: 'delete',
  })
}

export const getRoleUsers = (id: number, params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<{ id: number; username: string; nickname?: string; avatar?: string }>>>> => {
  return request({
    url: `/v1/roles/${id}/users`,
    method: 'get',
    params,
  })
}
