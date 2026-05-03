<template>
  <div v-if="visible" class="avatar-takeover-banner">
    <div class="banner-content">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" class="banner-icon">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"/>
      </svg>
      <span class="banner-text">分身已暂停，{{ remainingText }}后恢复</span>
    </div>
    <div class="banner-actions">
      <button class="banner-btn resume" @click="$emit('resume')">立即恢复</button>
      <button class="banner-btn extend" @click="$emit('extend')">继续暂停</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'

const props = defineProps<{
  takeoverUntil: string | null
}>()

defineEmits<{
  resume: []
  extend: []
}>()

const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  timer = setInterval(() => { now.value = Date.now() }, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const visible = computed(() => {
  if (!props.takeoverUntil) return false
  return new Date(props.takeoverUntil).getTime() > now.value
})

const remainingSeconds = computed(() => {
  if (!props.takeoverUntil) return 0
  const diff = new Date(props.takeoverUntil).getTime() - now.value
  return Math.max(0, Math.floor(diff / 1000))
})

const remainingText = computed(() => {
  const mins = Math.floor(remainingSeconds.value / 60)
  const secs = remainingSeconds.value % 60
  if (mins > 0) return `${mins}分${secs}秒`
  return `${secs}秒`
})
</script>

<style scoped>
.avatar-takeover-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: #FFF8E1;
  border-bottom: 1px solid rgba(255, 152, 0, 0.2);
  font-size: 13px;
}

.banner-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.banner-icon {
  color: #FF9800;
  flex-shrink: 0;
}

.banner-text {
  color: var(--text-primary);
}

.banner-actions {
  display: flex;
  gap: 8px;
}

.banner-btn {
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  border: none;
  transition: opacity 0.2s;
}

.banner-btn:hover {
  opacity: 0.85;
}

.banner-btn.resume {
  background: var(--primary-color);
  color: white;
}

.banner-btn.extend {
  background: transparent;
  color: #FF9800;
  border: 1px solid #FF9800;
}

[data-theme="elegant-dark"] .avatar-takeover-banner {
  background: rgba(255, 152, 0, 0.1);
  border-bottom-color: rgba(255, 152, 0, 0.3);
}
</style>
