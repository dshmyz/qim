<template>
  <div
    class="task-row"
    :class="{ 'is-completed': task.status === 'completed', 'is-selected': isSelected }"
    @click="$emit('click', task)"
    @contextmenu.prevent="$emit('contextmenu', $event, task)"
  >
    <button class="row-checkbox" @click.stop="$emit('toggleComplete', task)">
      <i v-if="task.status === 'completed'" class="fas fa-check-circle" style="color: #34d399;"></i>
      <i v-else class="far fa-circle" style="color: var(--border-color);"></i>
    </button>
    <span class="row-title">{{ task.title }}</span>
    <span v-if="task.tags.length" class="row-tags">
      <span
        v-for="tag in task.tags.slice(0, 2)"
        :key="tag.id"
        class="task-tag"
        :style="{ background: tag.color + '20', color: tag.color }"
      >{{ tag.name }}</span>
    </span>
    <span class="row-priority" :class="'priority-' + task.priority">
      {{ priorityLabel }}
    </span>
    <span v-if="task.due_date" class="row-due">{{ dueDateLabel }}</span>
    <div v-if="task.assignee" class="row-assignee">
      <div class="assignee-avatar" :style="{ background: avatarColor }">
        {{ task.assignee.name.charAt(0) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../../../../types/task'

const props = defineProps<{
  task: Task
  isSelected?: boolean
}>()

defineEmits<{
  click: [task: Task]
  contextmenu: [event: MouseEvent, task: Task]
  toggleComplete: [task: Task]
}>()

const priorityLabel = computed(() => {
  const map: Record<string, string> = { high: '高', medium: '中', low: '低' }
  return map[props.task.priority] || props.task.priority
})

const dueDateLabel = computed(() => {
  if (!props.task.due_date) return ''
  const date = new Date(props.task.due_date)
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const diff = Math.ceil((date.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))
  if (diff < 0) return '已过期'
  if (diff === 0) return '今天'
  if (diff === 1) return '明天'
  return `${date.getMonth() + 1}/${date.getDate()}`
})

const avatarColor = computed(() => {
  const colors = ['#8b5cf6', '#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']
  const index = props.task.assignee ? props.task.assignee.name.charCodeAt(0) % colors.length : 0
  return colors[index]
})
</script>

<style scoped>
.task-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background var(--animation-base) ease;
}
.task-row:hover { background: var(--hover-bg); }
.task-row.is-selected { background: var(--active-bg); }
.task-row.is-completed .row-title {
  text-decoration: line-through;
  color: var(--text-secondary);
}
.row-checkbox {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-size: 14px;
  display: flex;
  align-items: center;
}
.row-title {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.row-tags { display: flex; gap: 4px; }
.task-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}
.row-priority {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}
.row-priority.priority-high { background: #fef2f2; color: #ef4444; }
.row-priority.priority-medium { background: #fffbeb; color: #d97706; }
.row-priority.priority-low { background: #eff6ff; color: #3b82f6; }
.row-due {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}
.row-assignee { flex-shrink: 0; }
.assignee-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: #fff;
  font-weight: 500;
}
</style>
