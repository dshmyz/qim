// src/types/bot.ts

import type { AISource } from './index'

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
  use_creator_notes?: boolean // internal_ai 模式下是否读取创建者笔记作为知识库
  // 模型来源（internal_ai 模式）：true=用系统默认模型（推荐）；false 且 user_config_id 非空时用自定义配置
  use_system_config?: boolean
  user_config_id?: number | null
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
  // Bot 回复命中笔记时的知识来源（标题/分数）
  knowledge_sources?: AISource[]
}

export interface BotConversation {
  id: number
  botId: number
  userId: number
  conversationId: number
  createdAt: string
}
