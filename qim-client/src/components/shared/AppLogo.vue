<template>
  <img
    :src="src"
    :alt="alt"
    :style="imgStyle"
    class="app-logo"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  size?: number | 'small' | 'medium' | 'large' | 'extraLarge'
  src?: string
  alt?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'medium',
  src: './app-logo.png',
  alt: 'QIM Logo'
})

const sizeMap = {
  small: 32,
  medium: 48,
  large: 80,
  extraLarge: 120
}

const actualSize = computed(() => {
  if (typeof props.size === 'number') {
    return props.size
  }
  return sizeMap[props.size]
})

const imgStyle = computed(() => ({
  width: `${actualSize.value}px`,
  height: `${actualSize.value}px`
}))
</script>

<style scoped>
.app-logo {
  object-fit: contain;
}
</style>
