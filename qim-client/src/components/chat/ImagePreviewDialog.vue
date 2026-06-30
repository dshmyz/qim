<template>
  <div v-if="visible" class="image-viewer-source">
    <img :src="imageUrl" alt="预览图片" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, watch } from 'vue'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'

interface Props {
  visible: boolean
  imageUrl: string
}

const props = defineProps<Props>()
const emit = defineEmits<{ (e: 'close'): void }>()

let viewer: Viewer | null = null

const destroyViewer = () => {
  viewer?.destroy()
  viewer = null
}

const openViewer = async () => {
  if (!props.visible || !props.imageUrl) return
  await nextTick()
  const source = document.querySelector('.image-viewer-source') as HTMLElement | null
  if (!source) return
  destroyViewer()
  viewer = new Viewer(source, {
    inline: false,
    navbar: false,
    title: false,
    toolbar: {
      zoomIn: 1,
      zoomOut: 1,
      oneToOne: 1,
      reset: 1,
      rotateLeft: 1,
      rotateRight: 1,
    },
    hidden() {
      emit('close')
    },
  })
  viewer.show()
}

watch(() => [props.visible, props.imageUrl], () => {
  if (props.visible) {
    openViewer()
  } else {
    destroyViewer()
  }
}, { immediate: true })

onBeforeUnmount(() => {
  destroyViewer()
})
</script>

<style scoped>
.image-viewer-source {
  display: none;
}

.image-viewer-source img {
  max-width: 100%;
}
</style>

<style>
.viewer-container {
  background-color: rgba(15, 23, 42, 0.82);
  backdrop-filter: blur(14px);
}

.viewer-button {
  background-color: rgba(15, 23, 42, 0.72);
  border: 1px solid rgba(255, 255, 255, 0.18);
}

.viewer-toolbar > ul > li,
.viewer-navbar,
.viewer-title {
  background-color: rgba(15, 23, 42, 0.72);
}

.viewer-canvas img {
  border-radius: 14px;
}
</style>
