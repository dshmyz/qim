<template>
  <div class="mini-app-manager">
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
              <img :src="miniApp.icon || generateAvatar(miniApp.name || '小程序')" :alt="miniApp.name" />
            </div>
            <div class="mini-app-item-name">{{ miniApp.name }}</div>
            <div class="mini-app-item-actions">
              <button class="mini-app-action-btn" @click="handleSendMiniApp(miniApp)">发送</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <MiniAppDrawer
      :mini-app="activeMiniApp"
      @close="activeMiniApp = null"
      @show-toast="handleMiniAppToast"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { generateAvatar } from '../../utils/avatar'
import MiniAppDrawer from '../miniapp/MiniAppDrawer.vue'
import type { MiniAppData } from '../miniapp/MiniAppDrawer.vue'
import { useChatState } from '../../composables/useChatState'
import { useRequest } from '../../composables/useRequest'
import { useChatStore } from '../../stores/chat'
import { useCurrentUser } from '../../composables/useCurrentUser'
import { nextTick } from 'vue'

const { $message } = useChatState()
const { request } = useRequest()
const chatStore = useChatStore()
const { currentUser } = useCurrentUser()

const props = defineProps<{
  showMiniAppList: boolean
}>()

const emit = defineEmits<{
  'update:showMiniAppList': [value: boolean]
}>()

const miniApps = ref<MiniAppData[]>([])
const loading = ref(false)
const activeMiniApp = ref<MiniAppData | null>(null)

const closeMiniAppList = () => {
  emit('update:showMiniAppList', false)
}

const launchMiniApp = (miniApp: MiniAppData) => {
  activeMiniApp.value = miniApp
}

const handleSendMiniApp = async (miniApp: MiniAppData) => {
  console.log('发送小程序消息:', miniApp)
  const conversationId = chatStore.currentConversationId
  console.log('从 store 获取会话ID:', conversationId)
  
  if (!conversationId) {
    $message.warning('请先选择一个聊天会话')
    return
  }

  try {
    const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify({
        type: 'miniApp',
        content: JSON.stringify(miniApp)
      })
    })

    if (response.code === 0) {
      // 构建消息对象
      const newMessage = {
        id: response.data.id?.toString() || Date.now().toString(),
        content: response.data.content || JSON.stringify(miniApp),
        sender: {
          id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
          name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: 'miniApp',
        isSelf: true,
        isRead: false,
        conversationId: conversationId,
        miniAppData: miniApp
      }

      // 更新本地消息列表
      chatStore.receiveMessage(conversationId, newMessage as any, true)

      $message.success(`小程序 "${miniApp.name}" 已发送`)
      closeMiniAppList()
    } else {
      throw new Error(response.message || '发送失败')
    }
  } catch (error) {
    console.error('发送小程序消息失败:', error)
    $message.error('发送失败，请重试')
  }
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
    const response = await request('/api/v1/mini-apps', {
      method: 'GET',
      params: { status: 'active' }
    })

    if (response.code !== 0) {
      throw new Error('获取小程序列表失败')
    }

    const list = response.data?.list ?? response.data ?? []
    miniApps.value = list.map((item: any) => ({
      id: item.id,
      name: item.name,
      icon: item.icon || '',
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
.mini-app-manager {
  position: relative;
  z-index: auto;
}

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
