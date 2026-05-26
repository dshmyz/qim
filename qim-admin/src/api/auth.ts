import type { ApiResponse, UserInfo } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const login = (data: { username: string; password: string }): Promise<AxiosResponse<ApiResponse<{ token: string; user: UserInfo }>>> => {
  return request({
    url: '/v1/auth/login',
    method: 'post',
    data,
  })
}

export const logout = (): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/auth/logout',
    method: 'post',
  })
}

export const getCurrentUser = (): Promise<AxiosResponse<ApiResponse<UserInfo>>> => {
  return request({
    url: '/v1/users/me',
    method: 'get',
  })
}
