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
  { id: 'summary', icon: '<svg viewBox="0 0 24 24"><path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a1.003 1.003 0 000-1.42l-2.34-2.34a1.003 1.003 0 00-1.42 0l-1.83 1.83 3.75 3.75 1.84-1.82z"/></svg>', label: '总结对话', tooltip: '总结当前会话内容' },
  { id: 'translate', icon: '<svg viewBox="0 0 24 24"><path d="M12.87 15.07l-2.54-2.51.03-.03A17.52 17.52 0 0014.07 6H17V4h-7V2H8v2H1v2h11.17C11.5 7.92 10.44 9.75 9 11.35 8.07 10.32 7.3 9.19 6.69 8h-2c.73 1.63 1.73 3.17 2.98 4.56l-5.09 5.02L4 19l5-5 3.11 3.11.76-2.04zM18.5 10h-2L12 22h2l1.12-3h4.75L21 22h2l-4.5-12zm-2.62 7l1.62-4.33L19.12 17h-3.24z"/></svg>', label: '翻译', tooltip: '翻译选中文本' },
  { id: 'rewrite', icon: '<svg viewBox="0 0 24 24"><path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM5.92 19H5v-.92l9.06-9.06.92.92L5.92 19zM20.71 7.04a1.003 1.003 0 000-1.42l-2.34-2.34a1.003 1.003 0 00-1.42 0l-1.83 1.83 3.75 3.75 1.84-1.82z"/></svg>', label: '改写', tooltip: '改写输入框中的文本' },
  { id: 'polish', icon: '<svg viewBox="0 0 24 24"><path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/></svg>', label: '润色', tooltip: '润色文本语气和表达' },
  { id: 'code_review', icon: '<svg viewBox="0 0 24 24"><path d="M9.4 16.6L4.8 12l4.6-4.6L8 6l-6 6 6 6 1.4-1.4zm5.2 0l4.6-4.6-4.6-4.6L16 6l6 6-6 6-1.4-1.4z"/></svg>', label: '代码审查', tooltip: '审查代码片段' },
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
