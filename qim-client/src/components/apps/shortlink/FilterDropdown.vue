<template>
  <select
    v-model="selectedValue"
    class="filter-dropdown"
    @change="handleChange"
  >
    <option v-for="option in options" :key="option.value" :value="option.value">
      {{ option.label }}
    </option>
  </select>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

export interface FilterOption {
  label: string
  value: string
}

const props = defineProps<{
  modelValue?: string
  options: FilterOption[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()

const selectedValue = ref(props.modelValue || props.options[0]?.value || '')

watch(() => props.modelValue, (newValue) => {
  selectedValue.value = newValue || ''
})

const handleChange = () => {
  emit('update:modelValue', selectedValue.value)
  emit('change', selectedValue.value)
}
</script>

<style scoped>
.filter-dropdown {
  padding: 10px 16px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg, white);
  color: var(--text-primary, #1f2937);
  cursor: pointer;
  transition: all 0.2s;
  min-width: 120px;
}

.filter-dropdown:focus {
  outline: none;
  border-color: var(--accent-color, #667eea);
}

.filter-dropdown:hover {
  border-color: var(--accent-color, #667eea);
}
</style>
