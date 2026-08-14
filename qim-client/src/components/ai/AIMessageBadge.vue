<template>
  <div class="ai-message-badge" :class="{ compact }">
    <i class="fas fa-robot ai-badge-icon"></i>
    <span class="ai-generated-tag">由 AI助手 {{ assistantName || 'AI 助手' }} 生成</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  assistantName?: string
  compact?: boolean
}>()
</script>

<style scoped>
.ai-message-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(102, 126, 234, 0.05);
  /* color: rgba(102, 126, 234, 0.7); */
  font-size: var(--font-size-xxs);
  font-weight: 500;
}

.ai-message-badge.compact {
  padding: 1px 6px;
  font-size: var(--font-size-tiny);
  gap: 4px;
}

.ai-badge-icon {
  font-size: var(--font-size-xxxs);
  opacity: 0.8;
}

.ai-badge-text {
  font-weight: 500;
}

.ai-generated-tag {
  font-size: var(--font-size-tiny);
  opacity: 0.7;
  font-weight: 400;
}
</style>