<template>
  <div class="channel-avatar-wrapper">
    <img
      v-if="!imageError"
      :src="avatarUrl"
      :alt="`${name}的头像`"
      class="channel-avatar"
      :class="[`avatar-${size}`, `avatar-${shape}`]"
      @error="onImageError"
    />
    <CssAvatar
      v-else
      :name="name"
      :size="size"
      :shape="shape"
    />
    <span
      v-if="showTypeIcon"
      class="avatar-type-icon"
      :class="publishPermission === 'creator_only' ? 'broadcast' : 'collaborative'"
    >
      <i :class="publishPermission === 'creator_only' ? 'fas fa-bullhorn' : 'fas fa-comments'"></i>
    </span>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { API_BASE_URL } from '../../config'
import CssAvatar from '../shared/CssAvatar.vue'

interface Props {
  avatar?: string | null
  name: string
  publishPermission?: string
  size?: 'sm' | 'md' | 'lg'
  shape?: 'circle' | 'rounded'
  serverUrl?: string
}

const props = withDefaults(defineProps<Props>(), {
  publishPermission: '',
  size: 'md',
  shape: 'circle',
  serverUrl: () => localStorage.getItem('serverUrl') || API_BASE_URL
})

const imageError = ref(false)

const avatarUrl = computed(() => {
  return getAvatarUrl(props.avatar, props.name, props.serverUrl)
})

const showTypeIcon = computed(() => {
  return props.publishPermission === 'creator_only' || props.publishPermission === 'anyone'
})

const onImageError = () => {
  imageError.value = true
}
</script>

<style scoped>
.channel-avatar-wrapper {
  position: relative;
  flex-shrink: 0;
}

.channel-avatar {
  object-fit: cover;
}

.avatar-sm {
  width: 40px;
  height: 40px;
}

.avatar-md {
  width: 44px;
  height: 44px;
}

.avatar-lg {
  width: 48px;
  height: 48px;
}

.avatar-rounded {
  border-radius: var(--radius-md);
}

.avatar-circle {
  border-radius: 50%;
}

.avatar-type-icon {
  position: absolute;
  bottom: -3px;
  right: -3px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  border: 2px solid var(--card-bg, white);
}

.avatar-sm + .avatar-type-icon,
.avatar-sm ~ .avatar-type-icon {
  width: 16px;
  height: 16px;
  font-size: 7px;
  bottom: -2px;
  right: -2px;
}

.avatar-type-icon.broadcast {
  background: var(--primary-color);
  color: white;
}

.avatar-type-icon.collaborative {
  background: var(--success-color);
  color: white;
}
</style>
