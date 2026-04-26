import type { ApiResponse, SensitiveWord, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getSensitiveWords = (params?: PaginationParams & { category?: string; keyword?: string }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<SensitiveWord>>>> => {
  return request({
    url: '/v1/sensitive-words',
    method: 'get',
    params,
  })
}

export const createSensitiveWord = (data: { word: string; category: string; level: 'low' | 'medium' | 'high' }): Promise<AxiosResponse<ApiResponse<SensitiveWord>>> => {
  return request({
    url: '/v1/sensitive-words',
    method: 'post',
    data,
  })
}

export const updateSensitiveWord = (id: number, data: { category?: string; level?: 'low' | 'medium' | 'high' }): Promise<AxiosResponse<ApiResponse<SensitiveWord>>> => {
  return request({
    url: `/v1/sensitive-words/${id}`,
    method: 'put',
    data,
  })
}

export const deleteSensitiveWord = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/sensitive-words/${id}`,
    method: 'delete',
  })
}

export const toggleSensitiveWordStatus = (id: number, status: 'active' | 'inactive'): Promise<AxiosResponse<ApiResponse<SensitiveWord>>> => {
  return request({
    url: `/v1/sensitive-words/${id}/status`,
    method: 'patch',
    data: { status },
  })
}

export const batchCreateSensitiveWords = (data: { words: string[]; category: string; level: 'low' | 'medium' | 'high' }): Promise<AxiosResponse<ApiResponse<{ count: number }>>> => {
  return request({
    url: '/v1/sensitive-words/batch',
    method: 'post',
    data,
  })
}
