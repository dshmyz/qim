<template>
  <div class="notifications-page">
    <el-card shadow="never">
      <!-- 操作栏 -->
      <div class="action-bar">
        <div class="left-actions">
          <el-form :model="filterForm" inline>
            <el-form-item label="类型">
              <el-select v-model="filterForm.type" placeholder="全部类型" clearable style="width: 140px" @change="handleSearch">
                <el-option label="通知" value="notification" />
                <el-option label="警告" value="warning" />
                <el-option label="信息" value="info" />
              </el-select>
            </el-form-item>
            <el-form-item label="状态">
              <el-select v-model="filterForm.isRead" placeholder="全部状态" clearable style="width: 120px" @change="handleSearch">
                <el-option label="已读" :value="true" />
                <el-option label="未读" :value="false" />
              </el-select>
            </el-form-item>
          </el-form>
        </div>
        <el-button type="primary" @click="handleMarkAllAsRead">全部标记为已读</el-button>
      </div>

      <!-- 通知列表 -->
      <el-table :data="notifications" v-loading="loading">
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="typeColor(row.type)">
              {{ typeLabel(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="200" />
        <el-table-column label="内容预览" min-width="300" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="content-preview">{{ row.content }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isRead" size="small" type="info">已读</el-tag>
            <el-tag v-else size="small">未读</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="发送时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="!row.isRead"
              size="small"
              type="primary"
              @click="handleToggleRead(row)"
            >标记已读</el-button>
            <el-button
              v-else
              size="small"
              @click="handleToggleRead(row)"
            >标记未读</el-button>
            <el-popconfirm
              title="确定删除该通知吗？"
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
          @size-change="fetchNotifications"
          @current-change="fetchNotifications"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import type { Notification } from '@/types'
import { getNotifications, markAsRead, markAsUnread, markAllAsRead, deleteNotification, type GetNotificationsParams } from '@/api/notifications'

// 筛选和分页
const filterForm = reactive({
  type: '',
  isRead: null as boolean | null,
})
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const notifications = ref<Notification[]>([])
const loading = ref(false)

// 工具函数
const typeLabel = (type: string): string => {
  const map: Record<string, string> = { notification: '通知', warning: '警告', info: '信息' }
  return map[type] || type
}

const typeColor = (type: string): 'primary' | 'warning' | 'info' => {
  const map: Record<string, 'primary' | 'warning' | 'info'> = { notification: 'primary', warning: 'warning', info: 'info' }
  return map[type] || 'info'
}

// 获取通知列表
const fetchNotifications = async () => {
  loading.value = true
  try {
    const params: GetNotificationsParams = {
      page: pagination.page,
      pageSize: pagination.pageSize,
    }
    if (filterForm.type) params.type = filterForm.type
    if (filterForm.isRead !== null) params.isRead = filterForm.isRead

    const { data } = await getNotifications(params)
    notifications.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchNotifications()
}

// 标记已读/未读
const handleToggleRead = async (row: Notification) => {
  try {
    if (!row.isRead) {
      await markAsRead(row.id)
      ElMessage.success('已标记为已读')
    } else {
      await markAsUnread(row.id)
      ElMessage.success('已标记为未读')
    }
    fetchNotifications()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 全部标记为已读
const handleMarkAllAsRead = async () => {
  try {
    await markAllAsRead()
    ElMessage.success('全部标记为已读')
    fetchNotifications()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 删除通知
const handleDelete = async (id: number) => {
  try {
    await deleteNotification(id)
    ElMessage.success('删除成功')
    fetchNotifications()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchNotifications)
</script>

<style scoped>
.notifications-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.content-preview {
  color: var(--color-text-secondary);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
