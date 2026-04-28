<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="handlePageChange" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="keyword" label="关键词" placeholder="用户名或昵称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-permission="'user:create'" type="primary" @click="handleCreate">创建用户</el-button>
    </template>

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
    <el-table-column label="操作" width="260" fixed="right">
      <template #default="{ row }">
        <ActionButton v-permission="'user:update'" @click="handleEdit(row)">编辑</ActionButton>
        <ActionButton v-permission="'user:update'" @click="handleManageRoles(row)">管理角色</ActionButton>
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
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { FormRules } from 'element-plus'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import StatusTag from '@/components/data/StatusTag.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import EntityDialog from '@/components/forms/EntityDialog.vue'
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
  { name: 'avatar', label: '头像', type: 'input', props: { placeholder: '请输入头像URL' } },
  { name: 'phone', label: '手机号', type: 'input' },
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
