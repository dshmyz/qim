<template>
  <div v-if="visible" class="image-preview-modal" @click="$emit('close')">
    <div class="image-preview-content" @click.stop>
      <div class="image-preview-header">
        <button class="close-btn" @click.stop="$emit('close')">
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
defineEmits<{ (e: 'close'): void }>()
</script>
