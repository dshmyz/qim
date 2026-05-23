<template>
  <div class="channel-avatar-wrapper">
    <Avatar
      :src="avatar"
      :name="name"
      :server-url="serverUrl"
      :alt="`${name}的头像`"
      :size="size"
      :shape="shape"
      class="channel-avatar"
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
import { computed } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { getStoredServerUrl } from '../../composables/useServerUrl'

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
  serverUrl: () => getStoredServerUrl()
})

const showTypeIcon = computed(() => {
  return props.publishPermission === 'creator_only' || props.publishPermission === 'anyone'
})
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
