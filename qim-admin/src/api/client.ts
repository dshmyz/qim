import { request } from '@/utils/request'
import type { ClientVersion, VersionDistribution, CreateVersionParams, UpdateVersionParams } from '@/types/client'
import type { ApiResponse } from '@/types'
import type { AxiosResponse } from 'axios'

export function getVersions(): Promise<AxiosResponse<ApiResponse<ClientVersion[]>>> {
  return request({
    url: '/v1/versions',
    method: 'get',
  })
}

export function createVersion(data: CreateVersionParams): Promise<AxiosResponse<ApiResponse<ClientVersion>>> {
  return request({
    url: '/v1/versions',
    method: 'post',
    data,
  })
}

export function updateVersion(id: number, data: UpdateVersionParams): Promise<AxiosResponse<ApiResponse<ClientVersion>>> {
  return request({
    url: `/v1/versions/${id}`,
    method: 'put',
    data,
  })
}

export function deleteVersion(id: number): Promise<AxiosResponse<ApiResponse<void>>> {
  return request({
    url: `/v1/versions/${id}`,
    method: 'delete',
  })
}

export function getVersionDistribution(): Promise<AxiosResponse<ApiResponse<VersionDistribution[]>>> {
  return request({
    url: '/v1/versions/distribution',
    method: 'get',
  })
}
