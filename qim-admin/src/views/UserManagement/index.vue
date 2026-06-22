<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="handlePageChange" @refresh="fetchData"
    @selection-change="handleSelectionChange">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="keyword" label="关键词" placeholder="用户名或昵称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-role="'system_admin'" type="primary" @click="handleCreate">创建用户</el-button>
      <el-button
        v-role="'system_admin'"
        type="danger"
        :disabled="selectedIds.length === 0"
        :loading="batchDeleting"
        @click="handleBatchDelete"
      >
        批量删除{{ selectedIds.length > 0 ? `(${selectedIds.length})` : '' }}
      </el-button>
      <el-button
        v-role="'system_admin'"
        :disabled="selectedIds.length === 0"
        @click="handleBatchAssignRoles"
      >
        批量分配角色{{ selectedIds.length > 0 ? `(${selectedIds.length})` : '' }}
      </el-button>
    </template>

    <el-table-column type="selection" width="48" />
    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column label="用户名" min-width="150">
      <template #default="{ row }">
        <div class="user-cell">
          <el-avatar :size="32" :src="row.avatar">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
          <div class="user-info">
            <span class="username">{{ row.username }}</span>
            <span class="nickname">{{ row.nickname || '-' }}</span>
          </div>
        </div>
      </template>
    </el-table-column>
    <el-table-column prop="email" label="邮箱" min-width="180" />
    <el-table-column prop="phone" label="手机号" min-width="140" />
    <el-table-column label="角色" min-width="200">
      <template #default="{ row }">
        <el-tag v-for="role in (row.roles || []).slice(0, 3)" :key="role" size="small" class="role-tag">
          {{ getRoleName(role) }}
        </el-tag>
        <el-tag v-if="(row.roles || []).length > 3" size="small" type="info">
          +{{ (row.roles || []).length - 3 }}
        </el-tag>
        <span v-if="!row.roles || row.roles.length === 0" class="text-muted">未分配</span>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="100">
      <template #default="{ row }">
        <StatusTag :status="row.status" />
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="340" fixed="right">
      <template #default="{ row }">
        <ActionButton v-permission="'user:update'" @click="handleEdit(row)">编辑</ActionButton>
        <ActionButton v-permission="'user:update'" @click="handleManageRoles(row)">管理角色</ActionButton>
        <ActionButton v-permission="'user:update'" @click="handleManageAIConfig(row)">AI 配置</ActionButton>
        <el-popconfirm title="确定删除该用户吗？" @confirm="handleDelete(row.id)">
          <template #reference>
            <ActionButton v-permission="'user:delete'" type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <EntityDialog
    v-model="dialogVisible"
    :mode="dialogMode"
    :fields="entityFields"
    :rules="entityRules"
    :initial-data="formData"
    :create-title="'创建用户'"
    :edit-title="'编辑用户'"
    @save="handleSave"
  />

  <el-dialog v-model="roleDialogVisible" title="管理角色" width="400px">
    <p class="role-dialog-hint">为用户 <strong>{{ currentUser?.username }}</strong> 分配角色</p>
    <el-checkbox-group v-model="selectedRoles" class="role-checkbox-list">
        <el-checkbox v-for="role in roleOptions" :key="role.value" :label="role.value">
          {{ role.label }}
        </el-checkbox>
      </el-checkbox-group>
    <template #footer>
      <el-button @click="roleDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="roleSubmitting" @click="handleSaveRoles">保存</el-button>
    </template>
  </el-dialog>

  <AIConfigDialog
    v-model:visible="aiConfigDialogVisible"
    :user-id="aiConfigUserId"
    :username="aiConfigUsername"
  />

  <el-dialog v-model="batchRoleDialogVisible" title="批量分配角色" width="400px">
    <p class="role-dialog-hint">为选中的 <strong>{{ selectedIds.length }}</strong> 个用户分配角色</p>
    <el-checkbox-group v-model="batchSelectedRoles" class="role-checkbox-list">
      <el-checkbox v-for="role in roleOptions" :key="role.value" :label="role.value">
        {{ role.label }}
      </el-checkbox>
    </el-checkbox-group>
    <template #footer>
      <el-button @click="batchRoleDialogVisible = false">取消</el-button>
      <el-button type="primary" :loading="batchRoleSubmitting" @click="handleSaveBatchRoles">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormRules } from 'element-plus'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import EntityDialog from '@/components/forms/EntityDialog.vue'
import AIConfigDialog from './components/AIConfigDialog.vue'
import { useEntity } from '@/composables/useEntity'
import { getUsers, createUser, updateUser, deleteUser, assignRoles } from '@/api/users'
import { userFields, userRules, roleOptions } from './config'
import type { User } from '@/types'
import type { FormField as EntityFormField } from '@/composables/useEntity'
import type { FormField as RendererFormField } from '@/components/forms/FieldRenderer.vue'

const userApi = {
  getList: (params: Record<string, unknown>) => getUsers(params as any),
  create: (data: Record<string, unknown>) => createUser(data as any).then(() => {}),
  update: (id: number, data: Record<string, unknown>) => updateUser(id, data as any).then(() => {}),
  delete: (id: number) => deleteUser(id).then(() => {}),
}

const typedFields = userFields as EntityFormField[]

const {
  list, loading, pagination, searchForm,
  dialogVisible, dialogMode, formData,
  handleSearch, handleReset, handleCreate, handleEdit, handleDelete, handleSave,
  fetchData
} = useEntity<User>({
  api: userApi,
  searchFields: ['keyword'],
  formFields: typedFields,
  successMessages: { create: '创建成功', update: '更新成功', delete: '删除成功' }
})

const entityFields = computed<RendererFormField[]>(() => [
  { name: 'username', label: '用户名', type: 'input', required: true, props: { disabled: dialogMode.value === 'edit' } },
  { name: 'password', label: '密码', type: 'password', required: dialogMode.value === 'create', props: { showPassword: true, placeholder: dialogMode.value === 'edit' ? '留空表示不修改密码' : '请输入密码' } },
  { name: 'nickname', label: '昵称', type: 'input' },
  { name: 'email', label: '邮箱', type: 'input', required: true },
  { name: 'phone', label: '手机号', type: 'input' },
  { name: 'avatar', label: '头像', type: 'input', props: { placeholder: '请输入头像URL' } },
  { name: 'signature', label: '个性签名', type: 'textarea', props: { rows: 3, placeholder: '请输入个性签名' } },
  { name: 'status', label: '状态', type: 'select', options: [
    { label: '在线', value: 'online' },
    { label: '离线', value: 'offline' },
    { label: '禁用', value: 'banned' },
  ] },
])

const entityRules = computed<FormRules>(() => {
  const rules: FormRules = { ...userRules }
  if (dialogMode.value === 'edit') {
    delete rules.password
  }
  return rules
})

const keyword = computed({
  get: () => (searchForm.keyword as string) || '',
  set: (val: string) => { searchForm.keyword = val }
})

const roleDialogVisible = ref(false)
const currentUser = ref<User | null>(null)
const selectedRoles = ref<string[]>([])
const roleSubmitting = ref(false)

const getRoleName = (roleId: string): string => {
  const option = roleOptions.find((r) => r.value === roleId)
  return option?.label || `角色 #${roleId}`
}

const handleManageRoles = (row: User) => {
  currentUser.value = row
  selectedRoles.value = row.roles ? [...row.roles] : []
  roleDialogVisible.value = true
}

const handleSaveRoles = async () => {
  if (!currentUser.value) return
  roleSubmitting.value = true
  try {
    await assignRoles(currentUser.value.id, selectedRoles.value)
    ElMessage.success('角色更新成功')
    roleDialogVisible.value = false
    fetchData()
  } catch (error) {
    console.error('[UserManagement] save roles failed:', error)
    ElMessage.error('角色保存失败')
  } finally {
    roleSubmitting.value = false
  }
}

const handlePageChange = (page: number) => {
  pagination.page = page
  fetchData()
}

const aiConfigDialogVisible = ref(false)
const aiConfigUserId = ref(0)
const aiConfigUsername = ref('')

const handleManageAIConfig = (row: User) => {
  aiConfigUserId.value = row.id
  aiConfigUsername.value = row.username
  aiConfigDialogVisible.value = true
}

// ===== 批量操作 =====
const selectedRows = ref<User[]>([])
const selectedIds = computed(() => selectedRows.value.map(r => r.id))
const batchDeleting = ref(false)
const batchRoleDialogVisible = ref(false)
const batchSelectedRoles = ref<string[]>([])
const batchRoleSubmitting = ref(false)

const handleSelectionChange = (rows: unknown[]) => {
  selectedRows.value = rows as User[]
}

const handleBatchDelete = async () => {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(
      `确定删除选中的 ${selectedIds.value.length} 个用户吗？此操作不可恢复。`,
      '批量删除确认',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return // 用户取消
  }

  batchDeleting.value = true
  let successCount = 0
  let failCount = 0
  try {
    for (const id of selectedIds.value) {
      try {
        await deleteUser(id)
        successCount++
      } catch {
        failCount++
      }
    }
    if (failCount === 0) {
      ElMessage.success(`成功删除 ${successCount} 个用户`)
    } else {
      ElMessage.warning(`删除完成：成功 ${successCount} 个，失败 ${failCount} 个`)
    }
    fetchData()
  } finally {
    batchDeleting.value = false
  }
}

const handleBatchAssignRoles = () => {
  if (selectedIds.value.length === 0) return
  batchSelectedRoles.value = []
  batchRoleDialogVisible.value = true
}

const handleSaveBatchRoles = async () => {
  if (batchSelectedRoles.value.length === 0) {
    ElMessage.warning('请至少选择一个角色')
    return
  }
  batchRoleSubmitting.value = true
  let successCount = 0
  let failCount = 0
  try {
    for (const id of selectedIds.value) {
      try {
        await assignRoles(id, batchSelectedRoles.value)
        successCount++
      } catch {
        failCount++
      }
    }
    if (failCount === 0) {
      ElMessage.success(`成功为 ${successCount} 个用户分配角色`)
    } else {
      ElMessage.warning(`分配完成：成功 ${successCount} 个，失败 ${failCount} 个`)
    }
    batchRoleDialogVisible.value = false
    fetchData()
  } finally {
    batchRoleSubmitting.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-info {
  display: flex;
  flex-direction: column;
}

.username {
  font-weight: 600;
  color: var(--color-text-primary);
}

.nickname {
  font-size: 12px;
  color: var(--color-text-muted);
}

.role-tag {
  margin-right: var(--space-1);
}

.role-checkbox-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.role-dialog-hint {
  margin-bottom: var(--space-4);
  color: var(--color-text-secondary);
  font-weight: 500;
}
</style>
