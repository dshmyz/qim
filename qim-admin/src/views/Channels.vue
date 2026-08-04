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
        <el-form-item label="发布权限" prop="publish_permission">
          <el-select v-model="channelForm.publish_permission" placeholder="请选择发布权限">
            <el-option label="仅创建者可发布" value="creator_only" />
            <el-option label="所有订阅者可发布" value="all_subscribers" />
          </el-select>
        </el-form-item>
        <el-form-item label="评论权限" prop="comment_permission">
          <el-select v-model="channelForm.comment_permission" placeholder="请选择评论权限">
            <el-option label="所有订阅者可评论" value="all_subscribers" />
            <el-option label="关闭评论" value="disabled" />
          </el-select>
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
          <el-radio-group v-model="publishMode" size="small" class="publish-mode-toggle">
            <el-radio-button value="edit">编辑</el-radio-button>
            <el-radio-button value="preview">预览</el-radio-button>
          </el-radio-group>
          <el-input
            v-if="publishMode === 'edit'"
            v-model="publishForm.content"
            type="textarea"
            :rows="8"
            placeholder="支持 Markdown 语法（# 标题、**粗体**、[链接](url)、代码块等）"
          />
          <div
            v-else
            class="md-preview"
            v-html="previewContent"
          ></div>
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
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import { sanitizeMarkdown } from '@/utils/sanitize'
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
  publish_permission: 'creator_only' as 'creator_only' | 'all_subscribers',
  comment_permission: 'all_subscribers' as 'all_subscribers' | 'disabled',
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
const publishMode = ref<'edit' | 'preview'>('edit')

// 发布内容 Markdown 预览（复用 Docs.vue 同款 marked + sanitizeMarkdown 链路）
const previewContent = computed(() => {
  if (!publishForm.content) return '<p class="md-preview-empty">暂无内容</p>'
  const html = marked.parse(publishForm.content, { async: false }) as string
  return sanitizeMarkdown(html)
})

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
  channelForm.publish_permission = 'creator_only'
  channelForm.comment_permission = 'all_subscribers'
  channelDialogVisible.value = true
}

const handleEdit = (row: ChannelInfo) => {
  isEditing.value = true
  editingId.value = row.id
  channelForm.name = row.name
  channelForm.description = row.description
  channelForm.avatar = row.avatar
  channelForm.status = row.status as 'active' | 'inactive'
  channelForm.publish_permission = row.publish_permission
  channelForm.comment_permission = row.comment_permission
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
          publish_permission: channelForm.publish_permission,
          comment_permission: channelForm.comment_permission,
        })
        ElMessage.success('更新成功')
      } else {
        await createChannel({
          name: channelForm.name,
          description: channelForm.description,
          avatar: channelForm.avatar,
          publish_permission: channelForm.publish_permission,
          comment_permission: channelForm.comment_permission,
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
  publishMode.value = 'edit'
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

.publish-mode-toggle {
  margin-bottom: 8px;
}

.md-preview {
  width: 100%;
  min-height: 180px;
  max-height: 360px;
  overflow-y: auto;
  padding: 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background: var(--el-fill-color-light);
  font-size: 14px;
  line-height: 1.7;
  color: var(--el-text-color-primary);
  word-break: break-word;
}

.md-preview-empty {
  color: var(--el-text-color-placeholder);
  margin: 0;
}

.md-preview :deep(h1),
.md-preview :deep(h2),
.md-preview :deep(h3),
.md-preview :deep(h4) {
  font-weight: 600;
  margin: 0.8em 0 0.4em;
  line-height: 1.3;
}
.md-preview :deep(h1) { font-size: 1.5em; }
.md-preview :deep(h2) { font-size: 1.3em; }
.md-preview :deep(h3) { font-size: 1.15em; }
.md-preview :deep(p) { margin: 0.5em 0; }
.md-preview :deep(strong) { font-weight: 600; }
.md-preview :deep(em) { font-style: italic; }
.md-preview :deep(a) { color: var(--el-color-primary); text-decoration: none; }
.md-preview :deep(a:hover) { text-decoration: underline; }
.md-preview :deep(ul),
.md-preview :deep(ol) { padding-left: 1.6em; margin: 0.5em 0; }
.md-preview :deep(li) { margin: 0.25em 0; }
.md-preview :deep(blockquote) {
  margin: 0.5em 0;
  padding: 0.25em 1em;
  border-left: 3px solid var(--el-color-primary);
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 0 4px 4px 0;
}
.md-preview :deep(pre) {
  background: var(--el-fill-color-darker, #f0f2f5);
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
  margin: 0.5em 0;
}
.md-preview :deep(code) {
  background: var(--el-fill-color-darker, #f0f2f5);
  padding: 2px 5px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.92em;
}
.md-preview :deep(pre code) { background: transparent; padding: 0; border-radius: 0; }
.md-preview :deep(hr) { border: none; border-top: 1px solid var(--el-border-color); margin: 1em 0; }
.md-preview :deep(img) { max-width: 100%; border-radius: 6px; }
.md-preview :deep(table) { border-collapse: collapse; margin: 0.5em 0; }
.md-preview :deep(th),
.md-preview :deep(td) { border: 1px solid var(--el-border-color); padding: 6px 12px; text-align: left; }
.md-preview :deep(th) { background: var(--el-fill-color-light); font-weight: 600; }
</style>
