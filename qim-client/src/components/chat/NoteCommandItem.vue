<template>
  <!-- 笔记候选项：图标 + 标题 + 摘要/更新时间 -->
  <span class="note-cmd-item">
    <span class="note-cmd-item__icon">
      <i class="fas fa-file-alt"></i>
    </span>
    <span class="note-cmd-item__main">
      <span class="note-cmd-item__title">{{ note.title || '无标题笔记' }}</span>
      <span v-if="note.summary" class="note-cmd-item__summary">{{ note.summary }}</span>
    </span>
    <span v-if="note.updated_at" class="note-cmd-item__time">
      <i class="far fa-clock"></i>
      {{ timeLabel }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Note } from '../../types/note'

const props = defineProps<{
  item: Note
  active: boolean
}>()

const note = computed(() => props.item)

const timeLabel = computed(() => {
  const d = new Date(note.value.updated_at)
  if (isNaN(d.getTime())) return ''
  const now = new Date()
  const sameYear = d.getFullYear() === now.getFullYear()
  const opts: Intl.DateTimeFormatOptions = sameYear
    ? { month: 'numeric', day: 'numeric' }
    : { year: 'numeric', month: 'numeric', day: 'numeric' }
  return d.toLocaleDateString('zh-CN', opts)
})
</script>

<style scoped>
.note-cmd-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.note-cmd-item__icon {
  font-size: var(--font-size-xs);
  color: var(--primary-color, #3385ff);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  width: 16px;
}
.note-cmd-item__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.note-cmd-item__title {
  font-size: var(--font-size-xs);
  color: var(--text-color, #303133);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.note-cmd-item__summary {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #909399);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.note-cmd-item__time {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
</style>
