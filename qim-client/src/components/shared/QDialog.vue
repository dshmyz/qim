<template>
  <Teleport to="body">
    <Transition name="q-dialog-fade">
      <div v-if="visible" class="q-dialog-mask" @click.self="handleMaskClick">
        <div :class="['q-dialog', className]" :style="dialogStyle">
          <div class="q-dialog__header">
            <h3 class="q-dialog__title">{{ title }}</h3>
            <button v-if="showClose" class="q-dialog__close" @click="handleClose">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
          <div class="q-dialog__body">
            <slot></slot>
          </div>
          <div v-if="$slots.footer" class="q-dialog__footer">
            <slot name="footer"></slot>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  visible: boolean
  title?: string
  width?: string
  className?: string
  showClose?: boolean
  closeOnClickMask?: boolean
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
  width: '500px',
  className: '',
  showClose: true,
  closeOnClickMask: true
})

const emit = defineEmits<Emits>()

const dialogStyle = computed(() => ({
  width: props.width
}))

const handleClose = () => {
  emit('update:visible', false)
  emit('close')
}

const handleMaskClick = () => {
  if (props.closeOnClickMask) {
    handleClose()
  }
}
</script>

<style scoped>
.q-dialog-mask {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.q-dialog {
  background: var(--panel-bg);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-2xl);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.q-dialog__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4) var(--spacing-5);
  border-bottom: 1px solid var(--border-color);
}

.q-dialog__title {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.q-dialog__close {
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

.q-dialog__close:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.q-dialog__close svg {
  width: 18px;
  height: 18px;
}

.q-dialog__body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-5);
}

.q-dialog__footer {
  padding: var(--spacing-4) var(--spacing-5);
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-3);
}

.q-dialog-fade-enter-active,
.q-dialog-fade-leave-active {
  transition: all 0.3s ease;
}

.q-dialog-fade-enter-from,
.q-dialog-fade-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
