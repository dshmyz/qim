<template>
  <ModalContainer
    :visible="visible"
    title="智能搜索笔记"
    width="560px"
    :show-footer="false"
    @close="$emit('close')"
    @cancel="$emit('close')"
  >
    <div class="search-box">
      <input
        ref="searchInputRef"
        v-model="query"
        type="text"
        placeholder="输入搜索内容（支持语义理解）..."
        @keyup.enter="handleSearch"
      />
      <button class="search-btn" :disabled="!query.trim() || searching" @click="handleSearch">
        <span v-if="searching" class="spinner"></span>
        <span v-else>搜索</span>
      </button>
    </div>

    <div v-if="searching" class="searching-hint">
      <span class="spinner"></span>
      正在语义检索笔记...
    </div>

    <div v-else-if="results.length > 0" class="results-list">
      <div class="results-count">找到 {{ results.length }} 条相关结果</div>
      <div
        v-for="(item, index) in results"
        :key="index"
        class="result-item"
        @click="selectNote(item)"
      >
        <div class="result-title">{{ item.title }}</div>
        <div class="result-content">{{ item.content }}</div>
        <div class="result-meta">
          <span class="result-score">相似度: {{ (item.score * 100).toFixed(1) }}%</span>
        </div>
      </div>
    </div>

    <div v-else-if="searched" class="empty-result">
      <p>未找到相关的笔记内容</p>
      <p class="hint">可以尝试换个搜索词，或者先创建一些笔记</p>
    </div>

    <div v-else class="empty-hint">
      <p>输入关键词，系统会自动理解语义并检索最相关的笔记内容</p>
    </div>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { useNotes } from '../../../composables/useNotes'
import type { NoteVectorSearchResult } from '../../../types/note'
import ModalContainer from '../../shared/ModalContainer.vue'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  select: [noteId: number]
}>()

const { searchNotesSemantic } = useNotes()

const searchInputRef = ref<HTMLInputElement | null>(null)
const query = ref('')
const results = ref<NoteVectorSearchResult[]>([])
const searching = ref(false)
const searched = ref(false)

// 弹窗打开时聚焦输入框
watch(() => props.visible, async (newVal) => {
  if (newVal) {
    query.value = ''
    results.value = []
    searched.value = false
    await nextTick()
    searchInputRef.value?.focus()
  }
})

async function handleSearch() {
  if (!query.value.trim() || searching.value) return

  searching.value = true
  searched.value = true
  try {
    results.value = await searchNotesSemantic(query.value.trim())
  } finally {
    searching.value = false
  }
}

function selectNote(item: NoteVectorSearchResult) {
  const noteId = parseInt(item.note_id, 10)
  if (!isNaN(noteId)) {
    emit('select', noteId)
  }
}
</script>

<style scoped>
.search-box {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.search-box input {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  font-size: var(--font-size-sm);
  outline: none;
  transition: border-color 0.2s;
  color: var(--text-primary, #111827);
  background: var(--bg-color, #f9fafb);
}

.search-box input:focus {
  border-color: var(--primary-color, #3385ff);
}

.search-btn {
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  background: var(--primary-color, #3385ff);
  color: #fff;
  font-size: var(--font-size-sm);
  cursor: pointer;
  white-space: nowrap;
  transition: opacity 0.2s;
}

.search-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.search-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.searching-hint {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #6b7280);
  font-size: var(--font-size-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.spinner {
  display: inline-block;
  width: 16px;
  height: 16px;
  border: 2px solid var(--border-color, #e5e7eb);
  border-top-color: var(--primary-color, #3385ff);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.results-count {
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #6b7280);
  margin-bottom: 4px;
}

.result-item {
  padding: 12px 16px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.result-item:hover {
  border-color: var(--primary-color, #3385ff);
  background: rgba(51, 133, 255, 0.05);
}

.result-title {
  font-weight: 600;
  font-size: var(--font-size-sm);
  color: var(--text-primary, #111827);
  margin-bottom: 4px;
}

.result-content {
  font-size: var(--font-size-xs);
  color: var(--text-secondary, #6b7280);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.result-meta {
  margin-top: 6px;
}

.result-score {
  font-size: var(--font-size-xxs);
  color: var(--primary-color, #3385ff);
}

.empty-result {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #6b7280);
}

.empty-result p {
  margin: 0 0 8px;
  font-size: var(--font-size-sm);
}

.empty-result .hint {
  font-size: var(--font-size-xs);
  opacity: 0.7;
}

.empty-hint {
  text-align: center;
  padding: 40px 0;
  color: var(--text-secondary, #6b7280);
  font-size: var(--font-size-sm);
}
</style>
