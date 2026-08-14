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
    <span
      v-if="resolvedBadge"
      class="avatar-badge"
      :class="badgeClass"
      :style="badgeStyle"
      :title="resolvedBadge.title"
    >
      <i v-if="resolvedBadge.type === 'icon'" :class="resolvedBadge.icon"></i>
      <span v-else-if="resolvedBadge.type === 'text'" class="avatar-badge-text">{{ resolvedBadge.text }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { getAvatarColor, getInitial, isAbsoluteUrl } from '../../utils/avatar'
import type { AvatarBadge } from '../../utils/user'

/**
 * 角标：由调用方注入，Avatar 只负责渲染在头像右下角。
 * - type='status'：在线状态点（带颜色）
 * - type='icon' ：图标角标（FontAwesome）
 * - type='text' ：文字角标（如「群」「讨」）
 *
 * 接口定义集中在 utils/user.ts，便于业务层（buildConversationBadge 等）
 * 构造角标对象而不依赖组件文件。
 */
interface Props {
  src?: string | null
  name?: string
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  shape?: 'circle' | 'rounded'
  serverUrl?: string
  alt?: string
  /** 调用方注入的右下角角标；null 表示显式无角标 */
  badge?: AvatarBadge | null
}

const props = withDefaults(defineProps<Props>(), {
  name: '用户',
  size: 'md',
  shape: 'circle',
  serverUrl: '',
  alt: '头像',
  badge: undefined
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
    return cleanServerUrl + '/' + cleanAvatar
  }

  return props.src
})

const showImage = computed(() => {
  return !imageError.value && !!imageSrc.value
})

const initial = computed(() => getInitial(props.name || '用户'))
const fallbackStyle = computed(() => {
  const color = getAvatarColor(props.name || '用户')
  return { backgroundColor: color }
})

const sizeClass = computed(() => `avatar-${props.size}`)
const shapeClass = computed(() => `avatar-${props.shape}`)

// 统一以 resolvedBadge 作为单一数据源，避免 props.badge 与 resolvedBadge 混用造成风格不一致。
const resolvedBadge = computed<AvatarBadge | null>(() => props.badge || null)

const resolvedBadgeSize = computed(() => resolvedBadge.value?.size || 'md')

const badgeClass = computed(() => {
  const classes = [`badge-size-${resolvedBadgeSize.value}`]
  if (resolvedBadge.value?.type === 'status' && resolvedBadge.value.status) {
    classes.push(`status-${resolvedBadge.value.status}`)
  }
  if (resolvedBadge.value?.ring === false) {
    classes.push('badge-no-ring')
  }
  return classes
})

const badgeStyle = computed(() => {
  const color = resolvedBadge.value?.color
  return color ? { backgroundColor: color } : undefined
})

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
  position: relative;
}

.avatar-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: inherit;
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
  border-radius: inherit;
}

.avatar-xs {
  width: 28px;
  height: 28px;
  font-size: var(--font-size-xxs);
}

.avatar-sm {
  width: 32px;
  height: 32px;
  font-size: var(--font-size-sm);
}

.avatar-md {
  width: 40px;
  height: 40px;
  font-size: var(--font-size-lg);
}

.avatar-lg {
  width: 48px;
  height: 48px;
  font-size: var(--font-size-xl);
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

/* 右下角角标容器（通用）：由调用方注入内容，定位在头像右下角 */
.avatar-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1;
  overflow: hidden;
  color: #fff;
}

/* 去掉外圈白环 + 阴影：适用于角标较小或需要更干净的场景（badge.ring = false） */
.avatar-badge.badge-no-ring {
  border: none;
  box-shadow: none;
}

/* 角标尺寸 */
.badge-size-sm {
  width: 30%;
  height: 30%;
  font-size: 0.32em;
}

.badge-size-md {
  width: 35%;
  height: 35%;
  font-size: 0.4em;
}

.badge-size-lg {
  width: 44%;
  height: 44%;
  font-size: 0.5em;
}

/* type=status：纯色状态点（背景为状态色，无内容） */
.avatar-badge.status-online {
  background-color: #52c41a;
}

.avatar-badge.status-offline {
  background-color: #d9d9d9;
}

.avatar-badge.status-busy {
  background-color: #ff4d4f;
}

/* 图标/文字角标内容 */
.avatar-badge .fa-solid,
.avatar-badge .fas {
  line-height: 1;
}

.avatar-badge-text {
  line-height: 1;
  font-weight: 600;
}
</style>
