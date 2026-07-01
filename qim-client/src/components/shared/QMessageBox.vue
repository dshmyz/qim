<template>
  <div v-if="visible" class="q-message-box-mask" @click.self="handleClose">
    <div class="q-message-box">
      <div class="q-message-box__header">
        <h3 class="q-message-box__title">{{ title }}</h3>
        <button v-if="showClose" class="q-message-box__close" @click="handleClose">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      </div>
      <div class="q-message-box__body">
        <div v-if="type" :class="['q-message-box__icon', `q-message-box__icon--${type}`]">
          <svg v-if="type === 'warning'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
            <line x1="12" y1="9" x2="12" y2="13"/>
            <line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          <svg v-else-if="type === 'error'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M15 9l-6 6M9 9l6 6"/>
          </svg>
          <svg v-else-if="type === 'success'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M20 6L9 17l-5-5"/>
          </svg>
          <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="16" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12.01" y2="8"/>
          </svg>
        </div>
        <p class="q-message-box__message">{{ message }}</p>
        <div v-if="inputType" class="q-message-box__input">
          <input
            ref="inputRef"
            v-model="inputValue"
            :type="inputType"
            :placeholder="inputPlaceholder"
            class="q-message-box__input-field"
          />
        </div>
      </div>
      <div class="q-message-box__footer">
        <button
          v-if="showCancelButton"
          class="q-button q-button--default"
          @click="handleCancel"
        >
          {{ cancelButtonText }}
        </button>
        <button
          class="q-button q-button--primary"
          @click="handleConfirm"
        >
          {{ confirmButtonText }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'

interface MessageBoxOptions {
  title?: string
  message: string
  type?: 'warning' | 'error' | 'success' | 'info'
  confirmButtonText?: string
  cancelButtonText?: string
  showCancelButton?: boolean
  showClose?: boolean
  inputType?: 'text' | 'password' | ''
  inputPlaceholder?: string
}

interface MessageBoxResult {
  action: 'confirm' | 'cancel' | 'close'
  value?: string
}

const visible = ref(false)
const title = ref('提示')
const message = ref('')
const type = ref<MessageBoxOptions['type']>()
const confirmButtonText = ref('确定')
const cancelButtonText = ref('取消')
const showCancelButton = ref(true)
const showClose = ref(true)
const inputType = ref<MessageBoxOptions['inputType']>('')
const inputPlaceholder = ref('')
const inputValue = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
let resolve: ((result: MessageBoxResult) => void) | null = null

const show = (options: MessageBoxOptions): Promise<MessageBoxResult> => {
  title.value = options.title || '提示'
  message.value = options.message
  type.value = options.type
  confirmButtonText.value = options.confirmButtonText || '确定'
  cancelButtonText.value = options.cancelButtonText || '取消'
  showCancelButton.value = options.showCancelButton !== false
  showClose.value = options.showClose !== false
  inputType.value = options.inputType || ''
  inputPlaceholder.value = options.inputPlaceholder || ''
  inputValue.value = ''
  visible.value = true

  nextTick(() => {
    if (inputRef.value) {
      inputRef.value.focus()
    }
  })

  return new Promise((resolveFn) => {
    resolve = resolveFn
  })
}

const settle = (result: MessageBoxResult) => {
  visible.value = false
  resolve?.(result)
  resolve = null
}

const handleClose = () => {
  settle({ action: 'close' })
}

const handleCancel = () => {
  settle({ action: 'cancel' })
}

const handleConfirm = () => {
  settle({ action: 'confirm', value: inputValue.value })
}

const confirm = (message: string, title?: string, options?: Partial<MessageBoxOptions>) => {
  return show({
    title: title || '确认',
    message,
    type: 'warning',
    showCancelButton: true,
    ...options
  })
}

const alert = (message: string, title?: string) => {
  return show({
    title: title || '提示',
    message,
    type: 'info',
    showCancelButton: false
  })
}

const prompt = (message: string, title?: string, placeholder?: string) => {
  return show({
    title: title || '请输入',
    message,
    type: 'info',
    showCancelButton: true,
    inputType: 'text',
    inputPlaceholder: placeholder || '请输入内容'
  })
}

defineExpose({
  show,
  confirm,
  alert,
  prompt
})

if (!window.$QMessageBox) {
  window.$QMessageBox = {
    show,
    confirm,
    alert,
    prompt
  }
}
</script>

<style scoped>
.q-message-box-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10001;
}

.q-message-box {
  background: var(--panel-bg);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-2xl);
  width: 420px;
  max-width: 90%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
}

.q-message-box__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4) var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.q-message-box__title {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.q-message-box__close {
  width: 24px;
  height: 24px;
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

.q-message-box__close:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.q-message-box__close svg {
  width: 18px;
  height: 18px;
}

.q-message-box__body {
  display: flex;
  align-items: flex-start;
  gap: var(--spacing-3);
  padding: var(--spacing-6) var(--spacing-5);
  flex: 1;
  overflow-y: auto;
}

.q-message-box__icon {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
}

.q-message-box__icon svg {
  width: 24px;
  height: 24px;
}

.q-message-box__icon--warning {
  color: var(--color-warning-500);
}

.q-message-box__icon--error {
  color: var(--color-error-500);
}

.q-message-box__icon--success {
  color: var(--color-success-500);
}

.q-message-box__icon--info {
  color: var(--color-info-500);
}

.q-message-box__message {
  flex: 1;
  margin: 0;
  color: var(--text-color);
  font-size: var(--font-size-sm);
  line-height: var(--line-height-relaxed);
}

.q-message-box__input {
  width: 100%;
  margin-top: var(--spacing-4);
}

.q-message-box__input-field {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--input-bg);
  color: var(--text-color);
  font-size: var(--font-size-sm);
  outline: none;
  transition: border-color var(--transition-fast);
}

.q-message-box__input-field:focus {
  border-color: var(--primary-color);
}

.q-message-box__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
  padding: var(--spacing-4) var(--spacing-5);
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.q-button {
  padding: var(--spacing-2) var(--spacing-5);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  border: 1px solid transparent;
}

.q-button--primary {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.q-button--primary:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}

.q-button--default {
  background: var(--panel-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.q-button--default:hover {
  background: var(--color-gray-100);
}
</style>
