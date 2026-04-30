<template>
  <div class="apps-page">
    <el-card shadow="never">
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
          <el-form-item label="状态">
            <el-select v-model="searchForm.status" placeholder="请选择状态" clearable>
              <el-option label="正常" value="active" />
              <el-option label="停用" value="inactive" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
        <el-button type="success" @click="handleCreate">创建应用</el-button>
      </div>

      <el-table :data="apps" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="应用名称" min-width="180">
          <template #default="{ row }">
            <div class="app-cell">
              <el-avatar :size="32" :src="row.icon">{{ row.name.charAt(0) }}</el-avatar>
              <div class="app-info">
                <span class="app-name">{{ row.name }}</span>
                <el-tag v-if="row.isGlobal" size="small" type="warning" style="margin-left: 8px">全局</el-tag>
              </div>
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
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              :title="row.status === 'active' ? '确定停用该应用吗？' : '确定启用该应用吗？'"
              @confirm="handleToggleStatus(row)"
            >
              <template #reference>
                <el-button
                  size="small"
                  :type="row.status === 'active' ? 'warning' : 'success'"
                >
                  {{ row.status === 'active' ? '停用' : '启用' }}
                </el-button>
              </template>
            </el-popconfirm>
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
import { getApps, deleteApp, updateApp } from '@/api/apps'
import AppDialog from './components/AppDialog.vue'

const searchForm = reactive({ name: '', status: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const apps = ref<App[]>([])
const loading = ref(false)

const appDialogVisible = ref(false)
const currentApp = ref<App | null>(null)

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

const fetchApps = async () => {
  loading.value = true
  try {
    const { data } = await getApps({
      page: pagination.page,
      pageSize: pagination.pageSize,
      name: searchForm.name || undefined,
      status: searchForm.status || undefined,
    } as any)
    apps.value = data.data.list
    pagination.total = data.data.total
  } catch {
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
  searchForm.status = ''
  handleSearch()
}

const handleToggleStatus = async (row: App) => {
  const newStatus = row.status === 'active' ? 'inactive' : 'active'
  const actionText = newStatus === 'active' ? '启用' : '停用'

  try {
    await updateApp(row.id, { status: newStatus })
    ElMessage.success(`${actionText}成功`)
    fetchApps()
  } catch {
  }
}

const handleCreate = () => {
  currentApp.value = null
  appDialogVisible.value = true
}

const handleEdit = (row: App) => {
  currentApp.value = row
  appDialogVisible.value = true
}

const handleDelete = async (id: number) => {
  try {
    await deleteApp(id)
    ElMessage.success('删除成功')
    fetchApps()
  } catch {
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

.app-info {
  display: flex;
  align-items: center;
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
