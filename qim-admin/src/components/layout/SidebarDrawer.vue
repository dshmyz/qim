<template>
  <Transition name="slide">
    <div v-if="visible" class="sidebar-drawer">
      <Sidebar :collapsed="false" @toggle="$emit('close')" />
    </div>
  </Transition>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar/index.vue'

interface Props {
  visible: boolean
}

const props = defineProps<Props>()

interface Emits {
  (e: 'close'): void
}

const emit = defineEmits<Emits>()
</script>

<style scoped>
.sidebar-drawer {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: 260px;
  z-index: var(--z-drawer, 1000);
  box-shadow: 4px 0 24px rgba(0, 0, 0, 0.15);
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.slide-enter-from {
  transform: translateX(-100%);
}

.slide-leave-to {
  transform: translateX(-100%);
}
</style>
