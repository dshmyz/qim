import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'
import type { ApiResponse } from '@/types'
import type { FileStatistics, LargeFile } from '@/types/file'

export function getFileStatistics(): Promise<AxiosResponse<ApiResponse<FileStatistics>>> {
  return request({
    url: '/v1/admin/files/statistics',
    method: 'get',
  })
}

export function getLargeFiles(limit: number = 10): Promise<AxiosResponse<ApiResponse<LargeFile[]>>> {
  return request({
    url: '/v1/admin/files/large',
    method: 'get',
    params: { limit },
  })
}
