<template>
  <el-button :type="type" size="small" @click="$emit('click', $event)">
    <slot></slot>
  </el-button>
</template>

<script setup lang="ts">
interface Props {
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'default'
}

withDefaults(defineProps<Props>(), {
  type: 'default'
})

defineEmits<{
  'click': [event: MouseEvent]
}>()
</script>
