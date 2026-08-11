import type { ApiResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface AiThresholdSchema {
  key: string
  label: string
  description: string
  default: number
  min: number
  max: number
}

export const getAIThresholds = (): Promise<AxiosResponse<ApiResponse<Record<string, number>>>> => {
  return request({
    url: '/v1/ai/thresholds',
    method: 'get',
  })
}

export const getAIThresholdSchema = (): Promise<AxiosResponse<ApiResponse<AiThresholdSchema[]>>> => {
  return request({
    url: '/v1/ai/thresholds/schema',
    method: 'get',
  })
}

export const updateAIThresholds = (data: Record<string, number>): Promise<AxiosResponse<ApiResponse<null>>> => {
  return request({
    url: '/v1/ai/thresholds',
    method: 'put',
    data,
  })
}
