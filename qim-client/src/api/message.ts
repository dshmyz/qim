import { http } from './core'
import type { ApiResponse } from './core'

export interface Message {
  id: string
  conversation_id: string
  sender_id: string
  sender?: {
    id: string
    name: string
    avatar: string
  }
  type: 'text' | 'image' | 'file' | 'video' | 'audio' | 'markdown' | 'miniApp' | 'news'
  content: string
  file_name?: string
  file_size?: number
  file_url?: string
  thumbnail_url?: string
  timestamp: number
  isRecalled?: boolean
  isSelf?: boolean
  isRead?: boolean
  isFailed?: boolean
  quotedMessage?: Message
  miniAppData?: any
  newsData?: any
  metadata?: Record<string, any>
}

export interface MessageListParams {
  page?: number
  page_size?: number
  before?: number
  after?: number
}

export interface MessageListResponse {
  messages: Message[]
  pagination: {
    current_page: number
    total_pages: number
    total: number
    page_size: number
  }
}

export interface MessageFilterParams {
  conversation_id: string
  type?: string
  page?: number
  page_size?: number
  search?: string
  start_date?: string
  end_date?: string
}

export interface MessageFilterResponse {
  messages: Message[]
  total: number
}

export interface SendMessageRequest {
  type: 'text' | 'image' | 'file' | 'video' | 'audio' | 'markdown' | 'miniApp' | 'news'
  content: string
  quoted_message_id?: number
  file_size?: number
  file_name?: string
}

class MessageAPI {
  async getMessages(conversationId: string, params?: MessageListParams): Promise<MessageListResponse> {
    const response = await http.get<ApiResponse<MessageListResponse>>(
      `/api/v1/conversations/${conversationId}/messages`,
      { params }
    )
    return response.data
  }

  async getMessagesByFilter(params: MessageFilterParams): Promise<MessageFilterResponse> {
    const response = await http.get<ApiResponse<MessageFilterResponse>>(
      '/api/v1/messages',
      { params }
    )
    return response.data
  }

  async sendMessage(conversationId: string, data: SendMessageRequest): Promise<Message> {
    const response = await http.post<ApiResponse<Message>>(
      `/api/v1/conversations/${conversationId}/messages`,
      data
    )
    return response.data
  }

  async recallMessage(messageId: string): Promise<void> {
    await http.post(`/api/v1/messages/${messageId}/recall`)
  }

  async deleteMessage(messageId: string): Promise<void> {
    await http.delete(`/api/v1/messages/${messageId}`)
  }

  async getMessageReadUsers(messageId: string): Promise<{
    read_users: Array<{ id: string; name: string; avatar: string; read_at: number }>
    total_members: number
    total_read: number
  }> {
    const response = await http.get<ApiResponse<any>>(`/api/v1/messages/${messageId}/read-users`)
    return response.data
  }

  async markAsRead(conversationId: string): Promise<void> {
    await http.put(`/api/v1/conversations/${conversationId}/read`)
  }

  async searchMessages(params: {
    query: string
    conversation_id?: string
    start_time?: number
    end_time?: number
    sender_id?: string
    type?: string
    page?: number
    page_size?: number
  }): Promise<{
    messages: Message[]
    total: number
  }> {
    const response = await http.get<ApiResponse<any>>('/api/v1/messages/search', { params })
    return response.data
  }

  async batchOperation(messageIds: string[], action: 'delete' | 'archive' | 'mark_read'): Promise<{
    success_count: number
    failed_count: number
  }> {
    const response = await http.post<ApiResponse<any>>('/api/v1/messages/batch', {
      message_ids: messageIds,
      action
    })
    return response.data
  }
}

export const messageApi = new MessageAPI()
