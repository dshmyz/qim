<template>
  <div v-if="visible" class="miniapp-loader-overlay" @click="handleOverlayClick">
    <div class="miniapp-loader-container" @click.stop>
      <div class="miniapp-loader-header">
        <div class="miniapp-loader-title">
          <img v-if="miniApp?.icon" :src="miniApp.icon" :alt="miniApp.name" class="miniapp-icon" />
          <span>{{ miniApp?.name || '小程序' }}</span>
        </div>
        <button class="miniapp-close-btn" @click="close" title="关闭">×</button>
      </div>
      <div class="miniapp-loader-body">
        <div v-if="loading" class="miniapp-loading">
          <div class="loading-spinner"></div>
          <span>加载中...</span>
        </div>
        <div v-if="error" class="miniapp-error">
          <div class="error-icon">⚠</div>
          <p class="error-title">加载失败</p>
          <p class="error-message">{{ errorMessage }}</p>
          <button class="miniapp-retry-btn" @click="loadMiniApp">重试</button>
        </div>
        <iframe
          v-show="!loading && !error"
          ref="iframeRef"
          class="miniapp-iframe"
          :src="iframeSrc"
          :sandbox="shouldSandbox ? 'allow-scripts allow-same-origin allow-forms allow-popups allow-modals' : undefined"
          :allow="getIframeAllow()"
          @load="handleIframeLoad"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { getCurrentUser } from '../../utils/user'

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
  type: string
  payload?: any
}

const props = defineProps<{
  miniApp: MiniAppData | null
}>()

const emit = defineEmits<{
  close: []
  'show-toast': [message: string]
}>()

const visible = computed(() => !!props.miniApp)
const iframeRef = ref<HTMLIFrameElement | null>(null)
const loading = ref(false)
const error = ref(false)
const errorMessage = ref('')
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

const iframeSrc = computed(() => {
  if (!props.miniApp?.path) return ''
  const path = props.miniApp.path
  if (path.startsWith('http://') || path.startsWith('https://')) return path
  return `${getStoredServerUrl()}${path.startsWith('/') ? '' : '/'}${path}`
})

const hasPermission = (perm: string): boolean => {
  try {
    const perms = props.miniApp?.permissions ? JSON.parse(props.miniApp.permissions) as string[] : []
    return perms.includes(perm)
  } catch {
    return false
  }
}

const handleIframeLoad = () => {
  loading.value = false
  error.value = false
  injectBridgeScript()
}

const loadMiniApp = () => {
  if (!props.miniApp?.path) {
    error.value = true
    errorMessage.value = '小程序路径未配置'
    return
  }
  loading.value = true
  error.value = false
}

const close = () => {
  emit('close')
  loading.value = false
  error.value = false
  errorMessage.value = ''
}

const handleOverlayClick = () => {
  close()
}

const handleMiniAppMessage = (event: MessageEvent) => {
  if (!event.data || typeof event.data !== 'object') return

  const data = event.data as MiniAppBridgeMessage

  switch (data.type) {
    case 'miniapp-loaded':
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

const injectBridgeScript = () => {
  if (!iframeRef.value?.contentWindow) return

  iframeRef.value.contentWindow.postMessage({
    type: 'bridge-ready',
    payload: { appId: props.miniApp?.appID },
  }, '*')
}

onMounted(() => {
  window.addEventListener('message', handleMiniAppMessage)
})

onBeforeUnmount(() => {
  window.removeEventListener('message', handleMiniAppMessage)
})

watch(() => props.miniApp, (newVal) => {
  if (newVal) {
    loadMiniApp()
  }
}, { immediate: true })
</script>

<style scoped>
.miniapp-loader-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: overlayFadeIn 0.2s ease;
}

@keyframes overlayFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.miniapp-loader-container {
  background: var(--sidebar-bg, #1a1a2e);
  border-radius: 12px;
  width: 90%;
  max-width: 800px;
  height: 80vh;
  max-height: 600px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  animation: containerSlideIn 0.3s ease;
}

@keyframes containerSlideIn {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(-10px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.miniapp-loader-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.miniapp-loader-title {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-color, #fff);
  font-size: 16px;
  font-weight: 600;
}

.miniapp-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
}

.miniapp-close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: var(--text-secondary, #888);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.2s ease;
  line-height: 1;
}

.miniapp-close-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-color, #fff);
}

.miniapp-loader-body {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.miniapp-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

.miniapp-loading {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  color: var(--text-secondary, #888);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top-color: var(--primary-color, #4a90d9);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.miniapp-error {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
  color: var(--text-color, #fff);
}

.error-icon {
  font-size: 48px;
  color: #e6a23c;
}

.error-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
}

.error-message {
  font-size: 14px;
  color: var(--text-secondary, #888);
  margin: 0;
  max-width: 300px;
}

.miniapp-retry-btn {
  margin-top: 8px;
  padding: 8px 24px;
  background: var(--primary-color, #4a90d9);
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.miniapp-retry-btn:hover {
  opacity: 0.85;
}
</style>
