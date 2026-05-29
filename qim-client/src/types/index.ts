export interface User {
  id: string
  name: string
  avatar: string
  username?: string
  status?: 'online' | 'offline' | 'busy'
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
  role?: 'owner' | 'admin' | 'user' | 'guest'
  type?: 'user' | 'bot' | 'system' | 'api'
  isBot?: boolean
  tags?: string[]
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
  type: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'system' | 'markdown' | 'streaming'
  isSelf: boolean
  isRead?: boolean
  isRecalled?: boolean
  isFailed?: boolean
  isAtMention?: boolean
  isAvatarReply?: boolean
  is_avatar_reply?: boolean
  ai_type?: string
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
  originalData?: any
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
