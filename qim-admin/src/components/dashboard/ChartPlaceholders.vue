<template>
  <div class="charts-grid">
    <div class="chart-card">
      <div class="chart-header">
        <h3 class="chart-title">用户增长趋势</h3>
        <span class="chart-subtitle">最近 7 天</span>
      </div>
      <div class="bar-chart">
        <div
          v-for="(item, index) in userTrendData"
          :key="index"
          class="bar-item"
        >
          <div class="bar-wrapper">
            <div
              class="bar"
              :style="{
                height: item.percent + '%',
                background: getBarGradient(index)
              }"
            >
              <span class="bar-value">{{ item.value }}</span>
            </div>
          </div>
          <span class="bar-label">{{ item.label }}</span>
        </div>
      </div>
    </div>

    <div class="chart-card">
      <div class="chart-header">
        <h3 class="chart-title">消息活跃度</h3>
        <span class="chart-subtitle">今日按时段</span>
      </div>
      <div class="activity-bars">
        <div
          v-for="(item, index) in activityData"
          :key="index"
          class="activity-item"
        >
          <span class="activity-label">{{ item.label }}</span>
          <div class="activity-bar-wrapper">
            <div
              class="activity-bar"
              :style="{ width: item.percent + '%' }"
            >
              <span class="activity-value">{{ item.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getDashboardTrend } from '@/api/statistics'

const userTrendData = ref<{ label: string; value: number; percent: number }[]>([])
const activityData = ref<{ label: string; value: number; percent: number }[]>([])

const fetchTrendData = async () => {
  try {
    const { data } = await getDashboardTrend()
    if (data.data) {
      userTrendData.value = data.data.userTrend || []
      activityData.value = data.data.activityData || []
    }
  } catch {
    // 静默失败，图表显示为空
  }
}

const getBarGradient = (index: number): string => {
  const gradients = [
    'linear-gradient(180deg, #0ea5e9 0%, #0284c7 100%)',
    'linear-gradient(180deg, #3b82f6 0%, #2563eb 100%)',
    'linear-gradient(180deg, #6366f1 0%, #4f46e5 100%)',
    'linear-gradient(180deg, #8b5cf6 0%, #7c3aed 100%)',
    'linear-gradient(180deg, #0ea5e9 0%, #0284c7 100%)',
    'linear-gradient(180deg, #3b82f6 0%, #2563eb 100%)',
    'linear-gradient(180deg, #6366f1 0%, #4f46e5 100%)',
  ]
  return gradients[index % gradients.length]
}

onMounted(fetchTrendData)
</script>

<style scoped>
.charts-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-4);
}

@media (max-width: 1024px) {
  .charts-grid {
    grid-template-columns: 1fr;
  }
}

.chart-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: var(--space-5);
  box-shadow: var(--shadow-card);
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: var(--space-5);
}

.chart-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.chart-subtitle {
  font-size: 13px;
  color: var(--color-text-muted);
}

.bar-chart {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  height: 180px;
  gap: var(--space-3);
  padding: var(--space-4) 0;
}

.bar-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  height: 100%;
}

.bar-wrapper {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.bar {
  width: 100%;
  max-width: 40px;
  border-radius: var(--radius-md) var(--radius-md) 0 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: var(--space-2);
  min-height: 24px;
  transition: height var(--duration-normal) var(--ease-out);
}

.bar-value {
  font-size: 11px;
  font-weight: 600;
  color: white;
}

.bar-label {
  font-size: 12px;
  color: var(--color-text-muted);
  font-weight: 500;
}

.activity-bars {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-2) 0;
}

.activity-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.activity-label {
  width: 48px;
  font-size: 13px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.activity-bar-wrapper {
  flex: 1;
  height: 28px;
  background: var(--color-surface-hover);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.activity-bar {
  height: 100%;
  background: linear-gradient(90deg, #0ea5e9 0%, #6366f1 100%);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: var(--space-2);
  min-width: 60px;
  transition: width var(--duration-normal) var(--ease-out);
}

.activity-value {
  font-size: 12px;
  font-weight: 600;
  color: white;
}
</style>
