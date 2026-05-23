import { ref } from 'vue'
import { API_BASE_URL } from '../config'

export function getDefaultServerUrl(): string {
  return API_BASE_URL
}

export function getStoredServerUrl(): string {
  const url = localStorage.getItem('serverUrl') || API_BASE_URL
  return url.replace(/\/+$/, '')
}

const serverUrl = ref(getStoredServerUrl())

export function useServerUrl() {
  function setServerUrl(url: string) {
    const cleanUrl = url.replace(/\/+$/, '')
    serverUrl.value = cleanUrl
    localStorage.setItem('serverUrl', cleanUrl)
    if (window.electron?.ipcRenderer) {
      window.electron.ipcRenderer.send('set-server-url', cleanUrl)
    }
  }

  return {
    serverUrl,
    setServerUrl,
    getServerUrl: (): string => serverUrl.value
  }
}
