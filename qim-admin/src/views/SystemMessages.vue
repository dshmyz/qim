<template>
  <div class="messages-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>系统消息管理</h3>
        <div class="page-actions">
          <el-button type="primary" @click="handleCreate">创建消息</el-button>
        </div>
      </div>

      <!-- 消息列表 -->
      <el-table :data="messages" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="title" label="标题" min-width="200" />
        <el-table-column prop="content" label="内容" min-width="250" show-overflow-tooltip />
        <el-table-column label="目标类型" width="120">
          <template #default="{ row }">
            <el-tag>{{ targetTypeLabel(row.target_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '已发布' : '草稿' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定删除该消息吗？"
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
          @size-change="fetchMessages"
          @current-change="fetchMessages"
        />
      </div>
    </el-card>

    <!-- 创建/编辑消息对话框 -->
    <el-dialog
      v-model="messageDialogVisible"
      :title="isEdit ? '编辑消息' : '创建消息'"
      width="600px"
    >
      <el-form
        ref="messageFormRef"
        :model="messageForm"
        :rules="messageRules"
        label-width="80px"
      >
        <el-form-item label="标题" prop="title">
          <el-input v-model="messageForm.title" placeholder="请输入消息标题" />
        </el-form-item>
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="messageForm.content"
            type="textarea"
            :rows="5"
            placeholder="请输入消息内容"
          />
        </el-form-item>
        <el-form-item label="发送范围" prop="target_type">
          <el-select v-model="messageForm.target_type" placeholder="请选择发送范围" @change="handleTargetTypeChange">
            <el-option label="全员" value="all" />
            <el-option label="指定部门" value="department" />
            <el-option label="指定用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="messageForm.target_type !== 'all'" label="目标ID" prop="target_id">
          <el-input-number v-model="messageForm.target_id" :min="1" placeholder="请输入目标ID" />
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态">
          <el-select v-model="messageForm.status" placeholder="请选择状态">
            <el-option label="草稿" value="draft" />
            <el-option label="已发布" value="active" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="messageDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SystemMessage } from '@/types'
import { getSystemMessages, createSystemMessage, updateSystemMessage, deleteSystemMessage } from '@/api/systemMessages'

// 搜索和分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const messages = ref<SystemMessage[]>([])
const loading = ref(false)

// 消息表单
const messageDialogVisible = ref(false)
const isEdit = ref(false)
const messageFormRef = ref<FormInstance>()
const submitting = ref(false)
const messageForm = reactive({
  id: 0,
  title: '',
  content: '',
  target_type: 'all' as string,
  target_id: undefined as number | undefined,
  status: 'draft' as 'published' | 'draft' | 'active',
})

const messageRules: FormRules = {
  title: [{ required: true, message: '请输入消息标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入消息内容', trigger: 'blur' }],
  target_type: [{ required: true, message: '请选择发送范围', trigger: 'change' }],
}

// 工具函数
const targetTypeLabel = (type: string): string => {
  const map: Record<string, string> = {
    all: '全员',
    department: '指定部门',
    user: '指定用户',
  }
  return map[type] || type
}

const handleTargetTypeChange = () => {
  messageForm.target_id = undefined
}

// 获取消息列表
const fetchMessages = async () => {
  loading.value = true
  try {
    const { data } = await getSystemMessages({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    messages.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 创建消息
const handleCreate = () => {
  console.log('handleCreate 被调用')
  isEdit.value = false
  resetMessageForm()
  messageDialogVisible.value = true
  console.log('messageDialogVisible:', messageDialogVisible.value)
}

// 编辑消息
const handleEdit = (row: SystemMessage) => {
  isEdit.value = true
  messageForm.id = row.id
  messageForm.title = row.title
  messageForm.content = row.content
  messageForm.target_type = row.target_type || 'all'
  messageForm.target_id = row.target_id
  messageForm.status = row.status === 'active' ? 'published' : 'draft'
  messageDialogVisible.value = true
}

const resetMessageForm = () => {
  messageForm.id = 0
  messageForm.title = ''
  messageForm.content = ''
  messageForm.target_type = 'all'
  messageForm.target_id = undefined
  messageForm.status = 'draft'
}

const handleSubmit = async () => {
  if (!messageFormRef.value) return
  await messageFormRef.value.validate(async (valid) => {
    if (!valid) {
      ElMessage.warning('请检查表单填写是否完整')
      return
    }
    console.log('提交系统消息:', JSON.parse(JSON.stringify(messageForm)))
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateSystemMessage(messageForm.id, {
          title: messageForm.title,
          content: messageForm.content,
          status: messageForm.status === 'published' ? 'published' : 'draft',
        })
        ElMessage.success('更新成功')
      } else {
        await createSystemMessage({
          title: messageForm.title,
          content: messageForm.content,
          target_type: messageForm.target_type,
          target_id: messageForm.target_id,
        })
        ElMessage.success('创建成功')
      }
      messageDialogVisible.value = false
      fetchMessages()
    } catch (error: any) {
      console.error('提交系统消息失败:', error)
      ElMessage.error(error.response?.data?.message || '创建失败')
    } finally {
      submitting.value = false
    }
  })
}

// 删除消息
const handleDelete = async (id: number) => {
  try {
    await deleteSystemMessage(id)
    ElMessage.success('删除成功')
    fetchMessages()
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchMessages)
</script>

<style scoped>
.messages-page {
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

.page-actions {
  display: flex;
  gap: 8px;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
