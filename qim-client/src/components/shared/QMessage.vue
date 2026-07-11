<template>
  <TransitionGroup name="q-message">
    <div
      v-for="msg in messages"
      :key="msg.id"
      :class="['q-message', `q-message--${msg.type}`]"
    >
      <div class="q-message__accent"></div>
      <div class="q-message__icon">
        <svg v-if="msg.type === 'success'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 6L9 17l-5-5"/>
        </svg>
        <svg v-else-if="msg.type === 'error'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 6L6 18"/>
          <path d="M6 6l12 12"/>
        </svg>
        <svg v-else-if="msg.type === 'warning'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
          <circle cx="12" cy="12" r="10"/>
        </svg>
      </div>
      <div class="q-message__content">{{ msg.content }}</div>
      <button class="q-message__close" @click="remove(msg.id)" aria-label="关闭">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>
  </TransitionGroup>
</template>

<script setup lang="ts">
import { ref } from 'vue'

interface Message {
  id: number
  type: 'success' | 'error' | 'warning' | 'info'
  content: string
  duration: number
  timer: number | null
}

const messages = ref<Message[]>([])
let nextId = 0

const remove = (id: number) => {
  const index = messages.value.findIndex(m => m.id === id)
  if (index > -1) {
    const msg = messages.value[index]
    if (msg.timer) {
      clearTimeout(msg.timer)
    }
    messages.value.splice(index, 1)
  }
}

const showMessage = (type: Message['type'], content: string, duration = 3000) => {
  const id = nextId++
  const timer = duration > 0 ? window.setTimeout(() => remove(id), duration) : null
  messages.value.push({
    id,
    type,
    content,
    duration,
    timer
  })
}

const success = (content: string, duration?: number) => {
  showMessage('success', content, duration)
}

const error = (content: string, duration?: number) => {
  showMessage('error', content, duration)
}

const warning = (content: string, duration?: number) => {
  showMessage('warning', content, duration)
}

const info = (content: string, duration?: number) => {
  showMessage('info', content, duration)
}

defineExpose({
  success,
  error,
  warning,
  info
})

if (!window.$QMessage) {
  window.$QMessage = {
    success,
    error,
    warning,
    info
  }
}
</script>

<style scoped>
.q-message {
  position: fixed;
  top: 24px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px 12px 14px;
  border-radius: 12px;
  /* 毛玻璃背景 */
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(16px) saturate(180%);
  -webkit-backdrop-filter: blur(16px) saturate(180%);
  /* 多层柔和阴影：环境光 + 投影 */
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.04),
    0 4px 12px rgba(0, 0, 0, 0.06),
    0 12px 32px rgba(0, 0, 0, 0.08);
  min-width: 320px;
  max-width: 560px;
  z-index: 100000;
  overflow: hidden;
  transition: transform 0.25s ease, box-shadow 0.25s ease;
}

.q-message:hover {
  transform: translateX(-50%) translateY(-1px);
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.05),
    0 6px 16px rgba(0, 0, 0, 0.08),
    0 16px 40px rgba(0, 0, 0, 0.1);
}

/* 深色主题适配 */
:root[data-theme="dark"] .q-message,
@media (prefers-color-scheme: dark) {
  .q-message {
    background: rgba(32, 32, 40, 0.78);
  }
}

/* 左侧彩色色条 */
.q-message__accent {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 3px;
  border-radius: 12px 0 0 12px;
}

/* 图标容器：带柔和背景圆 */
.q-message__icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.q-message__icon svg {
  width: 16px;
  height: 16px;
}

/* 配色：色条 + 图标背景 + 图标颜色 */
.q-message--success .q-message__accent {
  background: linear-gradient(180deg, #34d399, #10b981);
}
.q-message--success .q-message__icon {
  background: rgba(16, 185, 129, 0.12);
  color: #059669;
}

.q-message--error .q-message__accent {
  background: linear-gradient(180deg, #fb7185, #f43f5e);
}
.q-message--error .q-message__icon {
  background: rgba(244, 63, 94, 0.12);
  color: #e11d48;
}

.q-message--warning .q-message__accent {
  background: linear-gradient(180deg, #fbbf24, #f59e0b);
}
.q-message--warning .q-message__icon {
  background: rgba(245, 158, 11, 0.14);
  color: #d97706;
}

.q-message--info .q-message__accent {
  background: linear-gradient(180deg, #60a5fa, #3b82f6);
}
.q-message--info .q-message__icon {
  background: rgba(59, 130, 246, 0.12);
  color: #2563eb;
}

.q-message__content {
  flex: 1;
  color: #1f2937;
  font-size: 14px;
  font-weight: 500;
  line-height: 1.5;
}

:root[data-theme="dark"] .q-message__content,
@media (prefers-color-scheme: dark) {
  .q-message__content {
    color: #f3f4f6;
  }
}

.q-message__close {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: #9ca3af;
  cursor: pointer;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.q-message__close:hover {
  background: rgba(0, 0, 0, 0.06);
  color: #4b5563;
}

:root[data-theme="dark"] .q-message__close:hover,
@media (prefers-color-scheme: dark) {
  .q-message__close {
    color: #6b7280;
  }
  .q-message__close:hover {
    background: rgba(255, 255, 255, 0.08);
    color: #d1d5db;
  }
}

.q-message__close svg {
  width: 14px;
  height: 14px;
}

/* 弹性入场动画 */
.q-message-enter-active {
  transition: all 0.45s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.q-message-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 1, 1);
  position: absolute;
}

.q-message-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(-24px) scale(0.96);
}

.q-message-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-12px) scale(0.98);
}

.q-message-move {
  transition: transform 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}
</style>
