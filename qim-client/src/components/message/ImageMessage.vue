<template>
  <div class="message-content-image">
    <img v-if="cachedUrl" :src="cachedUrl" class="message-image" @click="previewImage" @dblclick="previewImage" />
    <img v-else :src="placeholderUrl" class="message-image loading" @click="previewImage" @dblclick="previewImage" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useImageCache } from '../../composables/useImageCache'

const props = defineProps<{
  src: string
  isSelf?: boolean
  serverUrl: string
}>()

const emit = defineEmits<{
  preview: [url: string]
}>()

const { getCachedImage, cacheImage, preloadImage } = useImageCache()
const cachedUrl = ref<string | null>(null)
const isLoading = ref(false)

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

// 占位符URL
const placeholderUrl = computed(() => {
  return 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" width="100" height="100"%3E%3Crect fill="%23f0f0f0" width="100" height="100"/%3E%3Ctext x="50" y="50" text-anchor="middle" dy=".3em" fill="%23999"%3E%3C/animation%3E%3C/text%3E%3C/svg%3E'
})

// 加载图片
const loadImage = async () => {
  if (isLoading.value) return

  const url = fullImageUrl.value
  if (!url) return

  const cached = getCachedImage(url)
  if (cached) {
    cachedUrl.value = cached
    return
  }

  isLoading.value = true
  try {
    const result = await cacheImage(url)
    if (result) {
      cachedUrl.value = result
    }
  } finally {
    isLoading.value = false
  }
}

watch(() => props.src, () => {
  cachedUrl.value = null
  loadImage()
}, { immediate: true })

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

.message-image.loading {
  opacity: 0.7;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.7; }
  50% { opacity: 0.4; }
}

/* 自己的图片消息样式 */
.message-content-image.self .message-image {
  border: 1px solid rgba(255, 255, 255, 0.2);
}
</style>