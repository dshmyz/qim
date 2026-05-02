<template>
  <div class="file-search-bar">
    <i class="fas fa-search search-icon"></i>
    <input
      ref="inputRef"
      type="text"
      class="search-input"
      :value="modelValue"
      :placeholder="placeholder"
      @input="handleInput"
      @keydown.escape="handleClear"
    />
    <button
      v-if="modelValue"
      class="clear-btn"
      @click="handleClear"
    >
      <i class="fas fa-times"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

defineOptions({
  name: 'FileSearchBar'
})

interface Props {
  modelValue: string
  placeholder?: string
}

withDefaults(defineProps<Props>(), {
  placeholder: '搜索文件名...'
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'search', value: string): void
  (e: 'clear'): void
}>()

const inputRef = ref<HTMLInputElement | null>(null)

let debounceTimer: ReturnType<typeof setTimeout> | null = null

const handleInput = (event: Event) => {
  const value = (event.target as HTMLInputElement).value
  emit('update:modelValue', value)

  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    emit('search', value)
  }, 300)
}

const handleClear = () => {
  emit('update:modelValue', '')
  emit('search', '')
  emit('clear')
  inputRef.value?.focus()
}

defineExpose({
  focus: () => inputRef.value?.focus()
})
</script>

<style scoped>
.file-search-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border: 1px solid transparent;
  border-radius: 16px;
  background: var(--hover-color, #f0f2f5);
  transition: all 0.2s ease;
  flex: 1;
  max-width: 220px;
  min-width: 120px;
}

.file-search-bar:focus-within {
  border-color: var(--primary-color, #4f6ef7);
  background: var(--card-bg, #fff);
  box-shadow: 0 0 0 2px rgba(79, 110, 247, 0.08);
}

.search-icon {
  color: var(--text-secondary, #8c95a6);
  font-size: 12px;
  flex-shrink: 0;
}

.search-input {
  border: none;
  background: transparent;
  color: var(--text-color, #4a5568);
  font-size: 12px;
  outline: none;
  width: 100%;
  min-width: 0;
}

.search-input::placeholder {
  color: var(--text-secondary, #8c95a6);
}

.clear-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  background: var(--hover-color, #f0f2f5);
  border-radius: 50%;
  cursor: pointer;
  color: var(--text-secondary, #8c95a6);
  font-size: 9px;
  flex-shrink: 0;
  transition: all 0.15s ease;
}

.clear-btn:hover {
  background: var(--border-color, #e8ecf0);
  color: var(--text-color, #4a5568);
}

@media (max-width: 768px) {
  .file-search-bar {
    max-width: none;
    min-width: 0;
  }
}
</style>
