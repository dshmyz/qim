<template>
  <div class="crash-logs-page">
    <DataTable
      :data="list"
      :loading="loading"
      :pagination="pagination"
      @page-change="handlePageChange"
      @refresh="fetchData"
    >
      <template #search>
        <SearchForm @search="handleSearch" @reset="handleReset">
          <SearchField v-model="searchForm.platform" label="平台" placeholder="windows / macos / linux" />
          <SearchField v-model="searchForm.appVersion" label="版本号" placeholder="如 1.0.0" />
        </SearchForm>
      </template>

      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="platform" label="平台" width="120">
        <template #default="{ row }">
          <el-tag :type="platformTagType(row.platform)" size="small">{{ row.platform }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="appVersion" label="应用版本" width="120" />
      <el-table-column prop="crashType" label="崩溃类型" width="180">
        <template #default="{ row }">
          <span class="crash-type">{{ row.crashType }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="crashMessage" label="崩溃信息" min-width="250" show-overflow-tooltip />
      <el-table-column prop="createdAt" label="发生时间" width="180">
        <template #default="{ row }">
          {{ formatDate(row.createdAt) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" size="small" @click="handleViewDetail(row)">详情</el-button>
        </template>
      </el-table-column>
    </DataTable>

    <!-- 崩溃详情对话框 -->
    <el-dialog v-model="detailVisible" title="崩溃详情" width="700px" top="5vh">
      <div v-if="currentDetail" class="detail-content">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ID">{{ currentDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="平台">{{ currentDetail.platform }}</el-descriptions-item>
          <el-descriptions-item label="应用版本">{{ currentDetail.appVersion }}</el-descriptions-item>
          <el-descriptions-item label="崩溃类型">{{ currentDetail.crashType }}</el-descriptions-item>
          <el-descriptions-item label="发生时间" :span="2">{{ formatDate(currentDetail.createdAt) }}</el-descriptions-item>
          <el-descriptions-item label="崩溃信息" :span="2">{{ currentDetail.crashMessage }}</el-descriptions-item>
          <el-descriptions-item label="设备信息" :span="2">{{ currentDetail.deviceInfo || '无' }}</el-descriptions-item>
        </el-descriptions>

        <div class="stack-section">
          <h4>堆栈跟踪</h4>
          <pre class="stack-trace">{{ currentDetail.stackTrace || '无堆栈信息' }}</pre>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import { getCrashLogs, getCrashDetail } from '@/api/client'
import type { CrashLog } from '@/types/client'

const list = ref<CrashLog[]>([])
const loading = ref(false)
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

const searchForm = reactive({
  platform: '',
  appVersion: '',
})

const detailVisible = ref(false)
const currentDetail = ref<CrashLog | null>(null)

const fetchData = async () => {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: pagination.page,
      pageSize: pagination.pageSize,
    }
    if (searchForm.platform) params.platform = searchForm.platform
    if (searchForm.appVersion) params.appVersion = searchForm.appVersion

    const { data } = await getCrashLogs(params)
    list.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchData()
}

const handleReset = () => {
  searchForm.platform = ''
  searchForm.appVersion = ''
  handleSearch()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  fetchData()
}

const handleViewDetail = async (row: CrashLog) => {
  try {
    const { data } = await getCrashDetail(row.id)
    currentDetail.value = data.data
    detailVisible.value = true
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const platformTagType = (platform: string): 'primary' | 'success' | 'warning' => {
  const map: Record<string, 'primary' | 'success' | 'warning'> = {
    windows: 'primary',
    macos: 'success',
    linux: 'warning',
  }
  return map[platform] || 'primary'
}

const formatDate = (dateStr: string): string => {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchData)
</script>

<style scoped>
.crash-logs-page {
  padding: 0;
}

.crash-type {
  font-family: monospace;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.detail-content {
  max-height: 70vh;
  overflow-y: auto;
}

.stack-section {
  margin-top: 20px;
}

.stack-section h4 {
  margin-bottom: 12px;
  color: var(--color-text-primary);
}

.stack-trace {
  background: var(--color-bg-page, #f5f5f5);
  border-radius: 8px;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-primary);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow-y: auto;
}
</style>
