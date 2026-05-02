<template>
  <div class="search-bar">
    <div class="search-input-wrapper">
      <input
        v-model="searchQuery"
        type="text"
        class="search-input"
        placeholder="搜索链接..."
        @input="handleSearch"
      />
      <div class="search-icon">
        <i class="fas fa-search"></i>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  search: [query: string]
}>()

const searchQuery = ref(props.modelValue || '')

watch(() => props.modelValue, (newValue) => {
  searchQuery.value = newValue || ''
})

const handleSearch = () => {
  emit('update:modelValue', searchQuery.value)
  emit('search', searchQuery.value)
}
</script>

<style scoped>
.search-bar {
  flex: 1;
}

.search-input-wrapper {
  position: relative;
  width: 100%;
}

.search-input {
  width: 100%;
  padding: 10px 16px 10px 40px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  font-size: 14px;
  background: var(--input-bg, white);
  color: var(--text-primary, #1f2937);
  transition: all 0.2s;
}

.search-input:focus {
  outline: none;
  border-color: var(--accent-color, #667eea);
  box-shadow: 0 0 0 3px var(--accent-shadow-light, rgba(102, 126, 234, 0.1));
}

.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary, #9ca3af);
  font-size: 14px;
  pointer-events: none;
}
</style>
