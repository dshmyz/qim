<template>
  <div class="activity-card">
    <div class="card-header">
      <div class="header-left">
        <h3 class="card-title">最近注册用户</h3>
        <span class="card-count">{{ registrations.length }} 位新用户</span>
      </div>
      <el-button type="primary" text size="small" @click="emit('view-all')">
        查看全部
        <el-icon :size="14" style="margin-left: 4px"><ArrowRight /></el-icon>
      </el-button>
    </div>

    <div v-loading="loading" class="user-list">
      <div
        v-for="user in registrations"
        :key="user.id"
        class="user-item"
      >
        <el-avatar :size="40" :src="user.avatar" class="user-avatar">
          {{ getAvatarText(user.username) }}
        </el-avatar>
        <div class="user-info">
          <span class="username">{{ user.username }}</span>
          <span class="user-email">{{ user.email }}</span>
        </div>
        <span class="user-time">{{ formatTime(user.createdAt) }}</span>
      </div>
      <el-empty v-if="!loading && registrations.length === 0" description="暂无新用户" :image-size="48" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArrowRight } from '@element-plus/icons-vue'
import type { RecentRegistration } from '@/types'

defineProps<{
  registrations: RecentRegistration[]
  loading: boolean
}>()

const emit = defineEmits<{
  'view-all': []
}>()

const getAvatarText = (username: string): string => {
  if (!username) return '?'
  return username.charAt(0).toUpperCase()
}

const formatTime = (timeStr: string): string => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins} 分钟前`
  if (diffHours < 24) return `${diffHours} 小时前`
  if (diffDays < 7) return `${diffDays} 天前`

  return date.toLocaleDateString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
  })
}
</script>

<style scoped>
.activity-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: var(--space-5);
  box-shadow: var(--shadow-card);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.card-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.card-count {
  font-size: 13px;
  color: var(--color-text-muted);
}

.user-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-height: 200px;
}

.user-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  transition: background-color var(--duration-fast) var(--ease-out);
}

.user-item:hover {
  background-color: var(--color-surface-hover);
}

.user-avatar {
  background: var(--gradient-primary);
  color: white;
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.user-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.user-email {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.user-time {
  font-size: 12px;
  color: var(--color-text-secondary);
  white-space: nowrap;
}
</style>
