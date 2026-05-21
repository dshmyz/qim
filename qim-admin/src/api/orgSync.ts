import client from './client'
import type { OrgSyncConfig, OrgSyncLog } from '@/types/auth'

export const getOrgSyncConfigs = () => {
  return client.get<{ data: OrgSyncConfig[] }>('/admin/org/sync/configs')
}

export const createOrgSyncConfig = (data: Partial<OrgSyncConfig>) => {
  return client.post('/admin/org/sync/configs', data)
}

export const updateOrgSyncConfig = (id: number, data: Partial<OrgSyncConfig>) => {
  return client.put(`/admin/org/sync/configs/${id}`, data)
}

export const triggerOrgSync = (id: number) => {
  return client.post(`/admin/org/sync/trigger/${id}`)
}

export const getOrgSyncLogs = (configId: number, page = 1, pageSize = 20) => {
  return client.get<{ data: { total: number; items: OrgSyncLog[] } }>(
    `/admin/org/sync/logs?config_id=${configId}&page=${page}&page_size=${pageSize}`
  )
}
