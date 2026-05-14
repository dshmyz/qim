import request from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface MCPTool {
  name: string
  description: string
  parameters: Record<string, any>
  enabled: boolean
}

export interface MCPToolsResponse {
  tools: MCPTool[]
  total: number
}

export const getMCPTools = (): Promise<AxiosResponse<{ code: number; message: string; data: MCPToolsResponse }>> => {
  return request.get('/v1/admin/mcp/tools')
}

export const updateMCPToolConfig = (toolName: string, enabled: boolean): Promise<AxiosResponse<{ code: number; message: string; data: any }>> => {
  return request.put(`/v1/admin/mcp/tools/${toolName}`, { enabled })
}
