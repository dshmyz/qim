<template>
  <div class="server-monitor">
    <el-row :gutter="20">
      <el-col :span="6">
        <MetricCard
          title="CPU 使用率"
          :value="monitorStore.serverMetrics?.cpu || 0"
          type="cpu"
        />
      </el-col>
      
      <el-col :span="6">
        <MetricCard
          title="内存使用率"
          :value="monitorStore.serverMetrics?.memory || 0"
          type="memory"
        />
      </el-col>
      
      <el-col :span="6">
        <MetricCard
          title="磁盘使用率"
          :value="monitorStore.serverMetrics?.disk || 0"
          type="disk"
        />
      </el-col>
      
      <el-col :span="6">
        <MetricCard
          title="网络流量"
          :value="networkPercentage"
          type="network"
        />
      </el-col>
    </el-row>
    
    <el-card style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>性能趋势</span>
          <el-date-picker
            v-model="timeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            @change="loadHistoryData"
          />
        </div>
      </template>
      
      <LineChart
        v-if="historyData.length > 0"
        :x-axis-data="historyTimeLabels"
        :series-data="historySeriesData"
        height="400px"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import MetricCard from '@/components/monitor/MetricCard.vue'
import LineChart from '@/components/charts/LineChart.vue'

const monitorStore = useMonitorStore()
const timeRange = ref<Date[]>([])
const historyData = ref<any[]>([])
let refreshTimer: number | null = null

const networkPercentage = computed(() => {
  if (!monitorStore.serverMetrics?.network) return 0
  const { in: inBytes, out: outBytes } = monitorStore.serverMetrics.network
  const total = inBytes + outBytes
  const maxBandwidth = 100 * 1024 * 1024
  return Math.min(Math.round((total / maxBandwidth) * 100), 100)
})

const historyTimeLabels = computed(() => {
  return historyData.value.map(item => 
    new Date(item.timestamp).toLocaleTimeString('zh-CN')
  )
})

const historySeriesData = computed(() => {
  return [
    {
      name: 'CPU',
      data: historyData.value.map(item => item.cpu),
      color: '#409eff'
    },
    {
      name: '内存',
      data: historyData.value.map(item => item.memory),
      color: '#67c23a'
    },
    {
      name: '磁盘',
      data: historyData.value.map(item => item.disk),
      color: '#e6a23c'
    }
  ]
})

onMounted(() => {
  loadCurrentData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

async function loadCurrentData() {
  await monitorStore.loadServerMetrics()
}

function startAutoRefresh() {
  refreshTimer = window.setInterval(() => {
    loadCurrentData()
  }, 5000)
}

function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

async function loadHistoryData() {
  if (!timeRange.value || timeRange.value.length !== 2) return
  console.log('Load history data:', timeRange.value)
}
</script>

<style scoped>
.server-monitor {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>
