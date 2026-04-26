import type { ApiResponse, Channel, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateChannelParams {
  name: string
  description?: string
  icon?: string
  type?: 'text' | 'voice' | 'video'
}

export const getChannels = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Channel>>>> => {
  return request({
    url: '/v1/channels',
    method: 'get',
    params,
  })
}

export const createChannel = (data: CreateChannelParams): Promise<AxiosResponse<ApiResponse<Channel>>> => {
  return request({
    url: '/v1/channels',
    method: 'post',
    data,
  })
}

export const updateChannel = (id: number, data: Partial<CreateChannelParams>): Promise<AxiosResponse<ApiResponse<Channel>>> => {
  return request({
    url: `/v1/channels/${id}`,
    method: 'put',
    data,
  })
}

export const deleteChannel = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/channels/${id}`,
    method: 'delete',
  })
}
