<template>
  <div class="users-page">
    <el-card shadow="never">
      <!-- 搜索栏 -->
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="关键词">
            <el-input
              v-model="searchForm.keyword"
              placeholder="请输入用户名或昵称"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
        <el-button type="success" @click="handleCreate">创建用户</el-button>
      </div>

      <!-- 用户列表 -->
      <el-table :data="users" v-loading="loading">
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
            <el-tag v-for="role in (row.roles || [])" :key="role" size="small" class="role-tag">
              {{ roleLabel(role) }}
            </el-tag>
            <span v-if="!row.roles || row.roles.length === 0" class="text-muted">未分配</span>
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
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="warning" @click="handleManageRoles(row)">管理角色</el-button>
            <el-popconfirm
              title="确定删除该用户吗？"
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
          @size-change="fetchUsers"
          @current-change="fetchUsers"
        />
      </div>
    </el-card>

    <!-- 创建/编辑用户对话框 -->
    <el-dialog
      v-model="userDialogVisible"
      :title="isEdit ? '编辑用户' : '创建用户'"
      width="500px"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userRules"
        label-width="80px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input v-model="userForm.username" :disabled="isEdit" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="密码" prop="password">
          <el-input v-model="userForm.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="userForm.nickname" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userForm.email" />
        </el-form-item>
        <el-form-item label="头像" prop="avatar">
          <el-input v-model="userForm.avatar" placeholder="请输入头像URL" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="userForm.phone" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 管理角色对话框 -->
    <el-dialog
      v-model="roleDialogVisible"
      title="管理角色"
      width="400px"
    >
      <p class="role-dialog-hint">为用户 <strong>{{ currentUser?.username }}</strong> 分配角色</p>
      <el-checkbox-group v-model="selectedRoles">
        <el-checkbox label="system_admin">系统管理员</el-checkbox>
        <el-checkbox label="system_publisher">系统发布者</el-checkbox>
        <el-checkbox label="system_moderator">系统审核员</el-checkbox>
        <el-checkbox label="system_operator">系统运营</el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSaveRoles">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { User } from '@/types'
import { getUsers, createUser, updateUser, deleteUser, assignRoles } from '@/api/users'

// 搜索和分页
const searchForm = reactive({ keyword: '' })
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const users = ref<User[]>([])
const loading = ref(false)

// 用户表单
const userDialogVisible = ref(false)
const isEdit = ref(false)
const userFormRef = ref<FormInstance>()
const submitting = ref(false)
const userForm = reactive({
  id: 0,
  username: '',
  password: '',
  nickname: '',
  email: '',
  avatar: '',
  phone: '',
})

const userRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
}

// 角色管理
const roleDialogVisible = ref(false)
const currentUser = ref<User | null>(null)
const selectedRoles = ref<string[]>([])

// 工具函数
const roleLabel = (role: string): string => {
  const map: Record<string, string> = {
    system_admin: '系统管理员',
    system_publisher: '系统发布者',
    system_moderator: '系统审核员',
    system_operator: '系统运营',
  }
  return map[role] || role
}

const statusLabel = (status: string): string => {
  const map: Record<string, string> = { active: '正常', inactive: '停用', banned: '封禁' }
  return map[status] || status
}

const statusType = (status: string): 'success' | 'info' | 'danger' => {
  const map: Record<string, 'success' | 'info' | 'danger'> = { active: 'success', inactive: 'info', banned: 'danger' }
  return map[status] || 'info'
}

// 获取用户列表
const fetchUsers = async () => {
  loading.value = true
  try {
    const { data } = await getUsers({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined,
    })
    users.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchUsers()
}

const handleReset = () => {
  searchForm.keyword = ''
  handleSearch()
}

// 创建用户
const handleCreate = () => {
  isEdit.value = false
  resetUserForm()
  userDialogVisible.value = true
}

// 编辑用户
const handleEdit = (row: User) => {
  isEdit.value = true
  userForm.id = row.id
  userForm.username = row.username
  userForm.password = ''
  userForm.nickname = row.nickname || ''
  userForm.email = row.email
  userForm.avatar = row.avatar || ''
  userForm.phone = row.phone || ''
  userDialogVisible.value = true
}

const resetUserForm = () => {
  userForm.id = 0
  userForm.username = ''
  userForm.password = ''
  userForm.nickname = ''
  userForm.email = ''
  userForm.avatar = ''
  userForm.phone = ''
}

const handleSubmit = async () => {
  if (!userFormRef.value) return
  await userFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateUser(userForm.id, {
          nickname: userForm.nickname,
          email: userForm.email,
          avatar: userForm.avatar,
          phone: userForm.phone,
        })
        ElMessage.success('更新成功')
      } else {
        await createUser({
          username: userForm.username,
          password: userForm.password,
          nickname: userForm.nickname,
          email: userForm.email,
          avatar: userForm.avatar,
          phone: userForm.phone,
        })
        ElMessage.success('创建成功')
      }
      userDialogVisible.value = false
      fetchUsers()
    } catch (error) {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除用户
const handleDelete = async (id: number) => {
  try {
    await deleteUser(id)
    ElMessage.success('删除成功')
    fetchUsers()
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

// 角色管理
const handleManageRoles = (row: User) => {
  currentUser.value = row
  selectedRoles.value = row.roles ? [...row.roles] : []
  roleDialogVisible.value = true
}

const handleSaveRoles = async () => {
  if (!currentUser.value) return
  submitting.value = true
  try {
    await assignRoles(currentUser.value.id, selectedRoles.value)
    ElMessage.success('角色更新成功')
    roleDialogVisible.value = false
    fetchUsers()
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    submitting.value = false
  }
}

onMounted(fetchUsers)
</script>

<style scoped>
.users-page {
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
  margin-right: 4px;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.role-dialog-hint {
  margin-bottom: var(--space-4);
  color: var(--color-text-secondary);
  font-weight: 500;
}
</style>
