export interface LoginParams {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  user: UserInfo
}

export interface UserInfo {
  id: number
  username: string
  email: string
  avatar: string
  role: string
  createdAt: string
}

export interface User {
  id: number
  username: string
  nickname?: string
  email: string
  phone: string
  avatar?: string
  status: 'active' | 'inactive' | 'banned'
  roles?: string[]
  role?: string
  createdAt: string
  updatedAt: string
}

export interface Organization {
  id: number
  name: string
  code?: string
  parentId: number | null
  leaderId?: number | null
  description?: string
  status?: 'active' | 'inactive'
  createdAt?: string
  updatedAt?: string
  children?: Organization[]
}

export interface Group {
  id: number
  name: string
  description: string
  ownerId: number
  avatar?: string
  memberCount: number
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
}

export interface SystemMessage {
  id: number
  title: string
  content: string
  type: 'notification' | 'warning' | 'info'
  priority?: 'low' | 'medium' | 'high'
  target_type?: string
  target_id?: number
  senderId?: number
  sender?: any
  readCount?: number
  status: 'published' | 'draft' | 'active'
  createdAt: string
  updatedAt?: string
}

export interface Channel {
  id: number
  name: string
  type: 'text' | 'voice' | 'video'
  description: string
  icon?: string
  memberCount: number
  status: 'active' | 'inactive'
  createdAt: string
  updatedAt: string
}

export interface ConversationMember {
  id: number
  userId: number
  username: string
  nickname?: string
  avatar?: string
  role?: 'owner' | 'admin' | 'member'
  joinedAt: string
}

export interface BlacklistEntry {
  id: number
  userId: number
  username: string
  reason: string
  operatorId: number
  status: 'active' | 'removed'
  createdAt: string
  updatedAt: string
}

export interface StatisticsData {
  totalUsers: number
  activeUsers: number
  totalGroups: number
  totalChannels: number
  messagesToday: number
  growthRate: {
    users: number
    groups: number
    messages: number
  }
}

export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PaginationParams {
  page: number
  pageSize: number
}

export interface PaginatedResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface DashboardData {
  totalUsers: number
  onlineUsers: number
  totalGroups: number
  totalMessages: number
}

export interface RecentRegistration {
  id: number
  username: string
  email: string
  avatar: string
  createdAt: string
}

export interface App {
  id: number
  name: string
  icon?: string
  category: string
  url: string
  openType: 'in-app' | 'external'
  status: 'active' | 'inactive'
  isGlobal: boolean
  scopeType?: string       // 'all' | 'users' | 'organizations' | 'roles'
  scopeValue?: string      // 具体范围值（逗号分隔ID列表）
  availableOrgIDs?: string // 可用组织ID列表
  createdAt: string
  updatedAt: string
}

export interface MiniApp {
  id: number
  appID: string
  name: string
  icon?: string
  path: string
  description: string
  status: 'active' | 'inactive'
  permissions?: string
  createdAt: string
  updatedAt: string
}

export interface Notification {
  id: number
  type: 'notification' | 'warning' | 'info'
  title: string
  content: string
  isRead: boolean
  createdAt: string
  updatedAt: string
}

export interface Conversation {
  id: number
  type: 'single' | 'group' | 'discussion'
  name: string
  creatorId: number
  creatorName: string
  memberCount: number
  isPinned: boolean
  lastMessageAt: string
  createdAt: string
}

// 角色相关
export interface Role {
  id: number
  name: string
  code: string
  description: string
  permissions: string[]
  userCount: number
  createdAt: string
}

// 敏感词相关
export interface SensitiveWord {
  id: number
  word: string
  category: string
  level: 'low' | 'medium' | 'high'
  status: 'active' | 'inactive'
  createdAt: string
}

// 操作日志相关
export interface OperationLog {
  id: number
  operatorId: number
  operatorName: string
  action: string
  targetType: string
  targetId: number
  detail: string
  ip: string
  createdAt: string
}

// 系统配置相关
export interface SystemConfig {
  messageRecallTime: number
  maxFileSize: number
  imageQuality: number
  enableRegistration: boolean
  enable2FA: boolean
  enableFileUpload: boolean
  enableAI: boolean
  enableReadReceipt: boolean
}

// 版本管理相关
export interface Version {
  id: number
  version: string
  platform: 'windows' | 'macos' | 'linux'
  releaseDate: string
  updateNotes: string
  forceUpdate: boolean
  downloadUrl: string
  status: 'active' | 'inactive'
}

// AI 助手相关
export interface AIBot {
  id: number
  name: string
  avatar: string
  description: string
  systemPrompt: string
  status: 'active' | 'inactive'
  conversationCount: number
  createdAt: string
  // 新增审批相关字段
  approvalStatus?: 'pending' | 'approved' | 'rejected'
  creatorId?: number
  creatorName?: string
  creatorAvatar?: string
  rejectReason?: string
  isTemplate?: boolean
  creatorBotCount?: number
}

export interface BotApprovalItem {
  id: number
  name: string
  avatar: string
  description: string
  type: string
  creatorName: string
  creatorAvatar: string
  creatorBotCount: number
  approvalStatus: 'pending' | 'approved' | 'rejected'
  rejectReason?: string
  createdAt: string
}

export interface AIUsageLog {
  id: number
  userId: number
  botId: number
  messagePreview: string
  callType: string
  createdAt: string
}

// 权限和路由相关
export interface Permission {
  resource: string
  actions: string[]
}

export interface RouteMeta {
  title: string
  requiresAuth?: boolean
  permission?: string  // format: resource:action
  icon?: string
}
