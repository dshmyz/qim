<template>
  <div class="roles-page">
    <el-card shadow="never">
      <!-- 操作栏 -->
      <div class="action-bar">
        <el-button type="primary" @click="handleCreate">创建角色</el-button>
      </div>

      <!-- 角色列表 -->
      <el-table :data="roles" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="角色名称" min-width="140" />
        <el-table-column prop="code" label="角色编码" min-width="160" />
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="权限" min-width="240">
          <template #default="{ row }">
            <el-tag
              v-for="perm in row.permissions"
              :key="perm"
              size="small"
              class="perm-tag"
            >
              {{ permissionLabel(perm) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="用户数" width="100">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.userCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="primary" @click="handleViewUsers(row)">查看用户</el-button>
            <el-popconfirm title="确定删除该角色吗？" @confirm="handleDelete(row.id)">
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
          @size-change="fetchRoles"
          @current-change="fetchRoles"
        />
      </div>
    </el-card>

    <!-- 创建/编辑角色对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑角色' : '创建角色'"
      width="600px"
    >
      <el-form
        ref="roleFormRef"
        :model="roleForm"
        :rules="roleRules"
        label-width="80px"
      >
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="roleForm.name" placeholder="请输入角色名称" />
        </el-form-item>
        <el-form-item label="角色编码" prop="code">
          <el-input v-model="roleForm.code" :disabled="isEdit" placeholder="例如：system_admin" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="roleForm.description" type="textarea" placeholder="请输入角色描述" />
        </el-form-item>
        <el-form-item label="权限">
          <div class="permission-grid">
            <el-checkbox
              v-for="perm in availablePermissions"
              :key="perm.value"
              v-model="roleForm.permissions"
              :label="perm.value"
            >
              {{ perm.label }}
            </el-checkbox>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 查看角色用户对话框 -->
    <el-dialog
      v-model="userDialogVisible"
      :title="`拥有该角色的用户 - ${currentRole?.name || ''}`"
      width="500px"
    >
      <el-table :data="roleUsers" v-loading="usersLoading" size="small">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="用户名" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="28">{{ (row.nickname || row.username).charAt(0) }}</el-avatar>
              <span>{{ row.nickname || row.username }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { Role } from '@/types'
import { getRoles, createRole, updateRole, deleteRole, getRoleUsers } from '@/api/roles'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const roles = ref<Role[]>([])
const loading = ref(false)

// 表单
const dialogVisible = ref(false)
const isEdit = ref(false)
const roleFormRef = ref<FormInstance>()
const submitting = ref(false)
const roleForm = reactive({
  id: 0,
  name: '',
  code: '',
  description: '',
  permissions: [] as string[],
})

const roleRules: FormRules = {
  name: [{ required: true, message: '请输入角色名称', trigger: 'blur' }],
  code: [
    { required: true, message: '请输入角色编码', trigger: 'blur' },
    { pattern: /^[a-z_]+$/, message: '角色编码只能包含小写字母和下划线', trigger: 'blur' },
  ],
}

// 用户对话框
const userDialogVisible = ref(false)
const currentRole = ref<Role | null>(null)
const roleUsers = ref<{ id: number; username: string; nickname?: string; avatar?: string }[]>([])
const usersLoading = ref(false)

// 权限列表
const availablePermissions = [
  { value: 'user:read', label: '查看用户' },
  { value: 'user:write', label: '编辑用户' },
  { value: 'user:delete', label: '删除用户' },
  { value: 'group:read', label: '查看群组' },
  { value: 'group:write', label: '编辑群组' },
  { value: 'group:delete', label: '删除群组' },
  { value: 'message:read', label: '查看消息' },
  { value: 'message:write', label: '发送消息' },
  { value: 'message:delete', label: '删除消息' },
  { value: 'system:config', label: '系统配置' },
  { value: 'system:log', label: '查看日志' },
  { value: 'role:manage', label: '角色管理' },
]

// 权限标签映射
const permissionLabel = (perm: string): string => {
  const found = availablePermissions.find(p => p.value === perm)
  return found ? found.label : perm
}

// 获取角色列表
const fetchRoles = async () => {
  loading.value = true
  try {
    const { data } = await getRoles({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    roles.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 创建角色
const handleCreate = () => {
  isEdit.value = false
  resetRoleForm()
  dialogVisible.value = true
}

// 编辑角色
const handleEdit = (row: Role) => {
  isEdit.value = true
  roleForm.id = row.id
  roleForm.name = row.name
  roleForm.code = row.code
  roleForm.description = row.description
  roleForm.permissions = [...row.permissions]
  dialogVisible.value = true
}

const resetRoleForm = () => {
  roleForm.id = 0
  roleForm.name = ''
  roleForm.code = ''
  roleForm.description = ''
  roleForm.permissions = []
}

// 提交表单
const handleSubmit = async () => {
  if (!roleFormRef.value) return
  await roleFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateRole(roleForm.id, {
          name: roleForm.name,
          description: roleForm.description,
          permissions: roleForm.permissions,
        })
        ElMessage.success('更新成功')
      } else {
        await createRole({
          name: roleForm.name,
          code: roleForm.code,
          description: roleForm.description,
          permissions: roleForm.permissions,
        })
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      fetchRoles()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除角色
const handleDelete = async (id: number) => {
  try {
    await deleteRole(id)
    ElMessage.success('删除成功')
    fetchRoles()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 查看角色用户
const handleViewUsers = async (row: Role) => {
  currentRole.value = row
  userDialogVisible.value = true
  usersLoading.value = true
  try {
    const { data } = await getRoleUsers(row.id, { page: 1, pageSize: 50 })
    roleUsers.value = data.data.list
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    usersLoading.value = false
  }
}

onMounted(fetchRoles)
</script>

<style scoped>
.roles-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.action-bar {
  display: flex;
  justify-content: flex-end;
  padding-bottom: var(--space-4);
}

.perm-tag {
  margin-right: 4px;
  margin-bottom: 4px;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.permission-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px 16px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
