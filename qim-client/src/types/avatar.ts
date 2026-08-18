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
  selfMessagePause: number // 你发消息后，分身暂停回复的时间（分钟），0=关闭

  // 无显式会话级 session 时，分身是否默认在该会话激活（true=广覆盖，false=逐会话 opt-in）
  activateByDefault: boolean

  createdAt: string
  updatedAt: string
}

export interface AvatarKnowledgeScope {
  conversationHistory: boolean
  memory: boolean // 分身长期记忆
  knowledgeDocs: boolean
  notes: boolean
  tasks: boolean
}

export interface AvatarTriggerRules {
  // 触发时机（门）
  offlineOnly: boolean // 仅离线才回
  // 触发意图（门，可多选组合）
  requireMention: boolean // 群里被@才回（私聊无 @ 语义，自动降级为智能判断）
  smartDecide: boolean // LLM 判断该不该回
  keywordOnly: boolean // 关键词命中才回
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
  groupReplyTarget: 'group' | 'private' // 群聊回复落点：group=回群内，private=回触发者私聊
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
  lastLearnedAt?: string | null
}

export interface CreateAvatarConfigRequest {
  name: string
  useSystemConfig: boolean
  modelConfigId: number | null
  triggerRules: AvatarTriggerRules
  knowledgeScope: AvatarKnowledgeScope
  replyStrategy: AvatarReplyStrategy
  takeoverCooldown: number
  selfMessagePause: number
  activateByDefault: boolean
  customPersonaAddon: string
}

export const DEFAULT_AVATAR_CONFIG: CreateAvatarConfigRequest = {
  name: 'AI分身',
  useSystemConfig: true,
  modelConfigId: null,
  triggerRules: {
    offlineOnly: false,
    requireMention: true, // 默认「群里被 @ 才回」，对齐旧默认 mention
    smartDecide: false,
    keywordOnly: false,
    keywords: [],
    timeRanges: [],
    excludedConversations: []
  },
  knowledgeScope: {
    conversationHistory: true,
    memory: true,
    knowledgeDocs: false,
    notes: false,
    tasks: false
  },
  replyStrategy: {
    maxReplyLength: 'medium',
    replyDelay: 3,
    confidenceThreshold: 0.6,
    disclaimerStyle: 'badge',
    replyOutOfScope: false,
    groupReplyTarget: 'group'
  },
  takeoverCooldown: 10,
  selfMessagePause: 0,
  activateByDefault: true,
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
