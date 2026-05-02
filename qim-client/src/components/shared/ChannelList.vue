<template>
  <div class="channel-list">
    <div class="channel-header">
      <h3>频道</h3>
      <button v-if="hasAdminPermission" class="create-channel-btn" @click="showCreateChannelModal = true">
        <i class="fas fa-plus"></i> 创建频道
      </button>
    </div>
    
    <!-- 频道分类标签 -->
    <div class="channel-tabs">
      <button 
        class="tab-button" 
        :class="{ active: activeTab === 'subscribed' }" 
        @click="activeTab = 'subscribed'"
      >
        订阅频道
      </button>
      <button 
        class="tab-button" 
        :class="{ active: activeTab === 'discover' }" 
        @click="activeTab = 'discover'"
      >
        频道广场
      </button>
    </div>
    
    <div class="channel-list-content">
      <div v-if="loading" class="loading">
        <div class="loading-spinner"></div>
        <span>加载中...</span>
      </div>
      <div v-else-if="filteredChannels.length === 0" class="empty">
        <i class="fas fa-bullhorn empty-icon"></i>
        <h4>{{ activeTab === 'subscribed' ? '暂无订阅频道' : '暂无频道' }}</h4>
        <p>{{ activeTab === 'subscribed' ? '去频道广场订阅感兴趣的频道吧！' : '成为第一个创建频道的人吧！' }}</p>
        <button v-if="activeTab === 'subscribed'" class="switch-to-discover-btn" @click="activeTab = 'discover'">
          浏览频道广场
        </button>
      </div>
      <div v-else class="channels-grid">
        <div 
          v-for="channel in filteredChannels" 
          :key="channel.id" 
          class="channel-card"
          @click="selectChannel(channel)"
        >
          <div class="channel-card-header">
            <img :src="getAvatarUrl(channel.avatar, channel.name, serverUrl)" :alt="channel.name" class="channel-avatar" />
            <button 
              v-if="channel.is_subscribed" 
              class="subscribe-btn subscribed" 
              @click.stop="unsubscribeChannel(channel)"
            >
              <i class="fas fa-check"></i> 已订阅
            </button>
            <button 
              v-else 
              class="subscribe-btn" 
              @click.stop="subscribeChannel(channel)"
            >
              <i class="fas fa-plus"></i> 订阅
            </button>
          </div>
          <div class="channel-card-body">
            <h4 class="channel-name">{{ channel.name }}</h4>
            <p class="channel-description">{{ channel.description }}</p>
            <span v-if="channel.status === 'pending'" class="channel-status pending">待审批</span>
          </div>
          <div class="channel-card-footer">
            <span class="channel-creator">创建者: {{ channel.creator?.name || '未知' }}</span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 创建频道弹窗 -->
    <QDialog
      v-model:visible="showCreateChannelModal"
      title="创建频道"
      width="500px"
    >
      <div class="form-group">
        <label>频道名称</label>
        <input v-model="createChannelForm.name" type="text" placeholder="输入频道名称" class="form-input" />
      </div>
      <div class="form-group">
        <label>频道描述</label>
        <textarea v-model="createChannelForm.description" placeholder="输入频道描述" rows="3" class="form-textarea"></textarea>
      </div>
      <div class="form-group">
        <label>频道头像</label>
        <input v-model="createChannelForm.avatar" type="text" placeholder="输入头像URL" class="form-input" />
      </div>
      <template #footer>
        <button class="q-btn q-btn--default" @click="showCreateChannelModal = false">取消</button>
        <button class="q-btn q-btn--primary" @click="createChannel" :disabled="!createChannelForm.name">创建</button>
      </template>
    </QDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Channel } from '../../types'
import { API_BASE_URL } from '../../config'
import { getAvatarUrl } from '../../utils/avatar'
import QMessage from '../../utils/qmessage'
import QDialog from './QDialog.vue'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const props = defineProps<{
  currentUser: Record<string, any>
}>()

const emit = defineEmits<{
  (e: 'select-channel', channel: Channel): void
}>()

const channels = ref<Channel[]>([])
const loading = ref(false)
const showCreateChannelModal = ref(false)
const activeTab = ref('subscribed')

const createChannelForm = ref({
  name: '',
  description: '',
  avatar: ''
})

// 过滤后的频道列表
const filteredChannels = computed(() => {
  if (activeTab.value === 'subscribed') {
    return channels.value.filter(channel => channel.is_subscribed)
  } else {
    return channels.value
  }
})

const hasAdminPermission = computed(() => {
  return props.currentUser.isAdmin || props.currentUser.roles?.includes('system_admin')
})

const loadChannels = async () => {
  loading.value = true
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    })
    const data = await response.json()
    if (data.code === 0) {
      channels.value = data.data || []
    }
  } catch (error) {
    console.error('加载频道失败:', error)
  } finally {
    loading.value = false
  }
}

const createChannel = async () => {
  if (!createChannelForm.value.name) return
  
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(createChannelForm.value)
    })
    const data = await response.json()
    if (data.code === 0) {
      showCreateChannelModal.value = false
      createChannelForm.value = { name: '', description: '', avatar: '' }
      await loadChannels()
      QMessage.success('频道创建成功')
    }
  } catch (error) {
    console.error('创建频道失败:', error)
  }
}

const subscribeChannel = async (channel: Channel) => {
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/subscribe`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    })
    const data = await response.json()
    if (data.code === 0) {
      channel.is_subscribed = true
    }
  } catch (error) {
    console.error('订阅频道失败:', error)
  }
}

const unsubscribeChannel = async (channel: Channel) => {
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/unsubscribe`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    })
    const data = await response.json()
    if (data.code === 0) {
      channel.is_subscribed = false
    }
  } catch (error) {
    console.error('取消订阅失败:', error)
  }
}

const selectChannel = async (channel: Channel) => {
  // 加载频道消息
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/messages`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
        'Content-Type': 'application/json'
      }
    })
    const data = await response.json()
    if (data.code === 0) {
      channel.messages = data.data
    }
  } catch (error) {
    console.error('加载频道消息失败:', error)
  }
  
  // 触发选择事件
  emit('select-channel', channel)
}

const openCreateModal = () => {
  showCreateChannelModal.value = true
}

defineExpose({ openCreateModal })

onMounted(() => {
  loadChannels()
})
</script>

<style scoped>
.channel-list {
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  margin: 8px 8px;
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.channel-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--sidebar-bg);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

/* 频道分类标签 */
.channel-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  background: var(--sidebar-bg);
}

.tab-button {
  flex: 1;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
  border-bottom: 2px solid transparent;
}

.tab-button:hover {
  color: var(--primary-color);
  background: var(--hover-color);
}

.tab-button.active {
  color: var(--primary-color);
  border-bottom-color: var(--primary-color);
  background: var(--card-bg);
}

/* 频道状态标签 */
.channel-status {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  margin-top: 8px;
}

.channel-status.pending {
  background: #FFF8E1;
  color: #FF9800;
}

/* 切换到频道广场按钮 */
.switch-to-discover-btn {
  margin-top: 20px;
  padding: 10px 20px;
  border: 1px solid var(--primary-color);
  border-radius: 8px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
}

.switch-to-discover-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.channel-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
}

.create-channel-btn {
  background: var(--primary-color);
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.create-channel-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.channel-list-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  color: var(--text-secondary);
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  color: var(--primary-color);
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty h4 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.empty p {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.channels-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.channel-card {
  background: var(--card-bg);
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: all 0.3s ease;
  border: 1px solid var(--border-color);
}

.channel-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  border-color: var(--primary-color);
}

.channel-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.channel-avatar {
  width: 56px;
  height: 56px;
  border-radius: 28px;
  object-fit: cover;
  border: 2px solid var(--border-color);
  transition: all 0.3s ease;
}

.channel-card:hover .channel-avatar {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 4px rgba(0, 123, 255, 0.1);
}

.subscribe-btn {
  padding: 8px 16px;
  border: 1px solid var(--primary-color);
  border-radius: 20px;
  background: var(--secondary-color);
  color: var(--primary-color);
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.subscribe-btn.subscribed:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.channel-card-body {
  margin-bottom: 16px;
}

.channel-card-body h4 {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
  line-height: 1.3;
}

.channel-card-body p {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.channel-card-footer {
  font-size: 12px;
  color: var(--text-tertiary);
  border-top: 1px solid var(--border-color);
  padding-top: 12px;
}

.channel-detail-modal {
  max-width: 800px;
  max-height: 90vh;
}

.channel-header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.channel-header-avatar {
  width: 40px;
  height: 40px;
  border-radius: 20px;
  object-fit: cover;
  border: 2px solid var(--border-color);
}

.channel-detail-info {
  margin-bottom: 24px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
}

.channel-detail-text {
  flex: 1;
}

.channel-detail-description {
  color: var(--text-secondary);
  margin-bottom: 16px;
  line-height: 1.5;
  font-size: 14px;
}

.channel-detail-meta {
  font-size: 12px;
  color: var(--text-tertiary);
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.channel-messages {
  margin-bottom: 24px;
}

.channel-messages h5 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.channel-messages h5::before {
  content: '';
  width: 4px;
  height: 16px;
  background: var(--primary-color);
  border-radius: 2px;
}

.empty-messages {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  color: var(--text-secondary);
  text-align: center;
  background: var(--hover-color);
  border-radius: 8px;
  border: 2px dashed var(--border-color);
}

.empty-messages i {
  font-size: 32px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.empty-messages p {
  margin: 0;
  font-size: 14px;
}

.message-list {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 20px;
  background: var(--card-bg);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-item {
  display: flex;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
  transition: all 0.3s ease;
}

.message-item:hover {
  background: var(--hover-color);
  padding-left: 12px;
  border-radius: 8px;
}

.message-item:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 20px;
  object-fit: cover;
  margin-right: 16px;
  border: 2px solid var(--border-color);
  flex-shrink: 0;
}

.message-content {
  flex: 1;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.message-sender {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}

.message-time {
  font-size: 12px;
  color: var(--text-tertiary);
}

.message-text {
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.message-input-area {
  display: flex;
  gap: 12px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--border-color);
}

.message-textarea {
  flex: 1;
  padding: 12px;
  border: 2px solid var(--border-color);
  border-radius: 8px;
  resize: vertical;
  min-height: 80px;
  font-family: inherit;
  font-size: 14px;
  transition: all 0.3s ease;
  background: var(--input-bg);
  color: var(--text-color);
}

.message-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
}

.send-btn {
  align-self: flex-end;
  padding: 10px 24px;
  border: none;
  border-radius: 8px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.send-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.send-btn:disabled {
  background: var(--border-color);
  color: var(--text-secondary);
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

/* 滚动条样式 */
.channel-list-content::-webkit-scrollbar,
.message-list::-webkit-scrollbar {
  width: 6px;
}

.channel-list-content::-webkit-scrollbar-track,
.message-list::-webkit-scrollbar-track {
  background: var(--hover-color);
  border-radius: 3px;
}

.channel-list-content::-webkit-scrollbar-thumb,
.message-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.channel-list-content::-webkit-scrollbar-thumb:hover,
.message-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .channels-grid {
    grid-template-columns: 1fr;
  }
  
  .channel-detail-modal {
    width: 95%;
    max-height: 95vh;
  }
  
  .modal-content {
    width: 95%;
  }
}
</style>

<style>
.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.q-btn {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  min-width: 80px;
  border: 1px solid var(--border-color);
}

.q-btn--default {
  background: var(--right-content-bg);
  color: var(--text-color);
}

.q-btn--default:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.q-btn--primary {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.q-btn--primary:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.q-btn--primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>