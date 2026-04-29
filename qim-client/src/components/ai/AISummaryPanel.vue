<template>
  <Teleport to="body">
    <div v-if="visible" class="ai-summary-overlay" @click.self="close">
      <div class="ai-summary-panel">
        <div class="panel-header">
          <h3>会话摘要</h3>
          <button class="close-btn" @click="close">&times;</button>
        </div>

        <div v-if="isGenerating" class="generating-state">
          <div class="generating-spinner"></div>
          <p>正在分析会话内容...</p>
          <p class="generating-hint">这可能需要几秒钟</p>
        </div>

        <div v-else-if="summaryData" class="summary-content">
          <div class="summary-meta">
            <span>{{ summaryData.time_range }}</span>
            <span>{{ summaryData.messages_count }} 条消息</span>
          </div>
          <div v-html="renderMarkdown(summaryData.summary)"></div>
          <div class="summary-actions">
            <button @click="copySummary">复制摘要</button>
            <button @click="exportSummary">导出 Markdown</button>
          </div>
        </div>

        <div v-else class="error-state">
          <p>摘要生成失败</p>
          <button @click="generate">重试</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'

const props = defineProps<{
  visible: boolean
  conversationId: number
  timeRange?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const { generateSummary, isProcessing: isGenerating } = useAIActions()
const summaryData = ref<any>(null)

watch(() => props.visible, async (newVal) => {
  if (newVal && props.conversationId) {
    await generate()
  }
})

const generate = async () => {
  summaryData.value = null
  try {
    summaryData.value = await generateSummary(
      props.conversationId,
      props.timeRange || 'today'
    )
  } catch {
    // 错误已由 composable 处理
  }
}

const close = () => {
  emit('close')
}

const copySummary = async () => {
  if (summaryData.value?.summary) {
    await navigator.clipboard.writeText(summaryData.value.summary)
  }
}

const exportSummary = () => {
  if (summaryData.value?.summary) {
    const blob = new Blob([summaryData.value.summary], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `session-summary-${Date.now()}.md`
    a.click()
    URL.revokeObjectURL(url)
  }
}

const renderMarkdown = (text: string): string => {
  try {
    const result = marked.parse(text)
    if (result instanceof Promise) return text
    return sanitizeMarkdown(result as string)
  } catch {
    return text.replace(/\n/g, '<br>')
  }
}
</script>

<style scoped>
.ai-summary-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.ai-summary-panel {
  background: var(--card-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 750px;
  max-height: 85vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.panel-header h3 {
  margin: 0;
  font-size: 18px;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-secondary);
  border-radius: 6px;
  transition: background 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
}

.generating-state {
  padding: 60px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.generating-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

.generating-hint {
  font-size: 12px;
  margin-top: 8px;
  opacity: 0.7;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.summary-content {
  padding: 28px 32px;
  overflow-y: auto;
  line-height: 1.8;
}

.summary-content :deep(p) {
  margin-bottom: 16px;
}

.summary-content :deep(ul),
.summary-content :deep(ol) {
  margin-bottom: 16px;
  padding-left: 24px;
}

.summary-content :deep(li) {
  margin-bottom: 8px;
  line-height: 1.8;
}

.summary-content :deep(h1),
.summary-content :deep(h2),
.summary-content :deep(h3) {
  margin-top: 24px;
  margin-bottom: 14px;
  line-height: 1.4;
}

.summary-content :deep(strong),
.summary-content :deep(b) {
  font-weight: 600;
}

.summary-meta {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.summary-actions {
  display: flex;
  gap: 12px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.summary-actions button {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: transparent;
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.2s;
}

.summary-actions button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.error-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.error-state p {
  margin-bottom: 16px;
}

.error-state button {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
}

.error-state button:hover {
  opacity: 0.9;
}
</style>
