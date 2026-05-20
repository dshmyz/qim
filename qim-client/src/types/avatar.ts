export interface AvatarConfig {
  id: number
  userId: number
  name: string
  enabled: boolean

  autoLearnedPersona: string
  customPersonaAddon: string
  personaVersion: number
  lastLearnedAt: string | null

  knowledgeScope: AvatarKnowledgeScope
  triggerRules: AvatarTriggerRules
  replyStrategy: AvatarReplyStrategy

  modelConfigId: number | null
  useSystemConfig: boolean

  takeoverCooldown: number

  createdAt: string
  updatedAt: string
}

export interface AvatarKnowledgeScope {
  conversationHistory: boolean
  knowledgeDocs: boolean
  notes: boolean
  tasks: boolean
}

export interface AvatarTriggerRules {
  mode: 'mention' | 'offline' | 'keyword' | 'all' | 'smart'
  keywords: string[]
  timeRanges: AvatarTimeRange[]
  excludedConversations: number[]
}

export interface AvatarTimeRange {
  dayOfWeek: number[]
  startHour: number
  endHour: number
}

export interface AvatarReplyStrategy {
  maxReplyLength: 'short' | 'medium' | 'long'
  replyDelay: number
  confidenceThreshold: number
  disclaimerStyle: 'badge' | 'footer' | 'both'
  replyOutOfScope: boolean // 是否回复知识范围外的消息，false 时静默跳过
}

export interface AvatarSession {
  conversationId: number
  avatarEnabled: boolean
  takeoverUntil: string | null
  lastReplyAt: string | null
}

export interface AvatarLearnStatus {
  status: 'idle' | 'learning' | 'completed' | 'failed'
  progress: number
  messageCount: number
  error: string | null
}

export interface CreateAvatarConfigRequest {
  name: string
  useSystemConfig: boolean
  modelConfigId: number | null
  triggerRules: AvatarTriggerRules
  knowledgeScope: AvatarKnowledgeScope
  replyStrategy: AvatarReplyStrategy
  takeoverCooldown: number
  customPersonaAddon: string
}

export const DEFAULT_AVATAR_CONFIG: CreateAvatarConfigRequest = {
  name: '我的分身',
  useSystemConfig: true,
  modelConfigId: null,
  triggerRules: {
    mode: 'mention',
    keywords: [],
    timeRanges: [],
    excludedConversations: []
  },
  knowledgeScope: {
    conversationHistory: true,
    knowledgeDocs: false,
    notes: false,
    tasks: false
  },
  replyStrategy: {
    maxReplyLength: 'medium',
    replyDelay: 3,
    confidenceThreshold: 0.6,
    disclaimerStyle: 'badge',
    replyOutOfScope: false
  },
  takeoverCooldown: 10,
  customPersonaAddon: ''
}

// Avatar 审批状态类型
export type AvatarApprovalStatus = 'none' | 'pending' | 'approved' | 'rejected'

// 扩展 AvatarConfig 添加审批相关字段
export interface AvatarConfigWithApproval extends AvatarConfig {
  approvalStatus: AvatarApprovalStatus
  approvalRejectedReason?: string
  approvalAppliedAt?: string
  approvalReviewedAt?: string
}

// Avatar 审批申请记录（管理员视角）
export interface AvatarApprovalRecord {
  id: number
  userId: number
  username: string
  nickname: string
  avatar?: string
  status: AvatarApprovalStatus
  appliedAt: string
  reviewedAt?: string
  reviewedBy?: number
  reviewerName?: string
  rejectedReason?: string
}

// Avatar 工具绑定 - 用于 Avatar 与 AI工具的关联
export interface AvatarToolBinding {
  avatarId: string
  toolId: string
  enabled: boolean
  priority: number
}

// AI工具类型
export interface AITool {
  id: string
  name: string
  description?: string
  enabled: boolean
  icon?: string
}

// AvatarPersona 类型
export interface AvatarPersona {
  autoLearnedPersona: string
  customPersonaAddon: string
  personaVersion: number
  lastLearnedAt: string | null
}

// 带工具的Avatar - 包含可用工具列表的Avatar视图
export interface AvatarWithTools {
  id: string
  enabled: boolean
  persona: AvatarPersona
  availableTools: AITool[]
  lastActiveAt: Date
}
