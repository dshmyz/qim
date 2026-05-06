import type { ApiResponse, Role, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getRoles = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Role>>>> => {
  return request({
    url: '/v1/admin/roles',
    method: 'get',
    params,
  })
}

export const createRole = (data: { user_id: number; role: string }): Promise<AxiosResponse<ApiResponse<{ id: number; user_id: number; role: string }>>> => {
  return request({
    url: '/v1/admin/roles',
    method: 'post',
    data,
  })
}

export const updateRole = (id: number, data: { role?: string }): Promise<AxiosResponse<ApiResponse<{ id: number; role: string }>>> => {
  return request({
    url: `/v1/admin/roles/${id}`,
    method: 'put',
    data,
  })
}

export const deleteRole = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/roles/${id}`,
    method: 'delete',
  })
}

export const getRoleUsers = (role: string, params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<{ id: number; username: string; nickname?: string; avatar?: string }>>>> => {
  return request({
    url: `/v1/admin/roles/${role}/users`,
    method: 'get',
    params,
  })
}
