<template>
  <div class="operation-logs-page">
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ stats.total }}</div>
          <div class="stat-label">总操作数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card success">
          <div class="stat-value">{{ stats.success }}</div>
          <div class="stat-label">成功</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card error">
          <div class="stat-value">{{ stats.failed }}</div>
          <div class="stat-label">失败</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-value">{{ stats.avgDuration }}ms</div>
          <div class="stat-label">平均耗时</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 操作趋势图 -->
    <el-card shadow="never" class="chart-card">
      <div class="card-header">
        <h4>操作趋势（近 7 天）</h4>
      </div>
      <div ref="chartRef" class="chart-container" v-loading="chartLoading"></div>
    </el-card>

    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="操作人">
            <el-input
              v-model="searchForm.operatorName"
              placeholder="请输入操作人姓名"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item label="操作类型">
            <el-select v-model="searchForm.action" placeholder="请选择操作类型" clearable>
              <el-option
                v-for="act in actionOptions"
                :key="act"
                :label="actionLabel(act)"
                :value="act"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="模块">
            <el-select v-model="searchForm.module" placeholder="请选择模块" clearable>
              <el-option
                v-for="mod in moduleOptions"
                :key="mod"
                :label="moduleLabel(mod)"
                :value="mod"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="searchForm.status" placeholder="请选择状态" clearable>
              <el-option label="成功" value="success" />
              <el-option label="失败" value="failed" />
            </el-select>
          </el-form-item>
          <el-form-item label="日期范围">
            <el-date-picker
              v-model="searchForm.dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              value-format="YYYY-MM-DD"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
        <el-button @click="handleExport">导出日志</el-button>
      </div>

      <!-- 日志列表 -->
      <el-table :data="logs" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="username" label="操作人" min-width="120" />
        <el-table-column label="操作" min-width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="actionTagType(row.action)">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="模块" width="100">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ moduleLabel(row.module) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="isSuccess(row.response) ? 'success' : 'danger'">
              {{ isSuccess(row.response) ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="request_url" label="请求路径" min-width="180" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP 地址" width="140">
          <template #default="{ row }">
            <span class="text-mono">{{ row.ip }}</span>
          </template>
        </el-table-column>
        <el-table-column label="耗时" width="80">
          <template #default="{ row }">
            <span :class="row.duration > 1000 ? 'text-danger' : ''">{{ row.duration }}ms</span>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="操作时间" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handleViewDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchLogs"
          @current-change="fetchLogs"
        />
      </div>
    </el-card>

    <!-- 日志详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      title="操作日志详情"
      width="700px"
    >
      <el-descriptions :column="2" border>
        <el-descriptions-item label="操作人">{{ currentLog?.username }}</el-descriptions-item>
        <el-descriptions-item label="操作人ID">{{ currentLog?.user_id }}</el-descriptions-item>
        <el-descriptions-item label="操作">{{ currentLog ? actionLabel(currentLog.action) : '' }}</el-descriptions-item>
        <el-descriptions-item label="模块">{{ moduleLabel(currentLog?.module) }}</el-descriptions-item>
        <el-descriptions-item label="请求方法" :span="2">{{ currentLog?.request_method || '-' }}</el-descriptions-item>
        <el-descriptions-item label="请求 URL" :span="2">{{ currentLog?.request_url || '-' }}</el-descriptions-item>
        <el-descriptions-item label="IP 地址">{{ currentLog?.ip }}</el-descriptions-item>
        <el-descriptions-item label="耗时">{{ currentLog?.duration }}ms</el-descriptions-item>
        <el-descriptions-item label="状态" :span="2">
          <el-tag :type="isSuccess(currentLog?.response) ? 'success' : 'danger'">
            {{ isSuccess(currentLog?.response) ? '成功' : '失败' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="User-Agent" :span="2">{{ currentLog?.user_agent || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">请求参数</el-divider>
      <pre class="json-block">{{ formatJson(currentLog?.request_body) }}</pre>

      <el-divider content-position="left">响应结果</el-divider>
      <pre class="json-block">{{ formatJson(currentLog?.response) }}</pre>

      <div class="detail-footer">
        <span class="detail-time">操作时间：{{ currentLog?.created_at }}</span>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import type { OperationLog } from '@/types'
import { getOperationLogs, exportOperationLogs, getOperationLogStats } from '@/api/operationLogs'

const actionOptions = ['create', 'update', 'delete', 'login', 'logout', 'assign', 'revoke', 'export']
const moduleOptions = ['user', 'group', 'conversation', 'message', 'system', 'config', 'role', 'file', 'ai']

const actionLabel = (action: string): string => {
  const map: Record<string, string> = {
    create: '创建',
    update: '编辑',
    delete: '删除',
    login: '登录',
    logout: '退出登录',
    assign: '分配',
    revoke: '撤销',
    export: '导出',
  }
  return map[action] || action
}

const actionTagType = (action: string): string => {
  const map: Record<string, string> = {
    create: 'success',
    update: 'primary',
    delete: 'danger',
    login: 'info',
    logout: 'info',
  }
  return map[action] || ''
}

const moduleLabel = (mod: string): string => {
  const map: Record<string, string> = {
    user: '用户',
    group: '群组',
    conversation: '会话',
    message: '消息',
    system: '系统',
    config: '配置',
    role: '角色',
    file: '文件',
    ai: 'AI',
  }
  return map[mod] || mod
}

const isSuccess = (response: string): boolean => {
  if (!response) return true
  try {
    const res = JSON.parse(response)
    return res.code === 0 || res.code === 200
  } catch {
    return true
  }
}

const searchForm = reactive({
  operatorName: '',
  action: '',
  module: '',
  status: '',
  dateRange: [] as string[],
})
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const logs = ref<OperationLog[]>([])
const loading = ref(false)

const stats = reactive({ total: 0, success: 0, failed: 0, avgDuration: 0 })

const detailDialogVisible = ref(false)
const currentLog = ref<OperationLog | null>(null)

const chartRef = ref<HTMLElement>()
const chartLoading = ref(false)
let chartInstance: echarts.ECharts | null = null

const fetchLogs = async () => {
  loading.value = true
  try {
    const { data } = await getOperationLogs({
      page: pagination.page,
      pageSize: pagination.pageSize,
      operatorName: searchForm.operatorName || undefined,
      action: searchForm.action || undefined,
      module: searchForm.module || undefined,
      status: searchForm.status || undefined,
      startDate: searchForm.dateRange?.[0],
      endDate: searchForm.dateRange?.[1],
    })
    logs.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const fetchStats = async () => {
  try {
    const { data } = await getOperationLogStats({
      startDate: searchForm.dateRange?.[0],
      endDate: searchForm.dateRange?.[1],
    })
    Object.assign(stats, data.data)
  } catch {
    // 忽略统计错误
  }
}

const fetchChartData = async () => {
  chartLoading.value = true
  try {
    const { data } = await getOperationLogStats({
      startDate: searchForm.dateRange?.[0],
      endDate: searchForm.dateRange?.[1],
      trend: true,
    })
    renderChart(data.data.trend || [])
  } catch {
    // 忽略图表错误
  } finally {
    chartLoading.value = false
  }
}

const renderChart = (trendData: Array<{ date: string; count: number }>) => {
  if (!chartRef.value) return
  
  if (!chartInstance) {
    chartInstance = echarts.init(chartRef.value)
  }

  chartInstance.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: trendData.map(d => d.date),
      boundaryGap: false,
    },
    yAxis: { type: 'value' },
    series: [{
      name: '操作数',
      type: 'line',
      smooth: true,
      data: trendData.map(d => d.count),
      areaStyle: { opacity: 0.3 },
      itemStyle: { color: '#409eff' },
    }],
  })
}

const handleSearch = () => {
  pagination.page = 1
  fetchLogs()
  fetchStats()
  fetchChartData()
}

const handleReset = () => {
  searchForm.operatorName = ''
  searchForm.action = ''
  searchForm.module = ''
  searchForm.status = ''
  searchForm.dateRange = []
  handleSearch()
}

const handleViewDetail = async (row: OperationLog) => {
  try {
    const { data } = await getOperationLogDetail(row.id)
    currentLog.value = data.data
    detailDialogVisible.value = true
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const handleExport = async () => {
  try {
    const { data } = await exportOperationLogs({
      startDate: searchForm.dateRange?.[0],
      endDate: searchForm.dateRange?.[1],
    })
    window.open(data.data.url, '_blank')
    ElMessage.success('导出链接已生成')
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const formatJson = (str: string): string => {
  if (!str) return '-'
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

onMounted(() => {
  fetchLogs()
  fetchStats()
  fetchChartData()
})

onUnmounted(() => {
  chartInstance?.dispose()
})
</script>

<style scoped>
.operation-logs-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.stats-row {
  margin-bottom: var(--space-2);
}

.stat-card {
  text-align: center;
  padding: var(--space-4);
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.stat-label {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-top: var(--space-1);
}

.stat-card.success .stat-value {
  color: #67c23a;
}

.stat-card.error .stat-value {
  color: #f56c6c;
}

.chart-card {
  margin-bottom: var(--space-4);
}

.card-header h4 {
  margin: 0 0 var(--space-4);
  font-size: 16px;
  font-weight: 600;
}

.chart-container {
  height: 300px;
  width: 100%;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.text-mono {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.text-danger {
  color: #f56c6c;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.json-block {
  background: var(--el-fill-color-light);
  padding: var(--space-3);
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.5;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.detail-footer {
  margin-top: var(--space-4);
  display: flex;
  justify-content: flex-end;
}

.detail-time {
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
