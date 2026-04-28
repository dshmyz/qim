import request from '@/utils/request'
import type { MessageSearchParams, MessageSearchResult, Message } from '@/types/message'

export function searchMessages(params: MessageSearchParams) {
  return request.get<MessageSearchResult>('/api/messages/search', { params })
}

export function getMessageDetail(id: number) {
  return request.get<Message>(`/api/messages/${id}`)
}

export function exportMessages(params: Omit<MessageSearchParams, 'page' | 'pageSize'>) {
  return request.post('/api/messages/export', params)
}

export function getExportTaskStatus(taskId: string) {
  return request.get(`/api/messages/export/${taskId}`)
}
