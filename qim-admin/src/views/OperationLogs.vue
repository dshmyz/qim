<template>
  <div class="operation-logs-page">
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
        <el-table-column prop="operatorName" label="操作人" min-width="120" />
        <el-table-column label="操作" min-width="140">
          <template #default="{ row }">
            <el-tag size="small">{{ actionLabel(row.action) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="targetType" label="目标类型" width="120" />
        <el-table-column label="目标ID" width="100">
          <template #default="{ row }">
            <span class="text-mono">{{ row.targetId }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="操作详情" min-width="200" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP 地址" width="140">
          <template #default="{ row }">
            <span class="text-mono">{{ row.ip }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="操作时间" width="180" />
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
      width="500px"
    >
      <el-descriptions :column="1" border>
        <el-descriptions-item label="操作人">{{ currentLog?.operatorName }}</el-descriptions-item>
        <el-descriptions-item label="操作人ID">{{ currentLog?.operatorId }}</el-descriptions-item>
        <el-descriptions-item label="操作">{{ currentLog ? actionLabel(currentLog.action) : '' }}</el-descriptions-item>
        <el-descriptions-item label="目标类型">{{ currentLog?.targetType }}</el-descriptions-item>
        <el-descriptions-item label="目标ID">{{ currentLog?.targetId }}</el-descriptions-item>
        <el-descriptions-item label="操作详情">{{ currentLog?.detail }}</el-descriptions-item>
        <el-descriptions-item label="IP 地址">{{ currentLog?.ip }}</el-descriptions-item>
        <el-descriptions-item label="操作时间">{{ currentLog?.createdAt }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { OperationLog } from '@/types'
import { getOperationLogs, exportOperationLogs } from '@/api/operationLogs'

// 操作类型选项
const actionOptions = ['create', 'update', 'delete', 'login', 'logout', 'assign', 'revoke', 'export']

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

// 搜索和分页
const searchForm = reactive({
  operatorName: '',
  action: '',
  dateRange: [] as string[],
})
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const logs = ref<OperationLog[]>([])
const loading = ref(false)

// 详情对话框
const detailDialogVisible = ref(false)
const currentLog = ref<OperationLog | null>(null)

// 获取列表
const fetchLogs = async () => {
  loading.value = true
  try {
    const { data } = await getOperationLogs({
      page: pagination.page,
      pageSize: pagination.pageSize,
      operatorName: searchForm.operatorName || undefined,
      action: searchForm.action || undefined,
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

const handleSearch = () => {
  pagination.page = 1
  fetchLogs()
}

const handleReset = () => {
  searchForm.operatorName = ''
  searchForm.action = ''
  searchForm.dateRange = []
  handleSearch()
}

// 查看详情
const handleViewDetail = (row: OperationLog) => {
  currentLog.value = row
  detailDialogVisible.value = true
}

// 导出
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

onMounted(fetchLogs)
</script>

<style scoped>
.operation-logs-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
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

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
