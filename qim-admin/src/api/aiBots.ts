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

export const createAIBot = (data: { name: string; description: string; systemPrompt: string; avatar?: string }): Promise<AxiosResponse<ApiResponse<AIBot>>> => {
  return request({
    url: '/v1/ai-bots',
    method: 'post',
    data,
  })
}

export const updateAIBot = (id: number, data: { name?: string; description?: string; systemPrompt?: string; avatar?: string; status?: 'active' | 'inactive' }): Promise<AxiosResponse<ApiResponse<AIBot>>> => {
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

// Bot 审批相关
export const getBotApprovals = (params?: { status?: string; page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIBot>>>> => {
  return request({
    url: '/v1/admin/bot-approvals',
    method: 'get',
    params,
  })
}

export const approveBot = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/bot-approvals/${id}/approve`,
    method: 'patch',
  })
}

export const rejectBot = (id: number, data: { reason?: string }): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/bot-approvals/${id}/reject`,
    method: 'patch',
    data,
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
