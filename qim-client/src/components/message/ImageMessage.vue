<template>
  <div class="message-content-image">
    <ImagePlaceholder
      class="image-placeholder-overlay"
      :class="{ 'placeholder-hidden': imageLoaded && !loadError }"
      :is-loading="isLoading"
      :text="loadError ? '图片加载失败' : '加载中...'"
    />
    <img
      :src="cachedUrl || ''"
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
import { computed, ref, watch } from 'vue'
import { useImageCache } from '../../composables/useImageCache'
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

const { getCachedImage, cacheImage, preloadImage } = useImageCache()
const cachedUrl = ref<string | undefined>(undefined)
const isLoading = ref(false)
const imageLoaded = ref(false)
const loadError = ref(false)

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

const loadImage = async () => {
  if (isLoading.value || imageLoaded.value) return

  const url = fullImageUrl.value
  if (!url) {
    loadError.value = true
    return
  }

  const cached = getCachedImage(url)
  if (cached) {
    cachedUrl.value = cached
    imageLoaded.value = true
    emit('imageLoaded')
    return
  }

  isLoading.value = true
  loadError.value = false
  try {
    const result = await cacheImage(url)
    if (result) {
      cachedUrl.value = result
      imageLoaded.value = true
      emit('imageLoaded')
    } else {
      loadError.value = true
    }
  } catch (error) {
    console.error('Failed to load image:', error)
    loadError.value = true
  } finally {
    isLoading.value = false
  }
}

const onImageLoad = () => {
  imageLoaded.value = true
  loadError.value = false
  emit('imageLoaded')
}

const onImageError = () => {
  loadError.value = true
  isLoading.value = false
}

// 同步从缓存初始化：在 watch immediate 之前设置初始状态，避免闪烁
const tryInitFromCache = () => {
  const url = fullImageUrl.value
  if (!url) return false
  const cached = getCachedImage(url)
  if (cached) {
    cachedUrl.value = cached
    imageLoaded.value = true
    return true
  }
  return false
}

// 监听 src 变化，先检查缓存再决定是否加载
watch(
  () => props.src,
  (newSrc, oldSrc) => {
    if (newSrc !== oldSrc) {
      // 先同步检查缓存，有缓存则直接设置状态，跳过异步加载
      if (tryInitFromCache()) {
        return
      }
      // 没有缓存才重置状态并异步加载
      cachedUrl.value = undefined
      imageLoaded.value = false
      loadError.value = false
      isLoading.value = false
      loadImage()
    }
  },
  { immediate: true }
)

const previewImage = () => {
  emit('preview', cachedUrl.value || fullImageUrl.value)
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
  border: 1px solid var(--border-color);
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
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
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
