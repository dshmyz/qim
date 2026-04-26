import { onMounted, onUnmounted } from 'vue'
import { useWebSocket, type WebSocketMessage } from './useWebSocket'
import { logger } from '../utils/logger';

type MessageHandler = (data: any) => void

interface MessageHandlers {
  [key: string]: MessageHandler
}

export function useWebSocketManager(serverUrl: any) {
  const { connectWithHandlers, disconnect, sendMessage, addHandler, isConnected, getWs } = useWebSocket(serverUrl.value)

  const localWs = {
    get readyState() { return getWs()?.readyState ?? WebSocket.CLOSED },
    send: (data: string) => sendMessage(JSON.parse(data)),
    close: () => disconnect(),
    onclose: null as ((event: CloseEvent) => void) | null,
    onerror: null as ((event: Event) => void) | null,
    onopen: null as ((event: Event) => void) | null,
    onmessage: null as ((event: MessageEvent) => void) | null
  }

  /**
   * 连接WebSocket
   * @param handleReconnect 重连处理函数
   * @param showNetworkError 显示网络错误的ref
   * @param networkErrorMsg 网络错误消息的ref
   * @param sessionExpired 会话过期状态的ref
   * @param messageHandlers 消息处理器
   */
  const connectWebSocket = (
    _handleReconnect: () => void,
    showNetworkError: any,
    networkErrorMsg: any,
    sessionExpired: any,
    messageHandlers: MessageHandlers
  ) => {
    connectWithHandlers(
      showNetworkError,
      networkErrorMsg,
      sessionExpired,
      messageHandlers
    )

    // 设置 ws 的事件回调
    const ws = getWs()
    if (ws) {
      // @ts-ignore
      localWs.onclose = () => {
        logger.log('WebSocket连接关闭')
      }

      // @ts-ignore
      localWs.onopen = () => {
        logger.log('WebSocket连接成功')
      }
    }
  }

  /**
   * 暴露sendWebSocketMessage到全局
   */
  const exposeGlobalWebSocket = () => {
    if (typeof window !== 'undefined') {
      ;(window as any).sendWebSocketMessage = (message: any) => sendMessage(message)
    }
  }

  /**
   * 关闭WebSocket连接
   */
  const disconnectWebSocket = () => {
    disconnect()
  }

  /**
   * 添加消息处理器
   * @param handler 消息处理器
   * @param messageType 消息类型
   */
  const addMessageHandler = (handler: (message: WebSocketMessage) => boolean, messageType?: string) => {
    addHandler(handler as any, messageType)
  }

  onMounted(() => {
    exposeGlobalWebSocket()
  })

  onUnmounted(() => {
    disconnectWebSocket()
  })

  return {
    ws: localWs,
    isConnected,
    connectWebSocket,
    disconnectWebSocket,
    sendMessage,
    addHandler: addMessageHandler
  }
}