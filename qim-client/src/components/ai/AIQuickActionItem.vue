<template>
  <button
    class="ai-quick-action"
    :class="{ 'is-loading': loading, disabled }"
    :title="tooltip"
    @click="handleClick"
  >
    <span class="action-icon" v-html="icon"></span>
    <span class="action-label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
interface Props {
  icon: string
  label: string
  tooltip?: string
  loading?: boolean
  disabled?: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'click'): void }>()

const handleClick = () => {
  if (props.disabled || props.loading) return
  emit('click')
}
</script>

<style scoped>
.ai-quick-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 10px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  font-family: inherit;
  font-size: var(--font-size-xxs);
  line-height: 1;
  white-space: nowrap;
  outline: none;
  appearance: none;
  -webkit-appearance: none;
  box-sizing: border-box;
  flex-shrink: 0;
  transform: none;
  transition: opacity 0.2s;
}

.ai-quick-action:hover,
.ai-quick-action:active,
.ai-quick-action:focus {
  transform: none;
}

.ai-quick-action:hover:not(.disabled):not(.is-loading) {
  border-color: var(--primary-color);
  background: var(--primary-light);
}

.ai-quick-action.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.ai-quick-action.is-loading {
  cursor: wait;
  animation: btn-pulse 1.2s ease-in-out infinite;
}

@keyframes btn-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

.action-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  overflow: hidden;
}

.action-icon :deep(svg) {
  width: 14px;
  height: 14px;
  fill: currentColor;
  display: block;
}

.action-label {
  font-weight: 500;
  line-height: 1;
}
</style>
