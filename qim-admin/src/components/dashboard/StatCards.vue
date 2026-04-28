<template>
  <div class="stat-cards">
    <div
      v-for="card in statCards"
      :key="card.key"
      class="stat-card"
      :style="{ background: card.gradient }"
    >
      <div class="card-content">
        <div class="card-header">
          <span class="card-label">{{ card.label }}</span>
          <div class="card-icon">
            <el-icon :size="20">
              <component :is="card.icon" />
            </el-icon>
          </div>
        </div>
        <div class="card-value">
          <span v-if="loading" class="loading-skeleton"></span>
          <span v-else class="value-number">{{ formatNumber(card.value) }}</span>
        </div>
        <div class="card-footer">
          <span class="growth-badge" :class="card.growthClass">
            <el-icon :size="12">
              <component :is="card.growthIcon" />
            </el-icon>
            {{ card.growthText }}
          </span>
          <span class="growth-period">vs 上周</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { User, Connection, UserFilled, ChatDotRound, Top, Bottom, Minus } from '@element-plus/icons-vue'
import type { DashboardData } from '@/types'

interface StatCardConfig {
  key: string
  label: string
  icon: typeof User
  gradient: string
  value: number
  growth: number
  growthClass: string
  growthIcon: typeof Top
  growthText: string
}

const props = defineProps<{
  stats: DashboardData
  loading: boolean
}>()

const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(1) + 'M'
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(1) + 'K'
  }
  return num.toString()
}

const getGrowthInfo = (growth: number) => {
  if (growth > 0) {
    return {
      growthClass: 'growth-positive',
      growthIcon: Top,
      growthText: `+${growth}%`
    }
  } else if (growth < 0) {
    return {
      growthClass: 'growth-negative',
      growthIcon: Bottom,
      growthText: `${growth}%`
    }
  }
  return {
    growthClass: 'growth-neutral',
    growthIcon: Minus,
    growthText: '0%'
  }
}

const statCards = computed<StatCardConfig[]>(() => {
  const mockGrowth = [12, 8, -3, 25]
  
  return [
    {
      key: 'totalUsers',
      label: '总用户数',
      icon: User,
      gradient: 'linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%)',
      value: props.stats.totalUsers,
      growth: mockGrowth[0],
      ...getGrowthInfo(mockGrowth[0])
    },
    {
      key: 'onlineUsers',
      label: '在线用户',
      icon: Connection,
      gradient: 'linear-gradient(135deg, #10b981 0%, #0ea5e9 100%)',
      value: props.stats.onlineUsers,
      growth: mockGrowth[1],
      ...getGrowthInfo(mockGrowth[1])
    },
    {
      key: 'totalGroups',
      label: '群组总数',
      icon: UserFilled,
      gradient: 'linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%)',
      value: props.stats.totalGroups,
      growth: mockGrowth[2],
      ...getGrowthInfo(mockGrowth[2])
    },
    {
      key: 'totalMessages',
      label: '消息总数',
      icon: ChatDotRound,
      gradient: 'linear-gradient(135deg, #f59e0b 0%, #ef4444 100%)',
      value: props.stats.totalMessages,
      growth: mockGrowth[3],
      ...getGrowthInfo(mockGrowth[3])
    },
  ]
})
</script>

<style scoped>
.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

@media (max-width: 1200px) {
  .stat-cards {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .stat-cards {
    grid-template-columns: 1fr;
  }
}

.stat-card {
  border-radius: var(--radius-xl);
  padding: var(--space-5);
  color: white;
  transition: all var(--duration-normal) var(--ease-out);
  cursor: default;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 12px 24px -8px rgba(0, 0, 0, 0.15);
}

.card-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-label {
  font-size: 14px;
  font-weight: 600;
  opacity: 0.9;
}

.card-icon {
  width: 36px;
  height: 36px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-value {
  min-height: 36px;
}

.value-number {
  font-size: 32px;
  font-weight: 800;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.loading-skeleton {
  display: inline-block;
  width: 80px;
  height: 32px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.8; }
}

.card-footer {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.growth-badge {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.2);
}

.growth-positive {
  color: #d1fae5;
}

.growth-negative {
  color: #fecaca;
}

.growth-neutral {
  color: rgba(255, 255, 255, 0.8);
}

.growth-period {
  font-size: 12px;
  opacity: 0.7;
}
</style>
