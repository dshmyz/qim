<template>
  <div class="groups-page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="群组名称">
            <el-input
              v-model="searchForm.keyword"
              placeholder="请输入群组名称"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>

      <!-- 群组列表 -->
      <el-table :data="groups" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="群组名称" min-width="180">
          <template #default="{ row }">
            <div class="group-cell">
              <el-avatar :size="32" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
              <span class="group-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="memberCount" label="成员数" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '正常' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handleViewMembers(row)">查看成员</el-button>
            <el-popconfirm
              title="确定删除该群组吗？"
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
          @size-change="fetchGroups"
          @current-change="fetchGroups"
        />
      </div>
    </el-card>

    <!-- 查看群组成员对话框 -->
    <el-dialog
      v-model="memberDialogVisible"
      :title="`群组成员 - ${currentGroup?.name || ''}`"
      width="600px"
    >
      <el-table :data="members" v-loading="membersLoading" size="small" max-height="400">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="成员" min-width="150">
          <template #default="{ row }">
            <div class="member-cell">
              <el-avatar :size="28" :src="row.avatar">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
              <span>{{ row.nickname || row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="100">
          <template #default="{ row }">
            <el-tag size="small" v-if="row.role">{{ roleLabel(row.role) }}</el-tag>
            <span v-else class="text-muted">普通成员</span>
          </template>
        </el-table-column>
        <el-table-column prop="joinedAt" label="加入时间" width="180" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-popconfirm
              title="确定移除该成员吗？"
              @confirm="handleRemoveMember(row.userId)"
            >
              <template #reference>
                <el-button size="small" type="danger" text>移除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { Group, ConversationMember } from '@/types'
import { getGroups, getGroupMembers, removeGroupMember, deleteGroup } from '@/api/groups'

// 搜索和分页
const searchForm = reactive({ keyword: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const groups = ref<Group[]>([])
const loading = ref(false)

// 成员对话框
const memberDialogVisible = ref(false)
const currentGroup = ref<Group | null>(null)
const members = ref<ConversationMember[]>([])
const membersLoading = ref(false)

// 工具函数
const roleLabel = (role: string): string => {
  const map: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' }
  return map[role] || role
}

// 获取群组列表
const fetchGroups = async () => {
  loading.value = true
  try {
    const { data } = await getGroups({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined,
    })
    groups.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchGroups()
}

const handleReset = () => {
  searchForm.keyword = ''
  handleSearch()
}

// 查看成员
const handleViewMembers = async (row: Group) => {
  currentGroup.value = row
  memberDialogVisible.value = true
  await fetchMembers(row.id)
}

const fetchMembers = async (conversationId: number) => {
  membersLoading.value = true
  try {
    const { data } = await getGroupMembers(conversationId, { page: 1, pageSize: 100 })
    members.value = data.data.list
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    membersLoading.value = false
  }
}

// 移除成员
const handleRemoveMember = async (userId: number) => {
  if (!currentGroup.value) return
  try {
    await removeGroupMember(currentGroup.value.id, userId)
    ElMessage.success('移除成功')
    fetchMembers(currentGroup.value.id)
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

// 删除群组
const handleDelete = async (id: number) => {
  try {
    await deleteGroup(id)
    ElMessage.success('删除成功')
    fetchGroups()
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchGroups)
</script>

<style scoped>
.groups-page {
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

.group-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.group-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.member-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
