<template>
  <Teleport to="body">
    <div v-if="isVisible" ref="menuRef" class="universal-context-menu" :style="menuStyle" @click.stop>
      <template v-for="(item, i) in items" :key="i">
        <div v-if="item.visible !== false && item.divider" class="ucm-divider"></div>
        <div
          v-else-if="item.visible !== false"
          class="ucm-item"
          :class="{ danger: item.danger }"
          @click="handleClick(item)"
        >
          <span v-if="item.icon" class="ucm-icon"><i :class="item.icon" :style="item.iconColor ? { color: item.iconColor } : {}"></i></span>
          <span class="ucm-label">{{ item.label }}</span>
        </div>
      </template>
      <slot />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted, watch } from 'vue'
import type { ContextMenuItem } from './context-menu-types'
import { activeMenu, activeMenuPosition, openMenu, closeMenu, computeContextMenuPosition } from '../../composables/useUI'

const props = withDefaults(defineProps<{
  visible?: boolean
  x?: number
  y?: number
  items: ContextMenuItem[]
  /** 传入后自动管理全局 activeMenu，保证同一时间只有一个菜单 */
  menuId?: string
}>(), { visible: true, x: 0, y: 0, menuId: '' })

const emit = defineEmits<{
  'update:visible': [val: boolean]
}>()

const menuRef = ref<HTMLElement | null>(null)
const adjustedX = ref(props.x)
const adjustedY = ref(props.y)

// 实际坐标：有 menuId 时从 activeMenuPosition 读取
const posX = computed(() => props.menuId ? activeMenuPosition.value.x : props.x)
const posY = computed(() => props.menuId ? activeMenuPosition.value.y : props.y)

// menuId 模式：visible 变 true 时自动注册到 activeMenu
watch(() => props.visible, (val) => {
  if (val && props.menuId) {
    openMenu(props.menuId, props.x, props.y)
  }
})

// 实际可见性：有 menuId 时必须 activeMenu 匹配
const isVisible = computed(() => {
  if (!props.visible) return false
  if (props.menuId) return activeMenu.value === props.menuId
  return true
})

watch(() => [isVisible.value, posX.value, posY.value], () => {
  if (!isVisible.value) return
  adjustedX.value = posX.value
  adjustedY.value = posY.value
  nextTick(() => {
    if (!menuRef.value) return
    const rect = menuRef.value.getBoundingClientRect()
    const pos = computeContextMenuPosition(
      posX.value,
      posY.value,
      rect.width,
      rect.height,
    )
    adjustedX.value = pos.x
    adjustedY.value = pos.y
  })
}, { immediate: true })

const menuStyle = computed(() => ({
  left: adjustedX.value + 'px',
  top: adjustedY.value + 'px',
}))

const handleClick = (item: ContextMenuItem) => {
  item.action?.()
  if (props.menuId) closeMenu()
  else emit('update:visible', false)
}

const close = (e: MouseEvent) => {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    // 触发按钮自己管理 toggle 逻辑，跳过 document 监听器的关闭，避免
    // capturing 阶段先清空 activeMenu 导致按钮 handler 误判为"关闭"后重新打开
    if ((e.target as HTMLElement)?.closest?.('[data-menu-trigger]')) return
    if (props.menuId) {
      // 只在自己还是活跃菜单时关闭，避免旧菜单关掉新菜单
      if (activeMenu.value === props.menuId) closeMenu()
    } else {
      emit('update:visible', false)
    }
  }
}

onMounted(() => {
  setTimeout(() => {
    document.addEventListener('click', close, true)
    document.addEventListener('contextmenu', close, true)
  }, 0)
})

onUnmounted(() => {
  document.removeEventListener('click', close, true)
  document.removeEventListener('contextmenu', close, true)
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
  border-radius: 10px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);
  padding: 6px;
}
.ucm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 12.5px;
  color: var(--text-color, #333);
  transition: background 0.15s, color 0.15s;
}
.ucm-item:hover {
  background: var(--hover-color, #f3f4f6);
}
.ucm-item.danger {
  color: #e5484d;
}
.ucm-item.danger .ucm-icon {
  color: #e5484d;
}
.ucm-item.danger:hover {
  background: rgba(229, 72, 77, 0.08);
}
.ucm-icon {
  width: 18px;
  text-align: center;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--primary-color);
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
  margin: 4px 6px;
}
</style>
