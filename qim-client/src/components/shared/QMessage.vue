<template>
  <TransitionGroup name="q-message">
    <div
      v-for="msg in messages"
      :key="msg.id"
      :class="['q-message', `q-message--${msg.type}`]"
    >
      <div class="q-message__icon">
        <svg v-if="msg.type === 'success'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M20 6L9 17l-5-5"/>
        </svg>
        <svg v-else-if="msg.type === 'error'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <path d="M15 9l-6 6M9 9l6 6"/>
        </svg>
        <svg v-else-if="msg.type === 'warning'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
        <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
      </div>
      <div class="q-message__content">{{ msg.content }}</div>
      <button class="q-message__close" @click="remove(msg.id)">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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
  top: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3) var(--spacing-4);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  background: var(--panel-bg);
  min-width: 300px;
  max-width: 600px;
  z-index: 100000;
}

.q-message--success {
  border: 1px solid var(--color-success-200);
}

.q-message--success .q-message__icon {
  color: var(--color-success-500);
}

.q-message--error {
  border: 1px solid var(--color-error-200);
}

.q-message--error .q-message__icon {
  color: var(--color-error-500);
}

.q-message--warning {
  border: 1px solid var(--color-warning-200);
}

.q-message--warning .q-message__icon {
  color: var(--color-warning-500);
}

.q-message--info {
  border: 1px solid var(--color-info-200);
}

.q-message--info .q-message__icon {
  color: var(--color-info-500);
}

.q-message__icon {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
}

.q-message__content {
  flex: 1;
  color: var(--text-color);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-normal);
}

.q-message__close {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.q-message__close:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.q-message__close svg {
  width: 16px;
  height: 16px;
}

.q-message-enter-active,
.q-message-leave-active {
  transition: all 0.3s ease;
}

.q-message-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}

.q-message-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(-20px);
}

.q-message-move {
  transition: transform 0.3s ease;
}
</style>
