import { http } from './core'
import type { ApiResponse } from '../composables/useRequest'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

// 统一走 core.ts 的 http 封装：401 静默刷新、X-App-Version/Platform 头注入、
// ApiError 规范化（含 response.data.message 提取）均由拦截器处理，本文件不再手写。

export const aiConfigAPI = {
  async listMyConfigs(): Promise<UserAIConfig[]> {
    const res = await http.get<ApiResponse<any>>('/api/v1/ai/configs/my')
    const data = res.data
    return data?.list ?? (Array.isArray(data) ? data : [])
  },

  async createConfig(data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const res = await http.post<ApiResponse<{ id: number; is_verified: boolean }>>('/api/v1/ai/configs/my', data)
    return res.data
  },

  async updateConfig(id: number, data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const res = await http.put<ApiResponse<{ id: number; is_verified: boolean }>>(`/api/v1/ai/configs/my/${id}`, data)
    return res.data
  },

  async deleteConfig(id: number): Promise<void> {
    await http.delete(`/api/v1/ai/configs/my/${id}`)
  },

  async testConfig(id: number): Promise<{ success: boolean; message: string }> {
    const res = await http.post<ApiResponse<{ success: boolean; message: string }>>(`/api/v1/ai/configs/my/${id}/test`, {})
    return res.data
  }
}

// 侧边栏 AI 敏感工具（send_message）待确认执行。
// 工具执行只生成待确认记录，用户在确认条点击后调用本 API 真正执行/作废。
export interface PendingActionResult {
  already_handled: boolean
  status: 'pending' | 'confirmed' | 'cancelled' | 'expired'
  id: number
  target_name?: string
  message_id?: number
}

export const aiPendingAPI = {
  async confirm(id: number): Promise<PendingActionResult> {
    const res = await http.post<ApiResponse<PendingActionResult>>(`/api/v1/ai/pending-actions/${id}/confirm`, {})
    return res.data
  },

  async cancel(id: number): Promise<PendingActionResult> {
    const res = await http.post<ApiResponse<PendingActionResult>>(`/api/v1/ai/pending-actions/${id}/cancel`, {})
    return res.data
  }
}

// AI 回复用户反馈（👍=1 / 👎=-1 / 0=撤销），接入质量闭环
export const aiFeedbackAPI = {
  async set(messageId: number, rating: 1 | -1 | 0): Promise<void> {
    await http.post('/api/v1/ai/feedback', { message_id: messageId, rating })
  },

  async get(messageId: number): Promise<number> {
    const res = await http.get<ApiResponse<{ rating?: number }>>(`/api/v1/ai/feedback/${messageId}`)
    return res.data?.rating ?? 0
  },

  /** 批量恢复反馈选中态（列表层一次拉取，替代每条 AI 消息挂载时各自 GET 的 N+1） */
  async getBatch(messageIds: number[]): Promise<Map<number, number>> {
    if (!messageIds.length) return new Map()
    const res = await http.post<ApiResponse<{ ratings: Record<string, number> }>>('/api/v1/ai/feedback/batch', {
      message_ids: messageIds,
    })
    const ratings = new Map<number, number>()
    for (const [k, v] of Object.entries(res.data?.ratings ?? {})) {
      ratings.set(Number(k), v)
    }
    return ratings
  }
}

// 推荐提示词（admin 可配置；未配置返回空，客户端用内置默认）
export const aiPromptAPI = {
  async getSuggested(): Promise<string[]> {
    const res = await http.get<ApiResponse<{ prompts?: string[] }>>('/api/v1/ai/suggested-prompts')
    const list = res.data?.prompts
    return Array.isArray(list) ? list : []
  }
}
