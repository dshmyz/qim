import type { ApiResponse, Conversation, ConversationMember, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface GetConversationsParams extends PaginationParams {
  type?: 'single' | 'group' | 'discussion' | 'bot'
  keyword?: string
}

export const getConversations = (params: GetConversationsParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Conversation>>>> => {
  return request({
    url: '/v1/admin/conversations',
    method: 'get',
    params,
  })
}

export const getConversationMembers = (id: number, params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<ConversationMember>>>> => {
  return request({
    url: `/v1/admin/conversations/${id}/members`,
    method: 'get',
    params,
  })
}

export const deleteConversation = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/conversations/${id}`,
    method: 'delete',
  })
}
