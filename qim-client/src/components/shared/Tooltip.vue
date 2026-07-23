<template>
  <div class="tooltip-wrap" @mouseenter="onEnter" @mouseleave="onLeave" ref="wrapRef">
    <slot />
    <Teleport to="body">
      <div v-if="show" class="tooltip-popup" :style="popupStyle" ref="popupRef">
        {{ text }}
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'

const props = defineProps<{
  text: string
  delay?: number
  maxWidth?: number
}>()

const show = ref(false)
const wrapRef = ref<HTMLElement | null>(null)
const popupRef = ref<HTMLElement | null>(null)
const popupStyle = ref<Record<string, string>>({})
let timer: ReturnType<typeof setTimeout> | null = null

function onEnter() {
  timer = setTimeout(() => {
    show.value = true
    nextTick(() => updatePosition())
  }, props.delay ?? 0)
}

function onLeave() {
  if (timer) { clearTimeout(timer); timer = null }
  show.value = false
}

function updatePosition() {
  if (!wrapRef.value || !popupRef.value) return
  const rect = wrapRef.value.getBoundingClientRect()
  const popup = popupRef.value
  let x = rect.left
  let y = rect.top - popup.offsetHeight - 6
  if (y < 0) y = rect.bottom + 6
  if (x + popup.offsetWidth > window.innerWidth) x = window.innerWidth - popup.offsetWidth - 4
  if (x < 0) x = 4
  popupStyle.value = { left: x + 'px', top: y + 'px' }
}
</script>

<style scoped>
.tooltip-wrap {
  display: inline;
  position: relative;
}
</style>

<style>
.tooltip-popup {
  position: fixed;
  z-index: 9999;
  background: #1e1e1e;
  color: #fff;
  padding: 5px 10px;
  border-radius: 6px;
  font-size: 12px;
  max-width: 400px;
  word-break: break-all;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
  pointer-events: none;
}
</style>
