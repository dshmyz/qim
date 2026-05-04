<template>
  <div class="avatar-wrapper" :class="[sizeClass, shapeClass]">
    <img
      v-if="showImage && imageSrc"
      :src="imageSrc"
      :alt="alt"
      class="avatar-image"
      @error="handleError"
      @load="handleLoad"
    />
    <div v-else class="avatar-fallback" :style="fallbackStyle">
      {{ initial }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getAvatarColor, getInitial, isAbsoluteUrl } from '../../utils/avatar'

interface Props {
  src?: string | null
  name?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  shape?: 'circle' | 'rounded'
  serverUrl?: string
  alt?: string
}

const props = withDefaults(defineProps<Props>(), {
  name: '用户',
  size: 'md',
  shape: 'circle',
  serverUrl: '',
  alt: '头像'
})

const imageError = ref(false)
const imageLoaded = ref(false)

const imageSrc = computed(() => {
  if (!props.src || !props.src.trim()) {
    return null
  }
  
  if (isAbsoluteUrl(props.src)) {
    return props.src
  }
  
  if (props.serverUrl) {
    const cleanServerUrl = props.serverUrl.replace(/\/$/, '')
    const cleanAvatar = props.src.replace(/^\//, '')
    return `${cleanServerUrl}/${cleanAvatar}`
  }
  
  return null
})

const showImage = computed(() => {
  return imageSrc.value && !imageError.value
})

const initial = computed(() => getInitial(props.name || ''))
const color = computed(() => getAvatarColor(props.name || ''))

const fallbackStyle = computed(() => ({
  backgroundColor: color.value
}))

const sizeClass = computed(() => `avatar-${props.size}`)
const shapeClass = computed(() => `avatar-${props.shape}`)

const handleError = () => {
  console.warn(`[Avatar] 加载失败: ${imageSrc.value}`)
  imageError.value = true
}

const handleLoad = () => {
  imageLoaded.value = true
}

watch(() => props.src, () => {
  imageError.value = false
  imageLoaded.value = false
})
</script>

<style scoped>
.avatar-wrapper {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  overflow: hidden;
  position: relative;
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: #fff;
  text-transform: uppercase;
  user-select: none;
}

.avatar-sm {
  width: 32px;
  height: 32px;
  font-size: 14px;
}

.avatar-md {
  width: 40px;
  height: 40px;
  font-size: 18px;
}

.avatar-lg {
  width: 48px;
  height: 48px;
  font-size: 20px;
}

.avatar-xl {
  width: 64px;
  height: 64px;
  font-size: 28px;
}

.avatar-circle {
  border-radius: 50%;
}

.avatar-rounded {
  border-radius: 10px;
}
</style>
