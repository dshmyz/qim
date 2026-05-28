import type { ApiResponse, User, UserAIConfig, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateUserParams {
  username: string
  password: string
  nickname?: string
  email: string
  avatar?: string
  phone?: string
}

export interface UpdateUserParams {
  nickname?: string
  email?: string
  phone?: string
  avatar?: string
  signature?: string
  status?: 'active' | 'inactive' | 'banned'
}

export const getUsers = (params: PaginationParams & { keyword?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<User>>>> => {
  return request({
    url: '/v1/admin/users',
    method: 'get',
    params,
  })
}

export const getUserById = (id: number): Promise<AxiosResponse<ApiResponse<User>>> => {
  return request({
    url: `/v1/users/${id}`,
    method: 'get',
  })
}

export const createUser = (data: CreateUserParams): Promise<AxiosResponse<ApiResponse<User>>> => {
  return request({
    url: '/v1/users',
    method: 'post',
    data,
  })
}

export const updateUser = (id: number, data: UpdateUserParams): Promise<AxiosResponse<ApiResponse<User>>> => {
  return request({
    url: `/v1/admin/users/${id}`,
    method: 'put',
    data,
  })
}

export const deleteUser = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/users/${id}`,
    method: 'delete',
  })
}

export const assignRoles = (userId: number, roles: string[]): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/users/${userId}/roles`,
    method: 'put',
    data: { roles },
  })
}

export const removeRole = (userId: number, role: string): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/users/${userId}/roles/${role}`,
    method: 'delete',
  })
}

export const banUser = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/users/${id}/ban`,
    method: 'post',
  })
}

export const unbanUser = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/users/${id}/unban`,
    method: 'post',
  })
}

// 管理员操作用户AI配置
export interface UpdateUserAIConfigParams {
  config_name?: string
  provider?: string
  api_key?: string
  model_name?: string
  base_url?: string
  ai_enabled?: boolean
  daily_limit?: number
  max_tokens?: number
  temperature?: number
}

export const getUserAIConfigs = (userId: number, params?: { page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<{ list: UserAIConfig[]; total: number }>>> => {
  return request({
    url: `/v1/admin/users/${userId}/ai-configs`,
    method: 'get',
    params,
  })
}

export const updateUserAIConfig = (userId: number, configId: number, data: UpdateUserAIConfigParams): Promise<AxiosResponse<ApiResponse<UserAIConfig>>> => {
  return request({
    url: `/v1/admin/users/${userId}/ai-configs/${configId}`,
    method: 'put',
    data,
  })
}
