<template>
  <Transition name="preview-fade">
    <div v-if="visible" class="ai-preview-overlay" @click.self="emit('cancel')">
      <div class="ai-preview-dialog">
        <div class="ai-preview-header">
          <span class="ai-preview-title">
            <i :class="iconClass"></i>
            {{ title }}
          </span>
          <button class="ai-preview-close" @click="emit('cancel')">
            <i class="fas fa-times"></i>
          </button>
        </div>
        <div class="ai-preview-body">
          <div class="ai-preview-section">
            <div class="ai-preview-label">原文</div>
            <div class="ai-preview-text original">{{ originalText }}</div>
          </div>
          <div class="ai-preview-arrow">
            <i class="fas fa-arrow-down"></i>
          </div>
          <div class="ai-preview-section">
            <div class="ai-preview-label">结果</div>
            <div class="ai-preview-text result">{{ resultText }}</div>
          </div>
        </div>
        <div class="ai-preview-footer">
          <button class="ai-preview-btn cancel" @click="emit('cancel')">取消</button>
          <button class="ai-preview-btn replace" @click="emit('replace')">
            <i class="fas fa-check"></i>
            替换原文
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  visible: boolean
  originalText: string
  resultText: string
  actionType: 'rewrite' | 'polish' | 'translate'
}

const emit = defineEmits<{
  replace: []
  cancel: []
}>()

const props = defineProps<Props>()

const titleMap = {
  rewrite: '改写结果',
  polish: '润色结果',
  translate: '翻译结果',
}

const iconMap = {
  rewrite: 'fas fa-pen-fancy',
  polish: 'fas fa-star',
  translate: 'fas fa-language',
}

const title = computed(() => titleMap[props.actionType] || '结果')
const iconClass = computed(() => iconMap[props.actionType] || 'fas fa-check')
</script>

<style scoped>
.ai-preview-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.ai-preview-dialog {
  width: 520px;
  max-width: 90vw;
  max-height: 80vh;
  background: var(--card-bg, #fff);
  border-radius: 12px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ai-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.ai-preview-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.ai-preview-title i {
  color: var(--primary-color, #6366f1);
}

.ai-preview-close {
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #6b7280);
  transition: all 0.15s;
}

.ai-preview-close:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #1f2937);
}

.ai-preview-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-preview-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ai-preview-label {
  font-size: var(--font-size-xxs);
  font-weight: 600;
  color: var(--text-secondary, #9ca3af);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.ai-preview-text {
  padding: 12px 14px;
  border-radius: 8px;
  font-size: var(--font-size-sm);
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
}

.ai-preview-text.original {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-secondary, #6b7280);
}

.ai-preview-text.result {
  background: color-mix(in srgb, var(--primary-color, #6366f1) 6%, transparent);
  color: var(--text-color, #1f2937);
  border: 1px solid color-mix(in srgb, var(--primary-color, #6366f1) 20%, transparent);
}

.ai-preview-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #9ca3af);
  font-size: var(--font-size-sm);
}

.ai-preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid var(--border-color, #e5e7eb);
}

.ai-preview-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s;
}

.ai-preview-btn.cancel {
  background: transparent;
  color: var(--text-secondary, #6b7280);
  border: 1px solid var(--border-color, #e5e7eb);
}

.ai-preview-btn.cancel:hover {
  background: var(--hover-color, #f3f4f6);
}

.ai-preview-btn.replace {
  background: var(--primary-color, #6366f1);
  color: #fff;
}

.ai-preview-btn.replace:hover {
  opacity: 0.9;
}

/* Transition */
.preview-fade-enter-active,
.preview-fade-leave-active {
  transition: opacity 0.2s;
}

.preview-fade-enter-from,
.preview-fade-leave-to {
  opacity: 0;
}

.preview-fade-enter-active .ai-preview-dialog,
.preview-fade-leave-active .ai-preview-dialog {
  transition: transform 0.2s;
}

.preview-fade-enter-from .ai-preview-dialog,
.preview-fade-leave-to .ai-preview-dialog {
  transform: scale(0.95) translateY(10px);
}
</style>
