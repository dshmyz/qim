import axios from 'axios'
import { getStoredServerUrl } from '../composables/useServerUrl'
import type { UserAIConfig, CreateConfigRequest } from '../types/ai'

function getToken() {
  return localStorage.getItem('token')
}

const baseURL = () => getStoredServerUrl()

export const aiConfigAPI = {
  async listMyConfigs(): Promise<UserAIConfig[]> {
    const response = await axios.get(`${baseURL()}/api/v1/ai/configs/my`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    const data = response.data.data
    return data?.list ?? data ?? []
  },

  async createConfig(data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.post(`${baseURL()}/api/v1/ai/configs/my`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async updateConfig(id: number, data: CreateConfigRequest): Promise<{ id: number; is_verified: boolean }> {
    const response = await axios.put(`${baseURL()}/api/v1/ai/configs/my/${id}`, data, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
  },

  async deleteConfig(id: number): Promise<void> {
    await axios.delete(`${baseURL()}/api/v1/ai/configs/my/${id}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
  },

  async testConfig(id: number): Promise<{ success: boolean; message: string }> {
    const response = await axios.post(`${baseURL()}/api/v1/ai/configs/my/${id}/test`, {}, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data
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

function pendingActionError(err: any): Error {
  const message = err?.response?.data?.message || err?.message || '请求失败'
  return new Error(message)
}

export const aiPendingAPI = {
  async confirm(id: number): Promise<PendingActionResult> {
    try {
      const response = await axios.post(`${baseURL()}/api/v1/ai/pending-actions/${id}/confirm`, {}, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (err) {
      throw pendingActionError(err)
    }
  },

  async cancel(id: number): Promise<PendingActionResult> {
    try {
      const response = await axios.post(`${baseURL()}/api/v1/ai/pending-actions/${id}/cancel`, {}, {
        headers: { Authorization: `Bearer ${getToken()}` }
      })
      return response.data.data
    } catch (err) {
      throw pendingActionError(err)
    }
  }
}

// AI 回复用户反馈（👍=1 / 👎=-1 / 0=撤销），接入质量闭环
export const aiFeedbackAPI = {
  async set(messageId: number, rating: 1 | -1 | 0): Promise<void> {
    await axios.post(`${baseURL()}/api/v1/ai/feedback`, { message_id: messageId, rating }, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
  },

  async get(messageId: number): Promise<number> {
    const response = await axios.get(`${baseURL()}/api/v1/ai/feedback/${messageId}`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    return response.data.data?.rating ?? 0
  }
}

// 推荐提示词（admin 可配置；未配置返回空，客户端用内置默认）
export const aiPromptAPI = {
  async getSuggested(): Promise<string[]> {
    const response = await axios.get(`${baseURL()}/api/v1/ai/suggested-prompts`, {
      headers: { Authorization: `Bearer ${getToken()}` }
    })
    const list = response.data.data?.prompts
    return Array.isArray(list) ? list : []
  }
}
