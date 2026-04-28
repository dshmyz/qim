<template>
  <Teleport to="body">
    <div 
      v-if="visible" 
      class="modal-container-overlay" 
      @click="handleOverlayClick"
      :style="overlayStyle"
    >
      <div 
        class="modal-container-content" 
        @click.stop
        :style="contentStyle"
      >
        <div v-if="showHeader" class="modal-container-header">
          <h3 class="modal-container-title">{{ title }}</h3>
          <button v-if="closable" class="modal-container-close" @click="$emit('close')">×</button>
        </div>
        <div class="modal-container-body">
          <slot></slot>
        </div>
        <div v-if="showFooter" class="modal-container-footer">
          <slot name="footer">
            <button class="modal-btn modal-btn-cancel" @click="$emit('cancel')">取消</button>
            <button class="modal-btn modal-btn-confirm" @click="$emit('confirm')">确认</button>
          </slot>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  visible: boolean
  title?: string
  width?: string
  minWidth?: string
  maxWidth?: string
  maxHeight?: string
  closable?: boolean
  showHeader?: boolean
  showFooter?: boolean
  overlayStyle?: Record<string, string>
  contentStyle?: Record<string, string>
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  title: '',
  width: 'auto',
  minWidth: '400px',
  maxWidth: '90vw',
  maxHeight: '85vh',
  closable: true,
  showHeader: true,
  showFooter: true
})

const emit = defineEmits<{
  'close': []
  'cancel': []
  'confirm': []
}>()

const computedContentStyle = computed(() => ({
  width: props.width,
  minWidth: props.minWidth,
  maxWidth: props.maxWidth,
  maxHeight: props.maxHeight,
  ...props.contentStyle
}))

const handleOverlayClick = () => {
  // 点击遮罩层时关闭模态框
  emit('close')
}
</script>

<style scoped>
.modal-container-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  padding: 20px;
  box-sizing: border-box;
  animation: fadeIn 0.3s ease;
}

.modal-container-content {
  background-color: var(--card-bg, #fff);
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: 85vh;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
  animation: slideIn 0.3s ease;
  min-width: 400px;
}

.modal-container-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-color, #eee);
  background-color: var(--bg-color, #fff);
  flex-shrink: 0;
}

.modal-container-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary, #333);
  margin: 0;
}

.modal-container-close {
  width: 32px;
  height: 32px;
  border: none;
  background-color: transparent;
  color: var(--text-secondary, #999);
  font-size: 24px;
  font-weight: bold;
  cursor: pointer;
  border-radius: 50%;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-container-close:hover {
  background-color: var(--hover-color, #f5f5f5);
  color: var(--text-primary, #333);
}

.modal-container-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.modal-container-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color, #eee);
  background-color: var(--bg-color, #fff);
  flex-shrink: 0;
}

.modal-btn {
  padding: 8px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.modal-btn-cancel {
  background-color: var(--bg-color, #fff);
  color: var(--text-secondary, #666);
  border: 1px solid var(--border-color, #ddd);
}

.modal-btn-cancel:hover {
  background-color: var(--hover-color, #f5f5f5);
  color: var(--text-primary, #333);
  border-color: var(--primary-color, #409eff);
  transform: translateY(-1px);
}

.modal-btn-confirm {
  background-color: var(--primary-color, #409eff);
  color: white;
}

.modal-btn-confirm:hover {
  background-color: var(--active-color, #66b1ff);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.3);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from { transform: translateY(-20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

@media (max-width: 768px) {
  .modal-container-overlay {
    padding: 10px;
  }
  
  .modal-container-content {
    max-width: 100%;
    max-height: 90vh;
  }
  
  .modal-container-header {
    padding: 12px 16px;
  }
  
  .modal-container-title {
    font-size: 16px;
  }
  
  .modal-container-body {
    padding: 16px;
  }
  
  .modal-container-footer {
    padding: 12px 16px;
    flex-direction: column;
  }
  
  .modal-btn {
    width: 100%;
  }
}

@media (max-width: 480px) {
  .modal-container-content {
    width: 100%;
    min-width: auto;
    max-height: 95vh;
  }
}
</style>
