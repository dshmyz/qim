<template>
  <div v-if="visible" class="read-users-modal" @click="$emit('close')">
    <div class="read-users-content" @click.stop>
      <div class="read-users-header">
        <h3>已读用户 ({{ readUsers.read_users?.length || 0 }}/{{ Math.max(0, (readUsers.total_members || 0) - 1) }})</h3>
        <button class="close-btn" @click.stop="$emit('close')">&times;</button>
      </div>
      <div class="read-users-body">
        <div v-if="readUsers.read_users?.length === 0" class="empty-read">
          暂无已读用户
        </div>
        <div v-else class="read-users-list">
          <div v-for="user in readUsers.read_users" :key="user.id" class="read-user-item">
            <img :src="(user.avatar && user.avatar.startsWith('http')) ? user.avatar : (user.avatar ? serverUrl + user.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=' + user.id)" :alt="user.name || user.username" class="read-user-avatar" />
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

defineProps<Props>()
defineEmits<{ (e: 'close'): void }>()
</script>
