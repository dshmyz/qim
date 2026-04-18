<template>
  <div class="message-content-image">
    <img :src="src" class="message-image" @click="previewImage" @dblclick="previewImage" />
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  src: string
  isSelf?: boolean
}>()

const emit = defineEmits<{
  preview: [url: string]
}>()

const previewImage = () => {
  emit('preview', props.src)
}
</script>

<style scoped>
.message-content-image {
  display: flex;
  align-items: center;
  max-width: 100%;
}

.message-image {
  max-width: 100%;
  max-height: 250px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s ease;
  object-fit: cover;
}

.message-image:hover {
  transform: scale(1.05);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 自己的图片消息样式 */
.message-content-image.self .message-image {
  border: 1px solid rgba(255, 255, 255, 0.2);
}
</style>