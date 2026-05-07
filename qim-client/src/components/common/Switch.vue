<template>
  <button
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :aria-label="label || (modelValue ? '已开启' : '已关闭')"
    class="q-switch"
    :class="[
      `q-switch--${size}`,
      { 'q-switch--checked': modelValue, 'q-switch--disabled': disabled }
    ]"
    :disabled="disabled"
    @click="handleToggle"
  >
    <span class="q-switch__track">
      <span class="q-switch__thumb">
        <svg v-if="modelValue" class="q-switch__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
          <path d="M5 12l5 5L20 7" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <svg v-else class="q-switch__icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
          <path d="M18 6L6 18M6 6l12 12" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
      </span>
    </span>
    <span v-if="label" class="q-switch__label">{{ label }}</span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: boolean
  disabled?: boolean
  label?: string
  size?: 'small' | 'medium' | 'large'
}>(), {
  disabled: false,
  label: '',
  size: 'medium'
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
}>()

function handleToggle() {
  if (props.disabled) return
  const newValue = !props.modelValue
  emit('update:modelValue', newValue)
  emit('change', newValue)
}
</script>

<style scoped>
.q-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  background: transparent;
  cursor: pointer;
  padding: 0;
  outline: none;
  transition: transform 0.15s ease;
}

.q-switch:hover:not(:disabled) {
  transform: scale(1.02);
}

.q-switch:active:not(:disabled) {
  transform: scale(0.98);
}

.q-switch--disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.q-switch__track {
  position: relative;
  width: 44px;
  height: 24px;
  border-radius: 12px;
  background: linear-gradient(145deg, #e0e0e0, #bdbdbd);
  box-shadow: 
    inset 0 2px 4px rgba(0, 0, 0, 0.15),
    0 1px 2px rgba(0, 0, 0, 0.05);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.q-switch--small .q-switch__track {
  width: 36px;
  height: 20px;
  border-radius: 10px;
}

.q-switch--checked .q-switch__track {
  background: linear-gradient(145deg, var(--primary-color), var(--primary-color-dark, var(--primary-color)));
  box-shadow: 
    inset 0 2px 4px rgba(0, 0, 0, 0.2),
    0 4px 12px rgba(var(--primary-color-rgb, 66, 153, 225), 0.4);
}

.q-switch__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: linear-gradient(145deg, #ffffff, #f0f0f0);
  box-shadow: 
    0 2px 6px rgba(0, 0, 0, 0.15),
    0 1px 2px rgba(0, 0, 0, 0.05);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  align-items: center;
  justify-content: center;
}

.q-switch--small .q-switch__thumb {
  width: 16px;
  height: 16px;
}

.q-switch--checked .q-switch__thumb {
  left: 22px;
  background: linear-gradient(145deg, #ffffff, #fafafa);
  box-shadow: 
    0 2px 6px rgba(0, 0, 0, 0.2),
    0 1px 2px rgba(0, 0, 0, 0.1);
}

.q-switch--small.q-switch--checked .q-switch__thumb {
  left: 18px;
}

.q-switch__icon {
  width: 10px;
  height: 10px;
  transition: all 0.2s ease;
}

.q-switch--small .q-switch__icon {
  width: 8px;
  height: 8px;
}

.q-switch:not(.q-switch--checked) .q-switch__icon {
  color: #9e9e9e;
}

.q-switch--checked .q-switch__icon {
  color: var(--primary-color);
}

.q-switch__label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  user-select: none;
}
</style>
