<template>
  <div class="message-status" :class="statusClass">
    <!-- 发送中状态 -->
    <template v-if="status === 'sending'">
      <i class="fas fa-spinner fa-spin"></i>
      <span>发送中</span>
    </template>

    <!-- 已发送状态 -->
    <template v-else-if="status === 'sent'">
      <i class="fas fa-check"></i>
      <span>已发送</span>
    </template>

    <!-- 已读状态 -->
    <template v-else-if="status === 'read'">
      <i class="fas fa-check-double"></i>
      <span>
        {{ readCount !== undefined && totalCount !== undefined ? `${readCount}/${totalCount}人已读` : '已读' }}
      </span>
    </template>

    <!-- 发送失败状态 -->
    <template v-else-if="status === 'failed'">
      <i class="fas fa-exclamation-circle"></i>
      <span>发送失败</span>
      <span class="retry-btn" @click.stop="$emit('retry')" title="重新发送">
        <i class="fas fa-redo"></i>
      </span>
    </template>
  </div>
</template>

<script setup lang="ts">
export type MessageStatusType = 'sending' | 'sent' | 'failed' | 'read'

interface Props {
  /** 消息状态 */
  status: MessageStatusType
  /** 是否为自身发送的消息 */
  isSelf?: boolean
  /** 是否已读 */
  isRead?: boolean
  /** 已读人数 */
  readCount?: number
  /** 总人数 */
  totalCount?: number
}

const props = withDefaults(defineProps<Props>(), {
  isSelf: true,
  isRead: false,
  readCount: undefined,
  totalCount: undefined
})

const emit = defineEmits<{
  /** 重新发送消息事件 */
  retry: []
}>()

/**
 * 计算状态对应的 CSS 类名
 */
const statusClass = computed(() => {
  return {
    'message-status--sending': props.status === 'sending',
    'message-status--sent': props.status === 'sent',
    'message-status--read': props.status === 'read',
    'message-status--failed': props.status === 'failed'
  }
})
</script>

<style scoped>
.message-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  line-height: 1;
  color: var(--text-secondary, #999);
  opacity: 0.8;
  transition: all 0.2s ease;
}

.message-status i {
  font-size: 10px;
}

/* 发送中状态 */
.message-status--sending {
  color: var(--text-secondary, #999);
  opacity: 1;
}

.message-status--sending .fa-spinner {
  color: var(--primary-color, #3b82f6);
}

/* 已发送状态 */
.message-status--sent {
  color: var(--text-secondary, #999);
}

.message-status--sent .fa-check {
  color: var(--text-secondary, #999);
}

/* 已读状态 */
.message-status--read {
  color: var(--success-color, #4caf50);
  opacity: 1;
}

.message-status--read .fa-check-double {
  color: var(--success-color, #4caf50);
}

/* 发送失败状态 */
.message-status--failed {
  color: var(--error-color, #f56c6c);
  opacity: 1;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.message-status--failed .fa-exclamation-circle {
  color: var(--error-color, #f56c6c);
}

/* 重新发送按钮 */
.retry-btn {
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 2px 4px;
  border-radius: 3px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.retry-btn:hover {
  color: var(--error-color-hover, #f78989);
  background-color: var(--error-color-hover-bg, rgba(245, 108, 108, 0.1));
  transform: scale(1.1);
}

.retry-btn:active {
  transform: scale(0.95);
}

/* 暗黑主题适配 */
[data-theme="elegant-dark"] .message-status {
  color: var(--text-secondary, #888);
}

[data-theme="elegant-dark"] .message-status--sending .fa-spinner {
  color: var(--primary-color, #3b82f6);
}

[data-theme="ocean-blue"] .message-status--read .fa-check-double {
  color: var(--success-color, #4caf50);
}

[data-theme="elegant-purple"] .message-status--read {
  color: var(--success-color, #4caf50);
}

[data-theme="warm-amber"] .message-status--failed .fa-exclamation-circle {
  color: var(--error-color, #f56c6c);
}

[data-theme="crimson-red"] .message-status--read .fa-check-double {
  color: var(--success-color, #4caf50);
}

[data-theme="emerald-green"] .message-status--sent .fa-check {
  color: var(--success-color, #10b981);
}
</style>
