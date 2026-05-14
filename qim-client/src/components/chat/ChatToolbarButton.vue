<template>
  <button
    class="toolbar-btn"
    :class="buttonClass"
    :title="title"
    @click="$emit('click')"
    :disabled="disabled"
  >
    <i :class="icon"></i>
    <span v-if="$slots.default" class="toolbar-btn-text">
      <slot></slot>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  icon: string
  title?: string
  disabled?: boolean
  variant?: 'default' | 'primary' | 'ai'
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  variant: 'default'
})

defineEmits<{
  click: []
}>()

const buttonClass = computed(() => ({
  'toolbar-btn--primary': props.variant === 'primary',
  'toolbar-btn--ai': props.variant === 'ai',
  'toolbar-btn--disabled': props.disabled
}))
</script>

<style scoped>
.toolbar-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  color: var(--text-secondary, #666);
  font-size: 14px;
  transition: all 0.2s ease;
}

.toolbar-btn:hover:not(:disabled) {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
  color: var(--text-primary, #333);
}

.toolbar-btn:active:not(:disabled) {
  transform: scale(0.95);
}

.toolbar-btn--primary {
  background: var(--primary-color, #007AFF);
  color: white;
}

.toolbar-btn--primary:hover:not(:disabled) {
  background: var(--primary-color-hover, #0056CC);
  color: white;
}

.toolbar-btn--ai {
  color: var(--ai-color, #10A37F);
}

.toolbar-btn--ai:hover:not(:disabled) {
  background: rgba(16, 163, 127, 0.1);
}

.toolbar-btn--ai.ai-active {
  background: rgba(16, 163, 127, 0.15);
  color: var(--ai-color, #10A37F);
}

.toolbar-btn--disabled,
.toolbar-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.toolbar-btn-text {
  font-size: 13px;
}
</style>
