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
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.modal-container {
  background: var(--card-bg);
  border-radius: var(--radius-xl);
  width: 90%;
  max-width: 520px;
  max-height: 80vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-xl);
  animation: slideUp 0.3s ease;
  border: 1px solid var(--border-color);
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-4) var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
}

.modal-header h3 {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-bold);
  color: white;
}

.close-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  color: white;
  cursor: pointer;
  font-size: var(--font-size-base);
  padding: var(--spacing-2);
  border-radius: var(--radius-md);
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: rotate(90deg);
}

.modal-body {
  padding: var(--spacing-5);
  overflow-y: auto;
}

.result-section {
  margin-bottom: var(--spacing-5);
}

.result-section:last-child {
  margin-bottom: 0;
}

.result-section h4 {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  margin: 0 0 var(--spacing-2) 0;
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.result-section h4::before {
  content: '';
  width: 4px;
  height: 16px;
  background: var(--primary-color);
  border-radius: 2px;
}

.summary-text {
  font-size: var(--font-size-sm);
  color: var(--text-color);
  line-height: var(--line-height-relaxed);
  margin: 0;
  padding: var(--spacing-3);
  background: var(--content-bg);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-2);
}

.tag-item {
  font-size: var(--font-size-sm);
  padding: var(--spacing-2) var(--spacing-3);
  background: var(--btn-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: var(--font-weight-medium);
}

.tag-item:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--primary-light);
  transform: scale(1.05);
}

.tag-item.selected {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  box-shadow: 0 2px 8px rgba(51, 133, 255, 0.3);
}

.no-tags {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.action-list {
  margin: 0;
  padding-left: var(--spacing-5);
}

.action-list li {
  font-size: var(--font-size-sm);
  color: var(--text-color);
  line-height: var(--line-height-relaxed);
  padding: var(--spacing-2) 0;
  border-bottom: 1px dashed var(--border-color);
}

.action-list li:last-child {
  border-bottom: none;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
  padding: var(--spacing-4) var(--spacing-5);
  border-top: 1px solid var(--border-color);
  background: var(--content-bg);
}

.modal-btn {
  padding: var(--spacing-2) var(--spacing-5);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
  transition: all var(--transition-base);
}

.modal-btn.cancel {
  background: var(--btn-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-btn.cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--primary-light);
}

.modal-btn.confirm {
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  color: white;
  border: none;
  box-shadow: 0 2px 8px rgba(51, 133, 255, 0.3);
}

.modal-btn.confirm:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(51, 133, 255, 0.4);
}
</style>
