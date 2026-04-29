<template>
  <div v-if="showMiniAppList" class="mini-app-panel-container" @click="closeMiniAppList">
    <div class="mini-app-panel" @click.stop>
      <div class="mini-app-panel-header">
        <h4>小程序</h4>
        <button class="close-btn" @click="closeMiniAppList">×</button>
      </div>

      <div v-if="loading" class="mini-app-loading">
        <div class="loading-spinner"></div>
        <span>加载中...</span>
      </div>

      <div v-else-if="miniApps.length === 0" class="mini-app-empty">
        <span>暂无小程序</span>
      </div>

      <div v-else class="mini-app-grid">
        <div v-for="miniApp in miniApps" :key="miniApp.id" class="mini-app-item">
          <div class="mini-app-item-icon" @click="launchMiniApp(miniApp)">
            <img :src="miniApp.icon || defaultIcon" :alt="miniApp.name" />
          </div>
          <div class="mini-app-item-name">{{ miniApp.name }}</div>
          <div class="mini-app-item-actions">
            <button class="mini-app-action-btn" @click="sendMiniAppMessage(miniApp)">发送</button>
          </div>
        </div>
      </div>
    </div>
  </div>

  <MiniAppLoader
    :mini-app="activeMiniApp"
    @close="activeMiniApp = null"
    @show-toast="handleMiniAppToast"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { generateAvatar } from '../../utils/avatar'
import MiniAppLoader from '../miniapp/MiniAppLoader.vue'
import type { MiniAppData } from '../miniapp/MiniAppLoader.vue'
import { API_BASE_URL } from '../../config'
import { useChatState } from '../../composables/useChatState'

const { $message } = useChatState()

const props = defineProps<{
  showMiniAppList: boolean
}>()

const emit = defineEmits<{
  'update:showMiniAppList': [value: boolean]
  'send-mini-app-message': [miniApp: MiniAppData]
}>()

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)
const miniApps = ref<MiniAppData[]>([])
const loading = ref(false)
const activeMiniApp = ref<MiniAppData | null>(null)
const defaultIcon = generateAvatar('default')

const closeMiniAppList = () => {
  emit('update:showMiniAppList', false)
}

const launchMiniApp = (miniApp: MiniAppData) => {
  activeMiniApp.value = miniApp
}

const sendMiniAppMessage = (miniApp: MiniAppData) => {
  emit('send-mini-app-message', miniApp)
  closeMiniAppList()
}

const handleMiniAppToast = (message: string) => {
  $message.info(message)
}

const loadMiniApps = async () => {
  const hasData = miniApps.value.length > 0
  if (!hasData) {
    loading.value = true
  }

  try {
    const token = getToken()
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }

    const response = await fetch(`${serverUrl.value}/api/v1/mini-apps?status=active`, {
      method: 'GET',
      headers
    })

    if (!response.ok) {
      throw new Error('获取小程序列表失败')
    }

    const result = await response.json()
    if (result.code !== 0) {
      throw new Error('获取小程序列表失败')
    }

    const list = result.data?.list ?? result.data ?? []
    miniApps.value = list.map((item: any) => ({
      id: item.id,
      name: item.name,
      icon: item.icon || defaultIcon,
      path: item.path,
      description: item.description || '',
      status: item.status,
      permissions: item.permissions || '',
    }))
  } catch (error) {
    console.error('加载小程序列表失败:', error)
    if (!hasData) {
      miniApps.value = []
    }
  } finally {
    loading.value = false
  }
}

const getToken = () => {
  return localStorage.getItem('token')
}

watch(() => props.showMiniAppList, (visible) => {
  if (visible) {
    loadMiniApps()
  }
})
</script>

<style scoped>
.mini-app-panel-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
}

.mini-app-panel {
  background: var(--sidebar-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 600px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.mini-app-panel-header {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mini-app-panel-header h4 {
  margin: 0;
  color: var(--text-color);
}

.close-btn {
  background: none;
  border: none;
  font-size: 20px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.mini-app-grid {
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 20px;
  max-height: 60vh;
  overflow-y: auto;
}

.mini-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  cursor: pointer;
}

.mini-app-item-icon {
  width: 80px;
  height: 80px;
  border-radius: 16px;
  background: var(--content-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 8px;
  transition: all 0.2s ease;
}

.mini-app-item-icon:hover {
  transform: scale(1.05);
}

.mini-app-item-icon img {
  width: 50px;
  height: 50px;
  border-radius: 12px;
}

.mini-app-item-name {
  font-size: 14px;
  color: var(--text-color);
  margin-bottom: 8px;
  text-align: center;
}

.mini-app-item-actions {
  display: flex;
  gap: 4px;
}

.mini-app-action-btn {
  padding: 4px 8px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mini-app-action-btn:hover {
  opacity: 0.8;
}

.mini-app-loading,
.mini-app-empty {
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
