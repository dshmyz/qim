<template>
  <button class="ai-quick-action" @click="$emit('click')" :title="tooltip">
    <span class="action-icon" v-html="icon"></span>
    <span class="action-label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
interface Props {
  /** SVG markup string for the icon */
  icon: string
  label: string
  tooltip?: string
}

interface Emits {
  (e: 'click'): void
}

defineProps<Props>()
defineEmits<Emits>()
</script>

<style scoped>
.ai-quick-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
  transition: all 0.2s;
}

.ai-quick-action:hover {
  border-color: var(--primary-color);
  background: var(--primary-light);
  transform: translateY(-1px);
}

.action-icon {
  display: inline-flex;
  align-items: center;
}

.action-icon :deep(svg) {
  width: 16px;
  height: 16px;
  fill: currentColor;
}

.action-label {
  font-weight: 500;
}
</style>
