<template>
  <div class="unified-approval-panel">
    <!-- 审批配置卡片 -->
    <el-card class="config-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span class="title">审批配置</span>
          <el-tag type="info" size="small">控制各类操作的审批开关</el-tag>
        </div>
      </template>
      <div class="config-grid">
        <div v-for="config in approvalConfigs" :key="config.type" class="config-item">
          <div class="config-info">
            <span class="config-name">{{ getApprovalTypeName(config.type) }}</span>
            <span class="config-desc">{{ config.description || '暂无描述' }}</span>
          </div>
          <el-switch
            v-model="config.enabled"
            @change="handleConfigChange(config)"
            :loading="config.loading"
          />
        </div>
      </div>
    </el-card>

    <!-- 操作栏 -->
    <div class="action-bar">
      <div class="filter-left">
        <el-radio-group v-model="filterType" @change="handleFilterChange">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="avatar">Avatar</el-radio-button>
          <el-radio-button value="bot">Bot</el-radio-button>
          <el-radio-button value="group_ai">群聊AI</el-radio-button>
        </el-radio-group>
        <el-radio-group v-model="filterStatus" @change="handleFilterChange" style="margin-left: 16px">
          <el-radio-button value="all">全部</el-radio-button>
          <el-radio-button value="pending">待审批 ({{ pendingCount }})</el-radio-button>
          <el-radio-button value="approved">已通过</el-radio-button>
          <el-radio-button value="rejected">已拒绝</el-radio-button>
        </el-radio-group>
      </div>
      <el-button type="primary" @click="showEnableDialog" v-if="filterType === 'avatar' || filterType === 'all'">
        <el-icon><Plus /></el-icon>
        主动开启 Avatar
      </el-button>
    </div>

    <!-- 审批列表 -->
    <el-table :data="approvals" v-loading="loading" style="width: 100%">
      <el-table-column label="类型" width="120">
        <template #default="{ row }">
          <el-tag :type="getTypeTagType(row.type)" size="small">
            {{ getApprovalTypeName(row.type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="申请人" min-width="140">
        <template #default="{ row }">
          <div class="user-cell">
            <el-avatar :size="28" :src="row.creator_avatar">{{ row.creator_name?.charAt(0) }}</el-avatar>
            <span>{{ row.creator_name || '未知' }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="名称" min-width="120">
        <template #default="{ row }">
          <span class="entity-name">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="180" show-overflow-tooltip />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.approval_status)" size="small">
            {{ statusLabel(row.approval_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="申请时间" width="180">
        <template #default="{ row }">
          {{ row.applied_at ? formatTime(row.applied_at) : formatTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column prop="reject_reason" label="拒绝原因" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.reject_reason || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <template v-if="row.approval_status === 'pending'">
            <el-button size="small" type="success" @click="handleApprove(row)">通过</el-button>
            <el-button size="small" type="danger" @click="handleReject(row)">拒绝</el-button>
          </template>
          <span v-else class="reviewed-info">已处理</span>
        </template>
      </el-table-column>
    </el-table>

    <!-- 分页 -->
    <div class="pagination-container">
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        @size-change="fetchApprovals"
        @current-change="fetchApprovals"
      />
    </div>

    <!-- 拒绝原因输入对话框 -->
    <el-dialog v-model="rejectDialogVisible" title="拒绝原因" width="400px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" placeholder="请输入拒绝原因" />
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmReject" :disabled="!rejectReason.trim()">确认拒绝</el-button>
      </template>
    </el-dialog>

    <!-- 主动开启对话框 -->
    <el-dialog v-model="enableDialogVisible" title="主动开启用户分身" width="500px">
      <div class="enable-form">
        <p class="enable-hint">搜索用户并为其开启分身功能，用户将收到系统通知</p>
        <el-input
          v-model="searchKeyword"
          placeholder="输入用户名或昵称搜索..."
          :prefix-icon="Search"
          clearable
          @input="handleSearch"
        />
        <div v-if="searching" class="search-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>搜索中...</span>
        </div>
        <div v-else-if="searchResults.length > 0" class="search-results">
          <div
            v-for="user in searchResults"
            :key="user.id"
            class="search-result-item"
            @click="selectUser(user)"
          >
            <el-avatar :size="32" :src="user.avatar">{{ user.name?.charAt(0) || user.username?.charAt(0) }}</el-avatar>
            <div class="result-info">
              <span class="result-name">{{ user.name || user.username }}</span>
              <span class="result-username">@{{ user.username }}</span>
            </div>
            <el-icon class="add-icon"><Plus /></el-icon>
          </div>
        </div>
        <div v-else-if="searchKeyword && !searching" class="no-results">
          未找到匹配的用户
        </div>
        <div v-if="selectedUser" class="selected-user">
          <span>已选择：</span>
          <strong>{{ selectedUser.name || selectedUser.username }}</strong>
        </div>
      </div>
      <template #footer>
        <el-button @click="closeEnableDialog">取消</el-button>
        <el-button type="primary" @click="handleEnable" :disabled="!selectedUser" :loading="enabling">
          确认开启
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, Loading } from '@element-plus/icons-vue'
import { 
  getApprovals, 
  approveEntity, 
  rejectEntity, 
  enableAvatar, 
  getApprovalConfigs,
  updateApprovalConfig,
  type ApprovalItem, 
  type ApprovalType,
  type ApprovalConfig 
} from '@/api/approvals'
import { request } from '@/utils/request'

interface User {
  id: number
  username: string
  name: string
  avatar?: string
}

const filterType = ref<ApprovalType>('all')
const filterStatus = ref<'all' | 'pending' | 'approved' | 'rejected'>('all')
const loading = ref(false)
const approvals = ref<ApprovalItem[]>([])
const pendingCount = ref(0)

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

// 审批配置相关
const approvalConfigs = ref<(ApprovalConfig & { loading?: boolean })[]>([])

const getApprovalTypeName = (type: ApprovalType): string => {
  const names: Record<string, string> = {
    avatar: '分身功能',
    bot: '机器人创建',
    channel: '频道创建',
    group_ai: '群聊AI助手',
  }
  return names[type] || type
}

const getTypeTagType = (type: ApprovalType): string => {
  const types: Record<string, string> = {
    avatar: 'success',
    bot: 'primary',
    channel: 'warning',
    group_ai: 'info',
  }
  return types[type] || ''
}

const fetchApprovalConfigs = async () => {
  try {
    const { data } = await getApprovalConfigs()
    if (data.code === 0) {
      approvalConfigs.value = data.data.map(config => ({ ...config, loading: false }))
    }
  } catch (error) {
    console.error('获取审批配置失败:', error)
  }
}

const handleConfigChange = async (config: ApprovalConfig & { loading?: boolean }) => {
  config.loading = true
  try {
    const { data } = await updateApprovalConfig(config.type, {
      enabled: config.enabled,
      description: config.description,
    })
    if (data.code === 0) {
      ElMessage.success(`${getApprovalTypeName(config.type)}审批已${config.enabled ? '启用' : '关闭'}`)
    } else {
      // 恢复原状态
      config.enabled = !config.enabled
      ElMessage.error(data.message || '更新失败')
    }
  } catch (error) {
    // 恢复原状态
    config.enabled = !config.enabled
    ElMessage.error('更新审批配置失败')
  } finally {
    config.loading = false
  }
}

// 拒绝弹窗
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectingItem = ref<ApprovalItem | null>(null)

// 主动开启弹窗
const enableDialogVisible = ref(false)
const searchKeyword = ref('')
const searching = ref(false)
const searchResults = ref<User[]>([])
const selectedUser = ref<User | null>(null)
const enabling = ref(false)

// 搜索防抖
let searchTimer: ReturnType<typeof setTimeout> | null = null

const fetchApprovals = async () => {
  loading.value = true
  try {
    const { data } = await getApprovals({
      type: filterType.value,
      status: filterStatus.value,
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    approvals.value = data.data.list || []
    pagination.total = data.data.total || 0

    // 更新待审批计数
    if (filterStatus.value !== 'pending') {
      const { data: pendingData } = await getApprovals({ status: 'pending' })
      pendingCount.value = pendingData.data.total || 0
    } else {
      pendingCount.value = pagination.total
    }
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleFilterChange = () => {
  pagination.page = 1
  fetchApprovals()
}

const handleApprove = async (row: ApprovalItem) => {
  const typeName = row.type === 'avatar' ? '分身' : `机器人「${row.name}」`
  try {
    await ElMessageBox.confirm(`确定通过${row.creator_name}的${typeName}申请吗？`, '确认通过')
    await approveEntity(row.type, row.id)
    ElMessage.success('审批通过')
    fetchApprovals()
  } catch {
    // 用户取消或请求失败
  }
}

const handleReject = (row: ApprovalItem) => {
  rejectingItem.value = row
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

const confirmReject = async () => {
  if (!rejectingItem.value || !rejectReason.value.trim()) return
  try {
    await rejectEntity(rejectingItem.value.type, rejectingItem.value.id, rejectReason.value.trim())
    ElMessage.success('已拒绝')
    rejectDialogVisible.value = false
    fetchApprovals()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const showEnableDialog = () => {
  enableDialogVisible.value = true
  searchKeyword.value = ''
  searchResults.value = []
  selectedUser.value = null
}

const closeEnableDialog = () => {
  enableDialogVisible.value = false
  searchKeyword.value = ''
  searchResults.value = []
  selectedUser.value = null
}

const handleSearch = () => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }

  if (!searchKeyword.value.trim()) {
    searchResults.value = []
    return
  }

  searchTimer = setTimeout(async () => {
    searching.value = true
    try {
      const { data } = await request({
        url: `/v1/users/search?q=${encodeURIComponent(searchKeyword.value.trim())}`,
        method: 'get',
      }) as any
      searchResults.value = (data?.data?.list || data?.data || []) as User[]
    } catch {
      searchResults.value = []
    } finally {
      searching.value = false
    }
  }, 300)
}

const selectUser = (user: User) => {
  selectedUser.value = user
  searchResults.value = []
  searchKeyword.value = ''
}

const handleEnable = async () => {
  if (!selectedUser.value) return

  try {
    await ElMessageBox.confirm(
      `确定要为「${selectedUser.value.name || selectedUser.value.username}」开启分身功能吗？`,
      '确认开启'
    )
    enabling.value = true
    await enableAvatar(selectedUser.value.id)
    ElMessage.success('已开启分身功能')
    closeEnableDialog()
    fetchApprovals()
  } catch {
    // 用户取消或请求失败
  } finally {
    enabling.value = false
  }
}

const statusTagType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', approved: 'success', rejected: 'danger' }
  return map[status] || 'info'
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '待审批', approved: '已通过', rejected: '已拒绝' }
  return map[status] || status
}

const formatTime = (time: string) => {
  if (!time) return '-'
  const d = new Date(time)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

onMounted(() => {
  fetchApprovalConfigs()
  fetchApprovals()
})
</script>

<style scoped>
.unified-approval-panel {
  padding: 20px;
}

.config-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-header .title {
  font-size: 16px;
  font-weight: 600;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.config-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  transition: all 0.3s;
}

.config-item:hover {
  background: var(--el-fill-color);
}

.config-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.config-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.config-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filter-left {
  display: flex;
  align-items: center;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.entity-name {
  font-weight: 500;
  color: var(--color-text-primary);
}

.reviewed-info {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

/* 主动开启弹窗 */
.enable-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.enable-hint {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

.search-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.search-results {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.search-result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.2s;
}

.search-result-item:hover {
  background: var(--color-surface-hover);
}

.result-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.result-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.result-username {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.add-icon {
  color: var(--color-primary);
  font-size: 12px;
}

.no-results {
  padding: 20px;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 14px;
}

.selected-user {
  padding: 10px 12px;
  background: var(--color-primary-light);
  border-radius: var(--radius-md);
  font-size: 14px;
  color: var(--color-primary);
}
</style>
