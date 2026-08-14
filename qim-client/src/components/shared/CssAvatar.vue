<template>
  <div class="css-avatar" :style="avatarStyle" :class="[sizeClass, shapeClass]">
    {{ initial }}
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getAvatarColor, getInitial } from '../../utils/avatar'

interface Props {
  name: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
  shape?: 'circle' | 'rounded'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  shape: 'circle'
})

const initial = computed(() => getInitial(props.name || ''))
const color = computed(() => getAvatarColor(props.name || ''))

const avatarStyle = computed(() => ({
  backgroundColor: color.value
}))

const sizeClass = computed(() => `avatar-${props.size}`)
const shapeClass = computed(() => `avatar-${props.shape}`)
</script>

<style scoped>
.css-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
  overflow: hidden;
  text-transform: uppercase;
  user-select: none;
}

.avatar-sm {
  width: 32px;
  height: 32px;
  font-size: var(--font-size-sm);
}

.avatar-md {
  width: 40px;
  height: 40px;
  font-size: var(--font-size-lg);
}

.avatar-lg {
  width: 48px;
  height: 48px;
  font-size: var(--font-size-xl);
}

.avatar-xl {
  width: 64px;
  height: 64px;
  font-size: 28px;
}

.avatar-circle {
  border-radius: 50%;
}

.avatar-rounded {
  border-radius: 10px;
}
</style>
