# 小程序抽屉式弹出实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将小程序弹出方式从全屏遮罩改为右侧覆盖式抽屉，使用户可以边看聊天边使用工具

**架构：** 新建 MiniAppDrawer.vue 组件实现抽屉交互，修改 MiniAppManager.vue 将 MiniAppLoader 替换为 MiniAppDrawer。MiniAppLoader 保留用于聊天消息中的小程序卡片点击。

**技术栈：** Vue 3 + TypeScript + Composition API + CSS 动画

---

## 文件结构

| 文件 | 操作 | 职责 |
|------|------|------|
| `src/components/miniapp/MiniAppDrawer.vue` | 新建 | 抽屉组件：遮罩、滑入动画、iframe 加载、拖拽调整宽度 |
| `src/components/apps/MiniAppManager.vue` | 修改 | 将 MiniAppLoader 替换为 MiniAppDrawer |
| `src/components/miniapp/MiniAppLoader.vue` | 保留 | 不变，继续用于聊天消息中的小程序卡片 |

---

## 任务 1：创建 MiniAppDrawer.vue 组件

**文件：**
- 创建：`src/components/miniapp/MiniAppDrawer.vue`

- [ ] **步骤 1：创建组件模板**

```vue
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
            <button class="drawer-close-btn" @click="close" title="关闭">×</button>
          </div>
          <div class="drawer-resize-handle" @mousedown="startResize"></div>
          <div class="drawer-body">
            <div v-if="loading" class="drawer-loading">
              <div class="loading-spinner"></div>
              <span>加载中...</span>
            </div>
            <div v-if="error" class="drawer-error">
              <div class="error-icon">⚠</div>
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
```

- [ ] **步骤 2：创建组件脚本**

```vue
<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { request } from '../../composables/useRequest'
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
const drawerWidth = ref(400)

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

const resolvedPath = ref('')

const fetchLatestMiniApp = async () => {
  if (!props.miniApp?.appID) return
  try {
    const response = await request(`/api/v1/mini-apps/${props.miniApp.appID}`)
    if (response.code === 0 && response.data?.path) {
      resolvedPath.value = response.data.path
      return
    }
  } catch {}
  resolvedPath.value = props.miniApp?.path || ''
}

const iframeSrc = computed(() => {
  const path = resolvedPath.value || props.miniApp?.path
  if (!path) return ''
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

// 拖拽调整宽度
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

// ESC 键关闭
const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && visible.value) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('message', handleMiniAppMessage)
  window.addEventListener('keydown', handleKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('message', handleMiniAppMessage)
  window.removeEventListener('keydown', handleKeyDown)
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
})

watch(() => props.miniApp, async (newVal) => {
  if (newVal) {
    await fetchLatestMiniApp()
    loadMiniApp()
  }
}, { immediate: true })
</script>
```

- [ ] **步骤 3：创建组件样式**

```vue
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

.drawer-close-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-color, #fff);
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

/* 滑入/滑出动画 */
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
```

- [ ] **步骤 4：验证 TypeScript 类型**

运行：检查 IDE 是否有类型错误
预期：无类型错误

- [ ] **步骤 5：Commit**

```bash
git add src/components/miniapp/MiniAppDrawer.vue
git commit -m "feat(miniapp): add drawer component for mini app popup"
```

---

## 任务 2：修改 MiniAppManager.vue 使用抽屉组件

**文件：**
- 修改：`src/components/apps/MiniAppManager.vue`

- [ ] **步骤 1：替换模板中的 MiniAppLoader 为 MiniAppDrawer**

将：
```vue
<MiniAppLoader
  :mini-app="activeMiniApp"
  @close="activeMiniApp = null"
  @show-toast="handleMiniAppToast"
/>
```

改为：
```vue
<MiniAppDrawer
  :mini-app="activeMiniApp"
  @close="activeMiniApp = null"
  @show-toast="handleMiniAppToast"
/>
```

- [ ] **步骤 2：更新 import 语句**

将：
```typescript
import MiniAppLoader from '../miniapp/MiniAppLoader.vue'
import type { MiniAppData } from '../miniapp/MiniAppLoader.vue'
```

改为：
```typescript
import MiniAppDrawer from '../miniapp/MiniAppDrawer.vue'
import type { MiniAppData } from '../miniapp/MiniAppDrawer.vue'
```

- [ ] **步骤 3：验证组件通信**

确认：
- `activeMiniApp` 类型与 MiniAppDrawer 的 props 匹配
- `@close` 和 `@show-toast` 事件正确绑定

- [ ] **步骤 4：Commit**

```bash
git add src/components/apps/MiniAppManager.vue
git commit -m "refactor(miniapp): replace loader with drawer in manager"
```

---

## 任务 3：验证功能完整性

- [ ] **步骤 1：启动前端开发服务器**

运行：`cd qim-client && npm run dev`
预期：服务正常启动，无编译错误

- [ ] **步骤 2：测试抽屉弹出**

操作：
1. 点击小程序按钮
2. 点击任意小程序图标
预期：
- 抽屉从右侧滑入
- 小程序内容正常加载
- 点击遮罩或关闭按钮可关闭抽屉

- [ ] **步骤 3：测试拖拽调整宽度**

操作：
1. 打开抽屉
2. 拖拽左侧边缘
预期：
- 宽度可在 300px-600px 范围内调整
- 光标变为 col-resize

- [ ] **步骤 4：测试 ESC 键关闭**

操作：
1. 打开抽屉
2. 按 ESC 键
预期：抽屉滑出关闭

- [ ] **步骤 5：测试 Bridge 通信**

操作：
1. 打开需要权限的小程序（如密码生成器）
2. 测试剪贴板功能
预期：Bridge 通信正常，权限检查生效

- [ ] **步骤 6：验证 MiniAppLoader 仍可用**

操作：
1. 在聊天消息中点击小程序卡片
预期：MiniAppLoader 全屏弹出（不受影响）

- [ ] **步骤 7：最终 Commit**

```bash
git add -A
git commit -m "feat(miniapp): complete drawer implementation and testing"
```

---

## 规格覆盖度检查

| 规格需求 | 对应任务 | 状态 |
|---------|---------|------|
| 抽屉面板（400px 默认宽度） | 任务 1 | ✅ |
| 右侧滑入动画 | 任务 1 | ✅ |
| 半透明遮罩（点击关闭） | 任务 1 | ✅ |
| 拖拽调整宽度（300px-600px） | 任务 1 | ✅ |
| ESC 键关闭 | 任务 1 | ✅ |
| 替换 MiniAppManager 中的组件 | 任务 2 | ✅ |
| 保留 MiniAppLoader 用于聊天消息 | 无变更 | ✅ |
| Bridge 通信兼容 | 任务 1 | ✅ |
| 错误处理（加载失败、重试） | 任务 1 | ✅ |
