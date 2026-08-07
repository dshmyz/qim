import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse, PaginationParams, PaginatedResponse } from '@/types'
import type {
  AIProvider,
  CreateProviderParams,
  UpdateProviderParams,
  TestConnectionResult,
  AIRouterConfig,
  AIRouterResponse,
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

// ===== AI 模型路由（任务类型 → provider/model） =====
export const getAIRouter = (): Promise<AxiosResponse<ApiResponse<AIRouterResponse>>> => {
  return request({
    url: '/v1/admin/ai/router',
    method: 'get',
  })
}

export const saveAIRouter = (data: AIRouterConfig): Promise<AxiosResponse<ApiResponse<{ usingDb: boolean }>>> => {
  return request({
    url: '/v1/admin/ai/router',
    method: 'put',
    data,
  })
}

export const clearAIRouter = (): Promise<AxiosResponse<ApiResponse<{ usingDb: boolean }>>> => {
  return request({
    url: '/v1/admin/ai/router',
    method: 'delete',
  })
}
