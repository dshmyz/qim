<template>
  <div class="ai-message-badge" :class="{ compact }">
    <span class="ai-badge-icon">&#x1F916;</span>
    <span class="ai-badge-text">{{ assistantName || 'AI &#x52A9;&#x624B;' }}</span>
    <span v-if="!compact" class="ai-generated-tag">&#x7531; AI &#x751F;&#x6210;</span>
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
  padding: 4px 10px;
  border-radius: 12px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-size: 12px;
  font-weight: 500;
}

.ai-message-badge.compact {
  padding: 2px 8px;
  font-size: 11px;
}

.ai-badge-icon {
  display: flex;
  align-items: center;
  opacity: 0.9;
  font-size: 14px;
}

.ai-badge-text {
  font-weight: 600;
}

.ai-generated-tag {
  font-size: 11px;
  opacity: 0.8;
  font-weight: 400;
}

[data-theme="elegant-dark"] .ai-message-badge,
[data-theme="ocean-blue"] .ai-message-badge,
[data-theme="elegant-purple"] .ai-message-badge,
[data-theme="warm-amber"] .ai-message-badge,
[data-theme="crimson-red"] .ai-message-badge,
[data-theme="emerald-green"] .ai-message-badge {
  background: linear-gradient(135deg, #5a67d8 0%, #6b46c1 100%);
}
</style>
