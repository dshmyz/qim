import request from '@/utils/request'
import type { AxiosResponse } from 'axios'

// AI 工具注册表管理接口。
export interface AITool {
  name: string
  description: string
  parameters: Record<string, any>
  enabled: boolean
}

export interface AIToolsResponse {
  tools: AITool[]
  total: number
}

export const getAITools = (): Promise<AxiosResponse<{ code: number; message: string; data: AIToolsResponse }>> => {
  return request.get('/v1/admin/tool-registry/tools')
}

export const updateAIToolConfig = (toolName: string, enabled: boolean): Promise<AxiosResponse<{ code: number; message: string; data: any }>> => {
  return request.put(`/v1/admin/tool-registry/tools/${toolName}`, { enabled })
}

// ── 工具面作用域配置（各 AI 入口的白名单，admin 可覆盖/恢复默认，改完即生效） ──
export interface AIToolScope {
  scope: string
  name: string
  desc: string
  tools: string[]
  default_tools: string[]
  overridden: boolean
}

export function getAIToolScopes() {
  return request.get('/v1/admin/ai/tool-scopes')
}

export function updateAIToolScope(scope: string, payload: { tools?: string[]; reset?: boolean }) {
  return request.put(`/v1/admin/ai/tool-scopes/${scope}`, payload)
}
