import { ref, readonly } from 'vue'
import QMessage from '../utils/qmessage'
import { calculateReconnectDelay, shouldReconnect, DEFAULT_RECONNECT_CONFIG } from '../utils/websocketReconnect'
import { connectionMonitor } from '../utils/connectionMonitor'
import { messageQueue } from '../utils/messageQueue'

export interface WebSocketMessage {
  type: string
  data: any
}

export type MessageHandler = (message: WebSocketMessage) => void

let ws: WebSocket | null = null
let reconnectTimer: number | null = null
let heartbeatTimer: number | null = null
const handlers: Map<string, Set<MessageHandler>> = new Map()
const generalHandlers: Set<MessageHandler> = new Set()
const isConnected = ref(false)
const showNetworkError = ref(false)
const networkErrorMsg = ref('网络连接已断开')
let networkOnlineHandler: (() => void) | null = null

const HEARTBEAT_INTERVAL = 30000

let onSessionExpiredCallback: (() => void) | null = null
let externalShowNetworkError: typeof showNetworkError | null = null
let externalNetworkErrorMsg: typeof networkErrorMsg | null = null
let onConnectedCallback: (() => void) | null = null
let reconnectAttempts = 0

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
 * @param handlerMap 消息类型 -> 处理函数映射 (函数接收 data 参数而非完整 message)
 * @returns 清理函数
 */
export const addWsHandlers = (handlerMap: Record<string, (data: any) => void>): (() => void) => {
  const cleanups = Object.entries(handlerMap).map(([type, handler]) =>
    addWsHandler((message: WebSocketMessage) => {
      handler(message.data)
    }, type)
  )

  return () => {
    cleanups.forEach(cleanup => cleanup())
  }
}

/**
 * 发送消息
 */
export const sendMessage = (data: any) => {
  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(data))
  } else {
    messageQueue.enqueue(data)
    QMessage.error('网络连接已断开，消息已缓存')
  }
}

export function useWebSocket(wsUrl: string) {
  /**
   * 处理 WebSocket 消息
   */
  const handleMessage = (event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)

      if (message.type === 'pong') {
        connectionMonitor.recordPong()
        return
      }

      const typeHandlers = handlers.get(message.type)
      if (typeHandlers) {
        for (const handler of typeHandlers) {
          handler(message)
        }
      }

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
    connectionMonitor.start(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'ping' }))
      }
    })
  }

  /**
   * 停止心跳
   */
  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
    connectionMonitor.stop()
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
        
        if (onConnectedCallback) {
          onConnectedCallback()
        }

        if (!messageQueue.isEmpty()) {
          console.log(`[WebSocket] 刷新离线消息队列，共 ${messageQueue.size()} 条`)
          messageQueue.flush((data) => {
            if (ws && ws.readyState === WebSocket.OPEN) {
              ws.send(JSON.stringify(data))
              return true
            }
            return false
          })
        }

        if (!networkOnlineHandler) {
          networkOnlineHandler = () => {
            console.log('[WebSocket] 网络恢复，立即重连')
            reconnectAttempts = 0
            connect()
          }
          window.addEventListener('online', networkOnlineHandler)
        }
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
    
    if (!shouldReconnect(reconnectAttempts)) {
      setNetworkError(true, '网络连接失败，请手动重连')
      return
    }

    const delay = calculateReconnectDelay(reconnectAttempts)
    reconnectAttempts++
    
    console.log(`[WebSocket] 第 ${reconnectAttempts} 次重连，延迟 ${Math.round(delay)}ms`)
    setNetworkError(true, `网络连接已断开，${Math.round(delay / 1000)}秒后尝试重连... (${reconnectAttempts}/${DEFAULT_RECONNECT_CONFIG.maxAttempts})`)
    
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
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
    if (networkOnlineHandler) {
      window.removeEventListener('online', networkOnlineHandler)
      networkOnlineHandler = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
    isConnected.value = false

    handlers.clear()
    generalHandlers.clear()
    onSessionExpiredCallback = null
    externalShowNetworkError = null
    externalNetworkErrorMsg = null
    onConnectedCallback = null
    reconnectAttempts = 0
  }

  /**
   * 获取 WebSocket 实例
   */
  const getWs = () => ws

  /**
   * 设置连接成功回调
   * @param callback 连接成功时执行的回调函数
   */
  const setOnConnectedCallback = (callback: () => void) => {
    onConnectedCallback = callback
  }

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
    getWs,
    setOnConnectedCallback
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
