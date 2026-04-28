<template>
  <div class="message-content-image">
    <template v-if="imageLoaded && !loadError">
      <img
        :src="cachedUrl"
        class="message-image"
        @click="previewImage"
        @dblclick="previewImage"
        @load="onImageLoad"
        @error="onImageError"
      />
    </template>
    <template v-else>
      <ImagePlaceholder
        :is-loading="isLoading"
        :text="loadError ? '图片加载失败' : '加载中...'"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useImageCache } from '../../composables/useImageCache'
import ImagePlaceholder from './ImagePlaceholder.vue'

const props = defineProps<{
  src: string
  isSelf?: boolean
  serverUrl: string
}>()

const emit = defineEmits<{
  preview: [url: string]
}>()

const { getCachedImage, cacheImage, preloadImage } = useImageCache()
const cachedUrl = ref<string | undefined>(undefined)
const isLoading = ref(false)
const imageLoaded = ref(false)
const loadError = ref(false)

// 解析图片数据
const imageData = computed(() => {
  try {
    return JSON.parse(props.src)
  } catch {
    return { url: props.src }
  }
})

// 获取图片URL
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

// 加载图片
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
    return
  }

  isLoading.value = true
  loadError.value = false
  try {
    const result = await cacheImage(url)
    if (result) {
      cachedUrl.value = result
      imageLoaded.value = true
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
}

const onImageError = () => {
  loadError.value = true
  isLoading.value = false
}

watch(
  () => props.src,
  () => {
    cachedUrl.value = undefined
    imageLoaded.value = false
    loadError.value = false
    loadImage()
  },
  { immediate: true }
)

onMounted(() => {
  loadImage()
  preloadImage(fullImageUrl.value)
})

const previewImage = () => {
  emit('preview', cachedUrl.value || fullImageUrl.value)
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
  border: 1px solid var(--border-color);
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