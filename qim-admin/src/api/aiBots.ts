import type { ApiResponse, AIBot, AIUsageLog, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getAIBots = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIBot>>>> => {
  return request({
    url: '/v1/ai-bots',
    method: 'get',
    params,
  })
}

export const createAIBot = (data: { name: string; description: string; systemPrompt: string; avatar?: string; model?: string; temperature?: number; maxTokens?: number }): Promise<AxiosResponse<ApiResponse<AIBot>>> => {
  return request({
    url: '/v1/ai-bots',
    method: 'post',
    data,
  })
}

export const updateAIBot = (id: number, data: { name?: string; description?: string; systemPrompt?: string; avatar?: string; model?: string; temperature?: number; maxTokens?: number; status?: 'active' | 'inactive' }): Promise<AxiosResponse<ApiResponse<AIBot>>> => {
  return request({
    url: `/v1/ai-bots/${id}`,
    method: 'put',
    data,
  })
}

export const deleteAIBot = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/ai-bots/${id}`,
    method: 'delete',
  })
}

export const toggleAIBotStatus = (id: number, status: 'active' | 'inactive'): Promise<AxiosResponse<ApiResponse<AIBot>>> => {
  return request({
    url: `/v1/ai-bots/${id}/status`,
    method: 'patch',
    data: { status },
  })
}

// AI 使用审计日志
export const getAIUsageLogs = (params?: { userId?: string; botId?: string; callType?: string; page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIUsageLog>>>> => {
  return request({
    url: '/v1/admin/ai-usage-logs',
    method: 'get',
    params,
  })
}
