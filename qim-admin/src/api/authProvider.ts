import { request } from '@/utils/request'
import type { AuthProvider } from '@/types/auth'
import type { ApiResponse } from '@/types'
import type { AxiosResponse } from 'axios'

export const getAuthProviders = (): Promise<AxiosResponse<ApiResponse<AuthProvider[]>>> => {
  return request({
    baseURL: '/api/v1',
    url: '/admin/auth/providers',
    method: 'get',
  })
}

export const createAuthProvider = (data: Partial<AuthProvider>): Promise<AxiosResponse<ApiResponse<AuthProvider>>> => {
  return request({
    baseURL: '/api/v1',
    url: '/admin/auth/providers',
    method: 'post',
    data,
  })
}

export const updateAuthProvider = (id: number, data: Partial<AuthProvider>): Promise<AxiosResponse<ApiResponse<AuthProvider>>> => {
  return request({
    baseURL: '/api/v1',
    url: `/admin/auth/providers/${id}`,
    method: 'put',
    data,
  })
}

export const deleteAuthProvider = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    baseURL: '/api/v1',
    url: `/admin/auth/providers/${id}`,
    method: 'delete',
  })
}

export const testAuthProvider = (id: number, testData: { test_username: string; test_password: string }): Promise<AxiosResponse<ApiResponse<{ success: boolean; message: string }>>> => {
  return request({
    baseURL: '/api/v1',
    url: `/admin/auth/providers/${id}/test`,
    method: 'post',
    data: testData,
  })
}
