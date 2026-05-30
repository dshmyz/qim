import { request } from '../composables/useRequest'
import type {
  AvatarConfig,
  AvatarSession,
  AvatarLearnStatus,
  CreateAvatarConfigRequest,
  AvatarConfigWithApproval,
  AvatarWithTools
} from '../types/avatar'

export const avatarAPI = {
  async getConfig(): Promise<AvatarConfigWithApproval | null> {
    const response = await request<{ code: number; data: AvatarConfigWithApproval | null }>(
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

  async updateConfig(data: Partial<AvatarConfig>): Promise<AvatarConfigWithApproval> {
    const response = await request<{ code: number; data: AvatarConfigWithApproval }>(
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
    try {
      const response = await request<{ code: number; data: AvatarLearnStatus }>(
        '/api/v1/avatar/learn-status',
        { method: 'GET' }
      )
      return response!.data
    } catch (e: any) {
      if (e.message?.includes('配置不存在')) {
        return { status: 'idle', progress: 0, messageCount: 0, error: null }
      }
      throw e
    }
  },

  async getLearnedPersona(): Promise<string> {
    try {
      const response = await request<{ code: number; data: string }>(
        '/api/v1/avatar/learned-persona',
        { method: 'GET' }
      )
      return response!.data
    } catch (e: any) {
      if (e.message?.includes('配置不存在')) {
        return ''
      }
      throw e
    }
  },

  async clearLearnedPersona(): Promise<void> {
    await request('/api/v1/avatar/learned-persona', { method: 'DELETE' })
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
  },

  // Avatar 审批相关 API
  async applyForApproval(): Promise<AvatarConfigWithApproval> {
    const response = await request<{ code: number; data: AvatarConfigWithApproval }>(
      '/api/v1/avatar/apply',
      { method: 'POST' }
    )
    return response!.data
  },

  async cancelApplication(): Promise<AvatarConfigWithApproval> {
    const response = await request<{ code: number; data: AvatarConfigWithApproval }>(
      '/api/v1/avatar/cancel-apply',
      { method: 'POST' }
    )
    return response!.data
  },

  // 工具绑定相关 API
  async getAvailableTools(): Promise<any[]> {
    const response = await request<{ code: number; data: any[] }>(
      '/api/v1/avatar/tools',
      { method: 'GET' }
    )
    return response?.data ?? []
  },

  async getAvatarWithTools(): Promise<AvatarWithTools | null> {
    const [avatar, tools] = await Promise.all([
      this.getConfig(),
      this.getAvailableTools()
    ])
    if (!avatar) return null
    return {
      id: String(avatar.id),
      enabled: avatar.enabled,
      persona: avatar.persona,
      availableTools: tools,
      lastActiveAt: new Date()
    }
  },

  async bindToolToAvatar(toolId: string): Promise<void> {
    await request(
      `/api/v1/avatar/tools/${toolId}`,
      { method: 'POST' }
    )
  },

  async unbindToolFromAvatar(toolId: string): Promise<void> {
    await request(
      `/api/v1/avatar/tools/${toolId}`,
      { method: 'DELETE' }
    )
  }
}
