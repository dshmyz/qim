<template>
  <div v-if="visible" class="screenshot-preview-modal" @click="$emit('cancel')">
    <div class="screenshot-preview-content" @click.stop>
      <div class="screenshot-preview-header">
        <h3>截图预览</h3>
        <button class="close-btn" @click.stop="$emit('cancel')">&times;</button>
      </div>
      <div class="screenshot-preview-body">
        <div class="screenshot-image-container">
          <img :src="imageData" class="screenshot-image" alt="截图" />
        </div>
      </div>
      <div class="screenshot-preview-footer">
        <button class="screenshot-btn retake-btn" @click.stop="$emit('retake')">重新截图</button>
        <button class="screenshot-btn cancel-btn" @click.stop="$emit('cancel')">取消</button>
        <button class="screenshot-btn send-btn" @click.stop="$emit('send')">发送</button>
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
defineEmits<{
  (e: 'cancel'): void
  (e: 'retake'): void
  (e: 'send'): void
}>()
</script>
