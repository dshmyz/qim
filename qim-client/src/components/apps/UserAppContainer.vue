<template>
  <div class="user-app-container">
    <AppHeader :title="app.name" @back="$emit('back')">
    </AppHeader>
    <div v-if="app.url" class="user-app-content">
      <!-- Electron 环境使用 <webview>：独立会话（按应用隔离登录态）+ 逐应用弹窗策略；
           非 Electron（纯浏览器预览）降级为 <iframe>。 -->
      <webview
        v-if="isElectron"
        ref="webviewRef"
        class="user-app-webview"
        :src="app.url"
        :partition="webviewPartition"
        allowpopups
      ></webview>
      <iframe
        v-else
        :src="app.url"
        class="user-app-iframe"
        frameborder="0"
        allowfullscreen
      ></iframe>
    </div>
    <div v-else class="user-app-content">
      <div class="empty-user-app">
        <div class="empty-icon"><i class="fas fa-link"></i></div>
        <p>该应用没有配置URL</p>
        <p class="empty-hint">请在应用管理中编辑应用，添加URL地址</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import AppHeader from './AppHeader.vue'

const props = defineProps<{
  app: {
    id: string | number
    name: string
    url: string
  }
}>()

defineEmits(['back'])

// Electron 环境判断：window.electron 由 preload 注入，纯浏览器下不存在
const isElectron = !!window.electron

// 内嵌应用按应用隔离持久会话（persist: 保留登录态/cookie），互不影响
const webviewPartition = computed(() => `persist:userapp-${props.app.id}`)

// webview 为 Electron 原生 DOM 元素，其类型不在全局命名空间内，
// 用 any 表述以避免依赖 electron 全局类型（与工程内其它 electron 交互保持一致）。
const webviewRef = ref<any>(null)

// 内部/外部判断依据：与内嵌应用配置的 URL 同源（同协议+同域名+同端口）视为内部，
// 在应用内新窗口打开；其余（跨域或危险协议）一律交给系统默认浏览器。
const appOrigin = computed(() => {
  try {
    const u = new URL(props.app.url)
    return u.origin
  } catch {
    return ''
  }
})

function isSameOrigin(url: string): boolean {
  try {
    const u = new URL(url)
    if (u.protocol !== 'http:' && u.protocol !== 'https:') return false
    return appOrigin.value ? u.origin === appOrigin.value : false
  } catch {
    return false
  }
}

const openExternalLink = (url: string) => {
  try {
    if (window.electron?.shell?.openExternal) {
      window.electron.shell.openExternal(url)
    } else {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  } catch (e) {
    console.error('打开外部链接失败:', e)
    window.open(url, '_blank', 'noopener,noreferrer')
  }
}

// 绑定 webview 弹窗策略：同源内部跳转留在应用内，跨域外部链接走系统浏览器
const configurePopupPolicy = (webview: any) => {
  try {
    const contents = webview.getWebContents()
    if (!contents || typeof contents.setWindowOpenHandler !== 'function') return
    contents.setWindowOpenHandler(({ url }: { url: string }) => {
      if (isSameOrigin(url)) {
        return { action: 'allow' }
      }
      openExternalLink(url)
      return { action: 'deny' }
    })
  } catch (e) {
    console.error('配置内嵌应用弹窗策略失败:', e)
  }
}

const handleWebviewAttach = (event: Event) => {
  const webview = ((event.target as any) || webviewRef.value) as any
  if (webview) configurePopupPolicy(webview)
}

onMounted(() => {
  if (isElectron && webviewRef.value) {
    // webview 异步 attach，did-attach 后 webContents 才可用
    webviewRef.value.addEventListener('did-attach', handleWebviewAttach)
    // 已 attach 的情况（缓存复用）直接绑定
    configurePopupPolicy(webviewRef.value)
  }
})

onBeforeUnmount(() => {
  if (!webviewRef.value) return
  webviewRef.value.removeEventListener('did-attach', handleWebviewAttach)
  // 显式关闭 guest webContents，防止加载失败/未正确卸载的 webview 挂住，导致主窗口关不掉
  try {
    webviewRef.value.close()
  } catch (e) {
    console.error('关闭内嵌应用 webview 失败:', e)
  }
})
</script>

<style scoped>
.user-app-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.user-app-content {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.user-app-iframe {
  width: 100%;
  height: 100%;
  border: none;
  flex: 1;
}

.user-app-webview {
  width: 100%;
  height: 100%;
  flex: 1;
}

.empty-user-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-secondary);
}

.empty-user-app .empty-icon {
  font-size: 48px;
  color: var(--text-tertiary);
}

.empty-user-app p {
  margin: 0;
  font-size: var(--font-size-sm);
}

.empty-user-app .empty-hint {
  font-size: var(--font-size-xxs);
  color: var(--text-tertiary);
}
</style>
