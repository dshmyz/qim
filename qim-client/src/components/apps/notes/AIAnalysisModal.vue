<template>
  <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
    <div class="modal-container">
      <div class="modal-header">
        <h3>AI 分析结果</h3>
        <button class="close-btn" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>

      <div class="modal-body">
        <div class="result-section">
          <h4>摘要</h4>
          <p class="summary-text">{{ result?.summary || '暂无摘要' }}</p>
        </div>

        <div class="result-section">
          <h4>推荐标签</h4>
          <div class="tags-container">
            <span
              v-for="tag in result?.tags || []"
              :key="tag"
              :class="['tag-item', { selected: selectedTags.includes(tag) }]"
              @click="toggleTag(tag)"
            >
              {{ tag }}
            </span>
            <span v-if="!result?.tags?.length" class="no-tags">暂无推荐标签</span>
          </div>
        </div>

        <div class="result-section" v-if="result?.action_items?.length">
          <h4>提取的行动项</h4>
          <ul class="action-list">
            <li v-for="(item, index) in result.action_items" :key="index">
              {{ item }}
            </li>
          </ul>
        </div>
      </div>

      <div class="modal-footer">
        <button class="modal-btn cancel" @click="$emit('close')">取消</button>
        <button class="modal-btn confirm" @click="handleConfirm">
          保存摘要和标签
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AIAnalyzeResult } from '../../../types/note'

const props = defineProps<{
  visible: boolean
  result: AIAnalyzeResult | null
}>()

const emit = defineEmits<{
  close: []
  confirm: [summary: string, tags: string[]]
}>()

const selectedTags = ref<string[]>([])

watch(() => props.result, (newResult) => {
  if (newResult?.tags) {
    selectedTags.value = [...newResult.tags]
  }
}, { immediate: true })

function toggleTag(tag: string) {
  const index = selectedTags.value.indexOf(tag)
  if (index > -1) {
    selectedTags.value.splice(index, 1)
  } else {
    selectedTags.value.push(tag)
  }
}

function handleConfirm() {
  emit('confirm', props.result?.summary || '', selectedTags.value)
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-container {
  background: var(--bg-color);
  border-radius: 12px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 18px;
  color: var(--text-primary);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 18px;
  padding: 4px;
}

.close-btn:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 20px;
  overflow-y: auto;
}

.result-section {
  margin-bottom: 16px;
}

.result-section:last-child {
  margin-bottom: 0;
}

.result-section h4 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
}

.summary-text {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0;
  padding: 12px;
  background: var(--hover-color);
  border-radius: 6px;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  font-size: 13px;
  padding: 6px 12px;
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.tag-item.selected {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.no-tags {
  font-size: 13px;
  color: var(--text-tertiary);
}

.action-list {
  margin: 0;
  padding-left: 20px;
}

.action-list li {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.8;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
}

.modal-btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.modal-btn.cancel {
  background: var(--bg-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-btn.cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.modal-btn.confirm {
  background: var(--primary-color);
  color: white;
  border: none;
}

.modal-btn.confirm:hover {
  opacity: 0.9;
}
</style>
