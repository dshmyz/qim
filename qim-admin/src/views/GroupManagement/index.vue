<template>
  <div class="group-management-page">
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">群组管理</h1>
        <p class="page-subtitle">管理所有群组和成员信息</p>
      </div>
      <div class="header-stats">
        <div class="stat-item">
          <span class="stat-value">{{ pagination.total }}</span>
          <span class="stat-label">群组总数</span>
        </div>
      </div>
    </div>

    <div class="content-card">
      <div class="card-toolbar">
        <SearchForm @search="handleSearch" @reset="handleReset">
          <SearchField v-model="(searchForm.keyword as string)" label="群组名称" placeholder="请输入群组名称" />
        </SearchForm>
      </div>

      <el-table :data="list" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="群组名称" min-width="200">
          <template #default="{ row }">
            <div class="group-cell">
              <el-avatar :size="40" :src="row.avatar" class="group-avatar">
                {{ row.name?.charAt(0) || '?' }}
              </el-avatar>
              <div class="group-info">
                <span class="group-name">{{ row.name }}</span>
                <span class="group-desc">{{ row.description || '暂无描述' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="memberCount" label="成员数" width="100" align="center">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.memberCount || 0 }} 人</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <StatusTag :status="row.status" />
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handleViewMembers(row)">
              <el-icon><User /></el-icon>
              查看成员
            </el-button>
            <el-popconfirm title="确定删除该群组吗？" @confirm="handleDeleteGroup(row.id)">
              <template #reference>
                <el-button size="small" type="danger" plain>
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="card-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          layout="total, prev, pager, next"
          @current-change="handlePageChange"
        />
      </div>
    </div>

    <el-dialog
      v-model="memberDialogVisible"
      :title="`群组成员 - ${currentGroup?.name || ''}`"
      width="700px"
      :close-on-click-modal="false"
    >
      <div v-loading="membersLoading" class="members-grid">
        <div
          v-for="member in members"
          :key="member.id"
          class="member-card"
        >
          <el-avatar :size="48" :src="member.avatar" class="member-avatar">
            {{ (member.nickname || member.username)?.charAt(0)?.toUpperCase() || '?' }}
          </el-avatar>
          <div class="member-info">
            <span class="member-name">{{ member.nickname || member.username }}</span>
            <div class="member-meta">
              <el-tag v-if="member.role" size="small" :type="getRoleType(member.role)">
                {{ roleLabel(member.role) }}
              </el-tag>
              <span class="member-time">{{ member.joinedAt }}</span>
            </div>
          </div>
          <el-button
            v-if="member.userId !== currentGroup?.ownerId"
            size="small"
            type="danger"
            text
            @click="handleRemoveMember(member.userId)"
          >
            移除
          </el-button>
          <el-tag v-else size="small" type="warning">群主</el-tag>
        </div>
        <el-empty v-if="!membersLoading && members.length === 0" description="暂无成员" :image-size="64" />
      </div>
      <el-pagination
        v-if="memberPagination.total > memberPagination.pageSize"
        :current-page="memberPagination.page"
        :page-size="memberPagination.pageSize"
        :total="memberPagination.total"
        layout="total, prev, pager, next"
        small
        class="member-pagination"
        @current-change="handleMemberPageChange"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { User, Delete } from '@element-plus/icons-vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import { useEntity } from '@/composables/useEntity'
import { getGroups, getGroupMembers, removeGroupMember, deleteGroup } from '@/api/groups'
import type { Group, ConversationMember } from '@/types'

const groupApi = {
  getList: (params: Record<string, unknown>) => getGroups(params as any),
  create: async () => {},
  update: async () => {},
  delete: (id: number) => deleteGroup(id).then(() => {}),
}

const {
  list, loading, pagination, searchForm,
  handleSearch, handleReset, fetchData
} = useEntity<Group>({
  api: groupApi,
  searchFields: ['keyword'],
  formFields: [],
})

const memberDialogVisible = ref(false)
const currentGroup = ref<Group | null>(null)
const members = ref<ConversationMember[]>([])
const membersLoading = ref(false)

const memberPagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
})

const roleLabel = (role: string): string => {
  const map: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' }
  return map[role] || role
}

const getRoleType = (role: string): '' | 'success' | 'warning' | 'info' | 'danger' => {
  const map: Record<string, '' | 'success' | 'warning' | 'info' | 'danger'> = {
    owner: 'warning',
    admin: 'success',
    member: 'info'
  }
  return map[role] || 'info'
}

const handleViewMembers = async (row: Group) => {
  currentGroup.value = row
  memberDialogVisible.value = true
  memberPagination.value.page = 1
  await fetchMembers(1)
}

const fetchMembers = async (page = 1) => {
  if (!currentGroup.value) return
  membersLoading.value = true
  try {
    const { data } = await getGroupMembers(currentGroup.value.id, {
      page,
      pageSize: memberPagination.value.pageSize,
    })
    if (data.data) {
      members.value = data.data.list ?? []
      memberPagination.value.total = data.data.total ?? 0
      memberPagination.value.page = page
    }
  } catch (error) {
    console.error('[GroupManagement] fetch members failed:', error)
    ElMessage.error('获取成员列表失败')
  } finally {
    membersLoading.value = false
  }
}

const handleMemberPageChange = (page: number) => {
  fetchMembers(page)
}

const handleRemoveMember = async (userId: number) => {
  if (userId === currentGroup.value?.ownerId) {
    ElMessage.warning('无法移除群主')
    return
  }

  try {
    await removeGroupMember(currentGroup.value!.id, userId)
    ElMessage.success('成员移除成功')
    fetchMembers(memberPagination.value.page)
  } catch (error) {
    console.error('[GroupManagement] remove member failed:', error)
    ElMessage.error('移除成员失败')
  }
}

const handleDeleteGroup = async (id: number) => {
  try {
    await deleteGroup(id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (error) {
    console.error('[GroupManagement] delete group failed:', error)
    ElMessage.error('删除群组失败')
  }
}

onMounted(fetchData)

const handlePageChange = (page: number) => {
  pagination.page = page
  fetchData()
}
</script>

<style scoped>
.group-management-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  background: linear-gradient(135deg, #10b981 0%, #0ea5e9 100%);
  border-radius: var(--radius-xl);
  color: white;
}

.header-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
  color: white;
  letter-spacing: -0.02em;
  line-height: 1.2;
}

.page-subtitle {
  margin: 0;
  font-size: 14px;
  color: rgba(255, 255, 255, 0.85);
  font-weight: 500;
}

.header-stats {
  display: flex;
  gap: var(--space-5);
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat-value {
  font-size: 28px;
  font-weight: 800;
  color: white;
}

.stat-label {
  font-size: 12px;
  color: rgba(255, 255, 255, 0.8);
}

.content-card {
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-card);
  padding: var(--space-4);
}

.card-toolbar {
  margin-bottom: var(--space-3);
}

.group-cell {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.group-avatar {
  background: linear-gradient(135deg, #10b981 0%, #0ea5e9 100%);
  color: white;
  font-weight: 600;
  flex-shrink: 0;
}

.group-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.group-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.group-desc {
  font-size: 12px;
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border-light);
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-2);
  min-height: 200px;
}

.member-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-surface-hover);
  transition: all var(--duration-fast) var(--ease-out);
}

.member-card:hover {
  background: var(--color-surface-active);
}

.member-avatar {
  background: linear-gradient(135deg, #10b981 0%, #0ea5e9 100%);
  color: white;
  font-weight: 600;
  flex-shrink: 0;
}

.member-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.member-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.member-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.member-time {
  font-size: 11px;
  color: var(--color-text-muted);
}

.member-pagination {
  margin-top: var(--space-3);
  justify-content: center;
}

:deep(.el-table .el-table__cell) {
  padding: 8px 0;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-3);
  }

  .header-stats {
    width: 100%;
    justify-content: flex-start;
  }
}
</style>
