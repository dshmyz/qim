<template>
  <el-card class="metric-card">
    <div class="metric-header">
      <span class="metric-title">{{ title }}</span>
      <el-icon :size="20" :color="iconColor">
        <component :is="icon" />
      </el-icon>
    </div>
    <div class="metric-value">{{ value }}%</div>
    <el-progress
      :percentage="value"
      :color="progressColor"
      :stroke-width="8"
    />
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Cpu, Coin, Folder, Connection } from '@element-plus/icons-vue'

interface Props {
  title: string
  value: number
  type: 'cpu' | 'memory' | 'disk' | 'network'
}

const props = defineProps<Props>()

const iconMap = {
  cpu: Cpu,
  memory: Coin,
  disk: Folder,
  network: Connection
}

const icon = computed(() => iconMap[props.type])

const iconColor = computed(() => {
  if (props.value > 80) return '#f56c6c'
  if (props.value > 60) return '#e6a23c'
  return '#67c23a'
})

const progressColor = computed(() => {
  if (props.value > 80) return '#f56c6c'
  if (props.value > 60) return '#e6a23c'
  return '#67c23a'
})
</script>

<style scoped>
.metric-card {
  margin-bottom: 20px;
}

.metric-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}

.metric-title {
  font-size: 14px;
  color: #909399;
}

.metric-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
  margin: 10px 0;
}
</style>
