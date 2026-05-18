<template>
  <div class="screenshot-overlay" @contextmenu.prevent="onCancel">
    <div ref="reactRoot" class="screenshot-container"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { createRoot, Root } from 'react-dom/client'
import React from 'react'
import Screenshots from '../screenshots/Screenshots'
import type { Bounds } from '../screenshots/Screenshots/types'
import { invoke } from '@tauri-apps/api/core'

const reactRoot = ref<HTMLElement | null>(null)
const screenWidth = ref(window.screen.width)
const screenHeight = ref(window.screen.height)
let root: Root | null = null
let captureData: { dataUrl: string; displayInfo?: string } | null = null

const onOk = async (blob: Blob | null, bounds: Bounds) => {
  if (!blob) return
  const arrayBuffer = await blob.arrayBuffer()
  const uint8Array = new Uint8Array(arrayBuffer)
  const base64 = btoa(Array.from(uint8Array).map((b) => String.fromCharCode(b)).join(''))
  invoke('ok_screenshot_overlay', { dataUrl: `data:image/png;base64,${base64}`, boundsJson: JSON.stringify(bounds) })
}

const onCancel = () => {
  invoke('cancel_screenshot_overlay')
}

const onSave = async (blob: Blob | null, bounds: Bounds) => {
  if (!blob) return
  const arrayBuffer = await blob.arrayBuffer()
  const uint8Array = new Uint8Array(arrayBuffer)
  invoke('save_screenshot_overlay', { data: Array.from(uint8Array), boundsJson: JSON.stringify(bounds) })
}

const mountReact = () => {
  if (!reactRoot.value || !captureData?.dataUrl) return
  if (!root) {
    root = createRoot(reactRoot.value)
  }
  root.render(
    React.createElement(Screenshots, {
      url: captureData.dataUrl,
      width: screenWidth.value,
      height: screenHeight.value,
      onOk,
      onCancel,
      onSave,
    })
  )
}

onMounted(() => {
  captureData = (window as any).__SCREENSHOT_DATA__ as { dataUrl: string; displayInfo?: string } | null
  console.log('[screenshot] Overlay mounted, captureData:', captureData ? `dataUrl len=${captureData.dataUrl?.length}` : 'null')
  if (captureData?.dataUrl) {
    nextTick().then(() => {
      console.log('[screenshot] mounting React, screen:', screenWidth.value, 'x', screenHeight.value)
      mountReact()
    })
  } else {
    console.error('[screenshot] No capture data injected')
    invoke('cancel_screenshot_overlay')
  }
})

onUnmounted(() => {
  if (root) {
    root.unmount()
    root = null
  }
})
</script>

<style scoped>
.screenshot-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 99999;
  background: transparent;
}
.screenshot-container {
  width: 100%;
  height: 100%;
}
</style>