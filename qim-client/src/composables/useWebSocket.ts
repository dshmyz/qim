import { ref, readonly } from 'vue'
import QMessage from '../utils/qmessage'
import { calculateReconnectDelay, shouldReconnect, DEFAULT_RECONNECT_CONFIG } from '../utils/websocketReconnect'
import { connectionMonitor } from '../utils/connectionMonitor'
import { messageQueue } from '../utils/messageQueue'
import { APP_CONFIG } from '../config/appConfig'

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
let manualDisconnect = false
let sessionExpired = false
// 一次连接周期内是否已尝试过 refresh_token 自动刷新。防止刷新后仍被拒时
// 无限次 POST /auth/refresh 刷日志；每次建立新连接重置。
let refreshAttempted = false

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
  const handleMessage = async (event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data)

      if (message.type === 'pong') {
        connectionMonitor.recordPong()
        return
      }

      // 处理 WebSocket 认证响应
      if (message.type === 'auth_success') {
        isConnected.value = true
        setNetworkError(false, '')
        reconnectAttempts = 0
        console.log('[WS] 认证成功, user_id:', message.data?.user_id)

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
        return
      }

      if (message.type === 'auth_error') {
        console.error('[WS] 认证失败:', message.data?.message)
        // 每次连接最多自动尝试一次 refresh_token 刷新；若刷新后仍被拒，直接进入
        // 会话过期，避免无限次 POST /auth/refresh 造成日志刷屏。
        if (refreshAttempted) {
          sessionExpired = true
          if (onSessionExpiredCallback) {
            onSessionExpiredCallback()
          }
          disconnect()
          return
        }
        refreshAttempted = true
        // 尝试用 refresh_token 刷新后重新认证
        const refreshToken = localStorage.getItem('refresh_token')
        if (refreshToken) {
          try {
            const baseURL = localStorage.getItem('serverUrl') || wsUrl
            const cleanBaseURL = baseURL.replace(/\/+$/, '')
            const axios = (await import('axios')).default
            const refreshResponse = await axios.post(`${cleanBaseURL}/api/v1/auth/refresh`, {}, {
              headers: { 'Authorization': `Bearer ${refreshToken}` }
            })
            if (refreshResponse.data?.code === 0 && refreshResponse.data?.data?.token) {
              const newToken = refreshResponse.data.data.token
              const newRefreshToken = refreshResponse.data.data.refresh_token
              localStorage.setItem('token', newToken)
              if (newRefreshToken) {
                localStorage.setItem('refresh_token', newRefreshToken)
              }
              // 用新 token 重新发送 auth 消息
              if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({ type: 'auth', data: { token: newToken } }))
                console.log('[WS] token 刷新成功，重新发送认证消息')
                return
              }
            }
          } catch (e) {
            console.error('[WS] token 刷新失败:', e)
          }
        }
        // refresh 失败或无 refresh_token，触发会话过期
        sessionExpired = true
        if (onSessionExpiredCallback) {
          onSessionExpiredCallback()
        }
        disconnect()
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

    // 会话过期是终结态：只有重新登录（通常伴随页面刷新）才会重新进入 connect，
    // 这里不能重置该标志，否则自动重连会清掉它会话过期保护，导致
    // connect → auth_error → /auth/refresh → sessionExpired → 重连 → connect 的无限循环刷日志。
    if (sessionExpired) {
      setNetworkError(true, '会话已过期，请重新登录')
      return
    }

    // 新连接开启一次新的刷新尝试机会
    refreshAttempted = false

    const token = localStorage.getItem('token')
    if (!token) {
      setNetworkError(true, '未登录，请先登录')
      return
    }

    try {
      const storedUrl = localStorage.getItem('serverUrl')
      const serverUrl = storedUrl || wsUrl
      const cleanUrl = serverUrl.replace(/\/+$/, '')

      // 携带客户端版本和平台信息，用于版本分布统计
      const platform = navigator.userAgent.toLowerCase().includes('mac') ? 'macos'
        : navigator.userAgent.toLowerCase().includes('linux') ? 'linux'
        : 'windows'
      const version = APP_CONFIG.version
      const versionQuery = `&version=${encodeURIComponent(version)}&platform=${platform}`

      const wsFullUrl = cleanUrl.startsWith('ws')
        ? cleanUrl + (cleanUrl.includes('?') ? versionQuery : `?${versionQuery.slice(1)}`)
        : `ws${cleanUrl.startsWith('https') ? 's' : ''}://${cleanUrl.replace(/^https?:\/\//, '')}/api/v1/ws?${versionQuery.slice(1)}`

      console.log('[WS] connecting to', wsFullUrl, 'localStorage serverUrl:', storedUrl, 'wsUrl:', wsUrl)
      ws = new WebSocket(wsFullUrl)

      if (typeof window !== 'undefined') {
        ;(window as any).ws = ws
      }

      ws.onopen = () => {
        // 连接建立后，通过首条消息发送认证 token
        ws!.send(JSON.stringify({ type: 'auth', data: { token } }))
        startHeartbeat()
        console.log('WebSocket connected, sending auth message')
      }

      ws.onmessage = handleMessage

      ws.onclose = (event: CloseEvent) => {
        isConnected.value = false
        stopHeartbeat()

        // 主动断开或会话过期时不重连
        if (manualDisconnect || sessionExpired) {
          manualDisconnect = false
          return
        }

        setNetworkError(true, '网络连接已断开，正在尝试重新连接...')

        if (event.code === 4401 || (event.reason && event.reason.includes('401'))) {
          sessionExpired = true
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
    manualDisconnect = true
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
