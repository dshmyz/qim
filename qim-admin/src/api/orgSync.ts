import { request } from '@/utils/request'
import type { OrgSyncConfig, OrgSyncLog } from '@/types/auth'
import type { ApiResponse } from '@/types'
import type { AxiosResponse } from 'axios'

export const getOrgSyncConfigs = (): Promise<AxiosResponse<ApiResponse<OrgSyncConfig[]>>> => {
  return request({
    url: '/v1/admin/org/sync/configs',
    method: 'get',
  })
}

export const createOrgSyncConfig = (data: Partial<OrgSyncConfig>): Promise<AxiosResponse<ApiResponse<OrgSyncConfig>>> => {
  return request({
    url: '/v1/admin/org/sync/configs',
    method: 'post',
    data,
  })
}

export const updateOrgSyncConfig = (id: number, data: Partial<OrgSyncConfig>): Promise<AxiosResponse<ApiResponse<OrgSyncConfig>>> => {
  return request({
    url: `/v1/admin/org/sync/configs/${id}`,
    method: 'put',
    data,
  })
}

export const triggerOrgSync = (id: number): Promise<AxiosResponse<ApiResponse<{ success: boolean; message: string }>>> => {
  return request({
    url: `/v1/admin/org/sync/trigger/${id}`,
    method: 'post',
  })
}

export const getOrgSyncLogs = (configId: number, page = 1, pageSize = 20): Promise<AxiosResponse<ApiResponse<{ total: number; items: OrgSyncLog[] }>>> => {
  return request({
    url: '/v1/admin/org/sync/logs',
    method: 'get',
    params: {
      config_id: configId,
      page,
      page_size: pageSize,
    },
  })
}
