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
      </div>
      <div v-if="isLoading" class="loading-badge">加载中...</div>
    </div>

    <div class="messages" ref="messagesRef">
      <!-- 加载状态 -->
      <div v-if="isLoading && messages.length === 0" class="loading-state">
        <i class="fas fa-spinner fa-spin"></i>
        <span>加载历史消息...</span>
      </div>

      <!-- 消息列表 -->
      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['message', msg.senderType === 'user' ? 'user' : 'bot']"
      >
        <div class="content">
          <MarkdownRenderer
            v-if="msg.type === 'markdown' || msg.senderType === 'bot'"
            :content="msg.content"
          />
          <span v-else>{{ msg.content }}</span>
          <span v-if="msg.isStreaming" class="streaming-cursor"></span>
        </div>
        <div class="time">{{ formatTime(msg.timestamp) }}</div>
      </div>

      <!-- 思考指示器 -->
      <ThinkingIndicator v-if="isStreaming && !hasStreamingMessage" />

      <!-- 错误提示 -->
      <div v-if="error" class="error-message">
        <i class="fas fa-exclamation-circle"></i>
        <span>{{ error }}</span>
      </div>
    </div>

    <div class="input-area">
      <input
        v-model="input"
        :placeholder="`向 ${bot?.name || 'AI助手'} 提问...`"
        :disabled="isSending || isStreaming"
        @keyup.enter="sendMessage"
      >
      <button
        @click="sendMessage"
        class="send-btn"
        :disabled="isSending || isStreaming || !input.trim()"
      >
        <i class="fas fa-paper-plane"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import Avatar from '../../shared/Avatar.vue'
import MarkdownRenderer from '../../shared/MarkdownRenderer.vue'
import ThinkingIndicator from '../../shared/ThinkingIndicator.vue'
import type { BotMessage } from '../../../types/bot'

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
  error: string | null
}>()

const emit = defineEmits<{
  back: []
  send: [content: string]
  clearMessages: []
  newConversation: []
}>()

const input = ref('')
const messagesRef = ref<HTMLDivElement | null>(null)

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
 * 清空对话
 */
function handleClearMessages() {
  if (confirm('确定要清空对话记录吗？')) {
    emit('clearMessages')
  }
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

/**
 * 格式化时间
 */
function formatTime(date: Date) {
  return new Date(date).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 监听消息变化，自动滚动到底部
watch(() => props.messages, () => {
  scrollToBottom()
}, { deep: true })

// 监听流式状态变化
watch(() => props.isStreaming, () => {
  scrollToBottom()
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
  font-size: 16px;
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

.loading-badge {
  font-size: 12px;
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

.message {
  max-width: 80%;
  padding: 10px 14px;
  border-radius: 12px;
  position: relative;
}

.message.user {
  align-self: flex-end;
  background: var(--primary-color);
  color: white;
  border-bottom-right-radius: 4px;
}

.message.bot {
  align-self: flex-start;
  background: var(--sidebar-bg);
  border-bottom-left-radius: 4px;
}

.time {
  font-size: 11px;
  opacity: 0.6;
  margin-top: 4px;
  text-align: right;
}

.content {
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}

.streaming-cursor {
  display: inline-block;
  width: 2px;
  height: 1em;
  background: var(--primary-color);
  animation: blink 1s infinite;
  margin-left: 2px;
  vertical-align: text-bottom;
}

@keyframes blink {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0; }
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #d32f2f;
  background: #ffebee;
  padding: 10px 14px;
  border-radius: 8px;
  font-size: 13px;
}

.input-area {
  padding: 16px;
  border-top: 1px solid var(--border-color);
  display: flex;
  gap: 8px;
}

.input-area input {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 20px;
  background: var(--bg-color);
  color: var(--text-primary);
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
}

.input-area input:focus {
  border-color: var(--primary-color);
}

.input-area input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.send-btn {
  width: 40px;
  height: 40px;
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
</style>
