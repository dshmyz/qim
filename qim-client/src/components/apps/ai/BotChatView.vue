<template>
  <div class="bot-chat-view">
    <div class="chat-header">
      <button class="back-btn" @click="$emit('back')">
        <i class="fas fa-arrow-left"></i>
      </button>
      <h3>{{ bot?.name || 'AI助手' }}</h3>
    </div>

    <div class="messages" ref="messagesRef">
      <div
        v-for="msg in messages"
        :key="msg.id"
        :class="['message', msg.sender === 'user' ? 'user' : 'bot']"
      >
        <div class="content">{{ msg.content }}</div>
        <div class="time">{{ formatTime(msg.timestamp) }}</div>
      </div>
      <div v-if="thinking" class="message bot thinking">
        <div class="thinking-indicator">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </div>
      </div>
    </div>

    <div class="input-area">
      <input
        v-model="input"
        :placeholder="`向 ${bot?.name} 提问...`"
        @keyup.enter="sendMessage"
      >
      <button @click="sendMessage" class="send-btn">
        <i class="fas fa-paper-plane"></i>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'

interface Message {
  id: number
  content: string
  sender: 'user' | 'bot' | 'system'
  timestamp: Date
}

interface Bot {
  id: number
  name: string
  description?: string
  avatar?: string
}

const props = defineProps<{
  bot: Bot | null
  messages: Message[]
  thinking: boolean
}>()

const emit = defineEmits<{
  back: []
  send: [content: string]
  setThinking: [value: boolean]
}>()

const input = ref('')
const messagesRef = ref<HTMLDivElement | null>(null)

async function sendMessage() {
  if (!input.value.trim()) return
  emit('send', input.value.trim())
  input.value = ''
  emit('setThinking', true)
  await scrollToBottom()
}

async function scrollToBottom() {
  await nextTick()
  if (messagesRef.value) {
    messagesRef.value.scrollTop = messagesRef.value.scrollHeight
  }
}

function formatTime(date: Date) {
  return new Date(date).toLocaleTimeString('zh-CN')
}

watch(() => props.messages, () => {
  scrollToBottom()
}, { deep: true })
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

.messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
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

.send-btn:hover {
  background: var(--primary-hover);
}

.thinking-indicator {
  display: flex;
  gap: 4px;
  padding: 12px 14px;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--primary-color);
  animation: pulse 1.5s infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}
</style>
