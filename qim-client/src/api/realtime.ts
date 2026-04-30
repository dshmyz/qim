import { request } from '../composables/useRequest'

export interface PendingRequest {
  id: string
  session_id: string
  session_type: string
  conversation_id: number
  initiator_id: number
  initiator_name: string
  requested_at: string
}

/**
 * 获取待处理的共享请求
 */
export async function getPendingRequests(): Promise<PendingRequest[]> {
  return request<PendingRequest[]>('/api/v1/realtime/pending-requests')
}

/**
 * 响应共享请求（接受/拒绝）
 */
export async function respondToShareRequest(
  requestId: string,
  action: 'accept' | 'reject'
): Promise<void> {
  return request<void>(`/api/v1/realtime/pending-requests/${requestId}/respond`, {
    method: 'POST',
    body: JSON.stringify({ action })
  })
}
