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
