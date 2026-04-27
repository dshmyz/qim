<template>
  <div class="mini-apps-page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="小程序名称">
            <el-input
              v-model="searchForm.name"
              placeholder="请输入小程序名称"
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
        <el-button type="success" @click="handleCreate">创建小程序</el-button>
      </div>

      <!-- 小程序列表 -->
      <el-table :data="miniApps" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="appID" label="AppID" min-width="160" />
        <el-table-column label="名称" min-width="150">
          <template #default="{ row }">
            <div class="mini-app-cell">
              <el-avatar :size="32" :src="row.icon">{{ row.name.charAt(0) }}</el-avatar>
              <span class="mini-app-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="状态" width="120">
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
              :title="row.status === 'active' ? '确定停用该小程序吗？' : '确定上线该小程序吗？上线后用户将可见'"
              @confirm="handleToggleStatus(row)"
            >
              <template #reference>
                <el-button
                  size="small"
                  :type="row.status === 'active' ? 'warning' : 'success'"
                >
                  {{ row.status === 'active' ? '停用' : '上线' }}
                </el-button>
              </template>
            </el-popconfirm>
            <el-popconfirm
              title="确定删除该小程序吗？"
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
          @size-change="fetchMiniApps"
          @current-change="fetchMiniApps"
        />
      </div>
    </el-card>

    <!-- 创建/编辑小程序对话框 -->
    <MiniAppDialog
      v-model="miniAppDialogVisible"
      :mini-app="currentMiniApp"
      @saved="fetchMiniApps"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { MiniApp } from '@/types'
import { getMiniApps, deleteMiniApp, updateMiniApp } from '@/api/miniApps'
import MiniAppDialog from './components/MiniAppDialog.vue'

// 搜索和分页
const searchForm = reactive({ name: '', status: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const miniApps = ref<MiniApp[]>([])
const loading = ref(false)

// 对话框
const miniAppDialogVisible = ref(false)
const currentMiniApp = ref<MiniApp | null>(null)

// 工具函数
const statusLabel = (status: string): string => {
  const map: Record<string, string> = { active: '正常', inactive: '停用' }
  return map[status] || status
}

const statusType = (status: string): 'success' | 'info' => {
  const map: Record<string, 'success' | 'info'> = { active: 'success', inactive: 'info' }
  return map[status] || 'info'
}

// 获取小程序列表
const fetchMiniApps = async () => {
  loading.value = true
  try {
    const { data } = await getMiniApps({
      page: pagination.page,
      pageSize: pagination.pageSize,
      name: searchForm.name || undefined,
      status: searchForm.status || undefined,
    })
    miniApps.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchMiniApps()
}

const handleReset = () => {
  searchForm.name = ''
  searchForm.status = ''
  handleSearch()
}

// 切换小程序状态
const handleToggleStatus = async (row: MiniApp) => {
  const newStatus = row.status === 'active' ? 'inactive' : 'active'
  const actionText = newStatus === 'active' ? '上线' : '停用'

  try {
    await updateMiniApp(row.id, { status: newStatus })
    ElMessage.success(`${actionText}成功`)
    fetchMiniApps()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 创建小程序
const handleCreate = () => {
  currentMiniApp.value = null
  miniAppDialogVisible.value = true
}

// 编辑小程序
const handleEdit = (row: MiniApp) => {
  currentMiniApp.value = row
  miniAppDialogVisible.value = true
}

// 删除小程序
const handleDelete = async (id: number) => {
  try {
    await deleteMiniApp(id)
    ElMessage.success('删除成功')
    fetchMiniApps()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchMiniApps)
</script>

<style scoped>
.mini-apps-page {
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

.mini-app-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mini-app-name {
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
