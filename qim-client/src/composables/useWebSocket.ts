import { ref, onUnmounted, readonly } from 'vue'
import { ElMessage } from 'element-plus'

export interface WebSocketMessage {
  type: string
  data: any
}

export type MessageHandler = (message: WebSocketMessage) => void

// 模块级状态 - 跨组件共享
let ws: WebSocket | null = null
let reconnectTimer: number | null = null
let heartbeatTimer: number | null = null
const handlers: Map<string, MessageHandler[]> = new Map()
const generalHandlers: MessageHandler[] = []
const isConnected = ref(false)
const showNetworkError = ref(false)
const networkErrorMsg = ref('网络连接已断开')

const RECONNECT_INTERVAL = 3000
const HEARTBEAT_INTERVAL = 30000
const MAX_RECONNECT_ATTEMPTS = 5

// 重连回调函数
let onSessionExpiredCallback: (() => void) | null = null

export function useWebSocket(wsUrl: string) {
  let reconnectAttempts = 0

  /**
   * 处理 WebSocket 消息
   */
  const handleMessage = (event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)

      // 先处理特定类型的处理器
      const typeHandlers = handlers.get(message.type)
      if (typeHandlers) {
        for (const handler of typeHandlers) {
          handler(message)
        }
      }

      // 再处理通用处理器
      for (const handler of generalHandlers) {
        handler(message)
      }
    } catch (error) {
      console.error('WebSocket message parse error:', error)
    }
  }

  /**
   * 启动心跳
   */
  const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer = window.setInterval(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    }, HEARTBEAT_INTERVAL)
  }

  /**
   * 停止心跳
   */
  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  /**
   * 连接 WebSocket（带消息处理器）
   * @param showNetworkErrorRef 显示网络错误的 ref
   * @param networkErrorMsgRef 网络错误消息的 ref
   * @param sessionExpiredRef 会话过期的 ref
   * @param messageHandlers 消息处理器映射
   */
  const connectWithHandlers = (
    showNetworkErrorRef: typeof showNetworkError,
    networkErrorMsgRef: typeof networkErrorMsg,
    sessionExpiredRef: { value: boolean },
    messageHandlers: Record<string, (data: any) => void>
  ) => {
    // 保存 session 过期回调
    onSessionExpiredCallback = () => {
      sessionExpiredRef.value = true
      showNetworkErrorRef.value = true
      networkErrorMsgRef.value = '会话已过期，请重新登录'
    }

    // 注册消息处理器
    Object.entries(messageHandlers).forEach(([type, handler]) => {
      addHandler((message: WebSocketMessage) => {
        handler(message.data)
      }, type)
    })

    // 隐藏网络错误
    showNetworkErrorRef.value = false
    networkErrorMsgRef.value = '网络连接失败，正在尝试重新连接...'
    sessionExpiredRef.value = false

    // 执行连接
    connect()
  }

  /**
   * 连接 WebSocket
   */
  const connect = () => {
    if (ws && ws.readyState === WebSocket.OPEN) return

    const token = localStorage.getItem('token')
    if (!token) {
      showNetworkError.value = true
      networkErrorMsg.value = '未登录，请先登录'
      return
    }

    try {
      const wsFullUrl = wsUrl.startsWith('ws')
        ? `${wsUrl}?token=${token}`
        : `ws://${wsUrl.replace('http://', '')}/api/v1/ws?token=${token}`

      ws = new WebSocket(wsFullUrl)

      // 暴露到全局
      if (typeof window !== 'undefined') {
        ;(window as any).ws = ws
      }

      ws.onopen = () => {
        isConnected.value = true
        showNetworkError.value = false
        reconnectAttempts = 0
        startHeartbeat()
        console.log('WebSocket connected')
      }

      ws.onmessage = handleMessage

      ws.onclose = (event: CloseEvent) => {
        isConnected.value = false
        stopHeartbeat()
        showNetworkError.value = true
        networkErrorMsg.value = '网络连接已断开，正在尝试重新连接...'

        // 检查是否是会话过期（通过 CloseEvent code 或 reason）
        if (event.code === 4401 || (event.reason && event.reason.includes('401'))) {
          if (onSessionExpiredCallback) {
            onSessionExpiredCallback()
          }
        } else {
          scheduleReconnect()
        }
      }

      ws.onerror = (error: Event) => {
        isConnected.value = false
        console.error('WebSocket error:', error)

        // 检查是否是会话过期错误
        const errorObj = error as any
        if (errorObj.message && errorObj.message.includes('401')) {
          if (onSessionExpiredCallback) {
            onSessionExpiredCallback()
          }
        }
      }
    } catch (error) {
      console.error('WebSocket connection error:', error)
      showNetworkError.value = true
      networkErrorMsg.value = '网络连接失败'
      scheduleReconnect()
    }
  }

  /**
   * 安排重连
   */
  const scheduleReconnect = () => {
    if (reconnectTimer) return
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      networkErrorMsg.value = '重连次数过多，请检查网络或重新登录'
      return
    }

    reconnectAttempts++
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      connect()
    }, RECONNECT_INTERVAL * Math.min(reconnectAttempts, 3))
  }

  /**
   * 断开连接
   */
  const disconnect = () => {
    stopHeartbeat()
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
    isConnected.value = false
  }

  /**
   * 发送消息
   */
  const sendMessage = (data: any) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data))
    } else {
      ElMessage.error('网络连接已断开')
    }
  }

  /**
   * 添加特定类型消息处理器
   */
  const addHandler = (handler: MessageHandler, messageType?: string) => {
    if (messageType) {
      if (!handlers.has(messageType)) {
        handlers.set(messageType, [])
      }
      handlers.get(messageType)!.push(handler)
    } else {
      generalHandlers.push(handler)
    }
  }

  /**
   * 移除消息处理器
   */
  const removeHandler = (handler: MessageHandler, messageType?: string) => {
    if (messageType) {
      const typeHandlers = handlers.get(messageType)
      if (typeHandlers) {
        const index = typeHandlers.indexOf(handler)
        if (index !== -1) {
          typeHandlers.splice(index, 1)
        }
      }
    } else {
      const index = generalHandlers.indexOf(handler)
      if (index !== -1) {
        generalHandlers.splice(index, 1)
      }
    }
  }

  /**
   * 获取 WebSocket 实例
   */
  const getWs = () => ws

  onUnmounted(() => {
    // 只清理当前实例添加的处理器，不关闭连接
    // 因为连接是模块级共享的
  })

  return {
    // 状态（只读）
    isConnected: readonly(isConnected),
    showNetworkError: readonly(showNetworkError),
    networkErrorMsg: readonly(networkErrorMsg),

    // 内部状态（可修改）
    ws,

    // 方法
    connect,
    connectWithHandlers,
    disconnect,
    sendMessage,
    addHandler,
    removeHandler,
    getWs
  }
}

// 导出模块级函数，供外部使用
export const getWebSocketInstance = () => ws
export const isWebSocketConnected = () => isConnected.value

export const addWsHandler = (handler: MessageHandler, messageType?: string) => {
  if (messageType) {
    if (!handlers.has(messageType)) {
      handlers.set(messageType, [])
    }
    handlers.get(messageType)!.push(handler)
  } else {
    generalHandlers.push(handler)
  }
}

export const removeWsHandler = (handler: MessageHandler, messageType?: string) => {
  if (messageType) {
    const typeHandlers = handlers.get(messageType)
    if (typeHandlers) {
      const index = typeHandlers.indexOf(handler)
      if (index !== -1) {
        typeHandlers.splice(index, 1)
      }
    }
  } else {
    const index = generalHandlers.indexOf(handler)
    if (index !== -1) {
      generalHandlers.splice(index, 1)
    }
  }
}
