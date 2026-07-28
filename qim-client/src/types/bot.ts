// src/types/bot.ts

export interface Bot {
  id: number
  name: string
  avatar?: string
  description?: string
  type: 'system' | 'custom' | 'assistant' | 'group_assistant'
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

// Bot 的 webhook 路由配置（存于 model.Bot.Config JSON，与 AI BotConfig 共列）。
// mode 决定用户回复走内部 AI 还是转发外部 agent webhook。
export interface BotWebhookConfig {
  mode?: 'internal_ai' | 'external_webhook'
  webhook_url?: string
  webhook_secret?: string // 仅写入，服务端不回显
}

// Bot 访问令牌信息（列表用，不含明文/hash）。
export interface BotTokenInfo {
  id: number
  name: string
  created_at: string
  last_used_at: string | null
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
