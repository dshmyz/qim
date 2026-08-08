export interface User {
  id: string
  name: string
  avatar: string
  username?: string
  status?: 'online' | 'offline' | 'busy' | 'disabled'
  email?: string
  phone?: string
  nickname?: string
  gender?: 'male' | 'female' | 'other'
  birthday?: string
  bio?: string
  ip?: string
  location?: string
  createdAt?: number
  lastOnline?: number
  disabled?: boolean
  is_disabled?: boolean
  is_deleted?: boolean
  deletedAt?: string | number
  deleted_at?: string | number
  role?: 'owner' | 'admin' | 'user' | 'guest'
  type?: 'user' | 'bot' | 'system' | 'api'
  isBot?: boolean
  tags?: string[]
  // 后端 /api/v1/users/me 实际返回，部分模块依赖
  isAdmin?: boolean
  roles?: string[]
  signature?: string
  preferences?: {
    theme?: 'light' | 'dark' | 'system'
    language?: string
    notifications?: boolean
    autoAcceptCalls?: boolean
  }
}

export interface Message {
  id: string
  content: string
  sender: User
  timestamp: number
  type: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'system' | 'markdown' | 'streaming' | 'merged_forward'
  isSelf: boolean
  isRead?: boolean
  isRecalled?: boolean
  isFailed?: boolean
  isAtMention?: boolean
  mention_user_ids?: number[]
  isAvatarReply?: boolean
  is_avatar_reply?: boolean
  origin?: string
  isAIMessage?: boolean
  is_ai_message?: boolean
  ai_assistant_name?: string
  conversationId: string
  quotedMessage?: Message
  miniAppData?: {
    id: string
    name: string
    icon: string
    description: string
    path: string
  }
  newsData?: {
    title: string
    summary: string
    image: string
    url: string
  }
  shareData?: any
  isStreaming?: boolean
  // 卡片已点击的 action_id（服务端从 CardActionRecord 派生，跨设备一致）
  cardActionId?: string
  originalData?: any
  // 文件类消息附件信息（后端实际返回，部分模块依赖）
  file_name?: string
  file_size?: number
  file_url?: string
  avatar_name?: string
  // 分身回复的免责声明展示样式（badge/footer/both），来自 owner 的 ReplyStrategy.disclaimerStyle
  disclaimer_style?: string
  // Bot 回复命中创建者笔记时的知识来源（标题/分数），用于折叠「知识来源」标签
  knowledge_sources?: KnowledgeSource[]
  // 分身回复命中的知识来源（笔记/群知识/记忆），随 WS 实时下发，用于展示「依据」
  sources?: AvatarSource[]
  // 消息附加信息（JSON 字符串），撤回时保存原始内容用于重新编辑
  extra?: string
  // 外部工具调用记录：前端据此在气泡下方渲染独立工具调用卡片（不拼进正文）。
  // 实时由 ai_tool_call WS 事件追加，回放从 Extra 解析。参考 capability-console。
  tool_calls?: ToolCallRecord[]
}

// ToolCallRecord 外部 AI 工具调用的一条结构化记录（后端 ai_tool_call 事件 / Extra 持久化）。
export interface ToolCallRecord {
  tool_label: string
  args?: Record<string, unknown>
  status?: string // '' | 'ok' | 'error'
}

// KnowledgeSource Bot 回复命中笔记的最小展示结构
export interface KnowledgeSource {
  title: string
  score: number
}

// AvatarSource 分身回复命中的知识来源，用于在气泡下展示「依据」
export interface AvatarSource {
  type: 'note' | 'group' | 'memory'
  title?: string
  snippet?: string
}

export interface Conversation {
  id: string
  name: string
  avatar: string
  lastMessage?: Message
  unread_count: number
  timestamp: number
  type: 'single' | 'group' | 'bot' | 'discussion'
  members?: User[]
  muted?: boolean
  is_pinned?: boolean
  pinned?: boolean
  pinnedAt?: number
  ip?: string
  status?: 'online' | 'offline' | 'busy'
  signature?: string
  announcement?: string
  other_member_id?: number
  other_member_name?: string
  creator_id?: string | number
  owner_id?: string | number
  ownerId?: string | number
  user_id?: number
  is_deleted?: boolean
  ai_config?: {
    ai_enabled?: boolean
    ai_assistant_name?: string
    ai_reply_mode?: string
    ai_personality?: string
    ai_custom_prompt?: string
    ai_language?: string
    ai_max_length?: string
    ai_mention_reply_mode?: string
    ai_anti_spam_interval?: number
    ai_trigger_keywords?: string
    ai_learn_enabled?: boolean
    ai_extract_todos?: boolean
  }
  approval_status?: 'pending' | 'approved' | 'rejected'
  applied_at?: string
  approved_at?: string
  reject_reason?: string
  context_messages?: number
  invite_permission?: 'owner_admin' | 'all'
}

export interface Channel {
  id: string
  name: string
  description: string
  avatar: string
  creator_id: string
  status: string
  publish_permission: 'creator_only' | 'all_subscribers'
  comment_permission: 'all_subscribers' | 'disabled'
  created_at: number
  is_subscribed?: boolean
  is_default?: boolean
  creator?: User
  messages?: ChannelMessage[]
  subscriber_count?: number
  last_active_at?: number
  last_message?: ChannelMessage
  unread_count?: number
  category?: string
}

export interface ChannelMessage {
  id: string
  channel_id: string
  sender_id: string
  content: string
  type: string
  created_at: number
  sender?: User
}

export interface SystemMessage {
  id: string
  title: string
  content: string
  sender_id: string
  status: string
  target_type?: string
  target_id?: string
  created_at: number
  sender?: User
}

export interface Notification {
  id: string
  user_id: string
  type: string
  title: string
  content: string
  read: boolean
  read_at?: number
  created_at: number
}

export * from './ai'
export * from './avatar'
export * from './bot'
export * from './task'
