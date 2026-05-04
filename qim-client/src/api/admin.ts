import { request } from '../composables/useRequest'
import type { AvatarApprovalRecord, AvatarApprovalStatus } from '../types/avatar'

export interface AvatarApprovalsResponse {
  list: AvatarApprovalRecord[]
  total: number
}

export const adminAPI = {
  // 获取 Avatar 审批列表
  async getAvatarApprovals(status?: AvatarApprovalStatus): Promise<AvatarApprovalRecord[]> {
    const params = status ? { status } : {}
    const response = await request<{ code: number; data: AvatarApprovalsResponse }>(
      '/api/v1/admin/avatar-approvals',
      { method: 'GET', params }
    )
    return response?.data?.list ?? []
  },

  // 通过 Avatar 审批
  async approveAvatar(id: number): Promise<void> {
    await request(`/api/v1/admin/avatar-approvals/${id}/approve`, { method: 'POST' })
  },

  // 拒绝 Avatar 审批
  async rejectAvatar(id: number, reason: string): Promise<void> {
    await request(`/api/v1/admin/avatar-approvals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
      headers: { 'Content-Type': 'application/json' }
    })
  }
}
