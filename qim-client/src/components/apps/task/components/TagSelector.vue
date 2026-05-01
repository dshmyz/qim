<template>
  <div class="tag-selector">
    <label class="tag-label">标签</label>
    <div class="tag-list">
      <span
        v-for="tag in availableTags"
        :key="tag.id"
        class="tag-item"
        :class="{ selected: isSelected(tag.id) }"
        :style="isSelected(tag.id) ? { background: tag.color + '20', color: tag.color, borderColor: tag.color } : {}"
        @click="toggleTag(tag)"
      >
        {{ tag.name }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Tag } from '../../../../types/task'

const props = defineProps<{
  availableTags: Tag[]
  selectedTagIds: string[]
}>()

const emit = defineEmits<{
  update: [tagIds: string[]]
}>()

function isSelected(id: string) {
  return props.selectedTagIds.includes(id)
}

function toggleTag(tag: Tag) {
  const ids = [...props.selectedTagIds]
  const index = ids.indexOf(tag.id)
  if (index === -1) {
    ids.push(tag.id)
  } else {
    ids.splice(index, 1)
  }
  emit('update', ids)
}
</script>

<style scoped>
.tag-selector { margin-bottom: var(--spacing-3); }
.tag-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}
.tag-list { display: flex; flex-wrap: wrap; gap: var(--spacing-1); }
.tag-item {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--animation-fast) ease;
}
.tag-item:hover { border-color: var(--text-secondary); }
.tag-item.selected { font-weight: 500; }
</style>
