import type { ApiResponse, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateChannelParams {
  name: string
  description?: string
  avatar?: string
  publish_permission?: 'creator_only' | 'all_subscribers'
}

export interface UpdateChannelParams {
  name?: string
  description?: string
  avatar?: string
  status?: 'active' | 'inactive'
  publish_permission?: 'creator_only' | 'all_subscribers'
}

export interface ChannelInfo {
  id: number
  name: string
  avatar: string
  description: string
  status: string
  publish_permission: 'creator_only' | 'all_subscribers'
  creatorName: string
  memberCount: number
  createdAt: string
}

export const getChannels = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<ChannelInfo>>>> => {
  return request({
    url: '/v1/admin/channels',
    method: 'get',
    params,
  })
}

export const createChannel = (data: CreateChannelParams): Promise<AxiosResponse<ApiResponse<ChannelInfo>>> => {
  return request({
    url: '/v1/channels',
    method: 'post',
    data,
  })
}

export const updateChannel = (id: number, data: UpdateChannelParams): Promise<AxiosResponse<ApiResponse<ChannelInfo>>> => {
  return request({
    url: `/v1/admin/channels/${id}`,
    method: 'put',
    data,
  })
}

export const deleteChannel = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/channels/${id}`,
    method: 'delete',
  })
}

export interface CreateChannelMessageParams {
  content: string
  type?: string
}

export const createChannelMessage = (
  id: number,
  data: CreateChannelMessageParams,
): Promise<AxiosResponse<ApiResponse<unknown>>> => {
  return request({
    url: `/v1/channels/${id}/messages`,
    method: 'post',
    data,
  })
}
