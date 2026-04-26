<template>
  <div class="statistics-page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#409EFF"><User /></el-icon>
            <div class="stat-info">
              <div class="stat-value">{{ stats.totalUsers || 0 }}</div>
              <div class="stat-label">总用户数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#67C23A"><Connection /></el-icon>
            <div class="stat-info">
              <div class="stat-value">{{ stats.activeUsers || 0 }}</div>
              <div class="stat-label">活跃用户</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#E6A23C"><UserFilled /></el-icon>
            <div class="stat-info">
              <div class="stat-value">{{ stats.totalGroups || 0 }}</div>
              <div class="stat-label">群组总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon" color="#F56C6C"><ChatLineRound /></el-icon>
            <div class="stat-info">
              <div class="stat-value">{{ stats.messagesToday || 0 }}</div>
              <div class="stat-label">今日消息</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 增长率卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :xs="24" :sm="8">
        <el-card shadow="never">
          <div class="growth-item">
            <span class="growth-label">用户增长率</span>
            <span class="growth-value" :class="growthClass(stats.growthRate?.users)">
              {{ formatGrowth(stats.growthRate?.users) }}
            </span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="never">
          <div class="growth-item">
            <span class="growth-label">群组增长率</span>
            <span class="growth-value" :class="growthClass(stats.growthRate?.groups)">
              {{ formatGrowth(stats.growthRate?.groups) }}
            </span>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="8">
        <el-card shadow="never">
          <div class="growth-item">
            <span class="growth-label">消息增长率</span>
            <span class="growth-value" :class="growthClass(stats.growthRate?.messages)">
              {{ formatGrowth(stats.growthRate?.messages) }}
            </span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区域 -->
    <el-row :gutter="16" class="chart-row">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>用户增长趋势</span>
          </template>
          <div v-loading="chartLoading" class="chart-container">
            <div v-if="userTrendData.length > 0" class="bar-chart">
              <div
                v-for="(item, index) in userTrendData"
                :key="index"
                class="bar-item"
              >
                <div class="bar" :style="{ height: getBarHeight(item.value, userTrendMax) + '%' }">
                  <span class="bar-value">{{ item.value }}</span>
                </div>
                <span class="bar-label">{{ item.label }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无数据" :image-size="80" />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>消息发送趋势</span>
          </template>
          <div v-loading="chartLoading" class="chart-container">
            <div v-if="messageTrendData.length > 0" class="bar-chart">
              <div
                v-for="(item, index) in messageTrendData"
                :key="index"
                class="bar-item"
              >
                <div class="bar" :style="{ height: getBarHeight(item.value, messageTrendMax) + '%' }">
                  <span class="bar-value">{{ item.value }}</span>
                </div>
                <span class="bar-label">{{ item.label }}</span>
              </div>
            </div>
            <el-empty v-else description="暂无数据" :image-size="80" />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="chart-row">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>活动数据统计</span>
          </template>
          <div v-loading="chartLoading" class="activity-list">
            <div v-if="activityData.length > 0" class="activity-item" v-for="(item, index) in activityData" :key="index">
              <div class="activity-label">{{ item.label }}</div>
              <div class="activity-bar-wrapper">
                <div class="activity-bar" :style="{ width: getActivityPercent(item.value, activityMax) + '%' }">
                  <span class="activity-value">{{ item.value }}</span>
                </div>
              </div>
            </div>
            <el-empty v-else description="暂无数据" :image-size="80" />
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <span>系统概览</span>
          </template>
          <div v-loading="chartLoading" class="overview-grid">
            <div class="overview-item">
              <span class="overview-label">在线频道</span>
              <span class="overview-value">{{ stats.totalChannels || 0 }}</span>
            </div>
            <div class="overview-item">
              <span class="overview-label">活跃群组</span>
              <span class="overview-value">{{ stats.totalGroups || 0 }}</span>
            </div>
            <div class="overview-item">
              <span class="overview-label">活跃用户占比</span>
              <span class="overview-value">{{ calcActiveRate() }}%</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { User, Connection, UserFilled, ChatLineRound } from '@element-plus/icons-vue'
import { getStatistics } from '@/api/statistics'
import type { StatisticsData } from '@/types'

const stats = ref<StatisticsData>({
  totalUsers: 0,
  activeUsers: 0,
  totalGroups: 0,
  totalChannels: 0,
  messagesToday: 0,
  growthRate: { users: 0, groups: 0, messages: 0 },
})

const chartLoading = ref(false)

// 模拟趋势数据（实际应从 API 获取）
const userTrendData = ref<{ label: string; value: number }[]>([])
const messageTrendData = ref<{ label: string; value: number }[]>([])
const activityData = ref<{ label: string; value: number }[]>([])

const userTrendMax = computed(() => Math.max(...userTrendData.value.map((d) => d.value), 1))
const messageTrendMax = computed(() => Math.max(...messageTrendData.value.map((d) => d.value), 1))
const activityMax = computed(() => Math.max(...activityData.value.map((d) => d.value), 1))

// 格式化工具函数
const formatGrowth = (value?: number): string => {
  if (value === undefined || value === null) return '0%'
  return `${value > 0 ? '+' : ''}${value}%`
}

const growthClass = (value?: number): string => {
  if (value === undefined || value === null) return 'growth-neutral'
  if (value > 0) return 'growth-positive'
  if (value < 0) return 'growth-negative'
  return 'growth-neutral'
}

const calcActiveRate = (): string => {
  if (!stats.value.totalUsers) return '0'
  return ((stats.value.activeUsers / stats.value.totalUsers) * 100).toFixed(1)
}

const getBarHeight = (value: number, max: number): number => {
  return Math.max((value / max) * 100, 5)
}

const getActivityPercent = (value: number, max: number): number => {
  return Math.max((value / max) * 100, 5)
}

// 获取统计数据
const fetchStatistics = async () => {
  chartLoading.value = true
  try {
    const { data } = await getStatistics()
    if (data.data) {
      stats.value = data.data
    }
    // 模拟生成趋势数据（实际项目中应从后端获取）
    generateMockTrendData()
  } catch (error) {
    // 错误已在请求拦截器中处理
    generateMockTrendData()
  } finally {
    chartLoading.value = false
  }
}

const generateMockTrendData = () => {
  const days = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']
  userTrendData.value = days.map((day) => ({
    label: day,
    value: Math.floor(Math.random() * 50) + 10,
  }))
  messageTrendData.value = days.map((day) => ({
    label: day,
    value: Math.floor(Math.random() * 500) + 100,
  }))
  activityData.value = [
    { label: '登录次数', value: Math.floor(Math.random() * 1000) + 500 },
    { label: '消息发送', value: Math.floor(Math.random() * 5000) + 1000 },
    { label: '群组创建', value: Math.floor(Math.random() * 50) + 5 },
    { label: '频道订阅', value: Math.floor(Math.random() * 200) + 50 },
  ]
}

onMounted(fetchStatistics)
</script>

<style scoped>
.statistics-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.stats-row {
  margin-bottom: var(--space-2);
}

.stat-card {
  margin-bottom: var(--space-4);
  transition: all var(--duration-normal) var(--ease-out);
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: var(--shadow-card-hover) !important;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.stat-icon {
  font-size: 40px;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 30px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-top: var(--space-1);
  font-weight: 600;
}

.growth-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3) 0;
}

.growth-label {
  font-size: 14px;
  color: var(--color-text-secondary);
  font-weight: 600;
}

.growth-value {
  font-size: 22px;
  font-weight: 800;
}

.growth-positive {
  color: var(--color-success);
}

.growth-negative {
  color: var(--color-error);
}

.growth-neutral {
  color: var(--color-text-muted);
}

.chart-row {
  margin-bottom: var(--space-2);
}

.chart-container {
  min-height: 250px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding: var(--space-4) 0;
}

.bar-chart {
  display: flex;
  align-items: flex-end;
  justify-content: space-around;
  width: 100%;
  height: 200px;
  padding: 0 var(--space-4);
  gap: var(--space-2);
}

.bar-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
  height: 100%;
  justify-content: flex-end;
}

.bar {
  width: 100%;
  max-width: 40px;
  background: var(--gradient-primary);
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: var(--space-1);
  min-height: 20px;
  transition: height 0.3s ease;
}

.bar-value {
  font-size: 11px;
  color: #fff;
  white-space: nowrap;
  font-weight: 600;
}

.bar-label {
  margin-top: var(--space-2);
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
  font-weight: 500;
}

.activity-list {
  padding: var(--space-4);
}

.activity-item {
  margin-bottom: var(--space-4);
}

.activity-item:last-child {
  margin-bottom: 0;
}

.activity-label {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-2);
  font-weight: 600;
}

.activity-bar-wrapper {
  background: var(--color-bg-page);
  border-radius: var(--radius-sm);
  height: 32px;
  overflow: hidden;
}

.activity-bar {
  height: 100%;
  background: linear-gradient(90deg, #f59e0b 0%, #fbbf24 100%);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: var(--space-2);
  transition: width 0.3s ease;
  min-width: 40px;
}

.activity-value {
  font-size: 12px;
  color: #fff;
  font-weight: 700;
}

.overview-grid {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-height: 200px;
}

.overview-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-4);
  border-bottom: 2px solid var(--color-border-light);
}

.overview-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.overview-label {
  font-size: 14px;
  color: var(--color-text-secondary);
  font-weight: 600;
}

.overview-value {
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}
</style>
