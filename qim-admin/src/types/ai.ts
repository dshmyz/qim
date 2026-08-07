// AI provider management types

export type ProviderType = 'openai' | 'anthropic' | 'ollama' | 'azure' | 'custom'

export type ProviderStatus = 'connected' | 'error' | 'testing' | 'unknown'

export interface AIProvider {
  id: number
  name: string
  type: ProviderType
  apiKey: string
  apiEndpoint: string
  models: string[]
  status: ProviderStatus
  enabled: boolean
  lastTestAt?: string
  priority: number
  remark?: string
  createdAt: string
  updatedAt: string
}

export interface CreateProviderParams {
  name: string
  type: ProviderType
  apiKey: string
  apiEndpoint: string
  models: string[]
  enabled?: boolean
  priority?: number
  remark?: string
}

export interface UpdateProviderParams {
  name?: string
  type?: ProviderType
  apiKey?: string
  apiEndpoint?: string
  models?: string[]
  enabled?: boolean
  priority?: number
  remark?: string
}

export interface TestConnectionResult {
  success: boolean
  message: string
  models?: string[]
  responseTime?: number
}

// Provider type display labels
export const PROVIDER_TYPE_LABELS: Record<ProviderType, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  ollama: 'Ollama',
  azure: 'Azure OpenAI',
  custom: '自定义',
}

// Default API endpoints by provider type
export const DEFAULT_ENDPOINTS: Record<ProviderType, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  ollama: 'http://localhost:11434',
  azure: 'https://your-resource.openai.azure.com',
  custom: '',
}

// Common models by provider type
export const DEFAULT_MODELS: Record<ProviderType, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo'],
  anthropic: ['claude-3-opus', 'claude-3-sonnet', 'claude-3-haiku'],
  ollama: ['llama3', 'mistral', 'codellama', 'phi3'],
  azure: ['gpt-4o', 'gpt-4-turbo', 'gpt-35-turbo'],
  custom: [],
}

// ===== AI 模型路由（任务类型 → provider/model） =====

// 任务类型，与后端 ai.TaskType 对齐
export type TaskType =
  | 'chat'
  | 'intent_recognition'
  | 'analysis'
  | 'embedding'
  | 'tool_calling'
  | 'search'
  | 'digest'
  | 'vision'

// 单条路由规则
export interface AIRoute {
  provider: string
  model: string
  fallback?: string[]
}

// 路由配置（defaultTask + 任务→路由映射）
export interface AIRouterConfig {
  defaultTask: TaskType
  routes: Record<TaskType, AIRoute>
}

// GET /admin/ai/router 返回结构
export interface AIRouterResponse {
  defaultTask: TaskType
  routes: Record<TaskType, AIRoute>
  providers?: Array<{ id: number; name: string; type: string; models: string[] }>
  usingDb: boolean
}

// 任务类型下拉候选：值 + 中文标签 + 说明
export const TASK_TYPE_OPTIONS: Array<{ value: TaskType; label: string; desc: string }> = [
  { value: 'chat', label: 'Chat / 对话', desc: 'AI 助手日常对话' },
  { value: 'intent_recognition', label: '意图识别', desc: '判断用户意图' },
  { value: 'analysis', label: '分析 / 摘要生成', desc: '内容分析与总结' },
  { value: 'embedding', label: 'Embedding', desc: '向量化（语义检索）' },
  { value: 'tool_calling', label: '工具调用', desc: 'AI 调用内置工具' },
  { value: 'search', label: '搜索', desc: '知识库 / 语义检索' },
  { value: 'digest', label: '摘要 / 聚合', desc: '消息摘要、聚合总结' },
  { value: 'vision', label: '视觉理解', desc: '图片 / 多模态理解' },
]

// 直接可编辑的任务类型（embedding 为向量服务专用，通常无需改路由，但仍保留可编辑）
export const ROUTABLE_TASK_OPTIONS = TASK_TYPE_OPTIONS
