<template>
  <div v-if="visible" ref="menuRef" class="universal-context-menu" :style="{ left: pos.x + 'px', top: pos.y + 'px' }" @click.stop>
    <template v-for="(item, i) in items" :key="i">
      <div v-if="item.visible !== false && item.divider" class="ucm-divider"></div>
      <div
        v-else-if="item.visible !== false"
        class="ucm-item"
        :class="{ danger: item.danger }"
        @click="handleClick(item)"
      >
        <span v-if="item.icon" class="ucm-icon"><i :class="item.icon" :style="item.iconColor ? { color: item.iconColor, fontSize: item.iconColor ? '8px' : '' } : {}"></i></span>
        <span class="ucm-label">{{ item.label }}</span>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted } from 'vue'

export interface ContextMenuItem {
  label?: string
  icon?: string
  iconColor?: string
  action?: () => void
  visible?: boolean
  divider?: boolean
  danger?: boolean
}

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  items: ContextMenuItem[]
}>()

const emit = defineEmits<{
  'update:visible': [val: boolean]
}>()

const menuRef = ref<HTMLElement>()
const pos = ref({ x: props.x, y: props.y })

// 菜单显示时，nextTick 读 DOM 实际尺寸修正位置（不溢出屏幕）
watch(() => props.visible, (val) => {
  if (val) {
    pos.value = { x: props.x, y: props.y }
    nextTick(() => {
      const el = menuRef.value
      if (!el) return
      const w = el.offsetWidth
      const h = el.offsetHeight
      const ww = window.innerWidth
      const wh = window.innerHeight
      let x = props.x
      let y = props.y
      if (x + w > ww) x = ww - w - 10
      if (x < 0) x = 10
      if (y + h > wh) y = wh - h - 10
      if (y < 0) y = 10
      pos.value = { x, y }
    })
  }
})

const handleClick = (item: ContextMenuItem) => {
  item.action?.()
  emit('update:visible', false)
}

const close = () => emit('update:visible', false)

onMounted(() => {
  document.addEventListener('click', close)
})

onUnmounted(() => {
  document.removeEventListener('click', close)
})
</script>

<style scoped>
.universal-context-menu {
  position: fixed;
  z-index: 3000;
  min-width: 160px;
  max-width: 240px;
  max-height: calc(100vh - 20px);
  overflow-y: auto;
  background: var(--card-bg, #fff);
  border: 1px solid var(--border-color, #e5e7eb);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  padding: 6px 0;
}
.ucm-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-color, #333);
  transition: background 0.15s;
}
.ucm-item:hover {
  background: var(--hover-color, #f3f4f6);
}
.ucm-item.danger {
  color: #e5484d;
}
.ucm-item.danger:hover {
  background: rgba(229, 72, 77, 0.08);
}
.ucm-icon {
  width: 16px;
  text-align: center;
  flex-shrink: 0;
}
.ucm-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.ucm-divider {
  height: 1px;
  background: var(--border-color, #e5e7eb);
  margin: 4px 0;
}
</style>
