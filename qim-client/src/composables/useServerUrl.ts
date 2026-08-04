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

// 启动时从主进程拉取已持久化的服务器地址并归一到本地，保证与更新检查共用同一事实源。
// 仅当本地尚未保存地址时采用主进程值（避免覆盖用户正在设置的地址）。
function reconcileServerUrlFromMain() {
  if (!window.electron?.ipcRenderer) return
  window.electron.ipcRenderer.on('server-url', (_event: unknown, url: string) => {
    if (!url || typeof url !== 'string') return
    const clean = String(url).replace(/\/+$/, '')
    if (!localStorage.getItem('serverUrl')) {
      serverUrl.value = clean
      localStorage.setItem('serverUrl', clean)
    }
  })
  window.electron.ipcRenderer.send('get-server-url')
}

// 仅在非浏览器（存在主进程桥接）环境下执行启动归一
try {
  if (window.electron?.ipcRenderer) reconcileServerUrlFromMain()
} catch (error) {
  // 纯 Web 预览/SSR 等无 electron 环境静默跳过
}

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
