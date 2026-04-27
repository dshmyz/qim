<template>
  <div class="ai-quick-actions">
    <AIQuickActionItem
      v-for="action in actions"
      :key="action.id"
      :icon="action.icon"
      :label="action.label"
      :tooltip="action.tooltip"
      @click="handleAction(action.id)"
    />
    <div v-if="isProcessing" class="ai-processing">
      <span class="processing-spinner"></span>
      <span>处理中...</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AIQuickActionItem from './AIQuickActionItem.vue'

interface AIAction {
  id: string
  icon: string
  label: string
  tooltip?: string
}

interface Props {
  actions?: AIAction[]
  isProcessing: boolean
}

interface Emits {
  (e: 'action', actionId: string): void
}

const props = withDefaults(defineProps<Props>(), {
  actions: undefined
})

const emit = defineEmits<Emits>()

const defaultActions: AIAction[] = [
  { id: 'summary', icon: '📝', label: '总结对话', tooltip: '总结当前会话内容' },
  { id: 'translate', icon: '🌐', label: '翻译', tooltip: '翻译选中文本' },
  { id: 'rewrite', icon: '✍️', label: '改写', tooltip: '改写输入框中的文本' },
  { id: 'polish', icon: '✨', label: '润色', tooltip: '润色文本语气和表达' },
  { id: 'code_review', icon: '🔍', label: '代码审查', tooltip: '审查代码片段' },
]

const actions = computed(() => props.actions || defaultActions)

const handleAction = (actionId: string) => {
  emit('action', actionId)
}
</script>

<style scoped>
.ai-quick-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--card-bg);
  border-top: 1px solid var(--border-color);
  overflow-x: auto;
  scrollbar-width: thin;
}

.ai-processing {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  color: var(--text-secondary);
  font-size: 13px;
}

.processing-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
