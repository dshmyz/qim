import { request } from '../composables/useRequest'
import type {
  AvatarConfig,
  AvatarSession,
  AvatarLearnStatus,
  CreateAvatarConfigRequest
} from '../types/avatar'

export const avatarAPI = {
  async getConfig(): Promise<AvatarConfig | null> {
    const response = await request<{ code: number; data: AvatarConfig | null }>(
      '/api/v1/avatar/config',
      { method: 'GET' }
    )
    return response?.data ?? null
  },

  async createConfig(data: CreateAvatarConfigRequest): Promise<AvatarConfig> {
    const response = await request<{ code: number; data: AvatarConfig }>(
      '/api/v1/avatar/config',
      {
        method: 'POST',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async updateConfig(data: Partial<AvatarConfig>): Promise<AvatarConfig> {
    const response = await request<{ code: number; data: AvatarConfig }>(
      '/api/v1/avatar/config',
      {
        method: 'PUT',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async deleteConfig(): Promise<void> {
    await request('/api/v1/avatar/config', { method: 'DELETE' })
  },

  async triggerLearnPersona(): Promise<{ taskId: string }> {
    const response = await request<{ code: number; data: { taskId: string } }>(
      '/api/v1/avatar/learn-persona',
      {
        method: 'POST',
        body: JSON.stringify({}),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async getLearnStatus(): Promise<AvatarLearnStatus> {
    const response = await request<{ code: number; data: AvatarLearnStatus }>(
      '/api/v1/avatar/learn-status',
      { method: 'GET' }
    )
    return response!.data
  },

  async getLearnedPersona(): Promise<string> {
    const response = await request<{ code: number; data: string }>(
      '/api/v1/avatar/learned-persona',
      { method: 'GET' }
    )
    return response!.data
  },

  async getSessions(): Promise<AvatarSession[]> {
    const response = await request<{ code: number; data: AvatarSession[] }>(
      '/api/v1/avatar/sessions',
      { method: 'GET' }
    )
    return response?.data ?? []
  },

  async updateSession(convId: number, enabled: boolean): Promise<AvatarSession> {
    const response = await request<{ code: number; data: AvatarSession }>(
      `/api/v1/avatar/sessions/${convId}`,
      {
        method: 'PUT',
        body: JSON.stringify({ avatarEnabled: enabled }),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async takeoverSession(convId: number): Promise<AvatarSession> {
    const response = await request<{ code: number; data: AvatarSession }>(
      `/api/v1/avatar/sessions/${convId}/takeover`,
      {
        method: 'POST',
        body: JSON.stringify({}),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data
  },

  async previewReply(message: string): Promise<string> {
    const response = await request<{ code: number; data: { reply: string } }>(
      '/api/v1/avatar/preview',
      {
        method: 'POST',
        body: JSON.stringify({ message }),
        headers: { 'Content-Type': 'application/json' }
      }
    )
    return response!.data.reply
  }
}
