<template>
  <div v-if="visible" class="screenshot-preview-modal" @click="handleCancel">
    <div class="screenshot-preview-content" @click.stop>
      <div class="screenshot-preview-header">
        <h3>截图预览</h3>
        <button class="close-btn" @click.stop="handleCancel">&times;</button>
      </div>
      <div class="screenshot-preview-body">
        <div class="screenshot-image-container">
          <img :src="imageData" class="screenshot-image" alt="截图" />
        </div>
      </div>
      <div class="screenshot-preview-footer">
        <button class="screenshot-btn retake-btn" @click.stop="handleRetake">重新截图</button>
        <button class="screenshot-btn cancel-btn" @click.stop="handleCancel">取消</button>
        <button class="screenshot-btn send-btn" @click.stop="handleSend">发送</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  imageData: string
}

defineProps<Props>()
const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'retake'): void
  (e: 'send'): void
}>()

const handleCancel = () => {
  emit('cancel')
}

const handleRetake = () => {
  emit('retake')
}

const handleSend = () => {
  emit('send')
}
</script>

<style scoped>
/* 截图预览样式 */
.screenshot-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(5px);
}

.screenshot-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  animation: modalFadeIn 0.3s ease;
}

.screenshot-preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.screenshot-preview-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.screenshot-preview-body {
  flex: 1;
  padding: 24px;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--content-bg);
}

.screenshot-image-container {
  max-width: 100%;
  max-height: 60vh;
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  background: #fff;
  padding: 16px;
}

.screenshot-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 4px;
}

.screenshot-preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

.screenshot-btn {
  padding: 10px 24px;
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.retake-btn {
  background: var(--list-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.retake-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.cancel-btn {
  background: var(--list-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.cancel-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.send-btn {
  background: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.send-btn:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
  transform: translateY(-1px);
}
</style>
