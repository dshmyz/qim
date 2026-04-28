<!-- src/views/RoleManagement/index.vue -->
<template>
  <DataTable :data="list" :loading="loading" :pagination="pagination"
    @search="handleSearch" @page-change="fetchData" @refresh="fetchData">
    <template #search>
      <SearchForm @search="handleSearch" @reset="handleReset">
        <SearchField v-model="keyword" label="角色名称" placeholder="请输入角色名称" />
      </SearchForm>
    </template>
    <template #actions>
      <el-button v-permission="'role:create'" type="primary" :icon="Plus" @click="handleCreate">创建角色</el-button>
    </template>

    <el-table-column prop="id" label="ID" width="80" />
    <el-table-column prop="name" label="角色名称" min-width="150" />
    <el-table-column prop="code" label="角色代码" min-width="150" />
    <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
    <el-table-column label="权限" min-width="240">
      <template #default="{ row }">
        <div class="permissions-cell">
          <el-tag
            v-for="perm in (row.permissions || []).slice(0, 3)"
            :key="perm"
            size="small"
          >
            {{ permissionLabel(perm) }}
          </el-tag>
          <el-tag v-if="(row.permissions || []).length > 3" size="small" type="info">
            +{{ (row.permissions || []).length - 3 }}
          </el-tag>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="用户数" width="100">
      <template #default="{ row }">
        {{ row.userCount || 0 }}
      </template>
    </el-table-column>
    <el-table-column prop="createdAt" label="创建时间" width="180" />
    <el-table-column label="操作" width="200" fixed="right">
      <template #default="{ row }">
        <ActionButton v-permission="'role:update'" @click="handleEdit(row)">编辑</ActionButton>
        <el-popconfirm title="确定删除该角色吗？" @confirm="handleDelete(row.id)">
          <template #reference>
            <ActionButton v-permission="'role:delete'" type="danger">删除</ActionButton>
          </template>
        </el-popconfirm>
      </template>
    </el-table-column>
  </DataTable>

  <EntityDialog
    v-model="dialogVisible"
    :mode="dialogMode"
    :fields="entityFields"
    :rules="roleRules"
    :initial-data="formData"
    :create-title="'创建角色'"
    :edit-title="'编辑角色'"
    @save="handleSave"
  />
</template>

<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import DataTable from '@/components/data/DataTable.vue'
import SearchForm from '@/components/data/SearchForm.vue'
import SearchField from '@/components/data/SearchField.vue'
import EntityDialog from '@/components/forms/EntityDialog.vue'
import ActionButton from '@/components/common/ActionButton.vue'
import { useEntity } from '@/composables/useEntity'
import { getRoles, createRole, updateRole, deleteRole } from '@/api/roles'
import type { Role } from '@/types'
import { roleFields, roleRules, permissionLabelMap } from './config'

const roleApi = {
  getList: (params: Record<string, unknown>) => getRoles(params as any),
  create: (data: Record<string, unknown>) => createRole(data as any).then(() => {}),
  update: (id: number, data: Record<string, unknown>) => updateRole(id, data as any).then(() => {}),
  delete: (id: number) => deleteRole(id).then(() => {}),
}

const {
  list, loading, pagination, searchForm,
  dialogVisible, dialogMode, formData,
  handleSearch, handleReset, handleCreate, handleEdit, handleDelete, handleSave,
  fetchData
} = useEntity<Role>({
  api: roleApi,
  searchFields: ['keyword'],
  formFields: roleFields,
  successMessages: { create: '创建成功', update: '更新成功', delete: '删除成功' }
})

const keyword = computed({
  get: () => (searchForm.keyword as string) || '',
  set: (val: string) => { searchForm.keyword = val }
})

const entityFields = roleFields as any

const permissionLabel = (perm: string): string => {
  const label = permissionLabelMap[perm]
  if (!label && import.meta.env.DEV) {
    console.warn(`[RoleManagement] Unknown permission: ${perm}`)
  }
  return label || perm
}

onMounted(fetchData)
</script>

<style scoped>
.permissions-cell {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
</style>
