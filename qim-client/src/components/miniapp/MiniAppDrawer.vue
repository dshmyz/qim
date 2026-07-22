<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="visible" class="drawer-overlay" @click="handleOverlayClick">
        <div class="drawer-panel" :style="{ width: drawerWidth + 'px' }" @click.stop>
          <div class="drawer-header">
            <div class="drawer-title">
              <img v-if="miniApp?.icon" :src="miniApp.icon" :alt="miniApp.name" class="drawer-icon" />
              <span>{{ miniApp?.name || '小程序' }}</span>
            </div>
            <button
              class="drawer-close-btn"
              type="button"
              title="关闭"
              @pointerdown.stop.prevent="close"
              @click.stop.prevent="close"
            >
              ×
            </button>
          </div>
          <div class="drawer-resize-handle" @mousedown="startResize"></div>
          <div class="drawer-body">
            <div v-if="loading" class="drawer-loading">
              <div class="loading-spinner"></div>
              <span>加载中...</span>
            </div>
            <div v-if="error" class="drawer-error">
              <div class="error-icon"><i class="fas fa-triangle-exclamation"></i></div>
              <p class="error-title">加载失败</p>
              <p class="error-message">{{ errorMessage }}</p>
              <button class="drawer-retry-btn" @click="loadMiniApp">重试</button>
            </div>
            <iframe
              v-show="!loading && !error"
              ref="iframeRef"
              class="drawer-iframe"
              :src="iframeSrc"
              :sandbox="shouldSandbox ? 'allow-scripts allow-same-origin allow-forms allow-popups allow-modals' : undefined"
              :allow="getIframeAllow()"
              @load="handleIframeLoad"
              @error="handleIframeError"
            />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useMiniAppBridge, type MiniAppData } from '../../composables/useMiniAppBridge'

export type { MiniAppData }
export type { MiniAppBridgeMessage } from '../../composables/useMiniAppBridge'

const props = defineProps<{
  miniApp: MiniAppData | null
}>()

const emit = defineEmits<{
  close: []
  'show-toast': [message: string]
}>()

const {
  visible,
  iframeRef,
  loading,
  error,
  errorMessage,
  iframeSrc,
  shouldSandbox,
  getIframeAllow,
  loadMiniApp,
  close,
  handleOverlayClick,
  handleIframeLoad,
  handleIframeError,
} = useMiniAppBridge(props, emit)

// Drawer 独有：宽度拖拽
const drawerWidth = ref(400)
const isResizing = ref(false)
const startX = ref(0)
const startWidth = ref(0)

const startResize = (e: MouseEvent) => {
  isResizing.value = true
  startX.value = e.clientX
  startWidth.value = drawerWidth.value
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
  e.preventDefault()
}

const handleResize = (e: MouseEvent) => {
  if (!isResizing.value) return
  const delta = startX.value - e.clientX
  const newWidth = startWidth.value + delta
  drawerWidth.value = Math.min(600, Math.max(300, newWidth))
}

const stopResize = () => {
  isResizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// Drawer 独有：Esc 关闭
const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && visible.value) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeyDown)
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
})
</script>

<style scoped>
.drawer-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}

.drawer-panel {
  background: var(--sidebar-bg, #1a1a2e);
  height: 100%;
  display: flex;
  flex-direction: column;
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.15);
  border-radius: 12px 0 0 12px;
  position: relative;
  overflow: hidden;
}

.drawer-header {
  position: relative;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  flex-shrink: 0;
}

.drawer-title {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-color, #fff);
  font-size: 16px;
  font-weight: 600;
}

.drawer-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
}

.drawer-close-btn {
  position: relative;
  z-index: 21;
  width: 36px;
  height: 36px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 36px;
  background: transparent;
  border: none;
  font-size: 22px;
  font-family: Arial, Helvetica, sans-serif;
  color: var(--text-secondary, #888);
  cursor: pointer;
  pointer-events: auto;
  padding: 0;
  border-radius: 10px;
  line-height: 36px;
  text-align: center;
  transition: background 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.drawer-close-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-color, #fff);
}

.drawer-close-btn:active {
  transform: scale(0.96);
}

.drawer-resize-handle {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 8px;
  cursor: col-resize;
  z-index: 10;
}

.drawer-resize-handle:hover {
  background: rgba(74, 144, 217, 0.2);
}

.drawer-body {
  flex: 1;
  position: relative;
  overflow: hidden;
}

.drawer-iframe {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

.drawer-loading {
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

.drawer-error {
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

.drawer-retry-btn {
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

.drawer-retry-btn:hover {
  opacity: 0.85;
}

.drawer-enter-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.drawer-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.drawer-enter-from {
  opacity: 0;
}

.drawer-enter-from .drawer-panel {
  transform: translateX(100%);
}

.drawer-leave-to {
  opacity: 0;
}

.drawer-leave-to .drawer-panel {
  transform: translateX(100%);
}
</style>
