<template>
  <div v-if="members.length > 0" class="selected-bar">
    <div v-for="m in members" :key="m.id" class="chip">
      <Avatar :src="m.avatar" :name="m.name" :alt="m.name" size="sm" />
      <span class="chip-name">{{ m.name }}</span>
      <span class="chip-remove" @click="$emit('remove', m)">×</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import Avatar from './Avatar.vue'

defineProps<{
  members: { id: string | number; name: string; avatar?: string }[]
}>()

defineEmits<{
  remove: [member: any]
}>()
</script>

<style scoped>
.selected-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px;
  max-height: 84px;
  overflow-y: auto;
  border-bottom: 1px solid var(--border-color, #eee);
}
.chip {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px 3px 6px;
  background: var(--hover-color, rgba(99, 102, 241, 0.08));
  border-radius: 14px;
  font-size: var(--font-size-xxs);
}
.chip-name {
  max-width: 80px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-color, #333);
}
.chip-remove {
  cursor: pointer;
  color: var(--text-secondary, #999);
  font-size: var(--font-size-base);
  line-height: 1;
  padding: 0 3px;
  transition: color 0.15s;
}
.chip-remove:hover {
  color: var(--text-color, #333);
}
</style>
