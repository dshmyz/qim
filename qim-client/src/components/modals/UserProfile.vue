<template>
  <div v-if="visible" class="user-profile-modal" @click="close">
    <div class="user-profile-content" @click.stop>
      <div class="user-profile-header">
        <h3>用户资料</h3>
        <button class="close-btn" @click="close">×</button>
      </div>
      <div class="user-profile-body">
        <div class="profile-avatar">
          <Avatar
            :src="user.avatar"
            :name="user.name"
            :server-url="serverUrl"
            :alt="user.name"
            size="xl"
          />
        </div>
        <div class="profile-info">
          <div class="info-item">
            <label>姓名</label>
            <span class="info-value">{{ user.name }}</span>
          </div>
          <div class="info-item">
            <label>账号</label>
            <span class="info-value">{{ user.username || '无' }}</span>
          </div>
          <div class="info-item">
            <label>邮箱</label>
            <span class="info-value">{{ user.email || '无' }}</span>
          </div>
          <div class="info-item">
            <label>手机</label>
            <span class="info-value">{{ user.mobile || '无' }}</span>
          </div>
          <div class="info-item">
            <label>部门</label>
            <span class="info-value">{{ user.department || '无' }}</span>
          </div>
          <div class="info-item">
            <label>IP</label>
            <span class="info-value">{{ user.ip || '无' }}</span>
          </div>
        </div>
      </div>
      <div class="user-profile-footer">
        <button class="action-btn primary" @click="handleSendPrivateMessage">
          <i class="fas fa-comment"></i>
          <span>{{ isBot ? '开始对话' : '发起私聊' }}</span>
        </button>
        <button class="action-btn" @click="close">
          <span>关闭</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import Avatar from '../shared/Avatar.vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'

interface User {
  id: string | number
  name: string
  username?: string
  email?: string
  mobile?: string
  department?: string
  ip?: string
  avatar?: string
  type?: string
}

interface Props {
  visible: boolean
  user: User
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  sendPrivateMessage: [user: User]
}>()

const close = () => {
  emit('close')
}

const isBot = computed(() => {
  const t = props.user.type
  return t === 'bot'
})

const handleSendPrivateMessage = () => {
  emit('sendPrivateMessage', props.user)
  emit('close')
}

const serverUrl = getStoredServerUrl()

// 头像 URL
const avatarUrl = computed(() => {
  if (!props.user.avatar) {
    return generateAvatar(props.user.name)
  }
  if (isAbsoluteUrl(props.user.avatar)) {
    return props.user.avatar
  }
  return `${serverUrl}${props.user.avatar}`
})
</script>

<style scoped>
.user-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.user-profile-content {
  background-color: #fff;
  border-radius: 12px;
  width: 420px;
  max-width: 90%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
}

.user-profile-header {
  padding: 20px 24px;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #fafafa;
}

.user-profile-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.user-profile-header .close-btn {
  background: transparent;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #333;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  opacity: 0.6;
  line-height: 1;
}

.user-profile-header .close-btn:hover {
  background: #f0f0f0;
  color: #333;
  opacity: 1;
}

.user-profile-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.profile-avatar {
  display: flex;
  justify-content: center;
  margin-bottom: 20px;
}

.profile-avatar img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid #e8e8e8;
}

.profile-info {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: #f8f9fa;
  border-radius: 8px;
  transition: all 0.2s;
}

.info-item:hover {
  background-color: #f0f0f0;
}

.info-item label {
  font-size: 14px;
  color: #666;
  font-weight: 500;
}

.info-item .info-value {
  font-size: 14px;
  color: #333;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

.user-profile-footer {
  padding: 16px 24px;
  border-top: 1px solid #e8e8e8;
  display: flex;
  justify-content: center;
  gap: 12px;
  background-color: #fafafa;
}

.action-btn {
  padding: 10px 24px;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  background: #fff;
  color: #333;
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn i {
  font-size: 14px;
}

.action-btn:hover {
  background: #f0f0f0;
  border-color: #d0d0d0;
}

.action-btn.primary {
  background: var(--primary-color, #3b82f6);
  border-color: var(--primary-color, #3b82f6);
  color: #fff;
}

.action-btn.primary:hover {
  background: var(--active-color, #2563eb);
  border-color: var(--active-color, #2563eb);
}

/* 高雅紫主题 */
[data-theme="elegant-purple"] .user-profile-header {
  background-color: var(--sidebar-bg, #ffffff);
  border-bottom: 1px solid var(--border-color, #e9d5ff);
}

[data-theme="elegant-purple"] .user-profile-header h3 {
  color: var(--text-color, #5b21b6);
}

[data-theme="elegant-purple"] .action-btn.primary {
  background: var(--primary-color, #7e22ce);
  border-color: var(--primary-color, #7e22ce);
}

[data-theme="elegant-purple"] .action-btn.primary:hover {
  background: var(--active-color, #6b21a8);
  border-color: var(--active-color, #6b21a8);
}

/* 中国红主题 */
[data-theme="chinesered"] .user-profile-header {
  background-color: var(--sidebar-bg, #ffffff);
  border-bottom: 1px solid var(--border-color, #fecaca);
}

[data-theme="chinesered"] .user-profile-header h3 {
  color: var(--text-color, #5c1a1a);
}

[data-theme="chinesered"] .action-btn.primary {
  background: var(--primary-color, #c41e3a);
  border-color: var(--primary-color, #c41e3a);
  color: #fff;
}

[data-theme="chinesered"] .action-btn.primary:hover {
  background: var(--active-color, #a31d32);
  border-color: var(--active-color, #a31d32);
}
</style>
