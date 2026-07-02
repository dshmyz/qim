export interface Message {
  id: number
  conversationId: number
  senderId: number
  senderName: string
  receiverId?: number
  receiverName?: string
  groupId?: number
  groupName?: string
  channelId?: number
  channelName?: string
  messageType: 'text' | 'image' | 'file' | 'audio' | 'video'
  content: string
  createdAt: string
  updatedAt: string
}

export interface MessageSearchParams {
  keyword?: string
  senderId?: number
  receiverId?: number
  conversationType?: 'single' | 'group' | 'discussion' | 'bot'
  messageType?: string
  startTime?: string
  endTime?: string
  page: number
  pageSize: number
}

export interface MessageSearchResult {
  list: Message[]
  total: number
  page: number
  pageSize: number
}
