<template>
  <div v-if="visible" class="read-users-modal" @click="handleClose">
    <div class="read-users-content" @click.stop>
      <div class="read-users-header">
        <h3>已读用户 ({{ readUsers.read_users?.length || 0 }}/{{ Math.max(0, (readUsers.total_members || 0) - 1) }})</h3>
        <button class="close-btn" @click.stop="handleClose">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="read-users-body">
        <div v-if="readUsers.read_users?.length === 0" class="empty-read">
          暂无已读用户
        </div>
        <div v-else class="read-users-list">
          <div v-for="user in readUsers.read_users" :key="user.id" class="read-user-item">
            <img :src="getReadUserAvatar(user)" :alt="user.name || user.username" class="read-user-avatar" />
            <div class="read-user-info">
              <span class="read-user-name">{{ user.name || user.username }}</span>
            </div>
            <i class="fas fa-check read-icon"></i>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'

interface ReadUser {
  id: string | number
  name?: string
  username?: string
  avatar?: string
}

interface ReadUsersData {
  read_users?: ReadUser[]
  total_members?: number
}

interface Props {
  visible: boolean
  readUsers: ReadUsersData
  serverUrl: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'close'): void }>()

const handleClose = () => {
  emit('close')
}

const getReadUserAvatar = (user: ReadUser): string => {
  if (user.avatar && isAbsoluteUrl(user.avatar)) return user.avatar
  if (user.avatar) return props.serverUrl + user.avatar
  return generateAvatar(user.name || user.username || '用户')
}
</script>

<style scoped>
/* 已读用户列表弹窗样式 */
.read-users-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.read-users-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 360px;
  max-height: 480px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.read-users-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--panel-bg);
  border-bottom: 1px solid var(--border-color);
}

.read-users-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.read-users-header .close-btn {
  background: transparent;
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: var(--text-secondary);
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
  padding: 0;
}

.read-users-header .close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
  transform: scale(1.05);
}

.read-users-header .close-btn i {
  display: block !important;
  font-size: 16px !important;
  line-height: 1 !important;
}

.read-users-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.empty-read {
  text-align: center;
  color: var(--text-secondary);
  padding: 24px;
  font-size: 14px;
}

.read-users-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.read-user-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: var(--list-bg);
  border-radius: 8px;
  transition: all 0.2s;
}

.read-user-item:hover {
  background: var(--hover-color);
}

.read-user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  margin-right: 12px;
}

.read-user-info {
  flex: 1;
}

.read-user-name {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.read-icon {
  color: #4caf50;
  font-size: 14px;
}
</style>
