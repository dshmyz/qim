<template>
  <el-row :gutter="20">
    <el-col :xs="24" :md="12" v-for="chart in charts" :key="chart.key">
      <el-card class="chart-card">
        <template #header>
          <div class="card-header">
            <span class="header-title">{{ chart.title }}</span>
            <el-tag size="small" type="info">即将上线</el-tag>
          </div>
        </template>
        <div class="chart-container">
          <el-empty :description="chart.placeholder" :image-size="80" />
        </div>
      </el-card>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  showTrend?: boolean
  showDistribution?: boolean
}>()

interface ChartConfig {
  key: string
  title: string
  placeholder: string
}

const charts = computed<ChartConfig[]>(() => {
  const result: ChartConfig[] = []
  if (props.showTrend !== false) {
    result.push({
      key: 'userTrend',
      title: '用户增长趋势',
      placeholder: '图表组件待集成',
    })
  }
  if (props.showDistribution !== false) {
    result.push({
      key: 'activityDistribution',
      title: '活跃用户分布',
      placeholder: '图表组件待集成',
    })
  }
  return result
})
</script>

<style scoped>
.chart-card {
  margin-bottom: var(--space-4);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.chart-container {
  height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg-page);
  border-radius: var(--radius-lg);
}

:deep(.el-empty__description) {
  color: var(--color-text-muted);
  font-size: 13px;
}
</style>
