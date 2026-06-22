<template>
  <div class="blacklist-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>黑名单管理</h3>
        <el-button type="danger" @click="handleOpenAdd">
          <el-icon><Plus /></el-icon>
          添加到黑名单
        </el-button>
      </div>

      <!-- 黑名单列表 -->
      <el-table :data="blacklist" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="userId" label="用户ID" width="100" />
        <el-table-column prop="username" label="用户名" min-width="150" />
        <el-table-column prop="reason" label="封禁原因" min-width="200" show-overflow-tooltip />
        <el-table-column prop="operatorId" label="操作人ID" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'danger' : 'info'">
              {{ row.status === 'active' ? '封禁中' : '已移除' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="封禁时间" width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'active'"
              size="small"
              type="success"
              @click="handleRemove(row.id)"
            >
              移出黑名单
            </el-button>
            <span v-else class="text-muted">已移出</span>
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
          @size-change="fetchBlacklist"
          @current-change="fetchBlacklist"
        />
      </div>
    </el-card>

    <!-- 添加到黑名单对话框 -->
    <el-dialog v-model="addDialogVisible" title="添加到黑名单" width="480px" :close-on-click-modal="false">
      <el-form ref="addFormRef" :model="addForm" :rules="addRules" label-width="90px">
        <el-form-item label="用户" prop="userId">
          <el-select
            v-model="addForm.userId"
            filterable
            remote
            reserve-keyword
            placeholder="搜索用户名/昵称"
            :remote-method="searchUsers"
            :loading="userSearchLoading"
            style="width: 100%"
          >
            <el-option
              v-for="u in userOptions"
              :key="u.id"
              :label="`${u.nickname || u.username} (ID: ${u.id})`"
              :value="u.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="封禁原因" prop="reason">
          <el-input
            v-model="addForm.reason"
            type="textarea"
            :rows="3"
            placeholder="请输入封禁原因（可选）"
            maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="addSubmitting" @click="handleAddSubmit">确定封禁</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import type { BlacklistEntry, User } from '@/types'
import { getBlacklist, removeBlacklistEntry, addToBlacklist } from '@/api/blacklist'
import { getUsers } from '@/api/users'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const blacklist = ref<BlacklistEntry[]>([])
const loading = ref(false)

// 获取黑名单列表
const fetchBlacklist = async () => {
  loading.value = true
  try {
    const { data } = await getBlacklist({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    blacklist.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 移出黑名单
const handleRemove = async (id: number) => {
  try {
    await ElMessageBox.confirm('确定将该用户移出黑名单吗？', '提示', { type: 'warning' })
    await removeBlacklistEntry(id)
    ElMessage.success('移出成功')
    fetchBlacklist()
  } catch (error) {
    if (error !== 'cancel') {
      // 错误已在请求拦截器中处理
    }
  }
}

// 添加到黑名单
const addDialogVisible = ref(false)
const addSubmitting = ref(false)
const addFormRef = ref<FormInstance>()
const userOptions = ref<User[]>([])
const userSearchLoading = ref(false)
const addForm = reactive({
  userId: undefined as number | undefined,
  reason: '',
})
const addRules: FormRules = {
  userId: [{ required: true, message: '请选择用户', trigger: 'change' }],
}

const searchUsers = async (query: string) => {
  if (!query) {
    userOptions.value = []
    return
  }
  userSearchLoading.value = true
  try {
    const { data } = await getUsers({ keyword: query, page: 1, pageSize: 20 })
    userOptions.value = data.data.list ?? []
  } catch {
    userOptions.value = []
  } finally {
    userSearchLoading.value = false
  }
}

const handleOpenAdd = () => {
  addForm.userId = undefined
  addForm.reason = ''
  userOptions.value = []
  addDialogVisible.value = true
}

const handleAddSubmit = async () => {
  if (!addFormRef.value) return
  await addFormRef.value.validate(async (valid) => {
    if (!valid) return
    addSubmitting.value = true
    try {
      await addToBlacklist({ userId: addForm.userId!, reason: addForm.reason })
      ElMessage.success('已添加到黑名单')
      addDialogVisible.value = false
      fetchBlacklist()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      addSubmitting.value = false
    }
  })
}

onMounted(fetchBlacklist)
</script>

<style scoped>
.blacklist-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-4);
  border-bottom: 2px solid var(--color-border-light);
}

.page-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text-primary);
  letter-spacing: -0.02em;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
