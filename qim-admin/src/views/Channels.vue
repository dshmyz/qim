<template>
  <div class="channels-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>频道管理</h3>
        <el-button type="primary" @click="handleCreate">创建频道</el-button>
      </div>

      <!-- 频道列表 -->
      <el-table :data="channels" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="频道名称" min-width="180">
          <template #default="{ row }">
            <div class="channel-cell">
              <el-avatar :size="32" :src="row.icon">{{ row.name.charAt(0) }}</el-avatar>
              <span class="channel-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="channelType(row.type).type">
              {{ channelType(row.type).label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="订阅数" width="100">
          <template #default="{ row }">
            <el-tag effect="plain">{{ row.memberCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '正常' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-popconfirm
              title="确定删除该频道吗？"
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
          @size-change="fetchChannels"
          @current-change="fetchChannels"
        />
      </div>
    </el-card>

    <!-- 创建频道对话框 -->
    <el-dialog
      v-model="channelDialogVisible"
      title="创建频道"
      width="500px"
    >
      <el-form
        ref="channelFormRef"
        :model="channelForm"
        :rules="channelRules"
        label-width="80px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="channelForm.name" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="channelForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="图标" prop="icon">
          <el-input v-model="channelForm.icon" placeholder="请输入图标URL" />
        </el-form-item>
        <el-form-item label="类型" prop="type">
          <el-select v-model="channelForm.type" placeholder="请选择频道类型">
            <el-option label="文本频道" value="text" />
            <el-option label="语音频道" value="voice" />
            <el-option label="视频频道" value="video" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import type { Channel } from '@/types'
import { getChannels, createChannel, deleteChannel } from '@/api/channels'

// 分页
const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const channels = ref<Channel[]>([])
const loading = ref(false)

// 频道表单
const channelDialogVisible = ref(false)
const channelFormRef = ref<FormInstance>()
const submitting = ref(false)
const channelForm = reactive({
  name: '',
  description: '',
  icon: '',
  type: 'text' as 'text' | 'voice' | 'video',
})

const channelRules: FormRules = {
  name: [{ required: true, message: '请输入频道名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择频道类型', trigger: 'change' }],
}

// 工具函数
const channelType = (type: string): { label: string; type: 'success' | 'warning' | 'info' } => {
  const map: Record<string, { label: string; type: 'success' | 'warning' | 'info' }> = {
    text: { label: '文本', type: 'success' },
    voice: { label: '语音', type: 'warning' },
    video: { label: '视频', type: 'info' },
  }
  return map[type] || { label: type, type: 'info' }
}

// 获取频道列表
const fetchChannels = async () => {
  loading.value = true
  try {
    const { data } = await getChannels({
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    channels.value = data.data.list
    pagination.total = data.data.total
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

// 创建频道
const handleCreate = () => {
  channelForm.name = ''
  channelForm.description = ''
  channelForm.icon = ''
  channelForm.type = 'text'
  channelDialogVisible.value = true
}

const handleSubmit = async () => {
  if (!channelFormRef.value) return
  await channelFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      await createChannel({
        name: channelForm.name,
        description: channelForm.description,
        icon: channelForm.icon,
        type: channelForm.type,
      })
      ElMessage.success('创建成功')
      channelDialogVisible.value = false
      fetchChannels()
    } catch (error) {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

// 删除频道
const handleDelete = async (id: number) => {
  try {
    await deleteChannel(id)
    ElMessage.success('删除成功')
    fetchChannels()
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

onMounted(fetchChannels)
</script>

<style scoped>
.channels-page {
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

.channel-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.channel-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
