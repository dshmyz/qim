import { ref } from 'vue'
import { getPendingRequests, respondToShareRequest, type PendingRequest } from '../api/realtime'
import { QMessage, QDialog } from '../components/ui'

export function usePendingRequests() {
  const pendingDialogVisible = ref(false)
  const pendingRequests = ref<PendingRequest[]>([])

  const checkPendingRequests = async () => {
    try {
      const requests = await getPendingRequests()
      if (requests.length > 0) {
        pendingRequests.value = requests
        pendingDialogVisible.value = true
      }
    } catch (error) {
      console.error('查询待处理请求失败:', error)
    }
  }

  const handleRespond = async (requestId: string, action: 'accept' | 'reject') => {
    try {
      await respondToShareRequest(requestId, action)
      QMessage.success(action === 'accept' ? '已接受共享请求' : '已拒绝共享请求')
      // 从列表中移除
      pendingRequests.value = pendingRequests.value.filter(r => r.id !== requestId)
      // 如果列表为空，关闭对话框
      if (pendingRequests.value.length === 0) {
        pendingDialogVisible.value = false
      }
    } catch (error) {
      console.error('响应失败:', error)
      QMessage.error('操作失败')
    }
  }

  return {
    pendingDialogVisible,
    pendingRequests,
    checkPendingRequests,
    handleRespond
  }
}
