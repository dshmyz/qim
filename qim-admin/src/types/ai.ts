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
