export interface UserAIConfig {
  id: number
  config_name: string
  provider: string
  model_name: string
  base_url: string
  temperature: number
  max_tokens: number
  is_verified: boolean
  last_tested_at: string | null
  created_at: string
}

export interface CreateConfigRequest {
  config_name: string
  provider: string
  api_key: string
  model_name: string
  base_url?: string
}

export interface AIProvider {
  id: string
  name: string
  icon: string
  defaultModel: string
  defaultBaseURL: string
}

export const AI_PROVIDERS: AIProvider[] = [
  {
    id: 'openai',
    name: 'OpenAI',
    icon: 'fab fa-openai',
    defaultModel: 'gpt-4o',
    defaultBaseURL: 'https://api.openai.com/v1'
  },
  {
    id: 'alibaba',
    name: '\u963f\u91cc\u901a\u4e49\u5343\u95ee',
    icon: 'fas fa-wand-magic-sparkles',
    defaultModel: 'qwen-turbo',
    defaultBaseURL: 'https://dashscope.aliyuncs.com/api/v1'
  },
  {
    id: 'tencent',
    name: '\u817e\u8baf\u6df7\u5143',
    icon: 'fab fa-tencent',
    defaultModel: 'hunyuan-lite',
    defaultBaseURL: 'https://hunyuan.tencentcloudapi.com'
  },
  {
    id: 'bytedance',
    name: '\u5b57\u8282\u8c46\u5305',
    icon: 'fas fa-bullseye',
    defaultModel: 'doubao-pro-32k',
    defaultBaseURL: 'https://ark.cn-beijing.volces.com/api/v3'
  },
  {
    id: 'anthropic',
    name: 'Anthropic Claude',
    icon: 'fas fa-brain',
    defaultModel: 'claude-3-5-sonnet-20241022',
    defaultBaseURL: 'https://api.anthropic.com/v1'
  },
  {
    id: 'deepseek',
    name: 'DeepSeek',
    icon: 'fas fa-magnifying-glass',
    defaultModel: 'deepseek-chat',
    defaultBaseURL: 'https://api.deepseek.com/v1'
  }
]

export interface GroupAISettings {
  aiEnabled: boolean
  aiAssistantName: string
  aiReplyMode: string
  aiPersonality: string
  aiCustomPrompt: string
  aiLanguage: string
  aiMaxLength: string
  aiMentionReplyMode: string
  aiAntiSpamInterval: number
  aiTriggerKeywords: string[]
  aiLearnEnabled: boolean
  aiExtractTodos: boolean
}

export interface GroupDocument {
  id: number
  group_id: number
  file_id: number
  created_at: string
  file?: {
    id: number
    name: string
    size: number
    type: string
  }
  process_status?: 'pending' | 'processing' | 'completed' | 'failed'
  process_error?: string
  chunk_count?: number
}

// 群助手群级记忆条目
export interface GroupMemory {
  doc_id: string
  content: string
  metadata?: {
    conversation_id?: string
    remembered_at?: string
  }
  score?: number
}
