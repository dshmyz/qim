<template>
  <div
    class="note-card"
    :class="{ active: isActive }"
    @click="$emit('select')"
    @mouseenter="showActions = true"
    @mouseleave="showActions = false"
  >
    <div class="note-card-header">
      <h3 class="note-title">{{ note.title }}</h3>
      <div class="note-actions" v-show="showActions">
        <button class="note-action-btn" @click.stop="$emit('edit')" title="编辑">
          <i class="fas fa-edit"></i>
        </button>
        <button class="note-action-btn delete" @click.stop="$emit('delete')" title="删除">
          <i class="fas fa-trash"></i>
        </button>
      </div>
    </div>
    <p class="note-summary">{{ displaySummary }}</p>
    <div class="note-tags" v-if="note.tags && note.tags.length > 0">
      <span
        v-for="tag in note.tags"
        :key="tag"
        class="note-tag"
        @click.stop="$emit('filter-tag', tag)"
      >
        {{ tag }}
      </span>
    </div>
    <div class="note-footer">
      <span class="note-date">{{ formatDate(note.updated_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Note } from '../../../types/note'

const props = defineProps<{
  note: Note
  isActive?: boolean
}>()

defineEmits<{
  select: []
  edit: []
  delete: []
  'filter-tag': [tag: string]
}>()

const showActions = ref(false)

const displaySummary = computed(() => {
  if (props.note.summary) {
    return props.note.summary
  }
  const content = props.note.content || ''
  return content.length > 50 ? content.substring(0, 50) + '...' : content
})

const formatDate = (dateStr: string) => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.note-card {
  padding: 16px;
  margin-bottom: 8px;
  background: var(--bg-color);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.note-card:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.note-card.active {
  background: var(--hover-color);
  border-color: var(--primary-color);
}

.note-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 8px;
}

.note-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-actions {
  display: flex;
  gap: 4px;
}

.note-action-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.note-action-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.note-action-btn.delete:hover {
  color: var(--danger-color);
}

.note-summary {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0 0 8px 0;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.note-tag {
  font-size: 12px;
  padding: 2px 8px;
  background: var(--primary-light);
  color: var(--primary-color);
  border-radius: 10px;
  cursor: pointer;
}

.note-tag:hover {
  background: var(--primary-color);
  color: white;
}

.note-footer {
  display: flex;
  justify-content: flex-end;
}

.note-date {
  font-size: 12px;
  color: var(--text-tertiary);
}
</style>
