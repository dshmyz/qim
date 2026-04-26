<template>
  <div class="apps-page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="应用名称">
            <el-input
              v-model="searchForm.name"
              placeholder="请输入应用名称"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
        <el-button type="success" @click="handleCreate">创建应用</el-button>
      </div>

      <!-- 应用列表 -->
      <el-table :data="apps" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="应用名称" min-width="180">
          <template #default="{ row }">
            <div class="app-cell">
              <el-avatar :size="32" :src="row.icon">{{ row.name.charAt(0) }}</el-avatar>
              <span class="app-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="category" label="分类" min-width="120" />
        <el-table-column prop="url" label="链接地址" min-width="250" show-overflow-tooltip />
        <el-table-column label="打开方式" width="120">
          <template #default="{ row }">
            <el-tag :type="row.openType === 'in-app' ? '' : 'success'">
              {{ openTypeLabel(row.openType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">
              {{ statusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定删除该应用吗？"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
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
          @size-change="fetchApps"
          @current-change="fetchApps"
        />
      </div>
    </el-card>

    <!-- 创建/编辑应用对话框 -->
    <AppDialog
      v-model="appDialogVisible"
      :app="currentApp"
      @saved="fetchApps"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { App } from '@/types'
import { getApps, deleteApp } from '@/api/apps'
import AppDialog from './components/AppDialog.vue'

// 搜索和分页
const searchForm = reactive({ name: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const apps = ref<App[]>([])
const loading = ref(false)

// 对话框
const appDialogVisible = ref(false)
const currentApp = ref<App | null>(null)

// 工具函数
const openTypeLabel = (type: string): string => {
  const map: Record<string, string> = { 'in-app': '应用内', external: '外部' }
  return map[type] || type
}

const statusLabel = (status: string): string => {
  const map: Record<string, string> = { active: '正常', inactive: '停用' }
  return map[status] || status
}

const statusType = (status: string): 'success' | 'info' => {
  const map: Record<string, 'success' | 'info'> = { active: 'success', inactive: 'info' }
  return map[status] || 'info'
}

// 获取应用列表
const fetchApps = async () => {
  loading.value = true
  try {
    const { data } = await getApps({
      page: pagination.page,
      pageSize: pagination.pageSize,
      name: searchForm.name || undefined,
    })
    apps.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchApps()
}

const handleReset = () => {
  searchForm.name = ''
  handleSearch()
}

// 创建应用
const handleCreate = () => {
  currentApp.value = null
  appDialogVisible.value = true
}

// 编辑应用
const handleEdit = (row: App) => {
  currentApp.value = row
  appDialogVisible.value = true
}

// 删除应用
const handleDelete = async (id: number) => {
  try {
    await deleteApp(id)
    ElMessage.success('删除成功')
    fetchApps()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchApps)
</script>

<style scoped>
.apps-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.app-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.app-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
