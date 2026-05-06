import type { ApiResponse, BlacklistEntry, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getBlacklist = (params?: PaginationParams): Promise<AxiosResponse<ApiResponse<PaginatedResponse<BlacklistEntry>>>> => {
  return request({
    url: '/v1/users/blacklist',
    method: 'get',
    params,
  })
}

export const removeBlacklistEntry = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/users/blacklist/${id}`,
    method: 'delete',
  })
}
