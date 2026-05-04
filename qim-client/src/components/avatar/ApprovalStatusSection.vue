<template>
  <div class="approval-status-section">
    <div class="approval-status-header">
      <span class="status-label">审批状态</span>
      <span :class="['status-badge', approvalStatus]">
        <i :class="statusIcon"></i>
        {{ statusText }}
      </span>
    </div>

    <!-- 未申请状态 -->
    <div v-if="approvalStatus === 'none'" class="approval-action">
      <p class="approval-hint">分身功能需要管理员审批后才能启用</p>
      <button class="btn btn-primary" @click="$emit('apply')" :disabled="applying">
        <i class="fas fa-paper-plane"></i>
        {{ applying ? '申请中...' : '申请启用' }}
      </button>
    </div>

    <!-- 待审批状态 -->
    <div v-else-if="approvalStatus === 'pending'" class="approval-action">
      <p class="approval-hint">您的申请已提交，请等待管理员审批</p>
      <p class="approval-time" v-if="appliedAt">
        申请时间：{{ formatDate(appliedAt) }}
      </p>
      <button class="btn btn-secondary" @click="$emit('cancel')" :disabled="applying">
        <i class="fas fa-times"></i>
        取消申请
      </button>
    </div>

    <!-- 已通过状态 -->
    <div v-else-if="approvalStatus === 'approved'" class="approval-action approved">
      <p class="approval-hint success">
        <i class="fas fa-check-circle"></i>
        您的分身功能已通过审批，可以启用
      </p>
    </div>

    <!-- 已拒绝状态 -->
    <div v-else-if="approvalStatus === 'rejected'" class="approval-action rejected">
      <div class="reject-reason">
        <p class="approval-hint error">
          <i class="fas fa-exclamation-circle"></i>
          您的申请已被拒绝
        </p>
        <p class="reason-text" v-if="rejectReason">
          拒绝原因：{{ rejectReason }}
        </p>
        <p class="approval-time" v-if="approvedAt">
          审批时间：{{ formatDate(approvedAt) }}
        </p>
      </div>
      <button class="btn btn-primary" @click="$emit('apply')" :disabled="applying">
        <i class="fas fa-redo"></i>
        {{ applying ? '申请中...' : '重新申请' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AvatarApprovalStatus } from '../../types/avatar'
import { formatDate } from '../../utils/date'

const props = defineProps<{
  approvalStatus: AvatarApprovalStatus
  rejectReason?: string | null
  appliedAt?: string | null
  approvedAt?: string | null
  applying: boolean
}>()

defineEmits<{
  apply: []
  cancel: []
}>()

// 状态文本
const statusText = computed(() => {
  const texts: Record<AvatarApprovalStatus, string> = {
    none: '未申请',
    pending: '审批中',
    approved: '已通过',
    rejected: '已拒绝'
  }
  return texts[props.approvalStatus]
})

// 状态图标
const statusIcon = computed(() => {
  const icons: Record<AvatarApprovalStatus, string> = {
    none: 'fas fa-minus-circle',
    pending: 'fas fa-clock',
    approved: 'fas fa-check-circle',
    rejected: 'fas fa-times-circle'
  }
  return icons[props.approvalStatus]
})
</script>

<style scoped>
/* 审批状态区域 */
.approval-status-section {
  background: var(--hover-color);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}

.approval-status-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.status-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 500;
}

.status-badge.none {
  background: var(--color-gray-100);
  color: var(--text-secondary);
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

.approval-action {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.approval-hint {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0;
}

.approval-hint.success {
  color: #10B981;
}

.approval-hint.error {
  color: #EF4444;
}

.approval-time {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0;
}

.reject-reason {
  margin-bottom: 8px;
}

.reason-text {
  font-size: 13px;
  color: var(--text-secondary);
  background: rgba(239, 68, 68, 0.05);
  padding: 8px 12px;
  border-radius: 6px;
  margin: 8px 0 0 0;
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

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-color);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--hover-color);
}
</style>
