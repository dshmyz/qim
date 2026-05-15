import { ref } from 'vue'
import QMessage from '../utils/qmessage'

/**
 * 聊天状态管理 composable
 * 包含消息提示、确认对话框等状态管理功能
 */
export function useChatState() {
  // 显示消息提示（委托给 QMessage 实现，避免 DOM 泄漏）
  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration = 3000 } = options
    QMessage[type](message, duration)
  }

  // 便捷方法
  const $message = {
    success: (message: string, duration?: number) => showMessage({ message, type: 'success', duration }),
    warning: (message: string, duration?: number) => showMessage({ message, type: 'warning', duration }),
    error: (message: string, duration?: number) => showMessage({ message, type: 'error', duration }),
    info: (message: string, duration?: number) => showMessage({ message, type: 'info', duration })
  }

  // 确认对话框
  const showConfirmDialog = ref(false)
  const confirmDialogTitle = ref('确认操作')
  const confirmDialogMessage = ref('')
  const confirmDialogCallback = ref<(() => void) | null>(null)

  // 打开确认对话框
  const openConfirmDialog = (title: string, message: string, callback: () => void) => {
    confirmDialogTitle.value = title
    confirmDialogMessage.value = message
    confirmDialogCallback.value = callback
    showConfirmDialog.value = true
  }

  // 关闭确认对话框
  const closeConfirmDialog = () => {
    showConfirmDialog.value = false
    confirmDialogCallback.value = null
  }

  // 处理确认操作
  const handleConfirmAction = () => {
    if (confirmDialogCallback.value) {
      confirmDialogCallback.value()
    }
    closeConfirmDialog()
  }

  return {
    $message,
    showConfirmDialog,
    confirmDialogTitle,
    confirmDialogMessage,
    confirmDialogCallback,
    openConfirmDialog,
    closeConfirmDialog,
    handleConfirmAction
  }
}
