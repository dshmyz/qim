<template>
  <el-card shadow="never" class="distribution-chart">
    <template #header>
      <div class="chart-header">
        <span>版本分布</span>
        <el-button size="small" text @click="$emit('refresh')">
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>
    </template>
    <div v-loading="loading" class="chart-container">
      <PieChart
        v-if="chartData.length > 0"
        :data="chartData"
        height="280px"
      />
      <el-empty v-else description="暂无版本分布数据" :image-size="80" />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import PieChart from '@/components/charts/PieChart.vue'
import type { VersionDistribution } from '@/types/client'

interface Props {
  distribution: VersionDistribution[]
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
})

defineEmits<{
  'refresh': []
}>()

const chartData = computed(() =>
  props.distribution.map(item => ({
    name: item.version,
    value: item.count,
  }))
)
</script>

<style scoped>
.distribution-chart {
  height: 100%;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-container {
  min-height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
