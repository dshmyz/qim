<template>
  <el-card class="activity-card">
    <template #header>
      <div class="card-header">
        <div class="header-left">
          <el-icon class="header-icon" :size="18"><Clock /></el-icon>
          <span class="header-title">最近注册用户</span>
        </div>
        <el-button type="primary" text @click="emit('view-all')">
          查看全部
          <el-icon :size="14"><ArrowRight /></el-icon>
        </el-button>
      </div>
    </template>

    <el-table
      :data="registrations"
      v-loading="loading"
      :empty-text="loading ? '加载中...' : '暂无数据'"
      style="width: 100%"
      stripe
      class="activity-table"
    >
      <el-table-column label="用户" min-width="200">
        <template #default="{ row }">
          <div class="user-cell">
            <el-avatar :size="36" :src="row.avatar" class="user-avatar">
              {{ getAvatarText(row.username) }}
            </el-avatar>
            <div class="user-info">
              <span class="username">{{ row.username }}</span>
              <span class="email">{{ row.email }}</span>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="注册时间" width="180" sortable>
        <template #default="{ row }">
          <span class="time-text">{{ formatTime(row.createdAt) }}</span>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup lang="ts">
import { Clock, ArrowRight } from '@element-plus/icons-vue'
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
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<style scoped>
.activity-card {
  margin-top: var(--space-4);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  color: var(--color-primary);
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  background: var(--gradient-primary);
  color: white;
  font-size: 14px;
  font-weight: 600;
  flex-shrink: 0;
}

.user-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.username {
  font-size: 14px;
  color: var(--color-text-primary);
  font-weight: 600;
}

.email {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.time-text {
  font-size: 13px;
  color: var(--color-text-secondary);
  font-family: var(--font-mono);
}
</style>
