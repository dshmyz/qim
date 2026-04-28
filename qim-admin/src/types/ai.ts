// AI provider and model management types

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

export interface AIModel {
  id: number
  providerId: number
  providerName: string
  modelId: string
  name: string
  description?: string
  contextWindow: number
  maxTokens: number
  supportsVision: boolean
  supportsFunctionCall: boolean
  status: 'active' | 'inactive'
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

export interface CreateModelParams {
  providerId: number
  modelId: string
  name: string
  description?: string
  contextWindow: number
  maxTokens: number
  supportsVision: boolean
  supportsFunctionCall: boolean
}

export interface UpdateModelParams {
  name?: string
  description?: string
  contextWindow?: number
  maxTokens?: number
  supportsVision?: boolean
  supportsFunctionCall?: boolean
  status?: 'active' | 'inactive'
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

// AI Configuration
export interface AIConfig {
  id: number
  defaultProvider: string
  defaultModel: string
  temperature: number
  maxTokens: number
  topP: number
  frequencyPenalty: number
  presencePenalty: number
  timeout: number
}

// AI Quota
export interface AIQuota {
  id: number
  targetType: 'user' | 'role'
  targetId: number
  dailyLimit: number
  tokenLimit: number
  concurrentLimit: number
  overlimitStrategy: 'reject' | 'degrade' | 'notify'
}

// AI Usage
export interface AIUsage {
  id: number
  userId: number
  provider: string
  model: string
  promptTokens: number
  completionTokens: number
  totalTokens: number
  cost: number
  requestTime: number
  status: 'success' | 'error' | 'timeout'
  errorMessage?: string
  createdAt: string
}

// AI Usage Statistics
export interface AIUsageStatistics {
  totalCalls: number
  totalTokens: number
  totalCost: number
  byUser: Array<{
    userId: number
    userName: string
    calls: number
    tokens: number
    cost: number
  }>
  byModel: Array<{
    model: string
    calls: number
    tokens: number
    cost: number
  }>
}
