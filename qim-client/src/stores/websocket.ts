import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useWebSocketStore = defineStore('websocket', () => {
  const isConnected = ref(false)
  const showNetworkError = ref(false)
  const networkErrorMsg = ref('')
  const sessionExpired = ref(false)

  function setConnected(connected: boolean) {
    isConnected.value = connected
  }

  function setNetworkError(show: boolean, msg: string) {
    showNetworkError.value = show
    networkErrorMsg.value = msg
  }

  function setSessionExpired(expired: boolean) {
    sessionExpired.value = expired
  }

  function reset() {
    isConnected.value = false
    showNetworkError.value = false
    networkErrorMsg.value = ''
    sessionExpired.value = false
  }

  return {
    isConnected,
    showNetworkError,
    networkErrorMsg,
    sessionExpired,
    setConnected,
    setNetworkError,
    setSessionExpired,
    reset,
  }
})
