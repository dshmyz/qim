import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse, PaginationParams, PaginatedResponse } from '@/types'
import type {
  AIProvider,
  AIModel,
  CreateProviderParams,
  UpdateProviderParams,
  TestConnectionResult,
  CreateModelParams,
  UpdateModelParams,
  AIConfig,
  AIQuota,
  AIUsage,
  AIUsageStatistics,
} from '@/types/ai'

// Provider CRUD
export const getProviders = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIProvider>>>> => {
  return request({
    url: '/v1/admin/ai/providers',
    method: 'get',
    params,
  })
}

export const createProvider = (data: CreateProviderParams): Promise<AxiosResponse<ApiResponse<AIProvider>>> => {
  return request({
    url: '/v1/admin/ai/providers',
    method: 'post',
    data,
  })
}

export const updateProvider = (id: number, data: UpdateProviderParams): Promise<AxiosResponse<ApiResponse<AIProvider>>> => {
  return request({
    url: `/v1/admin/ai/providers/${id}`,
    method: 'put',
    data,
  })
}

export const deleteProvider = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/ai/providers/${id}`,
    method: 'delete',
  })
}

export const toggleProviderStatus = (id: number, enabled: boolean): Promise<AxiosResponse<ApiResponse<AIProvider>>> => {
  return request({
    url: `/v1/admin/ai/providers/${id}/status`,
    method: 'patch',
    data: { enabled },
  })
}

export const testProviderConnection = (id: number): Promise<AxiosResponse<ApiResponse<TestConnectionResult>>> => {
  return request({
    url: `/v1/admin/ai/providers/${id}/test`,
    method: 'post',
  })
}

// Model management under providers
export const getProviderModels = (providerId: number): Promise<AxiosResponse<ApiResponse<AIModel[]>>> => {
  return request({
    url: `/v1/admin/ai/providers/${providerId}/models`,
    method: 'get',
  })
}

export const addProviderModel = (providerId: number, data: CreateModelParams): Promise<AxiosResponse<ApiResponse<AIModel>>> => {
  return request({
    url: `/v1/admin/ai/providers/${providerId}/models`,
    method: 'post',
    data,
  })
}

export const updateProviderModel = (
  providerId: number,
  modelId: number,
  data: UpdateModelParams
): Promise<AxiosResponse<ApiResponse<AIModel>>> => {
  return request({
    url: `/v1/admin/ai/providers/${providerId}/models/${modelId}`,
    method: 'put',
    data,
  })
}

export const deleteProviderModel = (providerId: number, modelId: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/ai/providers/${providerId}/models/${modelId}`,
    method: 'delete',
  })
}

// AI Config APIs
export const getConfig = (): Promise<AxiosResponse<ApiResponse<AIConfig>>> => {
  return request({
    url: '/v1/admin/ai/config',
    method: 'get',
  })
}

export const updateConfig = (data: Partial<AIConfig>): Promise<AxiosResponse<ApiResponse<AIConfig>>> => {
  return request({
    url: '/v1/admin/ai/config',
    method: 'put',
    data,
  })
}

// AI Quota APIs
export const getQuota = (params?: { targetType?: string; targetId?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIQuota>>>> => {
  return request({
    url: '/v1/admin/ai/quota',
    method: 'get',
    params,
  })
}

export const updateQuota = (id: number, data: Partial<AIQuota>): Promise<AxiosResponse<ApiResponse<AIQuota>>> => {
  return request({
    url: `/v1/admin/ai/quota/${id}`,
    method: 'put',
    data,
  })
}

// AI Usage & Statistics APIs
export const getStatistics = (params?: { startDate?: string; endDate?: string }): Promise<AxiosResponse<ApiResponse<AIUsageStatistics>>> => {
  return request({
    url: '/v1/admin/ai/statistics',
    method: 'get',
    params,
  })
}

export const getUsage = (params?: PaginationParams & { userId?: number; provider?: string; status?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AIUsage>>>> => {
  return request({
    url: '/v1/admin/ai/usage',
    method: 'get',
    params,
  })
}
