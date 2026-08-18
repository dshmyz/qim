<template>
  <ModalContainer
    :visible="visible"
    title="AI 格式化预览"
    width="860px"
    min-width="640px"
    :show-footer="false"
    @close="$emit('close')"
  >
    <div class="ai-format-content">
      <div class="format-hint" v-if="truncated">
        <i class="fas fa-exclamation-triangle"></i>
        笔记内容较长，仅格式化了前一部分；确认替换后剩余内容保持不变。
      </div>
      <div class="format-panes">
        <div class="format-pane">
          <div class="pane-header">原文</div>
          <pre class="pane-original">{{ original }}</pre>
        </div>
        <div class="format-pane">
          <div class="pane-header">格式化后</div>
          <div class="pane-formatted markdown-content" v-html="formattedHtml"></div>
        </div>
      </div>
      <div class="action-buttons">
        <button class="btn-cancel" @click="$emit('close')">取消</button>
        <button class="btn-confirm" @click="$emit('confirm')" :disabled="!formatted">
          确认替换
        </button>
      </div>
    </div>
  </ModalContainer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModalContainer from '../../shared/ModalContainer.vue'
import { renderMarkdown } from '../../../composables/useMarkdownRender'
import '../../message/markdown-content.css'
const props = defineProps<{
  visible: boolean
  original: string
  formatted: string
  truncated?: boolean
}>()

const emit = defineEmits<{
  close: []
  confirm: []
}>()

// 格式化结果走统一 markdown 渲染（与 IM/AI 气泡同一份排版）
const formattedHtml = computed(() => renderMarkdown(props.formatted || ''))
</script>

<style scoped>
.ai-format-content {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-4);
}

.format-hint {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  font-size: var(--font-size-sm);
  color: var(--warning-color);
  background: var(--warning-bg);
  border-radius: var(--radius-md);
}

.format-panes {
  display: flex;
  gap: var(--spacing-4);
  min-height: 360px;
  max-height: 52vh;
}

.format-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
  min-width: 0;
}

.pane-header {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.pane-header::before {
  content: '';
  width: 4px;
  height: 14px;
  background: var(--primary-color);
  border-radius: 2px;
}

.pane-original {
  flex: 1;
  margin: 0;
  padding: var(--spacing-3);
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: var(--font-size-sm);
  font-family: var(--font-family-mono);
  line-height: var(--line-height-relaxed);
  color: var(--text-secondary);
  background: var(--content-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.pane-formatted {
  flex: 1;
  overflow: auto;
  padding: var(--spacing-3);
  background: var(--content-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
}

.btn-cancel {
  padding: var(--spacing-2) var(--spacing-5);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  background: transparent;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
}

.btn-cancel:hover {
  color: var(--text-color);
  border-color: var(--border-color-hover);
}

.btn-confirm {
  padding: var(--spacing-2) var(--spacing-5);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: #fff;
  background: var(--primary-color);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.btn-confirm:hover {
  filter: brightness(1.1);
}

.btn-confirm:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
