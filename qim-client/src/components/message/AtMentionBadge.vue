<template>
  <span v-if="isAtMention" class="at-mention-badge" :title="title">
    <slot>@</slot>
  </span>
</template>

<script setup lang="ts">
defineProps<{
  isAtMention: boolean
  title?: string
}>()
</script>

<style scoped>
.at-mention-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 4px;
  background: #ef4444;
  color: white;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
}
</style>
