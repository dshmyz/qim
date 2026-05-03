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
  mode: 'offline' | 'keyword' | 'mention' | 'all' | 'custom'
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
    disclaimerStyle: 'badge'
  },
  takeoverCooldown: 10,
  customPersonaAddon: ''
}
