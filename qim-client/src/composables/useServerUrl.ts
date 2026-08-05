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

// 启动时把渲染进程已持久化的服务器地址反向同步到主进程（config.json）。
// 更新检查在主进程读 config.json.serverUrl，而该值只在「登录成功」或「手动保存服务器设置」
// 时经 IPC 写入；老用户升级后靠 localStorage 的 token 自动登录，从不走这两个分支，导致
// config.json.serverUrl 为空 → 检查更新报“未配置更新服务器地址”。此处兜底：只要本地已有
// 显式保存的自定义地址，就把同一地址同步给主进程，让老用户无需重新登录/保存即可检查更新。
// 仅同步显式保存的值（localStorage.getItem 非空），避免把 API_BASE_URL 默认值写进 config.json。
function syncServerUrlToMain() {
  if (!window.electron?.ipcRenderer) return
  const saved = localStorage.getItem('serverUrl')
  if (!saved) return
  window.electron.ipcRenderer.send('set-server-url', saved.replace(/\/+$/, ''))
}

// 仅在非浏览器（存在主进程桥接）环境下执行启动归一
try {
  if (window.electron?.ipcRenderer) {
    syncServerUrlToMain()
    reconcileServerUrlFromMain()
  }
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
