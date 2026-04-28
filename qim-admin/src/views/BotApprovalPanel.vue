<template>
  <div class="bot-approval-panel">
    <!-- 筛选栏 -->
    <div class="filter-bar">
      <el-radio-group v-model="filterStatus" @change="fetchApprovals">
        <el-radio-button value="pending">待审批 ({{ pendingCount }})</el-radio-button>
        <el-radio-button value="approved">已通过</el-radio-button>
        <el-radio-button value="rejected">已拒绝</el-radio-button>
        <el-radio-button value="all">全部</el-radio-button>
      </el-radio-group>
    </div>

    <!-- 审批列表 -->
    <el-table :data="approvals" v-loading="loading" style="width: 100%">
      <el-table-column label="申请人" min-width="140">
        <template #default="{ row }">
          <div class="applicant-cell">
            <el-avatar :size="28" :src="row.creator_avatar">{{ row.creator_name?.charAt(0) }}</el-avatar>
            <span>{{ row.creator_name || '未知' }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Bot 名称" min-width="160">
        <template #default="{ row }">
          <div class="bot-cell">
            <el-avatar :size="28" :src="row.avatar">{{ row.name.charAt(0) }}</el-avatar>
            <span class="bot-name">{{ row.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="类型" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'ai' ? 'primary' : 'info'" size="small">
            {{ row.type === 'ai' ? 'AI' : '自定义' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="description" label="描述" min-width="200" show-overflow-tooltip />
      <el-table-column label="已创建" width="80">
        <template #default="{ row }">
          <el-tag size="small">{{ row.creator_bot_count }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusTagType(row.approval_status)" size="small">
            {{ statusLabel(row.approval_status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="createdAt" label="创建时间" width="180" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewPrompt(row)">提示词</el-button>
          <template v-if="row.approval_status === 'pending'">
            <el-button size="small" type="success" @click="handleApprove(row)">通过</el-button>
            <el-button size="small" type="danger" @click="handleReject(row)">拒绝</el-button>
          </template>
          <span v-else-if="row.approval_status === 'rejected'" class="reject-reason">
            {{ row.reject_reason || '无' }}
          </span>
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

    <!-- 查看提示词对话框 -->
    <el-dialog v-model="promptDialogVisible" title="Bot 配置详情" width="600px">
      <div v-if="selectedBot" class="prompt-content">
        <p><strong>名称：</strong>{{ selectedBot.name }}</p>
        <p><strong>描述：</strong>{{ selectedBot.description }}</p>
        <p><strong>类型：</strong>{{ selectedBot.type }}</p>
        <p><strong>系统提示词：</strong></p>
        <pre>{{ selectedBot.config }}</pre>
      </div>
    </el-dialog>

    <!-- 拒绝原因输入对话框 -->
    <el-dialog v-model="rejectDialogVisible" title="拒绝原因" width="400px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" placeholder="请输入拒绝原因（可选）" />
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmReject">确认拒绝</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getBotApprovals, approveBot, rejectBot } from '@/api/aiBots'
import type { AIBot } from '@/types'

const filterStatus = ref('pending')
const loading = ref(false)
const approvals = ref<AIBot[]>([])
const pendingCount = ref(0)
const selectedBot = ref<AIBot | null>(null)
const promptDialogVisible = ref(false)
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectingBotId = ref(0)

const pagination = reactive({ page: 1, pageSize: 10, total: 0 })

const fetchApprovals = async () => {
  loading.value = true
  try {
    const { data } = await getBotApprovals({
      status: filterStatus.value,
      page: pagination.page,
      pageSize: pagination.pageSize,
    })
    approvals.value = data.data.list
    pagination.total = data.data.total

    // 更新待审批计数
    if (filterStatus.value !== 'pending') {
      const { data: pendingData } = await getBotApprovals({ status: 'pending', page: 1, pageSize: 1 })
      pendingCount.value = pendingData.data.total
    } else {
      pendingCount.value = pagination.total
    }
  } catch {
    // 错误已在请求拦截器中处理
  } finally {
    loading.value = false
  }
}

const handleApprove = async (row: AIBot) => {
  try {
    await ElMessageBox.confirm(`确定通过「${row.name}」的申请吗？`, '确认通过')
    await approveBot(row.id)
    ElMessage.success('审批通过')
    fetchApprovals()
  } catch {
    // 用户取消或请求失败
  }
}

const handleReject = (row: AIBot) => {
  rejectingBotId.value = row.id
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

const confirmReject = async () => {
  try {
    await rejectBot(rejectingBotId.value, { reason: rejectReason.value })
    ElMessage.success('已拒绝')
    rejectDialogVisible.value = false
    fetchApprovals()
  } catch {
    // 错误已在请求拦截器中处理
  }
}

const handleViewPrompt = (row: AIBot) => {
  selectedBot.value = row
  promptDialogVisible.value = true
}

const statusTagType = (status: string) => {
  const map: Record<string, string> = { pending: 'warning', approved: 'success', rejected: 'danger' }
  return map[status] || 'info'
}

const statusLabel = (status: string) => {
  const map: Record<string, string> = { pending: '待审批', approved: '已通过', rejected: '已拒绝' }
  return map[status] || status
}

onMounted(fetchApprovals)
</script>

<style scoped>
.bot-approval-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.filter-bar {
  display: flex;
  justify-content: flex-end;
}

.applicant-cell, .bot-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.bot-name {
  font-weight: 600;
  color: var(--color-text-primary);
}

.reject-reason {
  font-size: 12px;
  color: var(--color-text-secondary);
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

.prompt-content pre {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.6;
}
</style>
