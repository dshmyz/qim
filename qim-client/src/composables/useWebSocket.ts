import { ref, readonly } from 'vue'
import QMessage from '../utils/qmessage'

export interface WebSocketMessage {
  type: string
  data: any
}

export type MessageHandler = (message: WebSocketMessage) => void

// 模块级状态 - 跨组件共享
let ws: WebSocket | null = null
let reconnectTimer: number | null = null
let heartbeatTimer: number | null = null
const handlers: Map<string, Set<MessageHandler>> = new Map()
const generalHandlers: Set<MessageHandler> = new Set()
const isConnected = ref(false)
const showNetworkError = ref(false)
const networkErrorMsg = ref('网络连接已断开')

const RECONNECT_INTERVAL = 5000
const HEARTBEAT_INTERVAL = 30000
const MAX_RECONNECT_ATTEMPTS = 3
const RECONNECT_JITTER_MIN = 2000
const RECONNECT_JITTER_MAX = 8000

// 重连回调函数
let onSessionExpiredCallback: (() => void) | null = null
let externalShowNetworkError: typeof showNetworkError | null = null
let externalNetworkErrorMsg: typeof networkErrorMsg | null = null

/**
 * 设置网络错误状态
 */
const setNetworkError = (show: boolean, msg: string) => {
  showNetworkError.value = show
  networkErrorMsg.value = msg
  if (externalShowNetworkError) {
    externalShowNetworkError.value = show
  }
  if (externalNetworkErrorMsg) {
    externalNetworkErrorMsg.value = msg
  }
}

/**
 * 注册消息处理器，返回清理函数
 * @param handler 消息处理函数
 * @param messageType 消息类型（可选，不传则处理所有消息）
 * @returns 清理函数，调用后自动移除该 handler
 */
export const addWsHandler = (handler: MessageHandler, messageType?: string): (() => void) => {
  let removed = false

  const cleanup = () => {
    if (removed) return
    removed = true

    if (messageType) {
      const typeHandlers = handlers.get(messageType)
      if (typeHandlers) {
        typeHandlers.delete(handler)
        if (typeHandlers.size === 0) {
          handlers.delete(messageType)
        }
      }
    } else {
      generalHandlers.delete(handler)
    }
  }

  if (messageType) {
    if (!handlers.has(messageType)) {
      handlers.set(messageType, new Set())
    }
    handlers.get(messageType)!.add(handler)
  } else {
    generalHandlers.add(handler)
  }

  return cleanup
}

/**
 * 批量注册多个消息处理器，返回统一的清理函数
 * @param handlerMap 消息类型 -> 处理函数映射
 * @returns 清理函数
 */
export const addWsHandlers = (handlerMap: Record<string, MessageHandler>): (() => void) => {
  const cleanups = Object.entries(handlerMap).map(([type, handler]) =>
    addWsHandler((message: WebSocketMessage) => {
      handler(message.data)
    }, type)
  )

  return () => {
    cleanups.forEach(cleanup => cleanup())
  }
}

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
   */
  const connectWithHandlers = (
    showNetworkErrorRef: typeof showNetworkError,
    networkErrorMsgRef: typeof networkErrorMsg,
    sessionExpiredRef: { value: boolean },
    messageHandlers: Record<string, (data: any) => void>
  ) => {
    externalShowNetworkError = showNetworkErrorRef
    externalNetworkErrorMsg = networkErrorMsgRef

    onSessionExpiredCallback = () => {
      sessionExpiredRef.value = true
      setNetworkError(true, '会话已过期，请重新登录')
    }

    // 使用新的 addWsHandler，每个 handler 自动注册
    Object.entries(messageHandlers).forEach(([type, handler]) => {
      addWsHandler((message: WebSocketMessage) => {
        handler(message.data)
      }, type)
    })

    setNetworkError(false, '网络连接失败，正在尝试重新连接...')
    sessionExpiredRef.value = false

    connect()
  }

  /**
   * 连接 WebSocket
   */
  const connect = () => {
    if (ws && ws.readyState === WebSocket.OPEN) return

    const token = localStorage.getItem('token')
    if (!token) {
      setNetworkError(true, '未登录，请先登录')
      return
    }

    try {
      const wsFullUrl = wsUrl.startsWith('ws')
        ? `${wsUrl}?token=${token}`
        : `ws://${wsUrl.replace('http://', '')}/api/v1/ws?token=${token}`

      ws = new WebSocket(wsFullUrl)

      if (typeof window !== 'undefined') {
        ;(window as any).ws = ws
      }

      ws.onopen = () => {
        isConnected.value = true
        setNetworkError(false, '')
        reconnectAttempts = 0
        startHeartbeat()
        console.log('WebSocket connected')
      }

      ws.onmessage = handleMessage

      ws.onclose = (event: CloseEvent) => {
        isConnected.value = false
        stopHeartbeat()
        setNetworkError(true, '网络连接已断开，正在尝试重新连接...')

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

        const errorObj = error as any
        if (errorObj.message && errorObj.message.includes('401')) {
          if (onSessionExpiredCallback) {
            onSessionExpiredCallback()
          }
        }
      }
    } catch (error) {
      console.error('WebSocket connection error:', error)
      setNetworkError(true, '网络连接失败')
      scheduleReconnect()
    }
  }

  /**
   * 安排重连
   */
  const scheduleReconnect = () => {
    if (reconnectTimer) return
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      setNetworkError(true, '重连次数过多，请检查网络或重新登录')
      return
    }

    reconnectAttempts++
    
    const baseDelay = RECONNECT_INTERVAL * Math.pow(2, reconnectAttempts - 1)
    const jitter = RECONNECT_JITTER_MIN + Math.random() * (RECONNECT_JITTER_MAX - RECONNECT_JITTER_MIN)
    const totalDelay = baseDelay + jitter
    
    console.log(`WebSocket 将在 ${Math.round(totalDelay / 1000)}s 后重连 (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`)
    setNetworkError(true, `网络连接已断开，${Math.round(totalDelay / 1000)}秒后尝试重连... (${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`)
    
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      connect()
    }, totalDelay)
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
      QMessage.error('网络连接已断开')
    }
  }

  /**
   * 获取 WebSocket 实例
   */
  const getWs = () => ws

  return {
    isConnected: readonly(isConnected),
    showNetworkError: readonly(showNetworkError),
    networkErrorMsg: readonly(networkErrorMsg),
    ws,
    connect,
    connectWithHandlers,
    disconnect,
    sendMessage,
    addHandler: addWsHandler,
    getWs
  }
}

// 导出模块级函数
export const getWebSocketInstance = () => ws
export const isWebSocketConnected = () => isConnected.value

// 保留向后兼容的导出（标记为 deprecated）
/** @deprecated 使用 addWsHandler 返回值进行清理，不要直接调用此函数 */
export const removeWsHandler = (handler: MessageHandler, messageType?: string) => {
  console.warn('removeWsHandler is deprecated, use the cleanup function returned by addWsHandler instead')
  if (messageType) {
    const typeHandlers = handlers.get(messageType)
    if (typeHandlers) {
      typeHandlers.delete(handler)
    }
  } else {
    generalHandlers.delete(handler)
  }
}
