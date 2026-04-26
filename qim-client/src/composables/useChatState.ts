import { ref } from 'vue'
import { logger } from '../utils/logger';

/**
 * 聊天状态管理 composable
 * 包含消息提示、确认对话框等状态管理功能
 */
export function useChatState() {
  // 显示消息提示
  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration = 3000 } = options
    logger.log('显示消息:', message, type)

    // 创建消息容器
    const messageElement = document.createElement('div')

    // 根据类型设置样式
    const typeStyles = {
      success: {
        background: '#f0f9eb',
        color: '#67c23a',
        border: '1px solid #e1f3d8'
      },
      warning: {
        background: '#fdf6ec',
        color: '#e6a23c',
        border: '1px solid #faecd8'
      },
      error: {
        background: '#fef0f0',
        color: '#f56c6c',
        border: '1px solid #fbc4c4'
      },
      info: {
        background: '#f4f4f5',
        color: '#909399',
        border: '1px solid #ebeef5'
      }
    }

    const style = typeStyles[type]

    // 设置样式
    messageElement.style.position = 'fixed'
    messageElement.style.top = '20px'
    messageElement.style.left = '50%'
    messageElement.style.transform = 'translateX(-50%)'
    messageElement.style.background = style.background
    messageElement.style.color = style.color
    messageElement.style.border = style.border
    messageElement.style.borderRadius = '4px'
    messageElement.style.padding = '12px 20px'
    messageElement.style.boxShadow = '0 2px 12px 0 rgba(0, 0, 0, 0.1)'
    messageElement.style.fontSize = '14px'
    messageElement.style.zIndex = '9999'
    messageElement.style.animation = 'messageFadeIn 0.3s ease'
    messageElement.style.pointerEvents = 'none'
    messageElement.style.minWidth = '300px'
    messageElement.style.maxWidth = '500px'
    messageElement.style.textAlign = 'center'

    // 添加图标
    const icon = document.createElement('span')
    icon.style.marginRight = '8px'

    switch (type) {
      case 'success':
        icon.textContent = '✓'
        icon.style.fontWeight = 'bold'
        break
      case 'warning':
        icon.textContent = '⚠️'
        break
      case 'error':
        icon.textContent = '✗'
        icon.style.fontWeight = 'bold'
        break
      case 'info':
        icon.textContent = 'ℹ️'
        break
    }

    messageElement.appendChild(icon)

    // 添加消息文本
    const text = document.createElement('span')
    text.textContent = message
    messageElement.appendChild(text)

    // 添加到 DOM
    document.body.appendChild(messageElement)
    logger.log('消息已添加到 DOM', messageElement)

    // 添加动画样式
    const animationStyle = document.createElement('style')
    animationStyle.textContent = `
      @keyframes messageFadeIn {
        from {
          opacity: 0;
          transform: translateX(-50%) translateY(-10px);
        }
        to {
          opacity: 1;
          transform: translateX(-50%) translateY(0);
        }
      }
    `
    document.head.appendChild(animationStyle)

    // 自动移除
    setTimeout(() => {
      messageElement.style.animation = 'messageFadeOut 0.3s ease'

      // 添加淡出动画
      const fadeOutStyle = document.createElement('style')
      fadeOutStyle.textContent = `
        @keyframes messageFadeOut {
          from {
            opacity: 1;
            transform: translateX(-50%) translateY(0);
          }
          to {
            opacity: 0;
            transform: translateX(-50%) translateY(-10px);
          }
        }
      `
      document.head.appendChild(fadeOutStyle)

      // 动画结束后移除元素
      setTimeout(() => {
        messageElement.remove()
        animationStyle.remove()
        fadeOutStyle.remove()
        logger.log('消息已移除')
      }, 300)
    }, duration)
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
