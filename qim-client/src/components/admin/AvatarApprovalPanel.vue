<template>
  <div class="avatar-approval-panel">
    <!-- 头部 -->
    <div class="panel-header">
      <div class="header-left">
        <button class="back-btn" @click="$emit('back')">
          <i class="fas fa-chevron-left"></i>
        </button>
        <button class="toggle-sidebar-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
        <div class="header-info">
          <h2>Avatar 审批管理</h2>
          <span class="subtitle">审核用户的分身功能启用申请</span>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="filter-bar">
      <div class="filter-tabs">
        <button
          v-for="tab in statusTabs"
          :key="tab.value"
          :class="['filter-tab', { active: currentStatus === tab.value }]"
          @click="currentStatus = tab.value"
        >
          {{ tab.label }}
          <span v-if="tab.count > 0" class="count">{{ tab.count }}</span>
        </button>
      </div>
    </div>

    <!-- 列表内容 -->
    <div class="panel-content">
      <div v-if="loading" class="loading-state">
        <LoadingSpinner />
        <span>加载中...</span>
      </div>

      <div v-else-if="approvals.length === 0" class="empty-state">
        <div class="empty-icon">
          <i class="fas fa-inbox"></i>
        </div>
        <p>暂无审批申请</p>
      </div>

      <div v-else class="approval-list">
        <div
          v-for="approval in approvals"
          :key="approval.id"
          class="approval-item"
        >
          <div class="approval-user">
            <div class="user-avatar">
              <CssAvatar :name="approval.nickname || approval.username" :avatar="approval.avatar" :size="40" />
            </div>
            <div class="user-info">
              <div class="user-name">
                {{ approval.nickname || approval.username }}
                <span class="username">@{{ approval.username }}</span>
              </div>
              <div class="apply-time">
                <i class="fas fa-clock"></i>
                {{ formatDate(approval.appliedAt) }}
              </div>
            </div>
          </div>

          <div class="approval-status">
            <span :class="['status-badge', approval.status]">
              {{ getStatusText(approval.status) }}
            </span>
          </div>

          <div class="approval-actions">
            <template v-if="approval.status === 'pending'">
              <button class="btn btn-success" @click="handleApprove(approval)" :disabled="processingId === approval.id">
                <i class="fas fa-check"></i>
                通过
              </button>
              <button class="btn btn-danger" @click="showRejectDialog(approval)" :disabled="processingId === approval.id">
                <i class="fas fa-times"></i>
                拒绝
              </button>
            </template>
            <template v-else>
              <span class="reviewed-info" v-if="approval.reviewerName">
                由 {{ approval.reviewerName }} 审批
              </span>
              <span class="reviewed-info" v-if="approval.rejectedReason">
                原因：{{ approval.rejectedReason }}
              </span>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- 拒绝原因弹窗 -->
    <QDialog
      v-model:visible="rejectDialogVisible"
      title="拒绝申请"
      width="400px"
    >
      <div class="reject-form">
        <p class="reject-hint">请输入拒绝原因，将通知申请人</p>
        <textarea
          v-model="rejectReason"
          class="reject-input"
          placeholder="请输入拒绝原因..."
          rows="4"
        ></textarea>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="closeRejectDialog">取消</button>
        <button class="btn btn-danger" @click="handleReject" :disabled="!rejectReason.trim() || rejecting">
          {{ rejecting ? '处理中...' : '确认拒绝' }}
        </button>
      </template>
    </QDialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { adminAPI } from '../../api/admin'
import type { AvatarApprovalRecord, AvatarApprovalStatus } from '../../types/avatar'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import CssAvatar from '../shared/CssAvatar.vue'
import QDialog from '../shared/QDialog.vue'

const emit = defineEmits<{
  back: []
  toggleSidebar: []
}>()

const loading = ref(false)
const approvals = ref<AvatarApprovalRecord[]>([])
const currentStatus = ref<AvatarApprovalStatus | 'all'>('pending')
const processingId = ref<number | null>(null)

// 拒绝弹窗相关
const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejecting = ref(false)
const selectedApproval = ref<AvatarApprovalRecord | null>(null)

// 状态选项卡
const statusTabs = computed(() => [
  { value: 'pending' as const, label: '待审批', count: approvals.value.filter(a => a.status === 'pending').length },
  { value: 'approved' as const, label: '已通过', count: approvals.value.filter(a => a.status === 'approved').length },
  { value: 'rejected' as const, label: '已拒绝', count: approvals.value.filter(a => a.status === 'rejected').length },
  { value: 'all' as const, label: '全部', count: approvals.value.length }
])

// 获取状态文本
function getStatusText(status: AvatarApprovalStatus): string {
  const texts: Record<AvatarApprovalStatus, string> = {
    none: '未申请',
    pending: '待审批',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return texts[status]
}

// 格式化日期
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 加载审批列表
async function loadApprovals() {
  loading.value = true
  try {
    const status = currentStatus.value === 'all' ? undefined : currentStatus.value
    approvals.value = await adminAPI.getAvatarApprovals(status)
  } catch (e: any) {
    window.$QMessage.error(e.response?.data?.message || '加载审批列表失败')
  } finally {
    loading.value = false
  }
}

// 通过审批
async function handleApprove(approval: AvatarApprovalRecord) {
  try {
    await window.$QMessageBox.confirm(
      `确定通过 ${approval.nickname || approval.username} 的分身启用申请吗？`,
      '审批确认'
    )
    processingId.value = approval.id
    try {
      await adminAPI.approveAvatar(approval.id)
      window.$QMessage.success('已通过申请')
      await loadApprovals()
    } catch (e: any) {
      window.$QMessage.error(e.response?.data?.message || '审批失败')
    } finally {
      processingId.value = null
    }
  } catch {
    // 用户取消
  }
}

// 显示拒绝弹窗
function showRejectDialog(approval: AvatarApprovalRecord) {
  selectedApproval.value = approval
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

// 关闭拒绝弹窗
function closeRejectDialog() {
  rejectDialogVisible.value = false
  selectedApproval.value = null
  rejectReason.value = ''
}

// 拒绝审批
async function handleReject() {
  if (!selectedApproval.value || !rejectReason.value.trim()) return

  rejecting.value = true
  try {
    await adminAPI.rejectAvatar(selectedApproval.value.id, rejectReason.value.trim())
    window.$QMessage.success('已拒绝申请')
    closeRejectDialog()
    await loadApprovals()
  } catch (e: any) {
    window.$QMessage.error(e.response?.data?.message || '拒绝失败')
  } finally {
    rejecting.value = false
  }
}

// 监听状态变化重新加载
watch(currentStatus, () => {
  loadApprovals()
})

onMounted(() => {
  loadApprovals()
})
</script>

<style scoped>
.avatar-approval-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
}

/* 头部样式 */
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  height: 72px;
  box-sizing: border-box;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn,
.toggle-sidebar-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--hover-color);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: var(--primary-color);
  transition: background 0.2s;
}

.back-btn:hover,
.toggle-sidebar-btn:hover {
  background: var(--primary-light);
}

.header-info h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
}

.subtitle {
  font-size: 13px;
  color: var(--text-secondary);
}

/* 筛选栏 */
.filter-bar {
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-color);
}

.filter-tabs {
  display: flex;
  gap: 8px;
}

.filter-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  background: var(--bg-color);
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.filter-tab:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.filter-tab.active {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.filter-tab .count {
  background: rgba(255, 255, 255, 0.2);
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 12px;
}

.filter-tab:not(.active) .count {
  background: var(--hover-color);
}

/* 内容区域 */
.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px;
  gap: 12px;
  color: var(--text-secondary);
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

/* 审批列表 */
.approval-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.approval-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px;
  background: var(--bg-color);
  border-radius: 8px;
  border: 1px solid var(--border-color);
  transition: all 0.2s;
}

.approval-item:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.approval-user {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.user-avatar {
  flex-shrink: 0;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.user-name {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-color);
}

.username {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: normal;
  margin-left: 4px;
}

.apply-time {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
}

.approval-status {
  flex-shrink: 0;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
}

.status-badge.pending {
  background: rgba(245, 158, 11, 0.1);
  color: #F59E0B;
}

.status-badge.approved {
  background: rgba(16, 185, 129, 0.1);
  color: #10B981;
}

.status-badge.rejected {
  background: rgba(239, 68, 68, 0.1);
  color: #EF4444;
}

.approval-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.reviewed-info {
  font-size: 13px;
  color: var(--text-secondary);
}

/* 按钮样式 */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-success {
  background: #10B981;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #059669;
}

.btn-danger {
  background: #EF4444;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #DC2626;
}

.btn-secondary {
  background: var(--bg-color);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--hover-color);
}

/* 拒绝弹窗 */
.reject-form {
  padding: 8px 0;
}

.reject-hint {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 12px 0;
}

.reject-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: 14px;
  resize: vertical;
  box-sizing: border-box;
}

.reject-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.reject-input::placeholder {
  color: var(--text-secondary);
}
</style>
