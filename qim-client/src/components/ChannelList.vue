<template>
  <div class="channel-list">
    <div class="channel-header">
      <h3>频道</h3>
      <button v-if="hasAdminPermission" class="create-channel-btn" @click="showCreateChannelModal = true">
        <i class="fas fa-plus"></i> 创建频道
      </button>
    </div>
    
    <div class="channel-list-content">
      <div v-if="loading" class="loading">
        <div class="loading-spinner"></div>
        <span>加载中...</span>
      </div>
      <div v-else-if="channels.length === 0" class="empty">
        <i class="fas fa-bullhorn empty-icon"></i>
        <h4>暂无频道</h4>
        <p>成为第一个创建频道的人吧！</p>
      </div>
      <div v-else class="channels-grid">
        <div 
          v-for="channel in channels" 
          :key="channel.id" 
          class="channel-card"
          @click="selectChannel(channel)"
        >
          <div class="channel-card-header">
            <img :src="channel.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=channel'" :alt="channel.name" class="channel-avatar" />
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
          </div>
          <div class="channel-card-footer">
            <span class="channel-creator">创建者: {{ channel.creator?.name || '未知' }}</span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 创建频道弹窗 -->
    <div v-if="showCreateChannelModal" class="modal-overlay" @click="showCreateChannelModal = false">
      <div class="modal-content create-channel-modal" @click.stop>
        <div class="modal-header">
          <h4>创建频道</h4>
          <button class="close-btn" @click="showCreateChannelModal = false">×</button>
        </div>
        <div class="modal-body">
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
        </div>
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="showCreateChannelModal = false">取消</button>
          <button class="btn btn-primary" @click="createChannel" :disabled="!createChannelForm.name">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Channel, User } from '../types'
import { API_BASE_URL } from '../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const props = defineProps<{
  currentUser: User
}>()

const emit = defineEmits<{
  (e: 'select-channel', channel: Channel): void
}>()

const channels = ref<Channel[]>([])
const loading = ref(false)
const showCreateChannelModal = ref(false)

const createChannelForm = ref({
  name: '',
  description: '',
  avatar: ''
})

const hasAdminPermission = computed(() => {
  return props.currentUser.roles?.includes('system_admin')
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

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.modal-content {
  background: var(--card-bg);
  border-radius: 16px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.15);
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from { transform: translateY(-20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.create-channel-modal {
  max-width: 500px;
}

.channel-detail-modal {
  max-width: 800px;
  max-height: 90vh;
}

.modal-header {
  padding: 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--card-bg);
  border-radius: 16px 16px 0 0;
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

.modal-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.3s ease;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.close-btn:hover {
  background: var(--hover-color);
  color: var(--text-primary);
}

.modal-body {
  padding: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-weight: 500;
  color: var(--text-primary);
  font-size: 14px;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 12px;
  border: 2px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.3s ease;
  background: var(--input-bg);
  color: var(--text-color);
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
  font-family: inherit;
}

.modal-footer {
  padding: 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background: var(--card-bg);
  border-radius: 0 0 16px 16px;
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.btn-secondary {
  background: var(--hover-color);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--border-color);
  transform: translateY(-1px);
}

.btn-primary {
  background: var(--primary-color);
  color: white;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.btn-primary:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.btn:disabled {
  background: var(--border-color);
  color: var(--text-secondary);
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
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