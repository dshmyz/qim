<template>
  <div class="channels-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>频道管理</h3>
        <div class="header-actions">
          <el-input
            v-model="keyword"
            placeholder="搜索频道..."
            clearable
            style="width: 200px; margin-right: 12px"
            @clear="fetchChannels"
            @keyup.enter="fetchChannels"
          />
          <el-button type="primary" @click="handleCreate">创建频道</el-button>
        </div>
      </div>

      <el-table :data="channels" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="频道名称" min-width="180">
          <template #default="{ row }">
            <div class="channel-cell">
              <el-avatar :size="32" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
              <span class="channel-name">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
        <el-table-column label="订阅数" width="100">
          <template #default="{ row }">
            <el-tag effect="plain">{{ row.memberCount }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="creatorName" label="创建者" width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status === 'active' ? '正常' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="180" />
        <el-table-column label="操作" width="260" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="handlePublish(row)">发布消息</el-button>
            <el-button size="small" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm
              title="确定删除该频道吗？删除后订阅和消息将一并清除。"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button size="small" type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

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

    <el-dialog
      v-model="channelDialogVisible"
      :title="isEditing ? '编辑频道' : '创建频道'"
      width="500px"
    >
      <el-form
        ref="channelFormRef"
        :model="channelForm"
        :rules="channelRules"
        label-width="100px"
      >
        <el-form-item label="名称" prop="name">
          <el-input v-model="channelForm.name" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input v-model="channelForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="头像" prop="avatar">
          <el-input v-model="channelForm.avatar" placeholder="请输入头像URL" />
        </el-form-item>
        <el-form-item v-if="isEditing" label="状态" prop="status">
          <el-select v-model="channelForm.status" placeholder="请选择状态">
            <el-option label="正常" value="active" />
            <el-option label="停用" value="inactive" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="publishDialogVisible"
      :title="`发布消息 - ${publishChannelName}`"
      width="500px"
    >
      <el-form
        ref="publishFormRef"
        :model="publishForm"
        :rules="publishRules"
        label-width="80px"
      >
        <el-form-item label="内容" prop="content">
          <el-input
            v-model="publishForm.content"
            type="textarea"
            :rows="5"
            placeholder="请输入要发布的消息内容"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="publishDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="publishing" @click="handlePublishSubmit">发布</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { getChannels, createChannel, updateChannel, deleteChannel, createChannelMessage } from '@/api/channels'
import type { ChannelInfo } from '@/api/channels'

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })
const channels = ref<ChannelInfo[]>([])
const loading = ref(false)
const keyword = ref('')

const channelDialogVisible = ref(false)
const channelFormRef = ref<FormInstance>()
const submitting = ref(false)
const isEditing = ref(false)
const editingId = ref<number | null>(null)

const channelForm = reactive({
  name: '',
  description: '',
  avatar: '',
  status: 'active' as 'active' | 'inactive',
})

const channelRules: FormRules = {
  name: [{ required: true, message: '请输入频道名称', trigger: 'blur' }],
}

const publishDialogVisible = ref(false)
const publishFormRef = ref<FormInstance>()
const publishing = ref(false)
const publishChannelId = ref<number | null>(null)
const publishChannelName = ref('')
const publishForm = reactive({ content: '' })

const publishRules: FormRules = {
  content: [{ required: true, message: '请输入消息内容', trigger: 'blur' }],
}

const fetchChannels = async () => {
  loading.value = true
  try {
    const { data } = await getChannels({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: keyword.value || undefined,
    } as any)
    channels.value = data.data.list || []
    pagination.total = data.data.total || 0
  } catch (error) {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleCreate = () => {
  isEditing.value = false
  editingId.value = null
  channelForm.name = ''
  channelForm.description = ''
  channelForm.avatar = ''
  channelForm.status = 'active'
  channelDialogVisible.value = true
}

const handleEdit = (row: ChannelInfo) => {
  isEditing.value = true
  editingId.value = row.id
  channelForm.name = row.name
  channelForm.description = row.description
  channelForm.avatar = row.avatar
  channelForm.status = row.status as 'active' | 'inactive'
  channelDialogVisible.value = true
}

const handleSubmit = async () => {
  if (!channelFormRef.value) return
  await channelFormRef.value.validate(async (valid) => {
    if (!valid) return
    submitting.value = true
    try {
      if (isEditing.value && editingId.value) {
        await updateChannel(editingId.value, {
          name: channelForm.name,
          description: channelForm.description,
          avatar: channelForm.avatar,
          status: channelForm.status,
        })
        ElMessage.success('更新成功')
      } else {
        await createChannel({
          name: channelForm.name,
          description: channelForm.description,
          avatar: channelForm.avatar,
        })
        ElMessage.success('创建成功')
      }
      channelDialogVisible.value = false
      fetchChannels()
    } catch (error) {
      // 错误已在请求拦截器中处理
    } finally {
      submitting.value = false
    }
  })
}

const handleDelete = async (id: number) => {
  try {
    await deleteChannel(id)
    ElMessage.success('删除成功')
    fetchChannels()
  } catch (error) {
    // 错误已在请求拦截器中处理
  }
}

const handlePublish = (row: ChannelInfo) => {
  publishChannelId.value = row.id
  publishChannelName.value = row.name
  publishForm.content = ''
  publishDialogVisible.value = true
}

const handlePublishSubmit = async () => {
  if (!publishFormRef.value) return
  await publishFormRef.value.validate(async (valid) => {
    if (!valid || !publishChannelId.value) return
    publishing.value = true
    try {
      await createChannelMessage(publishChannelId.value, { content: publishForm.content })
      ElMessage.success('发布成功')
      publishDialogVisible.value = false
    } catch (error) {
      // 错误已在请求拦截器中处理
    } finally {
      publishing.value = false
    }
  })
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

.header-actions {
  display: flex;
  align-items: center;
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
