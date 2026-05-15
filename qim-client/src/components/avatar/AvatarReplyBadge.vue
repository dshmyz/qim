<template>
  <span :class="['avatar-reply-badge', `avatar-reply-badge--${variant}`]">
    <i class="fas fa-robot"></i>
    <span v-if="variant === 'badge'" class="badge-text">{{ userName }}的分身{{ avatarName }}</span>
    <span v-else-if="variant === 'footer'" class="badge-text">该消息由 {{ userName }} 的分身{{ avatarName }}回复</span>
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'badge' | 'footer' | 'both'
  userName?: string
  avatarName?: string
}>(), {
  variant: 'badge',
  userName: '',
  avatarName: ''
})
</script>

<style scoped>
.avatar-reply-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
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
  font-size: 11px;
  padding: 0;
  opacity: 0.6;
}

.avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.1);
  color: #3b82f6;
}

.badge-text {
  font-size: 11px;
}

[data-theme="elegant-dark"] .avatar-reply-badge--badge,
[data-theme="elegant-dark"] .avatar-reply-badge--both {
  background: rgba(59, 130, 246, 0.2);
}
</style>