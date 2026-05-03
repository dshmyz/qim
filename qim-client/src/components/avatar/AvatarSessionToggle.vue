<template>
  <div class="avatar-session-toggle" :class="{ active: isActive }">
    <button
      class="toggle-btn"
      :title="isActive ? '分身已开启，点击关闭' : '点击开启分身'"
      @click="handleToggle"
    >
      <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
        <path d="M12 2a2 2 0 011 .26A2 2 0 0114 2h4a2 2 0 012 2v4a2 2 0 01-.26 1A2 2 0 0120 10v4a2 2 0 01-2 2h-4a2 2 0 01-1-.26A2 2 0 0112 16H8a2 2 0 01-2-2v-4a2 2 0 01.26-1A2 2 0 016 8V4a2 2 0 012-2h4zm0 2H8v4h4V4zm6 0h-4v4h4V4zm-6 6H8v4h4v-4zm6 0h-4v4h4v-4z"/>
      </svg>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAvatar } from '../../composables/useAvatar'

const props = defineProps<{
  conversationId: string | number
}>()

const { config, toggleSession, isAvatarActive } = useAvatar()

const isActive = computed(() => {
  return config.value?.enabled && isAvatarActive(props.conversationId)
})

async function handleToggle() {
  if (!config.value?.enabled) {
    window.$QMessage.warning('请先在分身设置中开启分身功能')
    return
  }
  try {
    await toggleSession(props.conversationId, !isActive.value)
    window.$QMessage.success(isActive.value ? '分身已关闭' : '分身已开启')
  } catch {
    window.$QMessage.error('操作失败')
  }
}
</script>

<style scoped>
.avatar-session-toggle {
  display: flex;
  align-items: center;
}

.toggle-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  cursor: pointer;
  border-radius: 6px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.toggle-btn:hover {
  background: var(--hover-color);
  color: var(--text-primary);
}

.avatar-session-toggle.active .toggle-btn {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.avatar-session-toggle.active .toggle-btn:hover {
  background: rgba(59, 130, 246, 0.2);
}
</style>
