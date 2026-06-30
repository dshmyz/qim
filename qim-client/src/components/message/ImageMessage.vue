<template>
  <div ref="observerTarget" class="message-content-image media-preview">
    <ImagePlaceholder
      class="image-placeholder-overlay"
      :class="{ 'placeholder-hidden': imageLoaded && !loadError }"
      :is-loading="isLoading"
      :text="loadError ? '图片加载失败' : '加载中...'"
    />
    <img
      v-if="isVisible && fullImageUrl"
      :src="fullImageUrl"
      class="message-image media-preview__image"
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
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  position: relative;
  border-radius: 14px;
  overflow: clip;
  border: 1px solid color-mix(in srgb, var(--border-color), transparent 20%);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.06);
  box-sizing: border-box;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.message-image {
  max-width: 100%;
  max-height: 250px;
  border-radius: 13px;
  border: none;
  cursor: pointer;
  transition: opacity 0.15s ease;
  object-fit: cover;
  box-sizing: border-box;
  display: block;
}

.message-content-image:hover {
  border-color: color-mix(in srgb, var(--primary-color), transparent 58%);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
}

.image-hidden {
  opacity: 0;
  pointer-events: none;
  position: absolute;
  left: 0;
  top: 0;
  visibility: hidden;
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
  border: none;
}

.message-content-image.self {
  border-color: color-mix(in srgb, var(--border-color), transparent 20%);
}

[data-theme="elegant-dark"] .message-content-image {
  box-shadow: none;
  border-color: rgba(255, 255, 255, 0.12);
}

[data-theme="elegant-dark"] .message-image {
  border: none;
}
</style>
