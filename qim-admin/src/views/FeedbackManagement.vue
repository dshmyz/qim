<template>
  <div class="feedback-management-page">
    <div class="page-header">
      <h2>用户反馈管理</h2>
      <p class="page-description">管理用户提交的意见反馈，及时处理和回复用户问题</p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stats-row">
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon blue">
            <el-icon><MessageSquare /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.total }}</div>
            <div class="stat-label">总反馈数</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon orange">
            <el-icon><Clock /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.pending }}</div>
            <div class="stat-label">待处理</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon green">
            <el-icon><CheckCircle /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.resolved }}</div>
            <div class="stat-label">已处理</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-icon purple">
            <el-icon><AlertCircle /></el-icon>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.bug }}</div>
            <div class="stat-label">Bug反馈</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 搜索筛选 -->
    <el-card shadow="never" class="search-card">
      <div class="search-bar">
        <el-form :model="searchForm" inline>
          <el-form-item label="反馈类型">
            <el-select v-model="searchForm.type" placeholder="请选择反馈类型" clearable>
              <el-option label="全部" value="" />
              <el-option label="Bug反馈" value="bug" />
              <el-option label="功能建议" value="feature" />
              <el-option label="其他" value="other" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="searchForm.status" placeholder="请选择状态" clearable>
              <el-option label="全部" value="" />
              <el-option label="待处理" value="pending" />
              <el-option label="已处理" value="resolved" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-select v-model="searchForm.priority" placeholder="请选择优先级" clearable>
              <el-option label="全部" value="" />
              <el-option label="高" value="high" />
              <el-option label="中" value="medium" />
              <el-option label="低" value="low" />
            </el-select>
          </el-form-item>
          <el-form-item label="用户ID">
            <el-input
              v-model="searchForm.userId"
              placeholder="请输入用户ID"
              clearable
              @keyup.enter="handleSearch"
            />
          </el-form-item>
          <el-form-item label="日期范围">
            <el-date-picker
              v-model="searchForm.dateRange"
              type="daterange"
              range-separator="至"
              start-placeholder="开始日期"
              end-placeholder="结束日期"
              value-format="YYYY-MM-DD"
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSearch">搜索</el-button>
            <el-button @click="handleReset">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-card>

    <!-- 反馈列表 -->
    <el-card shadow="never" class="list-card">
      <el-table :data="feedbacks" v-loading="loading" border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="user_id" label="用户ID" width="100" />
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag :type="getTypeTagType(row.type)">{{ getTypeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="反馈内容" min-width="300">
          <template #default="{ row }">
            <div class="content-preview">{{ row.content }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="priority" label="优先级" width="100">
          <template #default="{ row }">
            <el-tag :type="getPriorityTagType(row.priority)">{{ getPriorityLabel(row.priority) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">{{ getStatusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="180">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" width="180">
          <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button size="small" @click="openDetail(row)">查看详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          :current-page="pagination.page"
          :page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog
      v-model="showDetailModal"
      title="反馈详情"
      width="600px"
      @close="closeDetailModal"
    >
      <div v-if="currentFeedback" class="feedback-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="ID">{{ currentFeedback.id }}</el-descriptions-item>
          <el-descriptions-item label="用户ID">{{ currentFeedback.user_id }}</el-descriptions-item>
          <el-descriptions-item label="反馈类型">
            <el-tag :type="getTypeTagType(currentFeedback.type)">{{ getTypeLabel(currentFeedback.type) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="优先级">
            <el-tag :type="getPriorityTagType(currentFeedback.priority)">{{ getPriorityLabel(currentFeedback.priority) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(currentFeedback.status)">{{ getStatusLabel(currentFeedback.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatTime(currentFeedback.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatTime(currentFeedback.updated_at) }}</el-descriptions-item>
          <el-descriptions-item label="处理人ID">{{ currentFeedback.handler_id || '未分配' }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-section">
          <h4>反馈内容</h4>
          <p class="content-text">{{ currentFeedback.content }}</p>
        </div>

        <div v-if="currentFeedback.screenshot" class="detail-section">
          <h4>截图</h4>
          <img :src="currentFeedback.screenshot" alt="截图" class="screenshot-image" />
        </div>

        <div v-if="currentFeedback.reply" class="detail-section">
          <h4>管理员回复</h4>
          <p class="reply-text">{{ currentFeedback.reply }}</p>
        </div>

        <div class="reply-section">
          <h4>回复反馈</h4>
          <el-textarea
            v-model="replyContent"
            placeholder="请输入回复内容"
            rows="4"
            :disabled="currentFeedback.status === 'resolved'"
          />
          <div class="reply-actions">
            <el-select v-model="updateStatus" placeholder="选择状态">
              <el-option label="待处理" value="pending" />
              <el-option label="已处理" value="resolved" />
            </el-select>
            <el-button
              type="primary"
              @click="submitReply"
              :disabled="!replyContent.trim() || currentFeedback.status === 'resolved'"
            >
              提交回复
            </el-button>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { MessageSquare, Clock, CheckCircle, AlertCircle } from '@element-plus/icons-vue'
import { getFeedbacks, updateFeedback } from '../../api/client'
import type { UserFeedback } from '../../types/client'

const loading = ref(false)
const showDetailModal = ref(false)
const currentFeedback = ref<UserFeedback | null>(null)
const replyContent = ref('')
const updateStatus = ref('resolved')

const searchForm = reactive({
  type: '',
  status: '',
  priority: '',
  userId: '',
  dateRange: [] as string[]
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const feedbacks = ref<UserFeedback[]>([])

const stats = reactive({
  total: 0,
  pending: 0,
  resolved: 0,
  bug: 0
})

const handleSearch = () => {
  pagination.page = 1
  loadFeedbacks()
}

const handleReset = () => {
  searchForm.type = ''
  searchForm.status = ''
  searchForm.priority = ''
  searchForm.userId = ''
  searchForm.dateRange = []
  pagination.page = 1
  loadFeedbacks()
}

const handleSizeChange = (size: number) => {
  pagination.pageSize = size
  loadFeedbacks()
}

const handleCurrentChange = (page: number) => {
  pagination.page = page
  loadFeedbacks()
}

const loadFeedbacks = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      pageSize: pagination.pageSize
    }
    if (searchForm.type) params.type = searchForm.type
    if (searchForm.status) params.status = searchForm.status
    if (searchForm.priority) params.priority = searchForm.priority
    if (searchForm.userId) params.userId = searchForm.userId
    if (searchForm.dateRange && searchForm.dateRange.length === 2) {
      params.startDate = searchForm.dateRange[0]
      params.endDate = searchForm.dateRange[1]
    }

    const response = await getFeedbacks(params)
    if (response.data.code === 0) {
      feedbacks.value = response.data.data.list
      pagination.total = response.data.data.total
      pagination.page = response.data.data.page
      pagination.pageSize = response.data.data.pageSize
      updateStats()
    }
  } catch (error) {
    console.error('加载反馈列表失败:', error)
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  stats.total = pagination.total
  stats.pending = feedbacks.value.filter(f => f.status === 'pending').length
  stats.resolved = feedbacks.value.filter(f => f.status === 'resolved').length
  stats.bug = feedbacks.value.filter(f => f.type === 'bug').length
}

const openDetail = async (feedback: UserFeedback) => {
  currentFeedback.value = feedback
  replyContent.value = feedback.reply || ''
  updateStatus.value = feedback.status
  showDetailModal.value = true
}

const closeDetailModal = () => {
  showDetailModal.value = false
  currentFeedback.value = null
  replyContent.value = ''
}

const submitReply = async () => {
  if (!currentFeedback.value || !replyContent.value.trim()) return

  try {
    const response = await updateFeedback(currentFeedback.value.id as number, {
      reply: replyContent.value.trim(),
      status: updateStatus.value,
      handler_id: 1 // 当前管理员ID，实际应该从登录状态获取
    })

    if (response.data.code === 0) {
      closeDetailModal()
      loadFeedbacks()
      import('element-plus').then(El => {
        El.ElMessage.success('回复成功')
      })
    }
  } catch (error) {
    console.error('回复失败:', error)
    import('element-plus').then(El => {
      El.ElMessage.error('回复失败')
    })
  }
}

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    bug: 'Bug反馈',
    feature: '功能建议',
    other: '其他'
  }
  return labels[type] || type
}

const getTypeTagType = (type: string) => {
  const types: Record<string, string> = {
    bug: 'danger',
    feature: 'primary',
    other: 'info'
  }
  return types[type] || 'info'
}

const getPriorityLabel = (priority: string) => {
  const labels: Record<string, string> = {
    high: '高',
    medium: '中',
    low: '低'
  }
  return labels[priority] || priority
}

const getPriorityTagType = (priority: string) => {
  const types: Record<string, string> = {
    high: 'danger',
    medium: 'warning',
    low: 'success'
  }
  return types[priority] || 'info'
}

const getStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    pending: '待处理',
    resolved: '已处理'
  }
  return labels[status] || status
}

const getStatusTagType = (status: string) => {
  const types: Record<string, string> = {
    pending: 'warning',
    resolved: 'success'
  }
  return types[status] || 'info'
}

const formatTime = (timestamp: any) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

onMounted(() => {
  loadFeedbacks()
})
</script>

<style scoped>
.feedback-management-page {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0 0 8px 0;
  font-size: 24px;
  font-weight: 600;
}

.page-description {
  margin: 0;
  color: #909399;
  font-size: 14px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  display: flex;
  align-items: center;
  padding: 20px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
}

.stat-icon.blue {
  background: #e6f7ff;
  color: #1890ff;
}

.stat-icon.orange {
  background: #fff7e6;
  color: #fa8c16;
}

.stat-icon.green {
  background: #f6ffed;
  color: #52c41a;
}

.stat-icon.purple {
  background: #f9f0ff;
  color: #722ed1;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1f2329;
}

.stat-label {
  font-size: 14px;
  color: #646a73;
  margin-top: 4px;
}

.search-card {
  margin-bottom: 20px;
}

.search-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.content-preview {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  color: #646a73;
}

.pagination-container {
  margin-top: 20px;
  text-align: right;
}

.feedback-detail {
  padding: 10px 0;
}

.detail-section {
  margin-top: 20px;
}

.detail-section h4 {
  margin: 0 0 10px 0;
  font-size: 14px;
  font-weight: 500;
  color: #1f2329;
}

.content-text {
  padding: 12px;
  background: #f5f5f5;
  border-radius: 6px;
  margin: 0;
  line-height: 1.6;
}

.screenshot-image {
  max-width: 100%;
  max-height: 400px;
  border-radius: 8px;
}

.reply-text {
  padding: 12px;
  background: #e6f7ff;
  border-radius: 6px;
  margin: 0;
  line-height: 1.6;
  border-left: 4px solid #1890ff;
}

.reply-section {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #ebf0f5;
}

.reply-section h4 {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 500;
  color: #1f2329;
}

.reply-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 12px;
}

.reply-actions .el-select {
  width: 120px;
}
</style>
