<template>
  <span :class="['avatar-reply-badge', `avatar-reply-badge--${variant}`]">
    <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
      <path d="M12 2a2 2 0 011 .26A2 2 0 0114 2h4a2 2 0 012 2v4a2 2 0 01-.26 1A2 2 0 0120 10v4a2 2 0 01-2 2h-4a2 2 0 01-1-.26A2 2 0 0112 16H8a2 2 0 01-2-2v-4a2 2 0 01.26-1A2 2 0 016 8V4a2 2 0 012-2h4zm0 2H8v4h4V4zm6 0h-4v4h4V4zm-6 6H8v4h4v-4zm6 0h-4v4h4v-4z"/>
    </svg>
    <span class="badge-text">AI 代回复</span>
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'badge' | 'footer' | 'both'
}>(), {
  variant: 'badge'
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
