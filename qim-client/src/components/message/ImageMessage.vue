<template>
  <div ref="observerTarget" class="message-content-image">
    <ImagePlaceholder
      class="image-placeholder-overlay"
      :class="{ 'placeholder-hidden': imageLoaded && !loadError }"
      :is-loading="isLoading"
      :text="loadError ? '图片加载失败' : '加载中...'"
    />
    <img
      v-if="isVisible && fullImageUrl"
      :src="fullImageUrl"
      class="message-image"
      :class="{ 'image-hidden': !imageLoaded || loadError }"
      @click="previewImage"
      @dblclick="previewImage"
      @load="onImageLoad"
      @error="onImageError"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useIntersectionObserver } from '../../composables/useIntersectionObserver'
import ImagePlaceholder from './ImagePlaceholder.vue'

const props = defineProps<{
  src: string
  isSelf?: boolean
  serverUrl: string
}>()

const emit = defineEmits<{
  preview: [url: string]
  imageLoaded: []
}>()

const observerTarget = ref<HTMLElement | null>(null)
const { isVisible } = useIntersectionObserver(observerTarget)

const imageLoaded = ref(false)
const loadError = ref(false)

const isLoading = computed(() => isVisible.value && !imageLoaded.value && !loadError.value)

const imageData = computed(() => {
  try {
    return JSON.parse(props.src)
  } catch {
    return { url: props.src }
  }
})

const fullImageUrl = computed(() => {
  const url = imageData.value.url || props.src
  if (url.startsWith('http')) {
    return url
  } else {
    const cleanServerUrl = props.serverUrl.replace(/\/$/, '')
    const cleanUrl = url.replace(/^\//, '')
    return `${cleanServerUrl}/${cleanUrl}`
  }
})

const onImageLoad = () => {
  imageLoaded.value = true
  loadError.value = false
  emit('imageLoaded')
}

const onImageError = () => {
  loadError.value = true
}

const previewImage = () => {
  emit('preview', fullImageUrl.value)
}
</script>

<style scoped>
.message-content-image {
  display: flex;
  align-items: center;
  max-width: 100%;
  position: relative;
}

.message-image {
  max-width: 100%;
  max-height: 250px;
  border-radius: 4px;
  cursor: pointer;
  transition: opacity 0.15s ease;
  object-fit: cover;
}

.image-hidden {
  opacity: 0;
  pointer-events: none;
  position: absolute;
  left: 0;
  top: 0;
  visibility: hidden;
}

.message-image:hover {
  transform: scale(1.05);
}

.image-placeholder-overlay {
  transition: opacity 0.15s ease;
}

.placeholder-hidden {
  opacity: 0;
  pointer-events: none;
  position: absolute;
  left: 0;
  top: 0;
  z-index: -1;
  visibility: hidden;
}

.message-content-image.self .message-image {
  border: 1px solid rgba(255, 255, 255, 0.2);
}
</style>