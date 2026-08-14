<template>
  <Teleport to="body">
    <div v-if="visible" class="avatar-quick-menu" :style="menuStyle" @click.stop>
      <button class="avatar-quick-menu__item" @click="handleMention">
        <span class="avatar-quick-menu__icon">@</span>
        <span>{{ userName }}</span>
      </button>
    </div>
    <div v-if="visible" class="avatar-quick-menu__backdrop" @click="close" @contextmenu.prevent="close" />
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'

const visible = ref(false)
const position = ref({ x: 0, y: 0 })
const userName = ref('')
const userData = ref<any>(null)
let onMention: ((user: any) => void) | null = null

const menuStyle = computed(() => ({
  left: `${position.value.x}px`,
  top: `${position.value.y}px`,
}))

const open = (user: any, event: MouseEvent, callback: (user: any) => void) => {
  userData.value = user
  userName.value = user.name || user.username || 'TA'
  onMention = callback

  // 防止菜单超出视口
  const menuW = 140
  const menuH = 36
  const x = Math.min(event.clientX, window.innerWidth - menuW - 8)
  const y = Math.min(event.clientY, window.innerHeight - menuH - 8)
  position.value = { x, y }
  visible.value = true
}

const close = () => {
  visible.value = false
  userData.value = null
  onMention = null
}

const handleMention = () => {
  if (onMention && userData.value) {
    onMention(userData.value)
  }
  close()
}

// ESC 关闭
const handleEsc = (e: KeyboardEvent) => {
  if (e.key === 'Escape') close()
}
document.addEventListener('keydown', handleEsc)
onUnmounted(() => document.removeEventListener('keydown', handleEsc))

defineExpose({ open, close })
</script>

<style scoped>
.avatar-quick-menu__backdrop {
  position: fixed;
  inset: 0;
  z-index: 9998;
}
.avatar-quick-menu {
  position: fixed;
  z-index: 9999;
  min-width: 120px;
  padding: 4px;
  border-radius: 8px;
  background: var(--surface-elevated, #fff);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12), 0 0 0 1px rgba(0, 0, 0, 0.06);
  animation: avatarMenuIn 0.12s ease-out;
}
@keyframes avatarMenuIn {
  from { opacity: 0; transform: scale(0.95); }
  to   { opacity: 1; transform: scale(1); }
}
.avatar-quick-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 12px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-primary, #1a1a1a);
  font-size: var(--font-size-xs, 12px);
  cursor: pointer;
  transition: background 0.1s;
}
.avatar-quick-menu__item:hover {
  background: var(--fill-tsp-blue-1, rgba(37, 99, 235, 0.08));
}
.avatar-quick-menu__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border-radius: 4px;
  background: var(--accent-blue-bg, #EBF1FF);
  color: var(--accent-blue, #2563eb);
  font-size: var(--font-size-xxs);
  font-weight: 700;
  flex-shrink: 0;
}
</style>
