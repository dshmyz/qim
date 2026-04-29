<template>
  <div class="css-avatar" :style="avatarStyle" :class="sizeClass">
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

/* Size variants */
.avatar-sm {
  width: 32px;
  height: 32px;
  font-size: 14px;
}

.avatar-md {
  width: 40px;
  height: 40px;
  font-size: 18px;
}

.avatar-lg {
  width: 48px;
  height: 48px;
  font-size: 20px;
}

.avatar-xl {
  width: 64px;
  height: 64px;
  font-size: 28px;
}

/* Shape variants */
.avatar-circle {
  border-radius: 50%;
}

.avatar-rounded {
  border-radius: 10px;
}
</style>
