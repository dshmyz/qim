<template>
  <QDialog
    :visible="visible"
    :title="title"
    width="420px"
    :close-on-click-mask="false"
    @update:visible="handleClose"
  >
    <div class="confirm-body">
      <div v-if="danger" class="confirm-icon-wrap confirm-icon--danger">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
          <line x1="12" y1="9" x2="12" y2="13"/>
          <line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
      </div>
      <p class="confirm-message">{{ message }}</p>
    </div>
    <template #footer>
      <button class="q-btn q-btn--default" @click="handleCancel">
        {{ cancelText }}
      </button>
      <button :class="['q-btn', danger ? 'q-btn--danger' : 'q-btn--primary']" @click="handleConfirm">
        {{ confirmText }}
      </button>
    </template>
  </QDialog>
</template>

<script setup lang="ts">
import QDialog from './QDialog.vue'
interface Props {
  visible: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  confirmText: '确定',
  cancelText: '取消',
  danger: false
})

const emit = defineEmits<Emits>()

const handleClose = (value: boolean) => {
  emit('update:visible', value)
  if (!value) emit('cancel')
}

const handleCancel = () => {
  emit('update:visible', false)
  emit('cancel')
}

const handleConfirm = () => {
  emit('confirm')
}
</script>

<style scoped>
.confirm-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 14px;
}

.confirm-icon-wrap {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.confirm-icon--danger {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.confirm-icon--danger svg {
  width: 22px;
  height: 22px;
}

.confirm-message {
  margin: 0;
  font-size: var(--font-size-sm);
  line-height: 1.65;
  color: var(--text-secondary, #666);
}

.q-btn {
  padding: 8px 22px;
  border-radius: var(--radius-md, 8px);
  font-size: var(--font-size-sm);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast, 0.15s);
  min-width: 80px;
  border: 1px solid var(--border-color, #e5e7eb);
}

.q-btn--default {
  background: var(--right-content-bg, #fff);
  color: var(--text-color, #333);
}

.q-btn--default:hover {
  border-color: var(--primary-color, #409eff);
  color: var(--primary-color, #409eff);
}

.q-btn--primary {
  background: var(--primary-color, #409eff);
  color: white;
  border-color: var(--primary-color, #409eff);
}

.q-btn--primary:hover {
  background: var(--primary-dark, #337ecc);
  border-color: var(--primary-dark, #337ecc);
}

.q-btn--danger {
  background: #ef4444;
  color: white;
  border-color: #ef4444;
}

.q-btn--danger:hover {
  background: #dc2626;
  border-color: #dc2626;
}
</style>
