<template>
  <div class="avatar-group-reply-notice" @click="$emit('click')">
    <div class="notice-icon">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
      </svg>
    </div>
    <div class="notice-content">
      <span class="notice-text">你的分身在群聊「{{ groupName }}」中代你回复了{{ triggerName }}</span>
      <span class="notice-action">点击查看</span>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  groupName: string
  triggerName: string
}>()

defineEmits<{
  click: []
}>()
</script>

<style scoped>
.avatar-group-reply-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin: 4px 0;
  background: rgba(59, 130, 246, 0.06);
  border: 1px solid rgba(59, 130, 246, 0.15);
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.avatar-group-reply-notice:hover {
  background: rgba(59, 130, 246, 0.12);
}

.notice-icon {
  color: #3b82f6;
  flex-shrink: 0;
}

.notice-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.notice-text {
  font-size: 13px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notice-action {
  font-size: 12px;
  color: #3b82f6;
  white-space: nowrap;
  flex-shrink: 0;
}

[data-theme="elegant-dark"] .avatar-group-reply-notice {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.25);
}
</style>
