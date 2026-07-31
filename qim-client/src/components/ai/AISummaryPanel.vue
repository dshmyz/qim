<template>
  <Teleport to="body">
    <div v-if="visible" class="ai-summary-overlay" @click.self="close">
      <div class="ai-summary-panel">
        <div class="panel-header">
          <h3>会话摘要</h3>
          <button class="close-btn" @click="close">&times;</button>
        </div>

        <!-- Toolbar: time range selector + regenerate -->
        <div class="summary-toolbar">
          <div class="time-range-group">
            <button
              v-for="r in timeRanges"
              :key="r.value"
              class="range-btn"
              :class="{ active: currentRange === r.value }"
              :disabled="isGenerating"
              @click="selectRange(r.value)"
            >
              {{ r.label }}
            </button>
          </div>
          <button
            v-if="!isGenerating"
            class="regenerate-btn"
            title="重新生成"
            @click="generate"
          >
            <i class="fas fa-redo"></i>
          </button>
          <button
            v-else
            class="regenerate-btn"
            title="停止生成"
            @click="stop"
          >
            <i class="fas fa-stop"></i>
          </button>
        </div>

        <!-- Generating state with stream preview -->
        <div v-if="isGenerating" class="generating-state">
          <div class="generating-spinner"></div>
          <p>正在分析会话内容...</p>
        </div>

        <!-- Summary content -->
        <div v-else-if="summaryData || streamingContent" class="summary-content">
          <div class="summary-meta">
            <span v-if="summaryData?.time_range">{{ summaryData.time_range }}</span>
            <span v-else>{{ timeRangeText }}</span>
            <span>{{ messagesCount }} 条消息</span>
            <span v-if="activeMembers.length > 0">参与者: {{ activeMembers.join(', ') }}</span>
          </div>
          <div v-html="renderMarkdown(displaySummary)"></div>
          <div class="summary-actions">
            <button @click="copySummary">复制摘要</button>
            <button @click="exportSummary">导出 Markdown</button>
            <button @click="saveToNote" :disabled="saving">
              {{ saving ? '保存中...' : '保存到笔记' }}
            </button>
          </div>
        </div>

        <!-- Empty state: no messages -->
        <div v-else-if="noMessages" class="empty-state">
          <p>该时间段内没有可摘要的消息</p>
          <p class="empty-hint">换个时间范围，或等对话内容多一些再试试</p>
        </div>

        <!-- Error state -->
        <div v-else-if="error" class="error-state">
          <p>{{ error }}</p>
          <button @click="generate">重试</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { useAIActions } from '../../composables/useAIActions'
import { useNotes } from '../../composables/useNotes'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'
import QMessage from '../../utils/qmessage'

interface Props {
  visible: boolean
  conversationId: number
  timeRange?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

const {
  generateSummary,
  generateSummaryMeta,
  generateSummaryStream,
  abort: abortStream,
} = useAIActions()
const { createNote } = useNotes()

const currentRange = ref(props.timeRange || 'today')
const isGenerating = ref(false)
const streamingContent = ref('')
const summaryData = ref<any>(null)
const saving = ref(false)
const error = ref<string | null>(null)
const messagesCount = ref(0)
const activeMembers = ref<string[]>([])
const noMessages = ref(false)

const timeRanges = [
  { value: '1h', label: '最近 1 小时' },
  { value: 'today', label: '今天' },
  { value: '7d', label: '最近 7 天' },
]

const timeRangeText = computed(() => {
  switch (currentRange.value) {
    case '1h': return '最近 1 小时'
    case '7d': return '最近 7 天'
    default: return '今天'
  }
})

const displaySummary = computed(() => {
  return summaryData.value?.summary || streamingContent.value || ''
})

watch(() => props.visible, (newVal) => {
  if (newVal) {
    currentRange.value = props.timeRange || 'today'
    generate()
  }
})

const selectRange = (range: string) => {
  if (range === currentRange.value || isGenerating.value) return
  currentRange.value = range
  generate()
}

const stop = () => {
  abortStream()
  isGenerating.value = false
}

const generate = async () => {
  if (!props.conversationId) return

  isGenerating.value = true
  streamingContent.value = ''
  summaryData.value = null
  error.value = null
  noMessages.value = false

  // 先获取元数据（消息数、参与者等），再流式生成摘要
  try {
    const meta = await generateSummaryMeta(props.conversationId, currentRange.value)
    messagesCount.value = meta.messages_count || 0
    activeMembers.value = meta.active_members || []
    if (messagesCount.value === 0) {
      noMessages.value = true
      isGenerating.value = false
      return
    }
  } catch {
    messagesCount.value = 0
    activeMembers.value = []
  }

  generateSummaryStream(props.conversationId, currentRange.value, {
    onChunk: (content: string) => {
      streamingContent.value += content
    },
    onComplete: () => {
      isGenerating.value = false
      // 流结束后把内容当作最终结果
      summaryData.value = {
        summary: streamingContent.value,
        time_range: timeRangeText.value,
      }
    },
    onError: (err: Error) => {
      isGenerating.value = false
      error.value = err.message || '摘要生成失败'
    },
  })
}

const close = () => {
  if (isGenerating.value) {
    abortStream()
    isGenerating.value = false
  }
  emit('close')
}

const copySummary = async () => {
  if (!displaySummary.value) return
  await navigator.clipboard.writeText(displaySummary.value)
  QMessage.success('已复制')
}

const exportSummary = () => {
  if (!displaySummary.value) return
  const blob = new Blob([displaySummary.value], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `session-summary-${Date.now()}.md`
  a.click()
  URL.revokeObjectURL(url)
}

const saveToNote = async () => {
  if (!displaySummary.value || saving.value) return

  saving.value = true
  try {
    const result = await createNote({
      title: timeRangeText.value,
      content: displaySummary.value,
      type: 'note'
    })

    if (result) {
      QMessage.success('已保存到笔记')
    } else {
      QMessage.error('保存失败，请稍后重试')
    }
  } catch (error) {
    QMessage.error('保存失败，请稍后重试')
  } finally {
    saving.value = false
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

// ESC 关闭
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible) {
    e.preventDefault()
    close()
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))
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

.summary-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px;
  border-bottom: 1px solid var(--border-color);
}

.time-range-group {
  display: flex;
  gap: 8px;
}

.range-btn {
  padding: 5px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.range-btn:hover:not(:disabled) {
  border-color: var(--primary-color);
}

.range-btn.active {
  background: var(--primary-light);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.range-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.regenerate-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--card-bg);
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.regenerate-btn:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
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

@keyframes spin {
  to { transform: rotate(360deg); }
}

.summary-content {
  padding: 28px 32px;
  overflow-y: auto;
  line-height: 1.7;
  font-size: 14px;
}

.summary-content :deep(p) {
  margin-bottom: 12px;
}

.summary-content :deep(h1) {
  font-size: 18px;
}

.summary-content :deep(h2) {
  font-size: 16px;
}

.summary-content :deep(h3) {
  font-size: 15px;
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
  margin-top: 18px;
  margin-bottom: 10px;
  line-height: 1.4;
}

.summary-content :deep(strong),
.summary-content :deep(b) {
  font-weight: 600;
}

.summary-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
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

.empty-state,
.error-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-secondary);
}

.empty-hint {
  font-size: 12px;
  margin-top: 8px;
  opacity: 0.7;
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
