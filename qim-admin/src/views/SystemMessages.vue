<template>
  <div class="messages-page">
    <el-card shadow="never">
      <div class="page-header">
        <h3>系统消息管理</h3>
        <div class="page-actions">
          <el-button type="primary" @click="handleCreate">创建消息</el-button>
          <el-button @click="openChatDialog">群发私聊</el-button>
        </div>
      </div>

      <!-- 消息列表 -->
      <el-table :data="messages" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="title" label="标题" min-width="200" />
        <el-table-column prop="content" label="内容" min-width="250" show-overflow-tooltip />
        <el-table-column label="目标范围" min-width="160">
          <template #default="{ row }">
            <div class="target-info">
              <el-tag size="small" :type="row.target_type === 'all' ? '' : 'warning'" effect="plain">
                {{ targetTypeLabel(row.target_type) }}
              </el-tag>
              <span v-if="row.target_type === 'version' && row.target_version" class="target-detail">
                版本: {{ row.target_version }}{{ row.target_platform ? ` · ${clientPlatformLabel(row.target_platform)}` : '' }}
              </span>
              <span v-if="row.target_type !== 'all' && row.target_id" class="target-detail">
                ID: {{ row.target_id }}
              </span>
              <span v-if="row.target_type !== 'all' && !row.target_id && row.target_type !== 'version'" class="target-detail text-muted">
                未指定
              </span>
            </div>
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
      width="640px"
      :close-on-click-modal="false"
      class="system-message-dialog"
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
            <el-option label="指定版本(在线)" value="version" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="messageForm.target_type === 'user'" label="目标用户" prop="target_ids">
          <div class="multi-select-wrapper">
            <el-select
              v-model="messageForm.target_ids"
              multiple
              filterable
              remote
              collapse-tags
              collapse-tags-tooltip
              reserve-keyword
              placeholder="输入关键词搜索选择用户"
              :remote-method="searchUsers"
              :loading="searchUserLoading"
              style="width: 100%"
            >
              <el-option
                v-for="user in searchUserOptions"
                :key="user.id"
                :label="`${user.nickname || user.username}`"
                :value="user.id"
              >
                <div class="user-option-item">
                  <span class="user-option-name">{{ user.nickname || user.username }}</span>
                  <span class="user-option-meta">ID: {{ user.id }} · {{ user.email || '' }}</span>
                </div>
              </el-option>
            </el-select>
            <div v-if="messageForm.target_ids.length > 0" class="selected-summary">
              已选 <strong>{{ messageForm.target_ids.length }}</strong> 个用户
            </div>
          </div>
        </el-form-item>
        <el-form-item v-if="messageForm.target_type === 'department'" label="目标部门" prop="target_ids">
          <div class="multi-select-wrapper">
            <el-tree-select
              v-model="messageForm.target_ids"
              :data="departmentTree"
              :props="{ label: 'name', value: 'id', children: 'subDepartments' }"
              placeholder="请选择部门（可多选）"
              multiple
              filterable
              clearable
              collapse-tags
              collapse-tags-tooltip
              check-strictly
              style="width: 100%"
              render-after-expand
            />
            <div v-if="messageForm.target_ids.length > 0" class="selected-summary">
              已选 <strong>{{ messageForm.target_ids.length }}</strong> 个部门
            </div>
          </div>
        </el-form-item>
        <el-form-item v-if="messageForm.target_type === 'version'" label="目标版本" prop="target_version">
          <div class="multi-select-wrapper">
            <el-select v-model="messageForm.target_version" placeholder="请选择客户端版本" filterable style="width: 100%">
              <el-option
                v-for="item in versionOptions"
                :key="item.version"
                :label="`${item.version}（${item.count} 台在线）`"
                :value="item.version"
              />
            </el-select>
            <el-select v-model="messageForm.target_platform" placeholder="全部平台" clearable style="width: 100%; margin-top: var(--space-2)">
              <el-option label="全部平台" value="" />
              <el-option v-for="p in CLIENT_PLATFORM_OPTIONS" :key="p.value" :label="p.label" :value="p.value" />
            </el-select>
            <div class="selected-summary">仅推送给所选版本的<strong>在线</strong>用户；未上报版本的老客户端归入「未知版本」</div>
          </div>
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

    <!-- 群发私聊对话框：以系统账号向用户单聊会话发普通私聊消息（进最近会话），
         与上方的"系统消息/通知"不同 -->
    <el-dialog
      v-model="chatDialogVisible"
      title="群发私聊"
      width="640px"
      :close-on-click-modal="false"
      class="system-message-dialog"
    >
      <el-form
        ref="chatFormRef"
        :model="chatForm"
        :rules="chatRules"
        label-width="80px"
      >
        <el-form-item label="消息内容" prop="content">
          <el-input
            v-model="chatForm.content"
            type="textarea"
            :rows="5"
            placeholder="例如：新版本 v2.0.24 已发布，请关闭后重新打开以触发更新"
          />
        </el-form-item>
        <el-form-item label="发送范围" prop="scope">
          <el-select v-model="chatForm.scope" placeholder="请选择发送范围">
            <el-option label="全部用户" value="all" />
            <el-option label="指定用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="chatForm.scope === 'user'" label="目标用户" prop="target_user_ids">
          <div class="multi-select-wrapper">
            <el-select
              v-model="chatForm.target_user_ids"
              multiple
              filterable
              remote
              collapse-tags
              collapse-tags-tooltip
              reserve-keyword
              placeholder="输入关键词搜索选择用户"
              :remote-method="searchUsers"
              :loading="searchUserLoading"
              style="width: 100%"
            >
              <el-option
                v-for="user in searchUserOptions"
                :key="user.id"
                :label="`${user.nickname || user.username}`"
                :value="user.id"
              >
                <div class="user-option-item">
                  <span class="user-option-name">{{ user.nickname || user.username }}</span>
                  <span class="user-option-meta">ID: {{ user.id }} · {{ user.email || '' }}</span>
                </div>
              </el-option>
            </el-select>
            <div v-if="chatForm.target_user_ids.length > 0" class="selected-summary">
              已选 <strong>{{ chatForm.target_user_ids.length }}</strong> 个用户
            </div>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="chatDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="chatSubmitting" @click="handleSendChat">
          {{ chatForm.scope === 'all' ? '发送给全体用户' : '发送给所选用户' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SystemMessage, Organization } from '@/types'
import type { User } from '@/types'
import { getSystemMessages, createSystemMessage, updateSystemMessage, deleteSystemMessage, broadcastChat, getBroadcastChatJob } from '@/api/systemMessages'
import type { BroadcastChatJob } from '@/api/systemMessages'
import { getOrganizationTree } from '@/api/organization'
import { getUsers } from '@/api/users'
import { getVersionDistribution } from '@/api/versions'
import type { VersionDistribution } from '@/types/client'
import { CLIENT_PLATFORM_OPTIONS, clientPlatformLabel } from '@/constants/platforms'

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
  target_ids: [] as number[],
  target_version: '' as string,
  target_platform: '' as string,
  status: 'draft' as 'published' | 'draft' | 'active',
})

// 版本分布（在线客户端快照，含「未知版本」桶）供「指定版本(在线)」选择。
// 后端分布按 version|platform 拆分，此处聚合到版本级：同一版本跨平台合并台数，
// 避免下拉出现重复选项与误导性的分平台计数。
const versionOptions = ref<VersionDistribution[]>([])
const fetchVersionDistribution = async () => {
  try {
    const { data } = await getVersionDistribution()
    const byVersion = new Map<string, number>()
    for (const item of data.data || []) {
      byVersion.set(item.version, (byVersion.get(item.version) || 0) + item.count)
    }
    versionOptions.value = Array.from(byVersion.entries()).map(([version, count]) => ({ version, count }))
  } catch {
    // 请求拦截器已提示
  }
}


const messageRules: FormRules = {
  title: [{ required: true, message: '请输入消息标题', trigger: 'blur' }],
  content: [{ required: true, message: '请输入消息内容', trigger: 'blur' }],
  target_type: [{ required: true, message: '请选择发送范围', trigger: 'change' }],
  target_ids: [
    {
      validator: (_rule: any, value: number[], callback: Function) => {
        if (messageForm.target_type !== 'all' && messageForm.target_type !== 'version' && (!value || value.length === 0)) {
          callback(new Error(targetTypeRequiredMsg.value))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
  target_version: [
    {
      validator: (_rule: any, value: string, callback: Function) => {
        if (messageForm.target_type === 'version' && !value) {
          callback(new Error('请选择目标版本'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
}

// 动态错误提示（用于验证）
const targetTypeRequiredMsg = computed(() => {
  if (messageForm.target_type === 'user') return '请至少选择一个用户'
  if (messageForm.target_type === 'department') return '请至少选择一个部门'
  return ''
})

// 工具函数
const targetTypeLabel = (type: string): string => {
  const map: Record<string, string> = {
    all: '全员',
    department: '指定部门',
    user: '指定用户',
    version: '指定版本(在线)',
  }
  return map[type] || type
}


const handleTargetTypeChange = () => {
  messageForm.target_ids = []
}

// ---------- 部门树 ----------
const departmentTree = ref<Organization[]>([])

const fetchDepartmentTree = async () => {
  try {
    const { data } = await getOrganizationTree()
    // 接口返回 { departments, unassignedUsers }
    const resData = data.data as any
    const departments = Array.isArray(resData?.departments) ? resData.departments : []
    // 递归过滤掉无 ID 的虚拟节点（如"未分配用户"）
    const filterValid = (nodes: any[]): any[] => {
      return nodes.filter((n: any) => {
        if (!n.id) return false
        if (Array.isArray(n.subDepartments)) {
          n.subDepartments = filterValid(n.subDepartments)
        }
        return true
      })
    }
    departmentTree.value = filterValid(departments)
  } catch {
    // ignore
  }
}

// ---------- 用户搜索 ----------
const searchUserLoading = ref(false)
const searchUserOptions = ref<User[]>([])

const searchUsers = async (keyword: string) => {
  if (!keyword) {
    searchUserOptions.value = []
    return
  }
  searchUserLoading.value = true
  try {
    const { data } = await getUsers({ page: 1, pageSize: 20, keyword })
    searchUserOptions.value = data.data.list
  } catch {
    searchUserOptions.value = []
  } finally {
    searchUserLoading.value = false
  }
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
  isEdit.value = false
  resetMessageForm()
  messageDialogVisible.value = true
}

// 编辑消息
const handleEdit = (row: SystemMessage) => {
  isEdit.value = true
  messageForm.id = row.id
  messageForm.title = row.title
  messageForm.content = row.content
  messageForm.target_type = row.target_type || 'all'
  messageForm.target_id = row.target_id
  messageForm.target_ids = row.target_id ? [row.target_id] : []
  messageForm.target_version = row.target_version || ''
  messageForm.target_platform = row.target_platform || ''
  messageForm.status = row.status === 'active' ? 'published' : 'draft'
  messageDialogVisible.value = true
}

const resetMessageForm = () => {
  messageForm.id = 0
  messageForm.title = ''
  messageForm.content = ''
  messageForm.target_type = 'all'
  messageForm.target_id = undefined
  messageForm.target_ids = []
  messageForm.target_version = ''
  messageForm.target_platform = ''
  messageForm.status = 'draft'
}

const handleSubmit = async () => {
  if (!messageFormRef.value) return
  await messageFormRef.value.validate(async (valid) => {
    if (!valid) {
      ElMessage.warning('请检查表单填写是否完整')
      return
    }
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
          target_ids: messageForm.target_type !== 'all' ? messageForm.target_ids : undefined,
          target_version: messageForm.target_type === 'version' ? messageForm.target_version : undefined,
          target_platform: messageForm.target_type === 'version' ? messageForm.target_platform || undefined : undefined,
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

// ---------- 群发私聊 ----------
const chatDialogVisible = ref(false)
const chatSubmitting = ref(false)
const chatFormRef = ref<FormInstance>()
const chatForm = reactive({
  content: '',
  scope: 'all' as 'all' | 'user',
  target_user_ids: [] as number[],
})

const chatRules: FormRules = {
  content: [{ required: true, message: '请输入消息内容', trigger: 'blur' }],
  scope: [{ required: true, message: '请选择发送范围', trigger: 'change' }],
  target_user_ids: [
    {
      validator: (_rule: any, value: number[], callback: Function) => {
        if (chatForm.scope === 'user' && (!value || value.length === 0)) {
          callback(new Error('请至少选择一个用户'))
        } else {
          callback()
        }
      },
      trigger: 'change',
    },
  ],
}

const openChatDialog = () => {
  chatForm.content = ''
  chatForm.scope = 'all'
  chatForm.target_user_ids = []
  chatDialogVisible.value = true
}

const handleSendChat = async () => {
  if (!chatFormRef.value) return
  await chatFormRef.value.validate(async (valid) => {
    if (!valid) {
      ElMessage.warning('请检查表单填写是否完整')
      return
    }
    const scopeText = chatForm.scope === 'all' ? '全体用户' : `${chatForm.target_user_ids.length} 个用户`
    try {
      await ElMessageBox.confirm(
        `将以系统账号向${scopeText}的单聊会话发送一条消息，该消息会出现在对方的「最近会话」中。确认发送？`,
        '群发私聊确认',
        { type: 'warning', confirmButtonText: '确认发送', cancelButtonText: '取消' }
      )
    } catch {
      return // 用户取消
    }
    chatSubmitting.value = true
    try {
      // POST 提交（可能真失败：权限/参数）
      const { data } = await broadcastChat({
        content: chatForm.content,
        target_user_ids: chatForm.scope === 'user' ? chatForm.target_user_ids : undefined,
      })
      const job = data.data
      ElMessage.success(`群发已开始（共 ${job.total} 人，任务ID：${job.job_id}），后台处理中...`)
      chatDialogVisible.value = false
      // 轮询真实成败：POST 已成功，任何轮询错误（404/网络/超时）都表示"结果暂无法确认"而非发送失败
      try {
        const result = await pollBroadcastChatJob(job.job_id)
        ElMessage.success(`发送完成：成功 ${result.sent}，失败 ${result.failed}，跳过 ${result.skipped}（任务ID：${job.job_id}）`)
      } catch (pollErr: any) {
        const detail = pollErr instanceof Error ? pollErr.message : '后台仍在处理中，消息会陆续送达'
        ElMessage.warning({ message: `群发结果暂无法确认（任务ID：${job.job_id}）：${detail}`, duration: 6000 })
      }
    } catch (error: any) {
      console.error('群发私聊提交失败:', error)
      ElMessage.error(error.response?.data?.message || '群发失败')
    } finally {
      chatSubmitting.value = false
    }
  })
}

// 轮询群发任务直至完成（上限 ~60s）；超时说明后台仍在处理、消息会陆续送达
const pollBroadcastChatJob = async (jobId: string, timeoutMs = 60000): Promise<BroadcastChatJob> => {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    const { data } = await getBroadcastChatJob(jobId)
    const job = data.data
    if (job.status === 'done') return job
    await new Promise(r => setTimeout(r, 1000))
  }
  throw new Error('后台仍在处理中，消息会陆续送达，可稍后刷新页面确认')
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

onMounted(() => {
  fetchMessages()
  fetchDepartmentTree()
  fetchVersionDistribution()
})
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

/* 列表中的目标范围展示 */
.target-info {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.target-detail {
  font-size: 12px;
  color: var(--color-text-secondary, #909399);
  white-space: nowrap;
}

.target-detail.text-muted {
  color: #c0c4cc;
  font-style: italic;
}

/* 多选选择器 */
.multi-select-wrapper {
  width: 100%;
}

.selected-summary {
  margin-top: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #909399);
}

.selected-summary strong {
  color: var(--el-color-primary, #409eff);
  font-weight: 600;
}

/* 用户选项展示 */
.user-option-item {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
  padding: 2px 0;
}

.user-option-name {
  font-weight: 500;
  font-size: 14px;
  color: var(--color-text-primary, #303133);
}

.user-option-meta {
  font-size: 11px;
  color: var(--color-text-secondary, #909399);
  margin-top: 1px;
}

/* 对话框微调 */
:deep(.system-message-dialog .el-dialog__body) {
  padding-top: 10px;
}

:deep(.system-message-dialog .el-form-item:last-child) {
  margin-bottom: 0;
}

.pagination-container {
  margin-top: var(--space-5);
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}
</style>
