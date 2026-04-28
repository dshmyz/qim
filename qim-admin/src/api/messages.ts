import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import type { MessageSearchParams, MessageSearchResult, Message } from '@/types/message'

export function searchMessages(params: MessageSearchParams): Promise<AxiosResponse<ApiResponse<MessageSearchResult>>> {
  return request({
    url: '/messages/search',
    method: 'get',
    params,
  })
}

export function getMessageDetail(id: number): Promise<AxiosResponse<ApiResponse<Message>>> {
  return request({
    url: `/messages/${id}`,
    method: 'get',
  })
}

export function exportMessages(params: Omit<MessageSearchParams, 'page' | 'pageSize'>): Promise<AxiosResponse<ApiResponse<void>>> {
  return request({
    url: '/messages/export',
    method: 'post',
    data: params,
  })
}

export function getExportTaskStatus(taskId: string): Promise<AxiosResponse<ApiResponse<{ status: string; downloadUrl: string }>>> {
  return request({
    url: `/messages/export/${taskId}`,
    method: 'get',
  })
}
