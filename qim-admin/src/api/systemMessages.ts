import type { ApiResponse, SystemMessage, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateSystemMessageParams {
  title: string
  content: string
  target_type?: string
  target_id?: number
}

export interface UpdateSystemMessageParams {
  title?: string
  content?: string
  type?: 'notification' | 'warning' | 'info'
  priority?: 'low' | 'medium' | 'high'
  status?: 'published' | 'draft'
}

export const getSystemMessages = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<SystemMessage>>>> => {
  return request({
    url: '/v1/system-messages',
    method: 'get',
    params,
  })
}

export const createSystemMessage = (data: CreateSystemMessageParams): Promise<AxiosResponse<ApiResponse<SystemMessage>>> => {
  return request({
    url: '/v1/system-messages',
    method: 'post',
    data,
  })
}

export const updateSystemMessage = (id: number, data: UpdateSystemMessageParams): Promise<AxiosResponse<ApiResponse<SystemMessage>>> => {
  return request({
    url: `/v1/system-messages/${id}`,
    method: 'put',
    data,
  })
}

export const deleteSystemMessage = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/system-messages/${id}`,
    method: 'delete',
  })
}
