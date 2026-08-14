<template>
  <span :class="['avatar-reply-badge', `avatar-reply-badge--${variant}`]">
    <i class="fas fa-robot"></i>
    <span v-if="variant === 'badge'" class="badge-text">{{ userName }}的分身{{ avatarName }}</span>
    <span v-else-if="variant === 'footer'" class="badge-text">{{ footerText }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  variant?: 'badge' | 'footer' | 'both'
  userName?: string
  avatarName?: string
  isOwn?: boolean
}>(), {
  variant: 'badge',
  userName: '',
  avatarName: '',
  isOwn: false
})

// footer 文案：用空格把分身名字单独隔出，便于区分；无分身名时不留多余空格
const footerText = computed(() => {
  const owner = props.isOwn ? '您的' : `${props.userName}的`
  const avatar = props.avatarName ? ` ${props.avatarName} ` : ''
  return `由${owner}分身${avatar}回复`
})
</script>

<style scoped>
.avatar-reply-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: var(--font-size-xxxs);
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
}

.avatar-reply-badge--badge {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.avatar-reply-badge--footer {
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xxxs);
  padding: 0;
  opacity: 0.6;
}

.avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.badge-text {
  font-size: var(--font-size-xxxs);
}

[data-theme="elegant-dark"] .avatar-reply-badge--badge,
[data-theme="elegant-dark"] .avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.2);
}
</style>