<template>
  <Transition name="slide">
    <div v-if="visible" class="ai-sidebar-panel" :class="{ expanded: isExpanded }">
      <!-- Header -->
      <div class="ai-sidebar-header">
        <div class="ai-sidebar-title">
          <i class="fas fa-robot"></i>
          <span>AI 助手</span>
        </div>
        <div class="ai-sidebar-header-actions">
          <button class="icon-btn" :title="isExpanded ? '缩小' : '放大'" @click="isExpanded = !isExpanded">
            <i :class="isExpanded ? 'fas fa-compress' : 'fas fa-expand'"></i>
          </button>
          <button class="icon-btn" :title="confirmClear ? '再次点击确认' : '清空对话'" @click="handleClearClick">
            <i :class="confirmClear ? 'fas fa-check' : 'fas fa-trash-alt'"></i>
            <span v-if="confirmClear" class="confirm-hint">确认?</span>
          </button>
          <button class="icon-btn" title="关闭" @click="emit('close')">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>

      <!-- Context hint -->
      <div v-if="contextLinked && props.conversationName" class="ai-sidebar-context">
        <i class="fas fa-comment-dots"></i>
        <span>{{ props.conversationName }}</span>
        <button class="context-unlink-btn" title="取消关联，进入自由对话" @click="contextLinked = false">
          <i class="fas fa-xmark"></i>
        </button>
      </div>
      <div v-else-if="!contextLinked && props.conversationName" class="ai-sidebar-context unlinked">
        <i class="fas fa-feather"></i>
        <span>自由对话</span>
        <button class="context-relink-btn" title="重新关联当前会话" @click="contextLinked = true">
          <i class="fas fa-link"></i>
        </button>
      </div>

      <!-- Messages -->
      <div ref="messagesRef" class="ai-sidebar-messages" @scroll="onScroll">
        <!-- Empty State -->
        <div v-if="chatMessages.length === 0" class="ai-sidebar-empty">
          <div class="empty-avatar">
            <i class="fas fa-robot"></i>
          </div>
          <p class="empty-title">有什么可以帮你？</p>
          <p class="empty-sub">{{ conversationName ? `我已了解「${conversationName}」的上下文` : '随时问我任何问题' }}</p>
          <div class="ai-sidebar-suggestions">
            <button @click="sendSuggestion('帮我总结一下刚才的讨论要点')"><i class="fas fa-file-lines"></i>总结讨论</button>
            <button @click="sendSuggestion('三点钟提醒我开会')"><i class="fas fa-bell"></i>设置提醒</button>
            <button @click="sendSuggestion('帮我搜索一下相关的历史消息')"><i class="fas fa-magnifying-glass"></i>搜索信息</button>
            <button @click="sendSuggestion('我有哪些待办任务？')"><i class="fas fa-list-check"></i>查看待办</button>
          </div>
        </div>

        <!-- Chat Messages -->
        <div
          v-for="(msg, idx) in chatMessages"
          :key="idx"
          class="ai-msg"
          :class="msg.role"
        >
          <div class="ai-msg-bubble" :class="{ error: msg.isError }">
            <div v-if="msg.role === 'assistant'" class="ai-msg-content" v-html="renderMd(msg.content)"></div>
            <div v-else class="ai-msg-content" v-html="previewTextToHtml(msg.content)"></div>
          </div>
          <!-- Action row: copy / retry / timestamp -->
          <div v-if="msg.role === 'assistant'" class="ai-msg-actions">
            <span class="msg-time">{{ msg.time }}</span>
            <button
              class="msg-action-btn"
              :title="copiedIdx === idx ? '已复制' : '复制'"
              @click="copyMessage(idx)"
            >
              <i :class="copiedIdx === idx ? 'fas fa-check' : 'fas fa-copy'"></i>
            </button>
            <button
              v-if="msg.isError"
              class="msg-action-btn"
              title="重试"
              @click="retryLastMessage"
            >
              <i class="fas fa-redo"></i>
            </button>
          </div>
          <span v-else class="msg-time user-time">{{ msg.time }}</span>
        </div>

        <!-- Thinking / Streaming indicator -->
        <div v-if="isStreaming" class="ai-msg assistant">
          <div class="ai-msg-bubble streaming">
            <div v-if="!streamingContent" class="thinking-dots">
              <span></span><span></span><span></span>
            </div>
            <div v-else class="ai-msg-content" v-html="renderMd(streamingContent) + '<span class=stream-cursor>▌</span>'"></div>
          </div>
        </div>
      </div>

      <!-- Scroll to bottom button -->
      <Transition name="fade">
        <button v-if="showScrollBtn" class="scroll-bottom-btn" @click="scrollToBottom">
          <i class="fas fa-arrow-down"></i>
        </button>
      </Transition>

      <!-- Input -->
      <div class="ai-sidebar-input">
        <textarea
          ref="inputRef"
          v-model="inputText"
          placeholder="问我任何事…"
          rows="1"
          @keydown.enter.exact.prevent="sendMessage"
          @input="autoResize"
        ></textarea>
        <button
          v-if="isStreaming"
          class="send-btn stop-btn"
          title="停止生成"
          @click="handleStop"
        >
          <i class="fas fa-stop"></i>
        </button>
        <button
          v-else
          class="send-btn"
          :disabled="!inputText.trim()"
          @click="sendMessage"
        >
          <i class="fas fa-paper-plane"></i>
        </button>
      </div>
      <div class="ai-sidebar-input-hint">Enter 发送 · Shift+Enter 换行 · ESC 关闭</div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { marked } from 'marked'
import { sanitizeMarkdown } from '../../utils/sanitize'
import { useAIStream } from '../../composables/useAIStream'
import { getStoredServerUrl } from '../../composables/useServerUrl'
import { previewTextToHtml } from '../../utils/emoji'

interface Props {
  visible: boolean
  conversationId?: number | string | null
  conversationName?: string | null
}

const props = defineProps<Props>()
const emit = defineEmits<{ close: [] }>()

interface ChatMsg {
  role: 'user' | 'assistant'
  content: string
  isError?: boolean
  time: string
}


const chatMessages = ref<ChatMsg[]>([])
const inputText = ref('')
const isStreaming = ref(false)
const streamingContent = ref('')
const messagesRef = ref<HTMLDivElement>()
const inputRef = ref<HTMLTextAreaElement>()
const lastUserMessage = ref('')
const copiedIdx = ref(-1)
const confirmClear = ref(false)
const showScrollBtn = ref(false)
const isExpanded = ref(false)
const contextLinked = ref(true)
let clearTimer: ReturnType<typeof setTimeout> | null = null

const { stream, abort } = useAIStream()

// ── 时间格式化 ──
const formatTime = (): string => {
  const d = new Date()
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

// ── 自动滚动 ──
watch([chatMessages, streamingContent], () => {
  nextTick(() => {
    // 只在用户已处于底部时自动滚动
    if (messagesRef.value && !showScrollBtn.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  })
}, { deep: true })

// 滚动监听
const onScroll = () => {
  if (!messagesRef.value) return
  const { scrollTop, scrollHeight, clientHeight } = messagesRef.value
  const atBottom = scrollHeight - scrollTop - clientHeight < 60
  showScrollBtn.value = !atBottom && chatMessages.value.length > 0
}

const scrollToBottom = () => {
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    showScrollBtn.value = false
  }
}

// ── 面板开关 ──
watch(() => props.visible, (val) => {
  if (val) {
    nextTick(() => inputRef.value?.focus())
  } else {
    if (isStreaming.value) handleStop()
    confirmClear.value = false
  }
})

// 切换会话时自动重新关联
watch(() => props.conversationId, () => {
  contextLinked.value = true
})

// ── ESC 关闭 ──
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible) {
    e.preventDefault()
    emit('close')
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  if (clearTimer) clearTimeout(clearTimer)
})

// ── 发送消息 ──
const sendMessage = () => {
  const text = inputText.value.trim()
  if (!text || isStreaming.value) return

  lastUserMessage.value = text
  chatMessages.value.push({ role: 'user', content: text, time: formatTime() })
  inputText.value = ''
  autoResize()
  doStream(text)
}

const sendSuggestion = (text: string) => {
  if (isStreaming.value) return
  lastUserMessage.value = text
  chatMessages.value.push({ role: 'user', content: text, time: formatTime() })
  doStream(text)
}

// ── 重试 ──
const retryLastMessage = () => {
  if (!lastUserMessage.value || isStreaming.value) return
  // 移除最后一条错误消息
  const last = chatMessages.value[chatMessages.value.length - 1]
  if (last && last.isError) {
    chatMessages.value.pop()
  }
  doStream(lastUserMessage.value)
}

// ── 流式请求 ──
const doStream = (message: string) => {
  isStreaming.value = true
  streamingContent.value = ''

  const serverUrl = getStoredServerUrl()
  const body: Record<string, any> = {
    message,
    scope: 'current',
  }
  if (props.conversationId && contextLinked.value) {
    body.conversation_id = Number(props.conversationId)
  }

  stream({
    url: `${serverUrl}/api/v1/ai/sidebar/stream`,
    body,
    onChunk: (content: string) => {
      streamingContent.value += content
    },
    onComplete: () => {
      if (streamingContent.value) {
        chatMessages.value.push({
          role: 'assistant',
          content: streamingContent.value,
          time: formatTime(),
        })
      }
      streamingContent.value = ''
      isStreaming.value = false
    },
    onError: (err: Error) => {
      chatMessages.value.push({
        role: 'assistant',
        content: `⚠️ 请求失败: ${err.message || '未知错误'}`,
        isError: true,
        time: formatTime(),
      })
      streamingContent.value = ''
      isStreaming.value = false
    },
  })
}

// ── 停止生成 ──
const handleStop = () => {
  abort()
  if (streamingContent.value) {
    chatMessages.value.push({
      role: 'assistant',
      content: streamingContent.value + '\n\n*(已停止)*',
      time: formatTime(),
    })
  }
  streamingContent.value = ''
  isStreaming.value = false
}

// ── 清空（二次确认）──
const handleClearClick = () => {
  if (!confirmClear.value) {
    confirmClear.value = true
    clearTimer = setTimeout(() => { confirmClear.value = false }, 3000)
    return
  }
  if (isStreaming.value) abort()
  chatMessages.value = []
  streamingContent.value = ''
  isStreaming.value = false
  confirmClear.value = false
  if (clearTimer) { clearTimeout(clearTimer); clearTimer = null }
}

// ── 复制 ──
const copyMessage = (idx: number) => {
  const msg = chatMessages.value[idx]
  if (!msg) return
  navigator.clipboard.writeText(msg.content).then(() => {
    copiedIdx.value = idx
    setTimeout(() => { copiedIdx.value = -1 }, 1500)
  })
}

// ── Markdown 渲染 ──
const renderMd = (text: string): string => {
  if (!text) return ''
  try {
    // 将工具执行状态行转为小字样式
    const processed = text.replace(
      /^(🤔.*|\u2699\uFE0F.*|\u26A0\uFE0F.*)$/gm,
      '<span class="status-line">$1</span>'
    )
    const result = marked.parse(processed, { async: false }) as string
    return sanitizeMarkdown(result)
  } catch {
    return text.replace(/\n/g, '<br>')
  }
}

const autoResize = () => {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 120) + 'px'
}
</script>

<style scoped>
.ai-sidebar-panel {
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  width: 360px;
  display: flex;
  flex-direction: column;
  background: var(--card-bg, #fff);
  border-left: 1px solid var(--border-color, #e5e7eb);
  box-shadow: -4px 0 16px rgba(0, 0, 0, 0.06);
  z-index: 999;
  will-change: transform, width;
  transition: width 0.25s ease;
}

.ai-sidebar-panel.expanded {
  width: 680px;
  max-width: calc(100vw - 40px);
}

/* Slide transition */
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

/* Header */
.ai-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color, #e5e7eb);
}

.ai-sidebar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.ai-sidebar-title i {
  color: var(--primary-color, #6366f1);
}

.ai-sidebar-header-actions {
  display: flex;
  gap: 4px;
}

.icon-btn {
  height: 28px;
  width: auto;
  min-width: 28px;
  padding: 0 6px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-secondary, #6b7280);
  transition: all 0.15s;
}

.icon-btn:hover {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #1f2937);
}

.confirm-hint {
  font-size: 11px;
  font-weight: 500;
}


/* Context hint */
.ai-sidebar-context {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  font-size: 12px;
  color: var(--text-secondary, #9ca3af);
  border-bottom: 1px solid var(--border-color, #e5e7eb);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 4%, transparent);
}

.ai-sidebar-context i {
  font-size: 11px;
  opacity: 0.7;
}

.ai-sidebar-context span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.ai-sidebar-context.unlinked {
  background: transparent;
  color: var(--text-secondary, #b0b0b0);
}

.context-unlink-btn,
.context-relink-btn {
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 11px;
  color: var(--text-secondary, #9ca3af);
  padding: 2px 5px;
  border-radius: 4px;
  transition: all 0.15s;
  flex-shrink: 0;
}

.context-unlink-btn:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
}

.context-relink-btn:hover {
  color: var(--primary-color, #6366f1);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 8%, transparent);
}

/* Messages area */
.ai-sidebar-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  position: relative;
}

.ai-sidebar-messages::-webkit-scrollbar {
  width: 5px;
}

.ai-sidebar-messages::-webkit-scrollbar-track {
  background: transparent;
}

.ai-sidebar-messages::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.12);
  border-radius: 3px;
}

.ai-sidebar-messages::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.2);
}

.ai-sidebar-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-secondary, #9ca3af);
  text-align: center;
  padding: 20px;
  min-height: 0;
  overflow: hidden;
}

.empty-avatar {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: color-mix(in srgb, var(--primary-color, #6366f1) 10%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
}

.empty-avatar i {
  font-size: 22px;
  color: var(--primary-color, #6366f1);
}

.empty-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text-color, #1f2937);
}

.empty-sub {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary, #9ca3af);
}

.ai-sidebar-suggestions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
  margin-top: 16px;
  width: 100%;
}

.ai-sidebar-suggestions button {
  padding: 10px 12px;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  background: var(--card-bg, #fff);
  color: var(--text-color, #374151);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  gap: 6px;
  justify-content: center;
}

.ai-sidebar-suggestions button i {
  font-size: 12px;
  color: var(--primary-color, #6366f1);
  opacity: 0.8;
}

.ai-sidebar-suggestions button:hover {
  border-color: var(--primary-color, #6366f1);
  color: var(--primary-color, #6366f1);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 5%, transparent);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.1);
}

/* Message containers */
.ai-msg {
  display: flex;
  flex-direction: column;
  max-width: 88%;
  animation: msgIn 0.25s ease;
}

@keyframes msgIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.ai-msg.user {
  align-self: flex-end;
  align-items: flex-end;
}

.ai-msg.assistant {
  align-self: flex-start;
  align-items: flex-start;
}

.ai-msg-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
}

.ai-msg.user .ai-msg-bubble {
  background: var(--primary-color, #6366f1);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.ai-msg.assistant .ai-msg-bubble {
  background: var(--hover-color, #f3f4f6);
  color: var(--text-color, #1f2937);
  border-bottom-left-radius: 4px;
}

.ai-msg.assistant .ai-msg-bubble.streaming {
  border: 1px solid color-mix(in srgb, var(--primary-color, #6366f1) 20%, transparent);
}

/* Error message */
.ai-msg-bubble.error {
  background: rgba(239, 68, 68, 0.06);
  border: 1px solid rgba(239, 68, 68, 0.15);
}

/* Stream cursor blink */
.stream-cursor {
  animation: blink 1s step-end infinite;
  color: var(--primary-color, #6366f1);
  font-weight: 300;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* Action row (copy / retry / time) */
.ai-msg-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  padding: 0 2px;
  height: 22px;
  width: 100%;
  justify-content: flex-end;
}

.msg-action-btn {
  height: 22px;
  min-width: 22px;
  padding: 0 4px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  font-size: 10px;
  color: var(--text-secondary, #9ca3af);
  opacity: 0;
  transition: opacity 0.15s, color 0.15s, background 0.15s;
}

.ai-msg:hover .msg-action-btn {
  opacity: 1;
}

.msg-action-btn:hover {
  color: var(--primary-color, #6366f1);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 8%, transparent);
}

/* Timestamp */
.msg-time {
  font-size: 10px;
  color: var(--text-secondary, #9ca3af);
}

.user-time {
  margin-top: 2px;
  padding: 0 4px;
}

/* Thinking dots */
.thinking-dots {
  display: flex;
  gap: 4px;
  padding: 4px 0;
}

.thinking-dots span {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-secondary, #9ca3af);
  animation: thinking 1.4s infinite ease-in-out;
}

.thinking-dots span:nth-child(2) { animation-delay: 0.2s; }
.thinking-dots span:nth-child(3) { animation-delay: 0.4s; }

@keyframes thinking {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-6px); opacity: 1; }
}

/* Markdown content */
.ai-msg-content :deep(.emoji-img) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin: 0 1px;
}

.ai-msg-content :deep(p) { margin: 0 0 8px; }
.ai-msg-content :deep(p:last-child) { margin-bottom: 0; }
.ai-msg-content :deep(ul),
.ai-msg-content :deep(ol) { margin: 4px 0 8px; padding-left: 18px; }
.ai-msg-content :deep(li) { margin-bottom: 4px; }
.ai-msg-content :deep(code) {
  background: rgba(0, 0, 0, 0.06);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}
.ai-msg-content :deep(pre) {
  background: rgba(0, 0, 0, 0.05);
  padding: 10px;
  border-radius: 6px;
  overflow-x: auto;
  margin: 8px 0;
}
.ai-msg-content :deep(pre code) { background: none; padding: 0; }
.ai-msg-content :deep(strong) { font-weight: 600; }
.ai-msg-content :deep(h1),
.ai-msg-content :deep(h2),
.ai-msg-content :deep(h3) {
  margin: 12px 0 6px;
  font-size: 14px;
  font-weight: 600;
}
.ai-msg-content :deep(blockquote) {
  margin: 8px 0;
  padding: 6px 12px;
  border-left: 3px solid var(--primary-color, #6366f1);
  background: color-mix(in srgb, var(--primary-color, #6366f1) 4%, transparent);
  border-radius: 0 6px 6px 0;
  color: var(--text-secondary, #6b7280);
  font-size: 12px;
}
.ai-msg-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 8px 0;
  font-size: 12px;
}
.ai-msg-content :deep(th),
.ai-msg-content :deep(td) {
  border: 1px solid var(--border-color, #e5e7eb);
  padding: 6px 8px;
  text-align: left;
}
.ai-msg-content :deep(th) {
  background: var(--hover-color, #f3f4f6);
  font-weight: 600;
}
.ai-msg-content :deep(hr) {
  border: none;
  border-top: 1px solid var(--border-color, #e5e7eb);
  margin: 12px 0;
}

/* Scroll-to-bottom button */
.scroll-bottom-btn {
  position: absolute;
  bottom: 8px;
  left: 50%;
  transform: translateX(-50%);
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 50%;
  background: var(--card-bg, #fff);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: var(--text-color, #1f2937);
  z-index: 10;
  transition: all 0.15s;
}

.scroll-bottom-btn:hover {
  background: var(--hover-color, #f3f4f6);
  transform: translateX(-50%) scale(1.1);
}

.fade-enter-active,
.fade-leave-active { transition: opacity 0.2s, transform 0.2s; }
.fade-enter-from,
.fade-leave-to { opacity: 0; transform: translateX(-50%) translateY(8px); }

/* Input area */
.ai-sidebar-input {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: none;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.04);
}

.ai-sidebar-input textarea {
  flex: 1;
  resize: none;
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 10px;
  padding: 10px 12px;
  font-size: 13px;
  font-family: inherit;
  line-height: 1.5;
  outline: none;
  background: var(--sidebar-bg, #f9fafb);
  color: var(--text-color, #1f2937);
  transition: border-color 0.15s;
  max-height: 120px;
  overflow-y: auto;
  scrollbar-width: none;
}

.ai-sidebar-input textarea::-webkit-scrollbar {
  display: none;
}

.ai-sidebar-input textarea:focus {
  border-color: var(--primary-color, #6366f1);
}

.send-btn {
  width: 38px;
  height: 38px;
  border: none;
  border-radius: 10px;
  background: var(--primary-color, #6366f1);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.send-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.send-btn:not(:disabled):hover {
  opacity: 0.85;
}

.send-btn:not(:disabled):active {
  transform: scale(0.92);
}

.send-btn.stop-btn {
  background: #ef4444;
}

.send-btn.stop-btn:hover {
  opacity: 0.85;
}

/* Responsive */
@media (max-width: 768px) {
  .ai-sidebar-panel {
    width: 100%;
    min-width: unset;
  }
}

/* Input hint */
.ai-sidebar-input-hint {
  text-align: center;
  font-size: 10px;
  color: var(--text-secondary, #c0c0c0);
  padding: 0 16px 8px;
  user-select: none;
}

@media (max-width: 768px) {
  .ai-sidebar-input-hint {
    display: none;
  }
}

/* Status lines (tool execution progress) */
.ai-msg-content :deep(.status-line) {
  display: block;
  font-size: 12px;
  color: var(--text-secondary, #9ca3af);
  padding: 2px 0;
}
</style>
