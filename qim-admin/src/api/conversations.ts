import type { ApiResponse, Conversation, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface GetConversationsParams extends PaginationParams {
  type?: 'single' | 'group' | 'discussion'
  keyword?: string
}

export const getConversations = (params: GetConversationsParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<Conversation>>>> => {
  return request({
    url: '/v1/conversations',
    method: 'get',
    params,
  })
}

export const deleteConversation = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/conversations/${id}`,
    method: 'delete',
  })
}
