<template>
  <!-- 任务候选项：状态图标 + 标题 + 到期时间 -->
  <span class="task-cmd-item">
    <span class="task-cmd-item__icon" :class="`status-${task.status}`">
      <i :class="statusIcon"></i>
    </span>
    <span class="task-cmd-item__title">{{ task.title }}</span>
    <span
      v-if="task.due_date"
      class="task-cmd-item__due"
      :class="{ overdue: isOverdue }"
    >
      <i class="far fa-calendar"></i>
      {{ task.due_date }}
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../../types/task'

const props = defineProps<{
  item: Task
  active: boolean
}>()

const task = computed(() => props.item)

const statusIcon = computed(() => {
  switch (task.value.status) {
    case 'completed': return 'fas fa-check-circle'
    case 'in_progress': return 'fas fa-circle-half-stroke'
    default: return 'far fa-circle'
  }
})

const isOverdue = computed(() => {
  if (!task.value.due_date) return false
  return new Date(task.value.due_date).getTime() < Date.now() && task.value.status !== 'completed'
})
</script>

<style scoped>
.task-cmd-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.task-cmd-item__icon {
  font-size: 13px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  width: 16px;
}
.task-cmd-item__icon.status-completed { color: var(--color-success, #67c23a); }
.task-cmd-item__icon.status-in_progress { color: var(--color-warning, #e6a23c); }
.task-cmd-item__icon.status-todo { color: var(--color-info, #909399); }

.task-cmd-item__title {
  flex: 1;
  font-size: 13px;
  color: var(--text-color, #303133);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-cmd-item__due {
  font-size: 11px;
  color: var(--text-secondary, #909399);
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.task-cmd-item__due.overdue {
  color: var(--color-danger, #f56c6c);
}
</style>
