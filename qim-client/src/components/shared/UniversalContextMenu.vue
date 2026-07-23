<template>
  <Teleport to="body">
    <div v-if="visible" ref="menuRef" class="universal-context-menu" :style="menuStyle" @click.stop>
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
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import type { ContextMenuItem } from './context-menu-types'

const props = defineProps<{
  visible: boolean
  x: number
  y: number
  items: ContextMenuItem[]
}>()

const emit = defineEmits<{
  'update:visible': [val: boolean]
}>()

const menuRef = ref<HTMLElement | null>(null)
const adjustedX = ref(props.x)
const adjustedY = ref(props.y)

watch(() => [props.visible, props.x, props.y], () => {
  if (!props.visible) return
  adjustedX.value = props.x
  adjustedY.value = props.y
  nextTick(() => {
    if (!menuRef.value) return
    const rect = menuRef.value.getBoundingClientRect()
    let x = props.x
    let y = props.y
    if (x + rect.width > window.innerWidth) x = Math.max(0, window.innerWidth - rect.width - 4)
    if (y + rect.height > window.innerHeight) y = Math.max(0, window.innerHeight - rect.height - 4)
    adjustedX.value = x
    adjustedY.value = y
  })
}, { immediate: true })

const menuStyle = computed(() => ({
  left: adjustedX.value + 'px',
  top: adjustedY.value + 'px',
}))

const handleClick = (item: ContextMenuItem) => {
  item.action?.()
  emit('update:visible', false)
}

const close = () => emit('update:visible', false)

onMounted(() => {
  // 延迟注册，避免右键事件触发的 click 立即关闭菜单
  setTimeout(() => {
    document.addEventListener('click', close)
  }, 0)
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
