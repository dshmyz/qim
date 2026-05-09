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
      v-if="showStatusDot"
      class="avatar-status"
      :class="`status-${status}`"
      :title="statusTitle"
    ></span>
    <span
      v-else-if="showTypeIcon"
      class="avatar-type-icon"
      :class="`type-${userType}`"
      :title="typeTitle"
    >
      <i :class="typeIconClass"></i>
    </span>
    <span
      v-if="showConversationTypeIcon"
      class="avatar-conversation-icon"
      :class="`conv-type-${conversationType}`"
      :title="conversationTypeTitle"
    >
      <i :class="conversationTypeIconClass"></i>
    </span>
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
  status?: 'online' | 'offline' | 'busy'
  userType?: 'user' | 'bot' | 'system' | 'api'
  conversationType?: 'single' | 'group' | 'discussion' | 'bot'
}

const props = withDefaults(defineProps<Props>(), {
  name: '用户',
  size: 'md',
  shape: 'circle',
  serverUrl: '',
  alt: '头像',
  status: undefined,
  userType: 'user',
  conversationType: 'single'
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

const isNonUserType = computed(() => {
  return props.userType && props.userType !== 'user'
})

const showStatusDot = computed(() => {
  return props.status && !isNonUserType.value
})

const showTypeIcon = computed(() => {
  return isNonUserType.value
})

const statusTitle = computed(() => {
  const titleMap = {
    online: '在线',
    offline: '离线',
    busy: '忙碌'
  }
  return titleMap[props.status || 'offline']
})

const typeIconMap = {
  bot: 'fas fa-robot',
  system: 'fas fa-cog',
  api: 'fas fa-plug',
  user: ''
}

const typeTitleMap = {
  bot: '机器人',
  system: '系统',
  api: 'API',
  user: '用户'
}

const typeIconClass = computed(() => typeIconMap[props.userType || 'user'])
const typeTitle = computed(() => typeTitleMap[props.userType || 'user'])

const showConversationTypeIcon = computed(() => {
  return props.conversationType === 'group' || props.conversationType === 'discussion'
})

const conversationTypeIconMap = {
  single: '',
  group: 'fas fa-users',
  discussion: 'fas fa-comments',
  bot: 'fas fa-robot'
}

const conversationTypeTitleMap = {
  single: '单聊',
  group: '群聊',
  discussion: '讨论组',
  bot: '机器人'
}

const conversationTypeIconClass = computed(() => conversationTypeIconMap[props.conversationType || 'single'])
const conversationTypeTitle = computed(() => conversationTypeTitleMap[props.conversationType || 'single'])

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

.avatar-status {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 30%;
  height: 30%;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1);
  z-index: 1;
}

.status-online {
  background-color: #52c41a;
}

.status-offline {
  background-color: #d9d9d9;
}

.status-busy {
  background-color: #ff4d4f;
}

.avatar-type-icon {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 35%;
  height: 35%;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.4em;
  z-index: 1;
}

.type-bot {
  color: #fff;
  background: var(--primary-color, #1890ff);
}

.type-system {
  color: #fff;
  background: var(--color-warning-500, #faad14);
}

.type-api {
  color: #fff;
  background: var(--color-success-500, #52c41a);
}

.avatar-conversation-icon {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 35%;
  height: 35%;
  border-radius: 50%;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.4em;
  z-index: 1;
}

.conv-type-group {
  color: #fff;
  background: var(--primary-color, #1890ff);
}

.conv-type-discussion {
  color: #fff;
  background: var(--color-success-500, #52c41a);
}
</style>
