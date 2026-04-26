<template>
  <el-row :gutter="20">
    <el-col :xs="24" :sm="12" :md="6" v-for="card in statCards" :key="card.key">
      <el-card class="stat-card" shadow="hover">
        <div class="stat-content">
          <div class="stat-icon" :style="{ background: card.gradient }">
            <el-icon :size="26">
              <component :is="card.icon" />
            </el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-value">
              <el-statistic :value="loading ? 0 : card.value">
                <template #suffix>
                  <el-icon v-if="loading" class="is-loading"><Loading /></el-icon>
                </template>
              </el-statistic>
            </div>
            <div class="stat-label">{{ card.label }}</div>
          </div>
        </div>
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { User, Connection, UserFilled, ChatDotRound, Loading } from '@element-plus/icons-vue'
import type { DashboardData } from '@/types'

interface StatCardConfig {
  key: string
  label: string
  icon: typeof User
  gradient: string
  value: number
}

const props = defineProps<{
  stats: DashboardData
  loading: boolean
}>()

const statCards = computed<StatCardConfig[]>(() => [
  {
    key: 'totalUsers',
    label: '总用户数',
    icon: User,
    gradient: 'linear-gradient(135deg, #0ea5e9, #6366f1)',
    value: props.stats.totalUsers,
  },
  {
    key: 'onlineUsers',
    label: '在线用户',
    icon: Connection,
    gradient: 'linear-gradient(135deg, #10b981, #0ea5e9)',
    value: props.stats.onlineUsers,
  },
  {
    key: 'totalGroups',
    label: '总群组数',
    icon: UserFilled,
    gradient: 'linear-gradient(135deg, #6366f1, #8b5cf6)',
    value: props.stats.totalGroups,
  },
  {
    key: 'totalMessages',
    label: '总消息数',
    icon: ChatDotRound,
    gradient: 'linear-gradient(135deg, #f59e0b, #ef4444)',
    value: props.stats.totalMessages,
  },
])
</script>

<style scoped>
.stat-card {
  margin-bottom: var(--space-4);
  border-radius: var(--radius-xl) !important;
  overflow: hidden;
  transition: all var(--duration-normal) var(--ease-out);
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-lg) !important;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
  box-shadow: 0 4px 16px -2px rgba(0, 0, 0, 0.12);
}

.stat-info {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
  color: var(--color-text-primary);
  line-height: 1.2;
  letter-spacing: -0.02em;
}

:deep(.el-statistic__number) {
  font-size: 28px !important;
  font-weight: 800 !important;
  color: var(--color-text-primary) !important;
  letter-spacing: -0.02em !important;
}

.stat-label {
  font-size: 13px;
  color: var(--color-text-muted);
  margin-top: var(--space-1);
  font-weight: 600;
}
</style>
