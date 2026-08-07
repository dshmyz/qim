<template>
  <Transition name="fab">
    <button
      v-if="show"
      class="floating-ai-ball"
      :class="{ active: isSidebarOpen, dragging: isDragging }"
      :style="ballStyle"
      title="AI 助手"
      @mousedown.left="onMouseDown"
      @touchstart.passive="onTouchStart"
      @contextmenu.prevent="onContextMenu"
    >
      <i class="fas fa-robot"></i>
      <span v-if="!isSidebarOpen && !isDragging" class="fab-pulse"></span>

      <!-- 右键菜单 -->
      <UniversalContextMenu menuId="fab" :items="menuItems" />
    </button>
  </Transition>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useAISidebar } from '../../composables/useAISidebar'
import { openMenu } from '../../composables/useUI'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'

const { showFloatingAIBall: show, showAISidebar: isSidebarOpen, toggleSidebar, toggleFloatingBall } = useAISidebar()

// ── 右键菜单 ──
const menuItems = computed<ContextMenuItem[]>(() => [
  {
    label: isSidebarOpen.value ? '关闭 AI 助手' : '打开 AI 助手',
    icon: 'fas fa-robot',
    action: () => toggleSidebar(),
  },
  {
    label: '重置位置',
    icon: 'fas fa-crosshairs',
    action: () => resetPosition(),
  },
  { divider: true },
  {
    label: '隐藏悬浮球',
    icon: 'fas fa-eye-slash',
    action: () => toggleFloatingBall(false),
  },
])

const onContextMenu = (e: MouseEvent) => {
  // 如果正在拖动，不弹菜单
  if (isDragging.value) return
  openMenu('fab', e.clientX, e.clientY)
}

// ── 位置状态 ──
const ballLeft = ref<number | null>(null)
const ballTop = ref<number | null>(null)

const winW = ref(typeof window !== 'undefined' ? window.innerWidth : 1920)
const winH = ref(typeof window !== 'undefined' ? window.innerHeight : 1080)

const BALL_SIZE = 52
const MARGIN = 32
const MOBILE_SIZE = 48
const MOBILE_MARGIN = 20
// 默认位置在视口右下角再往上抬一段，避免与底部聊天输入区的发送按钮重叠。
// 输入区最小约 150px 高、球 52px，抬升 120px 让球浮在输入区上方但仍靠右下。
const DEFAULT_TOP_LIFT = 120
const MOBILE_TOP_LIFT = 100

const size = computed(() => winW.value <= 768 ? MOBILE_SIZE : BALL_SIZE)
const margin = computed(() => winW.value <= 768 ? MOBILE_MARGIN : MARGIN)
const topLift = computed(() => winW.value <= 768 ? MOBILE_TOP_LIFT : DEFAULT_TOP_LIFT)

const ballStyle = computed(() => {
  const s = size.value
  const m = margin.value
  const left = ballLeft.value ?? (winW.value - s - m)
  const top = ballTop.value ?? (winH.value - s - m - topLift.value)
  return {
    left: `${left}px`,
    top: `${top}px`,
    right: 'auto',
    bottom: 'auto',
  }
})

const resetPosition = () => {
  ballLeft.value = null
  ballTop.value = null
  localStorage.removeItem('fab_position')
}

// ── 拖动逻辑 ──
const isDragging = ref(false)
const hasMoved = ref(false)
const startX = ref(0)
const startY = ref(0)
const startLeft = ref(0)
const startTop = ref(0)

const clamp = (val: number, min: number, max: number) => Math.max(min, Math.min(val, max))

const onMouseDown = (e: MouseEvent) => {
  startDrag(e.clientX, e.clientY)
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

const onMouseMove = (e: MouseEvent) => {
  if (!isDragging.value) return
  e.preventDefault()
  doDrag(e.clientX, e.clientY)
}

const onMouseUp = () => {
  if (!isDragging.value) return
  finishDrag()
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  if (!hasMoved.value) {
    toggleSidebar()
  }
}

const onTouchStart = (e: TouchEvent) => {
  const t = e.touches[0]
  startDrag(t.clientX, t.clientY)
  document.addEventListener('touchmove', onTouchMove, { passive: false })
  document.addEventListener('touchend', onTouchEnd)
}

const onTouchMove = (e: TouchEvent) => {
  if (!isDragging.value) return
  e.preventDefault()
  const t = e.touches[0]
  doDrag(t.clientX, t.clientY)
}

const onTouchEnd = () => {
  if (!isDragging.value) return
  finishDrag()
  document.removeEventListener('touchmove', onTouchMove)
  document.removeEventListener('touchend', onTouchEnd)
  if (!hasMoved.value) {
    toggleSidebar()
  }
}

const startDrag = (clientX: number, clientY: number) => {
  isDragging.value = true
  hasMoved.value = false
  startX.value = clientX
  startY.value = clientY
  startLeft.value = ballLeft.value ?? (winW.value - size.value - margin.value)
  startTop.value = ballTop.value ?? (winH.value - size.value - margin.value)
}

const doDrag = (clientX: number, clientY: number) => {
  const dx = clientX - startX.value
  const dy = clientY - startY.value

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    hasMoved.value = true
  }

  const newLeft = startLeft.value + dx
  const newTop = startTop.value + dy
  const s = size.value
  const m = margin.value

  ballLeft.value = clamp(newLeft, m, winW.value - s - m)
  ballTop.value = clamp(newTop, m, winH.value - s - m)
}

const finishDrag = () => {
  isDragging.value = false
  if (hasMoved.value) {
    localStorage.setItem('fab_position', JSON.stringify({
      left: ballLeft.value,
      top: ballTop.value,
    }))
  }
}

// ── 位置恢复 ──
const restorePosition = () => {
  const stored = localStorage.getItem('fab_position')
  if (stored) {
    try {
      const pos = JSON.parse(stored)
      ballLeft.value = pos.left
      ballTop.value = pos.top
    } catch { /* ignore */ }
  }
}

const onResize = () => {
  winW.value = window.innerWidth
  winH.value = window.innerHeight
  const s = size.value
  const m = margin.value
  if (ballLeft.value !== null) {
    ballLeft.value = clamp(ballLeft.value, m, winW.value - s - m)
  }
  if (ballTop.value !== null) {
    ballTop.value = clamp(ballTop.value, m, winH.value - s - m)
  }
  if (ballLeft.value !== null || ballTop.value !== null) {
    localStorage.setItem('fab_position', JSON.stringify({
      left: ballLeft.value,
      top: ballTop.value,
    }))
  }
}

onMounted(() => {
  restorePosition()
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.removeEventListener('touchmove', onTouchMove)
  document.removeEventListener('touchend', onTouchEnd)
})
</script>

<style scoped>
.floating-ai-ball {
  position: fixed;
  width: 52px;
  height: 52px;
  border: none;
  border-radius: 50%;
  background: var(--primary-color, #6366f1);
  color: #fff;
  font-size: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  z-index: 999;
  box-shadow: 0 4px 16px rgba(99, 102, 241, 0.4);
  transition: transform 0.2s, box-shadow 0.2s;
  user-select: none;
  -webkit-user-select: none;
  touch-action: none;
}

.floating-ai-ball:hover {
  transform: scale(1.08);
  box-shadow: 0 6px 24px rgba(99, 102, 241, 0.5);
}

.floating-ai-ball:active {
  transform: scale(0.95);
}

.floating-ai-ball.dragging {
  transform: scale(1.12);
  cursor: grabbing;
  box-shadow: 0 8px 32px rgba(99, 102, 241, 0.6);
  transition: none;
}

.floating-ai-ball.active {
  background: var(--hover-color, #e0e0e0);
  color: var(--text-color, #333);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.fab-pulse {
  position: absolute;
  inset: -4px;
  border-radius: 50%;
  border: 2px solid var(--primary-color, #6366f1);
  animation: pulse 2s ease-out infinite;
}

@keyframes pulse {
  0% { opacity: 0.6; transform: scale(0.9); }
  70% { opacity: 0; transform: scale(1.3); }
  100% { opacity: 0; transform: scale(1.3); }
}

.fab-enter-active,
.fab-leave-active {
  transition: transform 0.25s ease, opacity 0.25s ease;
}
.fab-enter-from,
.fab-leave-to {
  transform: scale(0) translateY(20px);
  opacity: 0;
}

@media (max-width: 768px) {
  .floating-ai-ball {
    width: 48px;
    height: 48px;
    font-size: 20px;
  }
}
</style>
