import { ref, computed, watch, onMounted, onBeforeUnmount, type Ref } from 'vue'
import { getStoredServerUrl } from './useServerUrl'
import { request } from './useRequest'
import { getCurrentUser } from '../utils/user'

export interface MiniAppData {
  id: number | string
  appID: string
  name: string
  icon?: string
  path: string
  description?: string
  status?: string
  permissions?: string
}

export interface MiniAppBridgeMessage {
  type: 'miniapp-loaded' | 'request-keyboard' | 'miniapp-toast' | 'get-user-info' | 'get-token' | 'api-request' | 'clipboard-read' | 'clipboard-write' | string
  payload?: any
}

export interface UseMiniAppBridgeOptions {
  /** 是否启用缓存破坏（每次打开刷新 iframe），默认 false */
  useCacheBuster?: boolean
}

export interface UseMiniAppBridgeProps {
  miniApp: MiniAppData | null
}

export interface UseMiniAppBridgeEmits {
  (e: 'close'): void
  (e: 'show-toast', message: string): void
}

/**
 * 小程序 iframe 宿主-iframe 通信协议
 *
 * 封装宿主与 iframe 之间的 postMessage 通信、权限校验、API 代理、
 * 加载/错误状态管理、路径解析等逻辑。供 MiniAppDrawer / MiniAppLoader 共用。
 *
 * 通信协议（iframe → 宿主）：
 * - miniapp-loaded：iframe 加载完成
 * - request-keyboard：小程序声明需要键盘输入，宿主聚焦 iframe
 * - miniapp-toast：请求宿主显示 toast
 * - get-user-info：请求用户信息（需 user_info 权限）
 * - get-token：请求 token（需 token 权限）
 * - api-request：代理发起 API 请求（需 api_request 权限）
 * - clipboard-read：读剪贴板（需 clipboard 权限）
 * - clipboard-write：写剪贴板（需 clipboard 权限）
 */
export function useMiniAppBridge(
  props: UseMiniAppBridgeProps,
  emit: UseMiniAppBridgeEmits,
  options: UseMiniAppBridgeOptions = {}
) {
  const { useCacheBuster = false } = options

  const visible = computed(() => !!props.miniApp)
  const iframeRef = ref<HTMLIFrameElement | null>(null)
  const loading = ref(false)
  const error = ref(false)
  const errorMessage = ref('')
  // 小程序是否已声明需要键盘输入
  const keyboardRequested = ref(false)

  // 从后端获取最新路径，避免历史消息中嵌入的旧路径过期
  const resolvedPath = ref('')
  // 每次打开小程序刷新，用于绕过 iframe 缓存加载最新文件
  const cacheBuster = ref('')

  const hasClipboardPermission = computed(() => {
    try {
      const perms = props.miniApp?.permissions ? JSON.parse(props.miniApp.permissions) as string[] : []
      return perms.includes('clipboard')
    } catch {
      return false
    }
  })

  const getIframeAllow = (): string => {
    if (hasClipboardPermission.value) {
      return 'clipboard-read; clipboard-write'
    }
    return ''
  }

  const shouldSandbox = computed(() => true)

  const fetchLatestMiniApp = async () => {
    if (!props.miniApp?.appID) return
    try {
      const response = await request(`/api/v1/mini-apps/${props.miniApp.appID}`)
      if (response.code === 0 && response.data?.path) {
        resolvedPath.value = response.data.path
        return
      }
    } catch {}
    // fallback 到 prop 里的路径
    resolvedPath.value = props.miniApp?.path || ''
  }

  const iframeSrc = computed(() => {
    const path = resolvedPath.value || props.miniApp?.path
    if (!path) return ''
    const base = (path.startsWith('http://') || path.startsWith('https://'))
      ? path
      : `${getStoredServerUrl()}${path.startsWith('/') ? '' : '/'}${path}`
    if (!useCacheBuster || !cacheBuster.value) return base
    return `${base}${base.includes('?') ? '&' : '?'}_t=${cacheBuster.value}`
  })

  const hasPermission = (perm: string): boolean => {
    try {
      const perms = props.miniApp?.permissions ? JSON.parse(props.miniApp.permissions) as string[] : []
      return perms.includes(perm)
    } catch {
      return false
    }
  }

  const injectBridgeScript = () => {
    if (!iframeRef.value?.contentWindow) return
    iframeRef.value.contentWindow.postMessage({
      type: 'bridge-ready',
      payload: { appId: props.miniApp?.appID },
    }, '*')
  }

  const handleIframeLoad = () => {
    loading.value = false
    error.value = false
    injectBridgeScript()
  }

  const handleIframeError = () => {
    loading.value = false
    error.value = true
    errorMessage.value = '小程序加载失败，请检查网络或稍后重试'
  }

  const loadMiniApp = () => {
    if (!props.miniApp?.path) {
      error.value = true
      errorMessage.value = '小程序路径未配置'
      return
    }
    loading.value = true
    error.value = false
    // iframe 10 秒超时兜底
    setTimeout(() => {
      if (loading.value) {
        handleIframeError()
      }
    }, 10000)
  }

  const close = () => {
    emit('close')
    loading.value = false
    error.value = false
    errorMessage.value = ''
    // 关闭时归还焦点到主应用
    if (keyboardRequested.value) {
      keyboardRequested.value = false
      ;(document.activeElement as HTMLElement | null)?.blur?.()
    }
  }

  const handleOverlayClick = () => {
    close()
  }

  const handleApiRequest = async (payload: { method: string; url: string; body?: any }) => {
    if (!payload || !payload.url) return
    const token = localStorage.getItem('token') || ''
    const url = payload.url.startsWith('http') ? payload.url : `${getStoredServerUrl()}${payload.url.startsWith('/') ? '' : '/'}${payload.url}`

    try {
      const response = await fetch(url, {
        method: payload.method || 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: payload.body ? JSON.stringify(payload.body) : undefined,
      })
      const result = await response.json()
      iframeRef.value?.contentWindow?.postMessage({
        type: 'api-response',
        payload: result,
      }, '*')
    } catch (err: any) {
      iframeRef.value?.contentWindow?.postMessage({
        type: 'api-response',
        payload: { code: 500, message: err.message || '请求失败' },
      }, '*')
    }
  }

  const handleMiniAppMessage = (event: MessageEvent) => {
    if (!event.data || typeof event.data !== 'object') return
    const data = event.data as MiniAppBridgeMessage

    switch (data.type) {
      case 'miniapp-loaded':
        break
      case 'request-keyboard':
        // 小程序主动声明需要键盘输入，聚焦 iframe 让原生 keydown 生效
        keyboardRequested.value = true
        iframeRef.value?.contentWindow?.focus()
        break
      case 'miniapp-toast':
        emit('show-toast', data.payload?.message || '')
        break
      case 'get-user-info':
        if (!hasPermission('user_info')) {
          iframeRef.value?.contentWindow?.postMessage({
            type: 'user-info-response',
            payload: { error: '未授予 user_info 权限' },
          }, '*')
          return
        }
        const user = getCurrentUser()
        iframeRef.value?.contentWindow?.postMessage({
          type: 'user-info-response',
          payload: {
            id: user.id,
            username: user.username,
            nickname: user.nickname || '',
            avatar: user.avatar || '',
          },
        }, '*')
        break
      case 'get-token':
        if (!hasPermission('token')) {
          iframeRef.value?.contentWindow?.postMessage({
            type: 'token-response',
            payload: { error: '未授予 token 权限' },
          }, '*')
          return
        }
        const token = localStorage.getItem('token') || ''
        iframeRef.value?.contentWindow?.postMessage({
          type: 'token-response',
          payload: { token },
        }, '*')
        break
      case 'api-request':
        if (!hasPermission('api_request')) {
          iframeRef.value?.contentWindow?.postMessage({
            type: 'api-response',
            payload: { code: 403, message: '无权限调用此 API' },
          }, '*')
          return
        }
        handleApiRequest(data.payload)
        break
      case 'clipboard-read':
        if (!hasPermission('clipboard')) {
          iframeRef.value?.contentWindow?.postMessage({
            type: 'clipboard-read-response',
            payload: { error: '未授予 clipboard 权限' },
          }, '*')
          return
        }
        navigator.clipboard.readText()
          .then(text => {
            iframeRef.value?.contentWindow?.postMessage({
              type: 'clipboard-read-response',
              payload: { text },
            }, '*')
          })
          .catch(err => {
            iframeRef.value?.contentWindow?.postMessage({
              type: 'clipboard-read-response',
              payload: { error: err.message || '读取剪贴板失败' },
            }, '*')
          })
        break
      case 'clipboard-write':
        if (!hasPermission('clipboard')) {
          iframeRef.value?.contentWindow?.postMessage({
            type: 'clipboard-write-response',
            payload: { error: '未授予 clipboard 权限' },
          }, '*')
          return
        }
        navigator.clipboard.writeText(data.payload?.text || '')
          .then(() => {
            iframeRef.value?.contentWindow?.postMessage({
              type: 'clipboard-write-response',
              payload: { success: true },
            }, '*')
          })
          .catch(err => {
            iframeRef.value?.contentWindow?.postMessage({
              type: 'clipboard-write-response',
              payload: { error: err.message || '写入剪贴板失败' },
            }, '*')
          })
        break
    }
  }

  onMounted(() => {
    window.addEventListener('message', handleMiniAppMessage)
  })

  onBeforeUnmount(() => {
    window.removeEventListener('message', handleMiniAppMessage)
  })

  watch(() => props.miniApp, async (newVal) => {
    if (newVal) {
      if (useCacheBuster) {
        cacheBuster.value = String(Date.now())
      }
      await fetchLatestMiniApp()
      loadMiniApp()
    }
  }, { immediate: true })

  return {
    // 状态
    visible,
    iframeRef,
    loading,
    error,
    errorMessage,
    keyboardRequested,
    // 计算属性
    iframeSrc,
    shouldSandbox,
    hasClipboardPermission,
    // 方法
    getIframeAllow,
    loadMiniApp,
    close,
    handleOverlayClick,
    handleIframeLoad,
    handleIframeError,
    handleMiniAppMessage,
  }
}
