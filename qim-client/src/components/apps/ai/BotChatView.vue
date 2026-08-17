<template>
  <div class="bot-chat-view">
    <div class="chat-header">
      <button class="back-btn" @click="$emit('back')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <div class="bot-info">
        <Avatar :src="bot?.avatar" :name="bot?.name || 'AI助手'" :alt="bot?.name || 'AI助手'" size="sm" class="bot-avatar" />
        <span class="bot-name">{{ bot?.name || 'AI助手' }}</span>
      </div>
      <div class="header-actions">
        <button class="action-btn" @click="handleClearMessages" title="清空对话">
          <i class="fas fa-trash-alt"></i>
        </button>
        <button class="action-btn" @click="handleNewConversation" title="新建对话">
          <i class="fas fa-plus"></i>
        </button>
        <!-- 历史会话：当前 bot 的多会话线程在这里切回。
             入口固定显示（不因数据缺失而隐藏），避免用户找不到旧对话 -->
        <div v-if="bot" ref="historyRef" class="history-wrap" @click.stop>
          <button
            class="action-btn"
            :class="{ active: historyOpen }"
            title="历史会话"
            @click="toggleHistory"
          >
            <i class="fas fa-history"></i>
          </button>
          <div v-if="historyOpen" class="history-panel">
            <div class="history-panel-title">
              {{ bot.name }} 的 {{ historyThreads.length }} 段会话
            </div>
            <ul v-if="historyThreads.length" class="history-list">
              <li
                v-for="t in historyThreads"
                :key="t.id"
                :class="['history-item', { current: Number(t.id) === currentConversationId }]"
                @click="switchToThread(t.id)"
              >
                <i class="fas fa-comment-dots"></i>
                <span class="history-time">{{ formatRelativeTime(t.updated_at) }}</span>
                <i
                  v-if="Number(t.id) === currentConversationId"
                  class="fas fa-check history-check"
                ></i>
              </li>
            </ul>
            <div v-else class="history-empty">
              暂无历史会话，点「新建对话」开启新话题
            </div>
          </div>
        </div>
      </div>
      <div v-if="isLoading" class="loading-badge">加载中...</div>
    </div>

    <div class="messages" ref="messagesRef" @scroll="onScroll">
      <!-- 加载状态 -->
      <div v-if="isLoading && messages.length === 0" class="loading-state">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载历史消息...</span>
      </div>

      <!-- 空态欢迎语 + 示例提问 -->
      <div v-else-if="!error && messages.length === 0" class="empty-state">
        <div class="empty-icon"><i class="fas fa-robot"></i></div>
        <h3>{{ bot?.name ? `${bot.name} 为你服务` : '你好，我是 AI 助手' }}</h3>
        <p>直接向我提问，或从下面的示例开始：</p>
        <div class="suggestion-list">
          <button
            v-for="s in suggestions"
            :key="s"
            class="suggestion-chip"
            @click="useSuggestion(s)"
          >
            {{ s }}
          </button>
        </div>
      </div>

      <!-- 历史「加载更多」提示（滚动到顶部自动触发，也可点击） -->
      <button
        v-if="hasMoreMessages && messages.length > 0"
        class="load-more-btn"
        :disabled="isLoading"
        @click="emit('loadMore')"
      >
        <i class="fas fa-chevron-up"></i>
        {{ isLoading ? '加载中...' : '加载更早的消息' }}
      </button>

      <!-- 消息列表。注意：v-for 必须挂 <template> 而非 <div> 外层容器——否则
           .message-wrapper 成为匿名 div 的块级子元素，父容器不再是 .messages 这个
           flex 列，align-self: flex-end 失效，用户自己的消息无法贴右（右侧留大边距）。
           用 <template v-for> 让 message-wrapper / time-divider 直接作为 .messages
           的 flex 子项，用户消息恢复右对齐。 -->
      <template v-for="(msg, idx) in messages" :key="msg.id">
        <!-- 时间分隔线：与上一条间隔 > 5 分钟或跨天时插入（对齐 IM 消息列表） -->
        <div v-if="shouldShowDivider(idx, msg)" class="time-divider">
          <span class="time-divider-text">{{ chatUtils.formatTime(toBotTime(msg.timestamp)) }}</span>
        </div>
        <div :class="['message-wrapper', msg.senderType === 'user' ? 'user' : 'bot']">
          <Avatar
            v-if="msg.senderType === 'bot'"
            :src="msg.sender?.avatar || bot?.avatar"
            :name="msg.sender?.nickname || bot?.name || 'AI助手'"
            :alt="msg.sender?.nickname || bot?.name || 'AI助手'"
            size="sm"
            class="message-avatar"
          />
          <Avatar
            v-else-if="msg.senderType === 'user' && msg.sender"
            :src="msg.sender?.avatar"
            :name="msg.sender?.nickname || '用户'"
            :alt="msg.sender?.nickname || '用户'"
            size="sm"
            class="message-avatar"
          />
          <div class="message-column">
            <div class="message-bubble" :class="{ 'msg-failed': msg.isError || msg.isFailed }">
              <div class="content">
                <!-- bot 回答统一走 AIAnswerBubble（markdown 正文 + 思考/typing + 命中笔记来源标签），
                     与 IM 气泡 AI 渲染同一套能力 -->
                <AIAnswerBubble
                  v-if="msg.senderType === 'bot'"
                  :content="msg.content"
                  :is-streaming="Boolean(msg.isStreaming)"
                  variant="botchat"
                  :knowledge-sources="msg.knowledge_sources"
                />
                <span v-else v-html="previewTextToHtml(msg.content)"></span>
              </div>
            </div>
            <!-- AI 失败态：展示重试（重新填回上一次提问） -->
            <div v-if="msg.isError" class="failed-bar">
              <span class="failed-hint"><i class="fas fa-exclamation-triangle"></i> AI 回答失败</span>
              <button class="retry-btn" @click.stop="handleRetry(msg)">
                <i class="fas fa-redo"></i> 重试
              </button>
            </div>
            <!-- 用户消息发送失败态：展示重发 -->
            <div v-else-if="msg.isFailed" class="failed-bar">
              <span class="failed-hint"><i class="fas fa-exclamation-triangle"></i> 发送失败</span>
              <button class="retry-btn" @click.stop="emit('retryMessage', msg)">
                <i class="fas fa-redo"></i> 重试
              </button>
            </div>
            <!-- 底部 meta 行：时间 + 复制（位于气泡外、悬停出现，对齐 IM 消息列表惯例） -->
            <div class="message-meta">
              <span class="time">{{ chatUtils.formatTime(toBotTime(msg.timestamp)) }}</span>
              <button
                v-if="!msg.isStreaming && msg.content"
                class="meta-copy-btn"
                :title="copiedMessageId === String(msg.id) ? '已复制' : '复制'"
                @click.stop="copyMessage(msg)"
              >
                <i :class="copiedMessageId === String(msg.id) ? 'fas fa-check' : 'fas fa-copy'"></i>
              </button>
            </div>
          </div>
        </div>
      </template>

      <!-- 思考指示器：ai_reply_started 置位后显示，首个回复消息到达或安全超时后消失 -->
      <ThinkingIndicator v-if="aiThinking && !hasStreamingMessage" />

      <!-- 错误提示 -->
      <div v-if="error" class="error-message">
        <i class="fas fa-exclamation-circle"></i>
        <span>{{ error }}</span>
      </div>
    </div>

    <div class="input-area">
      <textarea
        ref="inputEl"
        v-model="input"
        :placeholder="`向 ${bot?.name || 'AI助手'} 提问...`"
        :disabled="isSending && !isStreaming"
        rows="1"
        @keydown="handleEnter"
      ></textarea>
      <!-- 流式中：发送按钮变「停止生成」 -->
      <button
        v-if="isStreaming"
        class="send-btn stop-btn"
        title="停止生成"
        @click="emit('stopStream')"
      >
        <i class="fas fa-stop"></i>
      </button>
      <button
        v-else
        @click="sendMessage"
        class="send-btn"
        :disabled="isSending || !input.trim()"
        title="发送"
      >
        <i class="fas fa-paper-plane"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch, onScopeDispose } from 'vue'
import Avatar from '../../shared/Avatar.vue'
import ThinkingIndicator from '../../shared/ThinkingIndicator.vue'
import AIAnswerBubble from '../../message/AIAnswerBubble.vue'
import { previewTextToHtml } from '../../../utils/emoji'
import { copyToClipboard } from '../../../utils/clipboard'
import { useChatUtils } from '../../../composables/useChatUtils'
import type { BotMessage } from '../../../types/bot'
import type { BotConversationThread } from '../../../composables/useBotChat'
import QMessageBox from '../../../utils/qmessagebox'
import { useChatStore } from '../../../stores/chat'

const chatUtils = useChatUtils()

const chatStore = useChatStore()

// AI 回复「思考中」占位：由后端 ai_reply_started 事件置位，首个回复消息到达或
// 90s 安全超时清除。独立于 isStreaming（后者驱动「停止生成」按钮且当前恒 false，
// 不可复用）。
const aiThinking = computed(() => {
  const cid = props.currentConversationId
  return cid != null && chatStore.isAiThinking(String(cid))
})

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
}

const props = defineProps<{
  bot: Bot | null
  messages: BotMessage[]
  isLoading: boolean
  isSending: boolean
  isStreaming: boolean
  hasMoreMessages: boolean
  error: string | null
  historyThreads: BotConversationThread[]
  currentConversationId: number | null
}>()

const emit = defineEmits<{
  back: []
  send: [content: string]
  clearMessages: []
  newConversation: []
  loadMore: []
  stopStream: []
  switchHistory: [id: number | string]
  retryMessage: [msg: BotMessage]
}>()

const input = ref('')
const inputEl = ref<HTMLTextAreaElement | null>(null)
const messagesRef = ref<HTMLDivElement | null>(null)
const copiedMessageId = ref<string | null>(null)

// 历史会话下拉（点外部/切换线程后收起）
const historyOpen = ref(false)
const historyRef = ref<HTMLDivElement | null>(null)

function toggleHistory() {
  historyOpen.value = !historyOpen.value
}

function switchToThread(id: number) {
  historyOpen.value = false
  emit('switchHistory', id)
}

/**
 * 点下拉外部收起（内部点击已被 wrap 的 @click.stop 拦截，不会走到这里）
 */
function onDocClick(e: MouseEvent) {
  const el = historyRef.value
  if (el && !el.contains(e.target as Node)) {
    historyOpen.value = false
  }
}

// 延迟挂载定时器句柄：组件在定时器触发前卸载时须取消，
// 否则监听器会挂到已卸载实例的闭包上永久残留（成对移除原则）。
let historyOpenTimer: ReturnType<typeof setTimeout> | null = null

watch(historyOpen, (open) => {
  if (open) {
    // 下一 tick 再挂，避免本次打开按钮的 click 冒泡到 document 立即触发关闭
    historyOpenTimer = setTimeout(() => {
      historyOpenTimer = null
      document.addEventListener('click', onDocClick)
    }, 0)
  } else {
    if (historyOpenTimer !== null) {
      clearTimeout(historyOpenTimer)
      historyOpenTimer = null
    }
    document.removeEventListener('click', onDocClick)
  }
})

onScopeDispose(() => {
  if (historyOpenTimer !== null) {
    clearTimeout(historyOpenTimer)
    historyOpenTimer = null
  }
  document.removeEventListener('click', onDocClick)
})

/**
 * 相对时间（历史会话下拉）
 */
function formatRelativeTime(ts: string) {
  if (!ts) return ''
  const time = new Date(ts).getTime()
  if (Number.isNaN(time)) return ''
  const diff = Date.now() - time
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return new Date(ts).toLocaleDateString('zh-CN')
}

// 空态示例提问
const suggestions = [
  '用三句话解释什么是 RESTful API',
  '帮我列一个项目启动检查清单',
  '写一段周报里本周进展的写法示例',
  '帮我规划一个 30 分钟的健身安排'
]

/**
 * 检查是否有正在流式传输的消息
 */
const hasStreamingMessage = computed(() => {
  return props.messages.some(msg => msg.isStreaming)
})

/**
 * 发送消息
 */
async function sendMessage() {
  if (!input.value.trim() || props.isSending || props.isStreaming) return
  emit('send', input.value.trim())
  input.value = ''
  await scrollToBottom()
}

/**
 * 回车发送 / Shift+Enter 换行 / 流式中回车退化为换行
 * 注意：必须显式判断 e.key === 'Enter'——此前漏判导致任意键（如 Ctrl+Z 撤销、
 * Ctrl+V 粘贴）在输入框非空时都会误触发发送。isComposing 用于跳过 IME 拼音
 * 候选确认的回车（keydown 仍报 Enter 但属组字过程），避免候选上屏即误发。
 */
function handleEnter(e: KeyboardEvent) {
  if (e.key !== 'Enter' || e.shiftKey || e.isComposing || props.isSending || props.isStreaming || !input.value.trim()) return
  e.preventDefault()
  sendMessage()
}

/**
 * 输入框自适应高度（最多 ~120px 滚动）
 */
function autoResize() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, 120)}px`
}
watch(input, () => autoResize())

/**
 * 复制消息（bot 回复 / 用户消息均可用）
 */
async function copyMessage(msg: BotMessage) {
  const ok = await copyToClipboard(msg.content)
  if (ok) {
    copiedMessageId.value = String(msg.id)
    setTimeout(() => {
      if (copiedMessageId.value === String(msg.id)) copiedMessageId.value = null
    }, 1500)
  }
}

/**
 * AI 失败重试：把上一次用户提问重新填回输入框，聚焦待发送
 */
function handleRetry(msg: BotMessage) {
  const idx = props.messages.indexOf(msg)
  for (let i = idx - 1; i >= 0; i--) {
    const prev = props.messages[i]
    if (prev.senderType === 'user') {
      input.value = prev.content
      autoResize()
      inputEl.value?.focus()
      return
    }
  }
}

/**
 * 使用示例提问
 */
function useSuggestion(text: string) {
  emit('send', text)
  scrollToBottom()
}

/**
 * 清空对话
 */
async function handleClearMessages() {
  const result = await QMessageBox.confirm(
    '确定要清空对话记录吗？',
    '清空对话',
    { confirmButtonText: '清空', type: 'warning' }
  )
  if (result.action !== 'confirm') return
  emit('clearMessages')
}

/**
 * 新建对话
 */
function handleNewConversation() {
  emit('newConversation')
}

/**
 * 滚动到底部
 */
async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

// 「加载更早的消息」prepend 前的滚动锚点：还原时保持视口停在原消息上，避免跳到底部
let pendingScrollAnchor: { height: number; top: number } | null = null
// 用户是否贴底（距底 < 50px）：只有贴底时新消息/流式增量才跟随滚动
let stickToBottom = true

/**
 * 滚动到顶部附近：触发加载更早的历史
 */
function onScroll() {
  const el = messagesRef.value
  if (!el) return
  stickToBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50
  if (props.isLoading) return
  if (el.scrollTop <= 30 && props.hasMoreMessages && props.messages.length > 0) {
    pendingScrollAnchor = { height: el.scrollHeight, top: el.scrollTop }
    emit('loadMore')
  }
}

/**
 * BotMessage.timestamp 是 Date，useChatUtils.formatTime 只接受 number|string，
 * 这里统一转成时间戳（ms）供分隔线/悬停时间复用 IM 的智能日期格式。
 */
function toBotTime(ts: Date): number {
  return ts instanceof Date ? ts.getTime() : Number(ts)
}

/**
 * 时间分隔线：与上一条间隔 > 5 分钟或跨天时显示（对齐 IM 消息列表的判定规则）。
 * BotMessage.timestamp 是 Date，这里本地直接比较，避免与 IM 的 shouldShowTimeDivider(Message) 强转类型。
 */
function shouldShowDivider(idx: number, msg: BotMessage): boolean {
  if (idx === 0) return true
  const prev = props.messages[idx - 1]
  if (!prev) return true
  if (msg.timestamp.getTime() - prev.timestamp.getTime() > 5 * 60 * 1000) return true
  const cur = msg.timestamp
  const p = prev.timestamp
  return cur.getFullYear() !== p.getFullYear()
    || cur.getMonth() !== p.getMonth()
    || cur.getDate() !== p.getDate()
}

// 首条消息变化 = 整组重载（切线程/首次加载）而非 prepend：重置贴底跟随。
// prepend 场景同样会先经过这里，但随后锚点还原触发的 scroll 事件会把 stickToBottom 修正回实际位置。
watch(() => props.messages[0], () => {
  stickToBottom = true
})

// 监听消息变化：加载更早消息时按锚点还原视口；其余仅在贴底时跟随滚动到底部
watch(() => props.messages, async () => {
  await nextTick()
  const el = messagesRef.value
  if (!el) return
  if (pendingScrollAnchor) {
    el.scrollTop = el.scrollHeight - pendingScrollAnchor.height + pendingScrollAnchor.top
    pendingScrollAnchor = null
  } else if (stickToBottom) {
    scrollToBottom()
  }
}, { deep: true })

// 监听流式状态变化
watch(() => props.isStreaming, () => {
  if (stickToBottom) {
    scrollToBottom()
  }
})
</script>

<style scoped>
.bot-chat-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s;
}

.back-btn:hover {
  background: var(--hover-color);
}

.bot-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.bot-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
}

.bot-name {
  font-size: var(--font-size-base);
  font-weight: 500;
  color: var(--text-primary);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--hover-color);
  color: var(--text-primary);
}

.action-btn.active {
  background: var(--hover-color);
  color: var(--primary-color);
}

/* 历史会话下拉 */
.history-wrap {
  position: relative;
}

.history-panel {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  width: 240px;
  max-height: 320px;
  overflow-y: auto;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
  z-index: 40;
  padding: 8px;
}

.history-panel-title {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  padding: 4px 8px 8px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 6px;
}

.history-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  transition: background 0.15s;
}

.history-item:hover {
  background: var(--hover-color);
}

.history-item.current {
  color: var(--primary-color);
  background: var(--hover-color);
}

.history-check {
  margin-left: auto;
  color: var(--primary-color);
}

.history-time {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-empty {
  padding: 16px 10px;
  text-align: center;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
}

.loading-badge {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 4px 8px;
  border-radius: 4px;
}

.messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--text-secondary);
  padding: 40px;
}

/* 空态欢迎语 */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: var(--text-secondary);
  gap: 8px;
  padding: 24px;
  min-height: 240px;
}

.empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--hover-color);
  color: var(--primary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-2xl);
  margin-bottom: 8px;
}

.empty-state h3 {
  color: var(--text-primary);
  font-size: var(--font-size-base);
  margin: 0;
}

.empty-state p {
  font-size: var(--font-size-xs);
  margin: 0;
}

.suggestion-list {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 12px;
  max-width: 440px;
}

.suggestion-chip {
  border: 1px solid var(--border-color);
  background: #ffffff;
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  padding: 8px 14px;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.2s;
}

.suggestion-chip:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

/* 历史加载更多 */
.load-more-btn {
  align-self: center;
  border: 1px solid var(--border-color);
  background: var(--bg-color);
  color: var(--text-secondary);
  font-size: var(--font-size-xxs);
  padding: 4px 12px;
  border-radius: 999px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s;
}

.load-more-btn:hover:not(:disabled) {
  color: var(--primary-color);
  border-color: var(--primary-color);
}

.load-more-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.message-wrapper {
  max-width: 80%;
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.message-wrapper.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}

.message-wrapper.user .message-bubble {
  background: var(--primary-color);
  color: white;
  border-bottom-right-radius: 4px;
}

.message-wrapper.bot {
  align-self: flex-start;
}

.message-wrapper.bot .message-bubble {
  /* 与 IM 窗口「对方/AI」气泡一致：用 --message-bubble-bg（浅色主题 #f3f4f6，
     深色主题 #334155）。此前用 --sidebar-bg（浅色主题纯白、深色主题与聊天背景
     同色），bot 消息在聊天背景上几乎看不出气泡轮廓。 */
  background: var(--message-bubble-bg);
  border-bottom-left-radius: 4px;
}

.message-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
}

/* 气泡 + 失败栏 + meta 行的纵向容器（对齐 IM MessageItem 的 .message-content）：
   作为 flex 子项按内容收缩，气泡/失败栏/meta 三个块级子元素同宽 */
.message-column {
  max-width: 100%;
  min-width: 0;
}

.message-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  position: relative;
  max-width: 100%;
}

/* AI 失败态气泡描边 */
.message-bubble.msg-failed {
  border: 1px solid #ffcdd2;
}

/* 底部 meta 行：时间 + 复制，位于气泡外（对齐 IM 消息列表惯例）。
   bot 消息左对齐、用户消息右对齐；整行悬停出现 */
.message-meta {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  margin-top: 4px;
  padding-left: 10px;
}

.message-wrapper.user .message-meta {
  justify-content: flex-end;
  padding-left: 0;
}

.time {
  font-size: var(--font-size-xxxs);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.message-wrapper:hover .time {
  opacity: 0.6;
}

/* 复制按钮：bot 与用户消息均可用，悬停出现 */
.meta-copy-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: rgba(0, 0, 0, 0.06);
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--font-size-xxs);
  opacity: 0;
  transition: opacity 0.15s;
}

.message-wrapper:hover .meta-copy-btn {
  opacity: 1;
}

.meta-copy-btn:hover {
  color: var(--primary-color);
  background: rgba(0, 0, 0, 0.12);
}

/* 时间分隔线（对齐 IM MessageListView） */
.time-divider {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 15px 0;
  position: relative;
}

.time-divider-text {
  background-color: var(--color-gray-200);
  color: var(--color-gray-500);
  font-size: var(--font-size-xxs);
  padding: 4px 12px;
  border-radius: 12px;
  text-align: center;
  font-weight: 400;
}

[data-theme="elegant-dark"] .time-divider-text {
  background-color: var(--sidebar-bg);
  color: var(--color-gray-700);
}

/* AI 失败态 / 用户发送失败态：重试栏，位于气泡外（bot 左对齐、用户右对齐） */
.failed-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding-top: 8px;
  padding-left: 10px;
  border-top: 1px dashed var(--border-color);
}

.message-wrapper.user .failed-bar {
  justify-content: flex-end;
  padding-left: 0;
}

.failed-hint {
  font-size: var(--font-size-xxs);
  color: #d32f2f;
  display: flex;
  align-items: center;
  gap: 4px;
}

.retry-btn {
  border: 1px solid #d32f2f;
  color: #d32f2f;
  background: transparent;
  font-size: var(--font-size-xxs);
  padding: 2px 10px;
  border-radius: 999px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: all 0.2s;
}

.retry-btn:hover {
  background: #d32f2f;
  color: #fff;
}

.content {
  font-size: var(--font-size-sm);
  line-height: 1.5;
  word-break: break-word;
}

.content :deep(.emoji-img) {
  width: 18px;
  height: 18px;
  vertical-align: middle;
  margin: 0 1px;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #d32f2f;
  background: #ffebee;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: var(--font-size-xs);
}

.input-area {
  padding: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

.input-area textarea {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  background: var(--bg-color);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  font-family: inherit;
  line-height: 1.5;
  outline: none;
  resize: none;
  min-height: 40px;
  max-height: 120px;
  overflow-y: auto;
  transition: border-color 0.2s;
}

.input-area textarea:focus {
  border-color: var(--primary-color);
}

.input-area textarea:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.send-btn {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s;
}

.send-btn:hover:not(:disabled) {
  background: var(--primary-hover);
}

.send-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 流式中：停止生成 */
.send-btn.stop-btn {
  background: #e5484d;
}

.send-btn.stop-btn:hover:not(:disabled) {
  background: #d8454a;
}
</style>