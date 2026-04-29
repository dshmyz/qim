export interface User {
  id: string
  name: string
  avatar: string
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
  role?: 'admin' | 'user' | 'guest'
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
}

export interface Conversation {
  id: string
  name: string
  avatar: string
  lastMessage?: Message
  unreadCount: number
  timestamp: number
  type: 'single' | 'group' | 'bot' | 'discussion'
  members?: User[]
  muted?: boolean
  pinned?: boolean
  ip?: string
  status?: 'online' | 'offline' | 'busy'
  signature?: string
  announcement?: string
  other_member_id?: number
  other_member_name?: string
  user_id?: number
}

export interface Channel {
  id: string
  name: string
  description: string
  avatar: string
  creator_id: string
  status: string
  created_at: number
  is_subscribed?: boolean
  creator?: User
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
