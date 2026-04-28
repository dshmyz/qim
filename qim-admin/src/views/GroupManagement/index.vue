<!-- src/views/GroupManagement/index.vue -->
<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="handlePageChange" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="(searchForm.keyword as string)" label="群组名称" placeholder="请输入群组名称" />
      </SearchForm>
    </template>

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
        <StatusTag :status="row.status" />
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="180" fixed="right">
      <template #default="{ row }">
        <el-button size="small" type="primary" @click="handleViewMembers(row)">查看成员</el-button>
        <el-popconfirm title="确定删除该群组吗？" @confirm="handleDeleteGroup(row.id)">
          <template #reference>
            <ActionButton type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <el-dialog v-model="memberDialogVisible" :title="`群组成员 - ${currentGroup?.name || ''}`" width="600px">
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
          <el-popconfirm title="确定移除该成员吗？" @confirm="handleRemoveMember(row.userId)">
            <template #reference>
              <el-button size="small" type="danger" text>移除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import ActionButton from '@/components/common/ActionButton.vue'
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

const roleLabel = (role: string): string => {
  const map: Record<string, string> = { owner: '群主', admin: '管理员', member: '成员' }
  return map[role] || role
}

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
    console.error('[GroupManagement] fetch members failed:', error)
    ElMessage.error('获取成员列表失败')
  } finally {
    membersLoading.value = false
  }
}

const handleRemoveMember = async (userId: number) => {
  if (!currentGroup.value) return
  try {
    await removeGroupMember(currentGroup.value.id, userId)
    ElMessage.success('移除成功')
    fetchMembers(currentGroup.value.id)
  } catch (error) {
    console.error('[GroupManagement] remove member failed:', error)
    ElMessage.error('移除成员失败')
  }
}

// 删除群组（直接调用 API，避免 useEntity.handleDelete 双重确认弹窗）
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
</style>
