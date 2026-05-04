import type { ApiResponse, PaginationParams } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export type ApprovalType = 'avatar' | 'bot' | 'channel' | 'all'
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
  }
}

export interface ApprovalListResponse {
  list: ApprovalItem[]
  total: number
  page: number
  pageSize: number
}

export const getApprovals = (params?: {
  type?: ApprovalType
  status?: ApprovalStatus
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

export const enableAvatar = (userId: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/admin/approvals/avatar/enable',
    method: 'post',
    data: { user_id: userId },
  })
}
