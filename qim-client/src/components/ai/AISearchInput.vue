<template>
  <div class="ai-search-input">
    <div class="search-wrapper" :class="{ 'focused': isFocused, 'processing': isProcessing }">
      <svg class="search-icon" viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
        <path d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
      </svg>
      <input
        v-model="query"
        type="text"
        :placeholder="placeholderText"
        @keyup.enter="handleSearch"
        @focus="isFocused = true"
        @blur="handleBlur"
      />
      <button v-if="query" class="clear-btn" @click="clear">&times;</button>
      <div v-if="isProcessing" class="search-spinner"></div>
    </div>

    <AISearchResults
      v-if="showResults && results.length > 0"
      :results="results"
      :conversation-id="conversationId"
      @select="handleSelect"
      @close="showResults = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import AISearchResults from './AISearchResults.vue'

const props = defineProps<{
  conversationId: number
  placeholder?: string
}>()

const emit = defineEmits<{
  select: [result: any]
}>()

const { searchMessages, isProcessing } = useAIActions()
const query = ref('')
const results = ref<any[]>([])
const showResults = ref(false)
const isFocused = ref(false)

const placeholderText = computed(() =>
  isProcessing.value ? '搜索中...' : (props.placeholder || '使用 AI 搜索消息...')
)

const handleSearch = async () => {
  if (!query.value.trim()) return

  try {
    const data = await searchMessages(props.conversationId, query.value.trim())
    results.value = data.results || []
    showResults.value = true
  } catch {
    // 错误处理已由 composable 处理
  }
}

const handleSelect = (result: any) => {
  emit('select', result)
  showResults.value = false
}

const clear = () => {
  query.value = ''
  results.value = []
  showResults.value = false
}

const handleBlur = () => {
  setTimeout(() => {
    showResults.value = false
  }, 200)
}
</script>

<style scoped>
.ai-search-input {
  position: relative;
  width: 100%;
}

.search-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  transition: border-color 0.2s;
}

.search-wrapper.focused {
  border-color: var(--primary-color);
}

.search-icon {
  color: var(--text-secondary);
  flex-shrink: 0;
}

.search-wrapper input {
  flex: 1;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
}

.clear-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 18px;
  cursor: pointer;
  padding: 0 4px;
}

.clear-btn:hover {
  color: var(--text-primary);
}

.search-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  flex-shrink: 0;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
