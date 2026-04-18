<template>
  <div class="statistics-app">
    <div class="statistics-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-arrow-left"></i>
        </button>
        <h2>统计报表</h2>
      </div>
      <div class="statistics-period-selector">
        <button 
          v-for="period in periods" 
          :key="period.value"
          class="period-btn"
          :class="{ active: selectedPeriod === period.value }"
          @click="changeStatisticsPeriod(period.value)"
        >
          {{ period.label }}
        </button>
      </div>
    </div>
    <div class="statistics-content">
      <div class="statistics-overview">
        <div class="stat-card">
          <div class="stat-icon messages-icon">
            <i class="fas fa-comment-dots"></i>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ statisticsData.totalMessages }}</div>
            <div class="stat-label">消息总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon files-icon">
            <i class="fas fa-file-alt"></i>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ statisticsData.totalFiles }}</div>
            <div class="stat-label">文件总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon notes-icon">
            <i class="fas fa-sticky-note"></i>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ statisticsData.totalNotes }}</div>
            <div class="stat-label">笔记总数</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon tasks-icon">
            <i class="fas fa-tasks"></i>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ statisticsData.totalTasks }}</div>
            <div class="stat-label">任务总数</div>
          </div>
        </div>
      </div>
      <div class="chart-container">
        <h3>消息趋势</h3>
        <div class="message-trend-chart">
          <div class="chart-axis-y">
            <div class="axis-label" v-for="i in 5" :key="i">{{ Math.round(statisticsData.maxMessages * (i / 5)) }}</div>
          </div>
          <div class="chart-bars">
            <div 
              v-for="(item, index) in statisticsData.messageTrend" 
              :key="index"
              class="chart-bar"
              :style="{ height: (item.count / statisticsData.maxMessages) * 100 + '%' }"
              :title="item.date + ': ' + item.count + ' 条消息'"
            >
              <div class="bar-label">{{ item.date }}</div>
            </div>
          </div>
        </div>
      </div>
      <div class="chart-container">
        <h3>文件类型分布</h3>
        <div class="file-distribution-chart">
          <div class="pie-chart">
            <svg viewBox="0 0 200 200">
              <circle 
                v-for="(item, index) in statisticsData.fileTypes" 
                :key="index"
                cx="100"
                cy="100"
                r="80"
                :fill="'none'"
                :stroke="getFileTypeColor(item.type)"
                :stroke-width="'30'"
                :stroke-dasharray="item.percentage * 502.4 + ' 502.4'"
                :stroke-dashoffset="getPieOffset(index, statisticsData.fileTypes)"
                :transform="'rotate(-90 100 100)'"
                class="pie-slice"
              />
            </svg>
            <div class="pie-center">
              <div class="pie-center-text">{{ statisticsData.totalFiles }}</div>
              <div class="pie-center-label">文件总数</div>
            </div>
          </div>
          <div class="file-types-legend">
            <div 
              v-for="item in statisticsData.fileTypes" 
              :key="item.type"
              class="legend-item"
            >
              <div class="legend-color" :style="{ backgroundColor: getFileTypeColor(item.type) }"></div>
              <div class="legend-label">{{ item.type }}</div>
              <div class="legend-value">{{ item.count }} ({{ item.percentage }}%)</div>
            </div>
          </div>
        </div>
      </div>
      <div class="chart-container">
        <h3>任务完成率</h3>
        <div class="completion-chart">
          <div class="completion-circle">
            <svg viewBox="0 0 200 200">
              <circle cx="100" cy="100" r="80" fill="none" stroke="#e0e0e0" stroke-width="15" />
              <circle 
                cx="100" 
                cy="100" 
                r="80" 
                fill="none" 
                stroke="var(--success-color)" 
                stroke-width="15" 
                :stroke-dasharray="statisticsData.taskCompletionRate * 502.4 + ' 502.4'"
                stroke-dashoffset="0"
                transform="rotate(-90 100 100)"
                class="completion-arc"
              />
              <text x="100" y="100" text-anchor="middle" dy=".3em" class="completion-value">{{ Math.round(statisticsData.taskCompletionRate * 100) }}%</text>
            </svg>
          </div>
          <div class="completion-details">
            <div class="completion-item">
              <div class="completion-label">已完成</div>
              <div class="completion-count">{{ statisticsData.completedTasks }}</div>
            </div>
            <div class="completion-item">
              <div class="completion-label">未完成</div>
              <div class="completion-count">{{ statisticsData.pendingTasks }}</div>
            </div>
            <div class="completion-item">
              <div class="completion-label">总计</div>
              <div class="completion-count">{{ statisticsData.totalTasks }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { API_BASE_URL } from '../../config'

// 服务器URL
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 统计报表相关状态
const selectedPeriod = ref('week')
const statisticsData = ref({
  totalMessages: 0,
  totalFiles: 0,
  totalNotes: 0,
  totalTasks: 0,
  completedTasks: 0,
  pendingTasks: 0,
  taskCompletionRate: 0,
  maxMessages: 0,
  messageTrend: [],
  fileTypes: []
})

// 时间周期选项
const periods = [
  { label: '日', value: 'day' },
  { label: '周', value: 'week' },
  { label: '月', value: 'month' },
  { label: '年', value: 'year' }
]

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 加载统计数据
const loadStatistics = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/statistics`, {
      params: { period: selectedPeriod.value },
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      }
    })
    statisticsData.value = response.data.data
  } catch (error) {
    console.error('加载统计数据失败:', error)
    ElMessage.error('加载统计数据失败，请稍后重试')
  }
}

// 切换统计周期
const changeStatisticsPeriod = (period: string) => {
  selectedPeriod.value = period
  loadStatistics()
}

// 获取文件类型颜色
const getFileTypeColor = (type: string) => {
  const colors = {
    '文档': '#3b82f6',
    '图片': '#10b981',
    '视频': '#f59e0b',
    '音频': '#8b5cf6',
    '其他': '#6b7280'
  }
  return colors[type] || '#6b7280'
}

// 获取饼图偏移量
const getPieOffset = (index: number, fileTypes: any[]) => {
  let offset = 0
  for (let i = 0; i < index; i++) {
    offset += fileTypes[i].percentage * 5.024
  }
  return offset
}

// 组件挂载时加载统计数据
onMounted(async () => {
  await loadStatistics()
})
</script>

<style scoped>
.statistics-app {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-color);
}

.statistics-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  height: 72px;
  box-sizing: border-box;
}

.statistics-header:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--primary-color);
}

.back-btn:hover {
  background: var(--primary-light);
}

.statistics-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  transition: color 0.3s ease;
}

.statistics-period-selector {
  display: flex;
  gap: 8px;
}

.period-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  background-color: var(--bg-color);
  color: var(--text-secondary);
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.period-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background-color: var(--hover-bg);
  transform: translateY(-1px);
}

.period-btn.active {
  background-color: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
  font-weight: 500;
}

.period-btn.active:hover {
  background-color: var(--primary-hover);
  transform: none;
}

.statistics-content {
  flex: 1;
  padding: 24px;
  gap: 24px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.statistics-overview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 16px;
}

.stat-card {
  background-color: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.stat-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
  border-radius: 4px 0 0 4px;
}

.stat-card:nth-child(1)::before {
  background-color: var(--primary-color);
}

.stat-card:nth-child(2)::before {
  background-color: var(--success-color);
}

.stat-card:nth-child(3)::before {
  background-color: var(--warning-color);
}

.stat-card:nth-child(4)::before {
  background-color: var(--error-color);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: white;
  transition: all 0.3s ease;
}

.messages-icon {
  background-color: var(--primary-color);
}

.files-icon {
  background-color: var(--success-color);
}

.notes-icon {
  background-color: var(--warning-color);
}

.tasks-icon {
  background-color: var(--error-color);
}

.stat-card:hover .stat-icon {
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
  transition: color 0.3s ease;
}

.stat-label {
  font-size: 14px;
  color: var(--text-secondary);
  transition: color 0.3s ease;
}

.chart-container {
  background-color: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  margin-bottom: 20px;
}

.chart-container:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.chart-container h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 16px 0;
  transition: color 0.3s ease;
}

.message-trend-chart {
  display: flex;
  align-items: flex-end;
  height: 300px;
  gap: 12px;
  padding: 20px 0;
  position: relative;
}

.chart-axis-y {
  display: flex;
  flex-direction: column-reverse;
  justify-content: space-between;
  height: 100%;
  width: 40px;
  font-size: 12px;
  color: var(--text-tertiary);
  transition: color 0.3s ease;
}

.axis-label {
  text-align: right;
  padding-right: 8px;
  transition: color 0.3s ease;
}

.chart-bars {
  flex: 1;
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 100%;
}

.chart-bar {
  flex: 1;
  background-color: var(--primary-light);
  border-radius: 4px 4px 0 0;
  position: relative;
  transition: all 0.3s ease;
  cursor: pointer;
  min-height: 20px;
}

.chart-bar:hover {
  background-color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(59, 130, 246, 0.3);
}

.bar-label {
  position: absolute;
  bottom: -24px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  transition: color 0.3s ease;
}

.file-distribution-chart {
  display: flex;
  align-items: center;
  gap: 40px;
  padding: 20px 0;
}

.pie-chart {
  position: relative;
  width: 200px;
  height: 200px;
  transition: all 0.3s ease;
}

.pie-chart:hover {
  transform: scale(1.05);
}

.pie-slice {
  transition: all 0.5s ease;
}

.pie-center {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 120px;
  height: 120px;
  background-color: var(--card-bg);
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  transition: all 0.3s ease;
}

.pie-center-text {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

.pie-center-label {
  font-size: 12px;
  color: var(--text-secondary);
  transition: color 0.3s ease;
}

.file-types-legend {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 12px;
  transition: all 0.3s ease;
}

.legend-item:hover {
  transform: translateX(4px);
}

.legend-color {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.legend-item:hover .legend-color {
  transform: scale(1.2);
}

.legend-label {
  flex: 1;
  font-size: 14px;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

.legend-value {
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 500;
  transition: color 0.3s ease;
}

.completion-chart {
  display: flex;
  align-items: center;
  gap: 40px;
  padding: 20px 0;
}

.completion-circle {
  position: relative;
  width: 200px;
  height: 200px;
  transition: all 0.3s ease;
}

.completion-circle:hover {
  transform: scale(1.05);
}

.completion-arc {
  transition: all 1s ease;
}

.completion-value {
  font-size: 32px;
  font-weight: 700;
  color: var(--success-color);
  transition: all 0.3s ease;
}

.completion-details {
  flex: 1;
  display: flex;
  gap: 32px;
}

.completion-item {
  flex: 1;
  text-align: center;
  transition: all 0.3s ease;
}

.completion-item:hover {
  transform: translateY(-2px);
}

.completion-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 8px;
  transition: color 0.3s ease;
}

.completion-count {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  transition: color 0.3s ease;
}

/* 动画效果 */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.stat-card,
.chart-container {
  animation: fadeIn 0.5s ease;
}

.stat-card:nth-child(1) {
  animation-delay: 0.1s;
}

.stat-card:nth-child(2) {
  animation-delay: 0.2s;
}

.stat-card:nth-child(3) {
  animation-delay: 0.3s;
}

.stat-card:nth-child(4) {
  animation-delay: 0.4s;
}

.chart-container:nth-child(2) {
  animation-delay: 0.5s;
}

.chart-container:nth-child(3) {
  animation-delay: 0.6s;
}

.chart-container:nth-child(4) {
  animation-delay: 0.7s;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .statistics-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
    padding: 16px 20px;
  }
  
  .statistics-period-selector {
    width: 100%;
    justify-content: space-between;
  }
  
  .period-btn {
    flex: 1;
    text-align: center;
  }
  
  .statistics-content {
    padding: 16px 20px;
  }
  
  .statistics-overview {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .chart-container {
    padding: 16px;
  }
  
  .chart-container h3 {
    font-size: 14px;
  }
  
  .message-trend-chart {
    height: 200px;
  }
  
  .chart-axis-y {
    width: 30px;
  }
  
  .axis-label {
    font-size: 10px;
  }
  
  .file-distribution-chart {
    flex-direction: column;
    gap: 24px;
  }
  
  .pie-chart {
    width: 150px;
    height: 150px;
  }
  
  .pie-center {
    width: 90px;
    height: 90px;
  }
  
  .pie-center-text {
    font-size: 18px;
  }
  
  .completion-chart {
    flex-direction: column;
    gap: 24px;
  }
  
  .completion-circle svg {
    width: 150px;
    height: 150px;
  }
  
  .completion-value {
    font-size: 24px;
  }
  
  .completion-details {
    width: 100%;
    gap: 16px;
  }
}

@media (max-width: 480px) {
  .statistics-overview {
    grid-template-columns: 1fr;
  }
  
  .stat-card {
    flex-direction: column;
    text-align: center;
    gap: 12px;
  }
  
  .completion-details {
    flex-direction: column;
    gap: 12px;
  }
  
  .completion-item {
    padding: 12px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background-color: var(--bg-color);
  }
}
</style>