import { ref, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'

interface WebSocketMessage {
  type: string
  data: any
}

type MessageHandler = (message: WebSocketMessage) => void

let ws: WebSocket | null = null
let reconnectTimer: number | null = null
const handlers: MessageHandler[] = []
const isConnected = ref(false)
const showNetworkError = ref(false)
const networkErrorMsg = ref('网络连接已断开')

const RECONNECT_INTERVAL = 3000

export function useWebSocket(wsUrl: string) {
  const connect = () => {
    if (ws && ws.readyState === WebSocket.OPEN) return

    const token = localStorage.getItem('token')
    if (!token) {
      showNetworkError.value = true
      networkErrorMsg.value = '未登录，请先登录'
      return
    }

    try {
      ws = new WebSocket(`${wsUrl}?token=${token}`)

      ws.onopen = () => {
        isConnected.value = true
        showNetworkError.value = false
      }

      ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          handlers.forEach(handler => handler(message))
        } catch (error) {
          console.error('WebSocket message parse error:', error)
        }
      }

      ws.onclose = () => {
        isConnected.value = false
        showNetworkError.value = true
        networkErrorMsg.value = '网络连接已断开，正在尝试重新连接...'
        scheduleReconnect(wsUrl)
      }

      ws.onerror = () => {
        isConnected.value = false
      }
    } catch (error) {
      console.error('WebSocket connection error:', error)
      showNetworkError.value = true
      networkErrorMsg.value = '网络连接失败'
      scheduleReconnect(wsUrl)
    }
  }

  const scheduleReconnect = (url: string) => {
    if (reconnectTimer) return
    reconnectTimer = window.setTimeout(() => {
      reconnectTimer = null
      connect()
    }, RECONNECT_INTERVAL)
  }

  const disconnect = () => {
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

  const sendMessage = (data: any) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(data))
    } else {
      ElMessage.error('网络连接已断开')
    }
  }

  const addHandler = (handler: MessageHandler) => {
    handlers.push(handler)
  }

  const removeHandler = (handler: MessageHandler) => {
    const index = handlers.indexOf(handler)
    if (index !== -1) {
      handlers.splice(index, 1)
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    isConnected,
    showNetworkError,
    networkErrorMsg,
    connect,
    disconnect,
    sendMessage,
    addHandler,
    removeHandler
  }
}
