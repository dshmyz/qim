<template>
  <div v-if="visible" class="image-preview-modal" @click="handleClose">
    <div class="image-preview-content" @click.stop>
      <div class="image-preview-header">
        <button class="close-btn" @click.stop="handleClose">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="image-preview-body">
        <img :src="imageUrl" alt="预览图片" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
  imageUrl: string
}

defineProps<Props>()
const emit = defineEmits<{ (e: 'close'): void }>()

const handleClose = () => {
  emit('close')
}
</script>

<style scoped>
/* 图片预览弹窗样式 */
.image-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(8px);
}

.image-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 800px;
  max-width: 90%;
  max-height: 80vh;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 8px 25px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.image-preview-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.image-preview-header .close-btn {
  background: #f0f0f0;
  border: 1px solid #ddd;
  color: #333;
  font-size: 18px;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  z-index: 10;
}

.image-preview-header .close-btn i {
  display: block !important;
  font-size: 16px !important;
  line-height: 1 !important;
}

.image-preview-body {
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  max-height: 60vh;
}

.image-preview-body img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}
</style>
