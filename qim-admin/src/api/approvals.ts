import type { ApiResponse, PaginationParams } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export type ApprovalType = 'avatar' | 'bot' | 'channel' | 'group_ai' | 'all'
export type ApprovalStatus = 'pending' | 'approved' | 'rejected'

export interface ApprovalItem {
  id: number
  type: ApprovalType
  creator_id: number
  creator_name: string
  creator_avatar: string
  name: string
  description: string
  approval_status: ApprovalStatus
  applied_at: string | null
  approved_at: string | null
  reject_reason: string
  created_at: string
  extra?: {
    bot_type?: string
    creator_bot_count?: number
    group_id?: number
    conversation_id?: number
    assistant_name?: string
  }
}

export interface ApprovalListResponse {
  list: ApprovalItem[]
  total: number
  page: number
  pageSize: number
}

export interface ApprovalConfig {
  id: number
  type: ApprovalType
  enabled: boolean
  description: string
  created_at: string
  updated_at: string
}

export const getApprovals = (params?: {
  type?: ApprovalType
  status?: ApprovalStatus | 'all'
  page?: number
  pageSize?: number
}): Promise<AxiosResponse<ApiResponse<ApprovalListResponse>>> => {
  return request({
    url: '/v1/admin/approvals',
    method: 'get',
    params,
  })
}

export const approveEntity = (
  type: ApprovalType,
  id: number
): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/approvals/${type}/${id}/approve`,
    method: 'post',
  })
}

export const rejectEntity = (
  type: ApprovalType,
  id: number,
  reason: string
): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/approvals/${type}/${id}/reject`,
    method: 'post',
    data: { reason },
  })
}

// 审批配置相关
export const getApprovalConfigs = (): Promise<AxiosResponse<ApiResponse<ApprovalConfig[]>>> => {
  return request({
    url: '/v1/admin/approvals/configs',
    method: 'get',
  })
}

export const updateApprovalConfig = (
  type: ApprovalType,
  data: { enabled?: boolean; description?: string }
): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/approvals/configs/${type}`,
    method: 'put',
    data,
  })
}
