<template>
  <div class="ai-bubble" :class="{ 'is-self': isSelf }">
    <div class="ai-header">
      <span class="ai-icon">🤖</span>
      <span class="ai-label">AI 助手</span>
    </div>
    <div class="ai-content">
      <TextMessage :content="content" :is-self="false" />
    </div>
    <div class="ai-footer">
      <span class="ai-source">基于知识库</span>
      <div class="ai-actions">
        <button class="ai-action-btn" @click="$emit('copy', content)" title="复制">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import TextMessage from './TextMessage.vue'

defineProps<{
  content: string
  isSelf?: boolean
}>()

defineEmits<{
  (e: 'copy', content: string): void
}>()
</script>

<style scoped>
.ai-bubble {
  max-width: 80%;
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid rgba(14, 165, 233, 0.2);
  background: rgba(14, 165, 233, 0.05);
}

.ai-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(14, 165, 233, 0.1);
  border-bottom: 1px solid rgba(14, 165, 233, 0.1);
}

.ai-icon {
  font-size: 16px;
}

.ai-label {
  font-size: 12px;
  font-weight: 600;
  color: #0ea5e9;
}

.ai-content {
  padding: 8px 12px;
}

.ai-content :deep(.text-message) {
  background: transparent;
  padding: 0;
  margin: 0;
}

.ai-content :deep(.text-message p) {
  margin: 0;
  padding: 0;
}

.ai-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px 8px;
}

.ai-source {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.ai-actions {
  display: flex;
  gap: 4px;
}

.ai-action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
}

.ai-action-btn:hover {
  background: rgba(14, 165, 233, 0.1);
  color: #0ea5e9;
  border-color: #0ea5e9;
}
</style>
