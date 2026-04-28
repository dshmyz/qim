<template>
  <div class="ai-assistant-page">
    <el-card shadow="never">
      <!-- 操作栏 -->
      <div class="action-bar">
        <el-button type="primary" @click="handleCreate">创建 AI 助手</el-button>
      </div>

      <!-- AI 助手列表 -->
      <el-table :data="aiBots" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="名称" min-width="160">
          <template #default="{ row }">
            <div class="bot-cell">
              <el-avatar :size="32" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
              <span class="bot-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="对话数" width="100">
          <template #default="{ row }">
            <el-tag type="info" size="small">{{ row.conversationCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-switch
              v-model="row.status"
              active-value="active"
              inactive-value="inactive"
              @change="handleToggleStatus(row)"
            />
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-button size="small" type="primary" @click="handleViewPrompt(row)">提示词</el-button>
            <el-popconfirm title="确定删除该 AI 助手吗？" @confirm="handleDelete(row.id)">
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
          @size-change="fetchAIBots"
          @current-change="fetchAIBots"
        />
      </div>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑 AI 助手' : '创建 AI 助手'"
      width="600px"
    >
      <el-form
        ref="botFormRef"
        :model="botForm"
        :rules="botRules"
        label-width="100px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="botForm.name" placeholder="请输入 AI 助手名称" />
        </el-form-item>
        <el-form-item label="头像URL" prop="avatar">
          <el-input v-model="botForm.avatar" placeholder="请输入头像URL" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="botForm.description" type="textarea" placeholder="请输入描述" />
        </el-form-item>
        <el-form-item label="系统提示词" prop="systemPrompt">
          <el-input
            v-model="botForm.systemPrompt"
            type="textarea"
            :rows="8"
            placeholder="请输入系统提示词，用于定义 AI 助手的行为和回答风格"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 查看提示词对话框 -->
    <el-dialog
      v-model="promptDialogVisible"
      :title="`系统提示词 - ${currentBot?.name || ''}`"
      width="600px"
    >
      <div class="prompt-content">
        <pre>{{ currentBot?.systemPrompt }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { AIBot } from '@/types'
import { getAIBots, createAIBot, updateAIBot, deleteAIBot, toggleAIBotStatus } from '@/api/aiBots'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const aiBots = ref<AIBot[]>([])
const loading = ref(false)

// 表单
const dialogVisible = ref(false)
const isEdit = ref(false)
const botFormRef = ref<FormInstance>()
const submitting = ref(false)
const botForm = reactive({
  id: 0,
  name: '',
  avatar: '',
  description: '',
  systemPrompt: '',
})

const botRules: FormRules = {
  name: [{ required: true, message: '请输入 AI 助手名称', trigger: 'blur' }],
  systemPrompt: [{ required: true, message: '请输入系统提示词', trigger: 'blur' }],
}

// 提示词对话框
const promptDialogVisible = ref(false)
const currentBot = ref<AIBot | null>(null)

// 获取列表
const fetchAIBots = async () => {
  loading.value = true
  try {
    const { data } = await getAIBots({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    aiBots.value = data.data.list
    pagination.total = data.data.total
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 创建
const handleCreate = () => {
  isEdit.value = false
  resetBotForm()
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: AIBot) => {
  isEdit.value = true
  botForm.id = row.id
  botForm.name = row.name
  botForm.avatar = row.avatar || ''
  botForm.description = row.description
  botForm.systemPrompt = row.systemPrompt
  dialogVisible.value = true
}

const resetBotForm = () => {
  botForm.id = 0
  botForm.name = ''
  botForm.avatar = ''
  botForm.description = ''
  botForm.systemPrompt = ''
}

// 提交
const handleSubmit = async () => {
  if (!botFormRef.value) return
  await botFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEdit.value) {
        await updateAIBot(botForm.id, {
          name: botForm.name,
          description: botForm.description,
          systemPrompt: botForm.systemPrompt,
          avatar: botForm.avatar,
        })
        ElMessage.success('更新成功')
      } else {
        await createAIBot({
          name: botForm.name,
          description: botForm.description,
          systemPrompt: botForm.systemPrompt,
          avatar: botForm.avatar,
        })
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      fetchAIBots()
    } catch {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除
const handleDelete = async (id: number) => {
  try {
    await deleteAIBot(id)
    ElMessage.success('删除成功')
    fetchAIBots()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

// 切换状态
const handleToggleStatus = async (row: AIBot) => {
  try {
    await toggleAIBotStatus(row.id, row.status)
    ElMessage.success('状态更新成功')
  } catch {
    // 错误已在请求拦截器中处理
    row.status = row.status === 'active' ? 'inactive' : 'active'
  }
}

// 查看提示词
const handleViewPrompt = (row: AIBot) => {
  currentBot.value = row
  promptDialogVisible.value = true
}

onMounted(fetchAIBots)
</script>

<style scoped>
.ai-assistant-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.action-bar {
  display: flex;
  justify-content: flex-end;
  padding-bottom: var(--space-4);
}

.bot-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.bot-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.prompt-content {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  max-height: 400px;
  overflow-y: auto;
}

.prompt-content pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text-primary);
}
</style>
