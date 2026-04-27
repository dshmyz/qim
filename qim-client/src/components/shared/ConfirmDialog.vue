<template>
  <QDialog
    :visible="visible"
    :title="title"
    width="420px"
    :close-on-click-mask="false"
    @update:visible="handleClose"
  >
    <p class="confirm-message">{{ message }}</p>
    <template #footer>
      <button class="q-btn q-btn--default" @click="handleCancel">
        {{ cancelText }}
      </button>
      <button class="q-btn q-btn--primary" @click="handleConfirm">
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
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  confirmText: '确定',
  cancelText: '取消'
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
.confirm-message {
  margin: 0;
  font-size: var(--font-size-base);
  line-height: 1.6;
  color: var(--text-secondary);
}

.q-btn {
  padding: 8px 20px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  min-width: 80px;
  border: 1px solid var(--border-color);
}

.q-btn--default {
  background: var(--right-content-bg);
  color: var(--text-color);
}

.q-btn--default:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.q-btn--primary {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.q-btn--primary:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
}
</style>
