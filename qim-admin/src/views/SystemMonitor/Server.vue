<template>
  <div class="server-monitor">
    <!-- 系统指标 -->
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

    <!-- 系统信息 -->
    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="8">
        <el-card>
          <div class="info-card">
            <div class="info-icon">
              <el-icon :size="32" color="#409eff"><Timer /></el-icon>
            </div>
            <div class="info-content">
              <div class="info-label">运行时长</div>
              <div class="info-value">{{ formatUptime(monitorStore.serverMetrics?.uptime || 0) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="8">
        <el-card>
          <div class="info-card">
            <div class="info-icon">
              <el-icon :size="32" color="#67c23a"><Cpu /></el-icon>
            </div>
            <div class="info-content">
              <div class="info-label">Goroutines</div>
              <div class="info-value">{{ monitorStore.serverMetrics?.goRoutines || 0 }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="8">
        <el-card>
          <div class="info-card">
            <div class="info-icon">
              <el-icon :size="32" color="#e6a23c"><Connection /></el-icon>
            </div>
            <div class="info-content">
              <div class="info-label">网络 I/O</div>
              <div class="info-value">
                <span class="io-in">↓ {{ formatBytes(monitorStore.serverMetrics?.network?.in || 0) }}</span>
                <span class="io-out">↑ {{ formatBytes(monitorStore.serverMetrics?.network?.out || 0) }}</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 服务状态 -->
    <el-card style="margin-top: 20px;">
      <template #header>
        <div class="card-header">
          <span>服务状态</span>
          <el-button type="primary" size="small" @click="loadServiceStatus">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </template>
      
      <el-table :data="monitorStore.serviceStatus" style="width: 100%">
        <el-table-column prop="name" label="服务名称" width="180" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="消息" />
        <el-table-column label="延迟" width="120">
          <template #default="{ row }">
            {{ row.latency }} ms
          </template>
        </el-table-column>
        <el-table-column prop="lastCheck" label="最后检查" width="180" />
      </el-table>
    </el-card>
    
    <!-- 性能趋势 -->
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
      <el-empty v-else description="暂无历史数据" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import MetricCard from '@/components/monitor/MetricCard.vue'
import LineChart from '@/components/charts/LineChart.vue'
import { Timer, Cpu, Connection, Refresh } from '@element-plus/icons-vue'

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

// 格式化运行时长
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  if (days > 0) {
    return `${days}天 ${hours}小时 ${minutes}分钟`
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟 ${secs}秒`
  }
  return `${secs}秒`
}

// 格式化字节数
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

// 获取状态类型
function getStatusType(status: string): 'success' | 'warning' | 'danger' | 'info' {
  switch (status) {
    case 'healthy':
      return 'success'
    case 'warning':
      return 'warning'
    case 'unhealthy':
      return 'danger'
    default:
      return 'info'
  }
}

// 获取状态文本
function getStatusText(status: string): string {
  switch (status) {
    case 'healthy':
      return '健康'
    case 'warning':
      return '警告'
    case 'unhealthy':
      return '不健康'
    default:
      return '未知'
  }
}

onMounted(() => {
  loadCurrentData()
  loadServiceStatus()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

async function loadCurrentData() {
  await monitorStore.loadServerMetrics()
}

async function loadServiceStatus() {
  await monitorStore.loadServiceStatus()
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

.info-card {
  display: flex;
  align-items: center;
  gap: 16px;
}

.info-icon {
  flex-shrink: 0;
}

.info-content {
  flex: 1;
}

.info-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 4px;
}

.info-value {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
}

.io-in {
  color: #67c23a;
  margin-right: 12px;
}

.io-out {
  color: #409eff;
}
</style>
