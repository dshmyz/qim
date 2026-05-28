import { ref } from 'vue'
import QMessage from '../utils/qmessage'

export function useNetwork() {
  // 网络连接状态
  const sessionExpired = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const reconnectTimer = ref<number | null>(null)
  const baseReconnectDelay = 1000 // 1秒

  /**
   * 重置网络连接状态
   */
  const resetNetworkState = () => {
    sessionExpired.value = false
    reconnectAttempts.value = 0
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
  }

  /**
   * 重新连接网络
   * @param connectFn 连接函数
   */
  const reconnect = (connectFn: () => void) => {
    // 重置重连尝试次数
    reconnectAttempts.value = 0
    // 清除之前的定时器
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
    // 重新连接
    connectFn()
  }

  /**
   * 处理网络重连
   * @param connectFn 连接函数
   * @param showNetworkError 显示网络错误的ref
   * @param networkErrorMsg 网络错误消息的ref
   * @returns 是否继续重连
   */
  const handleReconnect = (connectFn: () => void, showNetworkError: any, networkErrorMsg: any): boolean => {
    if (reconnectAttempts.value < maxReconnectAttempts) {
      // 指数退避策略
      const delay = baseReconnectDelay * Math.pow(2, reconnectAttempts.value)
      console.log(`网络重连尝试 ${reconnectAttempts.value + 1}/${maxReconnectAttempts}，延迟 ${delay}ms`)
      
      // 显示网络错误提示
      showNetworkError.value = true
      networkErrorMsg.value = `网络连接失败，正在尝试重新连接... (${reconnectAttempts.value + 1}/${maxReconnectAttempts})`
      
      reconnectTimer.value = window.setTimeout(() => {
        reconnectAttempts.value++
        connectFn()
      }, delay)
      return true
    } else {
      console.log('网络重连失败，已达到最大重试次数')
      // 显示最终错误提示
      showNetworkError.value = true
      networkErrorMsg.value = '网络连接失败，请检查网络设置或稍后重试'
      return false
    }
  }

  /**
   * 处理会话过期
   * @param showNetworkError 显示网络错误的ref
   * @param networkErrorMsg 网络错误消息的ref
   */
  const handleSessionExpired = (showNetworkError: any, networkErrorMsg: any) => {
    sessionExpired.value = true
    showNetworkError.value = true
    networkErrorMsg.value = '会话已过期，请重新登录'
  }

  /**
   * 跳转到登录页
   */
  const gotoLogin = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    window.location.reload()
  }

  /**
   * 清理网络相关资源
   */
  const cleanupNetwork = () => {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
  }

  return {
    // 状态
    sessionExpired,
    reconnectAttempts,
    maxReconnectAttempts,
    reconnectTimer,
    baseReconnectDelay,
    
    // 方法
    resetNetworkState,
    reconnect,
    handleReconnect,
    handleSessionExpired,
    gotoLogin,
    cleanupNetwork
  }
}