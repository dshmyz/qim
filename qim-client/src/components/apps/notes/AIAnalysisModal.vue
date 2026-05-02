<template>
  <ModalContainer
    :visible="visible"
    title="AI 分析结果"
    width="520px"
    :show-footer="false"
    @close="$emit('close')"
  >
    <div class="ai-analysis-content">
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

      <div class="action-buttons">
        <button class="btn-cancel" @click="$emit('close')">取消</button>
        <button class="btn-confirm" @click="handleConfirm">
          保存摘要和标签
        </button>
      </div>
    </div>
  </ModalContainer>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import ModalContainer from '../../shared/ModalContainer.vue'
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
.ai-analysis-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-5);
}

.result-section {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.result-section h4 {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  margin: 0;
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

.action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
  padding-top: var(--spacing-4);
  border-top: 1px solid var(--border-color);
  margin-top: var(--spacing-2);
}

.btn-cancel,
.btn-confirm {
  padding: var(--spacing-2) var(--spacing-5);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  cursor: pointer;
  transition: all var(--transition-base);
}

.btn-cancel {
  background: var(--btn-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.btn-cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  background: var(--primary-light);
}

.btn-confirm {
  background: linear-gradient(135deg, var(--primary-color), var(--color-primary-600));
  color: white;
  border: none;
  box-shadow: 0 2px 8px rgba(51, 133, 255, 0.3);
}

.btn-confirm:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(51, 133, 255, 0.4);
}
</style>
