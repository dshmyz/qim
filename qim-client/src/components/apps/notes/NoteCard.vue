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
  padding: var(--spacing-4);
  margin-bottom: var(--spacing-2);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-base);
  box-shadow: var(--shadow-xs);
}

.note-card:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.note-card.active {
  background: var(--primary-light);
  border-color: var(--primary-color);
  box-shadow: var(--shadow-sm);
}

.note-card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--spacing-2);
}

.note-title {
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  margin: 0;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-actions {
  display: flex;
  gap: var(--spacing-1);
  opacity: 0;
  transition: opacity var(--transition-fast);
}

.note-card:hover .note-actions {
  opacity: 1;
}

.note-action-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--btn-bg);
  color: var(--text-secondary);
  cursor: pointer;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
}

.note-action-btn:hover {
  background: var(--primary-light);
  color: var(--primary-color);
}

.note-action-btn.delete:hover {
  background: var(--color-error-100);
  color: var(--danger-color);
}

.note-summary {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  margin: 0 0 var(--spacing-2) 0;
  line-height: var(--line-height-normal);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.note-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-1);
  margin-bottom: var(--spacing-2);
}

.note-tag {
  font-size: var(--font-size-xs);
  padding: 2px 10px;
  background: var(--primary-light);
  color: var(--primary-color);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-weight: var(--font-weight-medium);
}

.note-tag:hover {
  background: var(--primary-color);
  color: white;
  transform: scale(1.05);
}

.note-footer {
  display: flex;
  justify-content: flex-end;
}

.note-date {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}
</style>
