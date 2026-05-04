import type { ApiResponse, PaginationParams, PaginatedResponse } from '@/types'
import { request } from '@/utils/request'
import type { AxiosResponse } from 'axios'

export interface AvatarApproval {
  id: number
  user_id: number
  user_name: string
  user_avatar: string
  name: string
  approval_status: 'none' | 'pending' | 'approved' | 'rejected'
  applied_at: string | null
  reject_reason: string
  created_at: string
}

export const getAvatarApprovals = (params?: { status?: string; page?: number; pageSize?: number }): Promise<AxiosResponse<ApiResponse<PaginatedResponse<AvatarApproval>>>> => {
  return request({
    url: '/v1/admin/avatar-approvals',
    method: 'get',
    params,
  })
}

export const approveAvatar = (id: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/avatar-approvals/${id}/approve`,
    method: 'post',
  })
}

export const rejectAvatar = (id: number, data: { reason: string }): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: `/v1/admin/avatar-approvals/${id}/reject`,
    method: 'post',
    data,
  })
}

export const enableAvatar = (userId: number): Promise<AxiosResponse<ApiResponse<void>>> => {
  return request({
    url: '/v1/admin/avatar-approvals/enable',
    method: 'post',
    data: { user_id: userId },
  })
}
