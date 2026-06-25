// src/types/bot.ts

export interface Bot {
  id: number
  name: string
  avatar?: string
  description?: string
  type: 'system' | 'custom' | 'assistant'
  config?: BotConfig
  approvalStatus: 'pending' | 'approved' | 'rejected'
  creatorId: number
  creatorName: string
  virtualUserId?: number
  isActive: boolean
  createdAt: string
  updatedAt: string
}

export interface BotConfig {
  systemPrompt?: string
  temperature?: number
  maxTokens?: number
  model?: string
}

export interface BotMessage {
  id: number
  conversationId: number | null
  senderId: number
  senderType: 'user' | 'bot' | 'system' | 'api'
  sender?: {
    id: number
    nickname: string
    avatar?: string
    type: string
  }
  type: 'text' | 'markdown'
  content: string
  timestamp: Date
  isStreaming?: boolean
}

export interface BotConversation {
  id: number
  botId: number
  userId: number
  conversationId: number
  createdAt: string
}
