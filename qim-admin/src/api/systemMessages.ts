import type { ApiResponse, SystemMessage, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface CreateSystemMessageParams {
  target_ids?: number[]
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

export interface BroadcastChatParams {
  content: string
  target_user_ids?: number[]
  exclude_user_ids?: number[]
}

// 群发私聊：以系统账号向用户(默认全员，可指定)的单聊会话发送普通私聊消息。
// 消息会出现在目标用户的「最近会话」列表中（区别于 createSystemMessage 的通知红点）。
export const broadcastChat = (data: BroadcastChatParams): Promise<AxiosResponse<ApiResponse<{ total: number; sent: number; failed: number; skipped: number }>>> => {
  return request({
    url: '/v1/system-messages/broadcast-chat',
    method: 'post',
    data,
  })
}
