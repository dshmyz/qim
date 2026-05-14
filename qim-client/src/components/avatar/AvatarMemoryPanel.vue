<template>
  <div class="avatar-memory-panel">
    <div class="panel-header">
      <h4>分身记忆</h4>
      <div class="memory-count">
        <span class="count-value">{{ memories.length }}</span>
        <span class="count-label">条记忆</span>
      </div>
    </div>

    <div class="search-section">
      <input 
        v-model="searchQuery" 
        class="search-input" 
        placeholder="搜索记忆..."
        @input="handleSearch"
      />
      <i class="fas fa-search search-icon"></i>
    </div>

    <div class="memory-list" v-if="!loading && memories.length > 0">
      <div 
        v-for="memory in filteredMemories" 
        :key="memory.doc_id" 
        class="memory-item"
      >
        <div class="memory-content">
          <p>{{ memory.content }}</p>
        </div>
        <div class="memory-meta">
          <span class="memory-time">{{ formatTime(memory.metadata?.remembered_at) }}</span>
          <button class="forget-btn" @click="handleForgetMemory(memory)" title="删除记忆">
            <i class="fas fa-trash"></i>
          </button>
        </div>
      </div>
    </div>

    <div class="empty-state" v-else-if="!loading && memories.length === 0">
      <i class="fas fa-brain"></i>
      <p>分身还没有记忆</p>
      <small>分身会在对话中自动记住重要信息</small>
    </div>

    <div class="loading-state" v-if="loading">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { request } from '../../composables/useRequest'

function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return (...args: Parameters<T>) => {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    timeoutId = setTimeout(() => {
      func(...args)
    }, wait)
  }
}

interface MemoryRecord {
  doc_id: string
  content: string
  metadata: {
    conversation_id?: string
    remembered_at?: string
    type?: string
  }
}

const props = defineProps<{
  userId: number
}>()

const memories = ref<MemoryRecord[]>([])
const loading = ref(false)
const searchQuery = ref('')
const searchResults = ref<MemoryRecord[]>([])
const isSearching = ref(false)

const filteredMemories = computed(() => {
  if (isSearching.value && searchResults.value.length > 0) {
    return searchResults.value
  }
  return memories.value
})

async function loadMemories() {
  loading.value = true
  try {
    const data = await request<{ code: number; data: MemoryRecord[] }>('/api/v1/avatar/memories', { method: 'GET' })
    if (data && data.code === 0) {
      memories.value = data.data || []
    }
  } catch (e) {
    console.error('加载记忆失败', e)
  } finally {
    loading.value = false
  }
}

async function performSearch() {
  if (!searchQuery.value || searchQuery.value.length < 2) {
    isSearching.value = false
    searchResults.value = []
    return
  }

  try {
    const data = await request<{ code: number; data: MemoryRecord[] }>('/api/v1/avatar/memory/search', {
      method: 'POST',
      body: JSON.stringify({
        query: searchQuery.value,
        top_k: 10
      })
    })
    if (data && data.code === 0) {
      searchResults.value = data.data || []
      isSearching.value = true
    }
  } catch (e) {
    console.error('搜索记忆失败', e)
  }
}

const handleSearch = debounce(performSearch, 300)

async function handleForgetMemory(memory: MemoryRecord) {
  if (!confirm('确定要删除这条记忆吗？')) return

  try {
    const data = await request<{ code: number }>(`/api/v1/avatar/memory/${memory.doc_id}`, { method: 'DELETE' })
    if (data && data.code === 0) {
      memories.value = memories.value.filter(m => m.doc_id !== memory.doc_id)
    }
  } catch (e) {
    console.error('删除记忆失败', e)
  }
}

function formatTime(timestamp?: string): string {
  if (!timestamp) return '未知时间'
  const date = new Date(parseInt(timestamp) * 1000)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadMemories()
})
</script>

<style scoped>
.avatar-memory-panel {
  padding: 16px;
  background: var(--card-bg);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.panel-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.memory-count {
  display: flex;
  align-items: center;
  gap: 4px;
}

.count-value {
  font-size: 18px;
  font-weight: 600;
  color: var(--primary-color);
}

.count-label {
  font-size: 12px;
  color: var(--text-secondary);
}

.search-section {
  position: relative;
  margin-bottom: 16px;
}

.search-input {
  width: 100%;
  padding: 8px 12px 8px 32px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--bg-color);
  color: var(--text-color);
  font-size: 14px;
}

.search-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.search-icon {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-secondary);
  font-size: 14px;
}

.memory-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 400px;
  overflow-y: auto;
}

.memory-item {
  padding: 12px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
}

.memory-content p {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-color);
}

.memory-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}

.memory-time {
  font-size: 12px;
  color: var(--text-secondary);
}

.forget-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 4px;
  font-size: 12px;
}

.forget-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

.empty-state {
  text-align: center;
  padding: 32px;
  color: var(--text-secondary);
}

.empty-state i {
  font-size: 40px;
  margin-bottom: 8px;
  display: block;
}

.empty-state p {
  margin: 8px 0 4px;
  font-size: 14px;
}

.empty-state small {
  font-size: 12px;
  color: var(--text-secondary);
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  color: var(--text-secondary);
}

.loading-state i {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>
