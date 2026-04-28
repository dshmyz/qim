<template>
  <Transition name="fade">
    <div v-if="visible" class="mobile-overlay" @click="$emit('close')" />
  </Transition>
</template>

<script setup lang="ts">
interface Props {
  visible: boolean
}

const props = defineProps<Props>()

defineEmits<{
  close: []
}>()
</script>

<style scoped>
.mobile-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--overlay-bg, rgba(15, 23, 42, 0.5));
  z-index: var(--z-overlay, 999);
  backdrop-filter: blur(2px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
