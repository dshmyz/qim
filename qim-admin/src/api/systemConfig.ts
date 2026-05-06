import type { ApiResponse, SystemConfig } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export const getSystemConfig = (): Promise<AxiosResponse<ApiResponse<SystemConfig>>> => {
  return request({
    url: '/v1/system/config',
    method: 'get',
  })
}

export const updateSystemConfig = (data: Partial<SystemConfig>): Promise<AxiosResponse<ApiResponse<SystemConfig>>> => {
  return request({
    url: '/v1/system/config',
    method: 'put',
    data,
  })
}
