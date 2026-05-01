<template>
  <div class="list-view">
    <div class="list-header">
      <div class="list-header-cell checkbox-cell"></div>
      <div class="list-header-cell title-cell" @click="toggleSort('title')">
        任务名称
        <i v-if="sortBy === 'title'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell priority-cell" @click="toggleSort('priority')">
        优先级
        <i v-if="sortBy === 'priority'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell due-cell" @click="toggleSort('due_date')">
        截止日期
        <i v-if="sortBy === 'due_date'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell assignee-cell">指派人</div>
    </div>
    <div class="list-body">
      <TaskRow
        v-for="task in sortedTasks"
        :key="task.id"
        :task="task"
        :is-selected="task.id === selectedTaskId"
        @click="emit('taskClick', $event)"
        @contextmenu="(...args: any[]) => emit('taskContextmenu', ...args)"
        @toggle-complete="onToggleComplete"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Task } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import TaskRow from '../components/TaskRow.vue'

const store = useTaskStore()
const selectedTaskId = computed(() => store.selectedTaskId)

const sortBy = ref<'title' | 'priority' | 'due_date'>('due_date')
const sortDir = ref<'asc' | 'desc'>('asc')

const priorityOrder: Record<string, number> = { high: 0, medium: 1, low: 2 }

const sortedTasks = computed(() => {
  const tasks = [...store.filteredTasks]
  tasks.sort((a, b) => {
    let cmp = 0
    if (sortBy.value === 'title') {
      cmp = a.title.localeCompare(b.title)
    } else if (sortBy.value === 'priority') {
      cmp = (priorityOrder[a.priority] ?? 1) - (priorityOrder[b.priority] ?? 1)
    } else if (sortBy.value === 'due_date') {
      const aDate = a.due_date ? new Date(a.due_date).getTime() : Infinity
      const bDate = b.due_date ? new Date(b.due_date).getTime() : Infinity
      cmp = aDate - bDate
    }
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return tasks
})

function toggleSort(field: 'title' | 'priority' | 'due_date') {
  if (sortBy.value === field) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = field
    sortDir.value = 'asc'
  }
}

async function onToggleComplete(task: Task) {
  const newStatus = task.status === 'completed' ? 'todo' : 'completed'
  await store.changeStatus(task.id, newStatus)
}

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
}>()
</script>

<style scoped>
.list-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.list-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 2px solid var(--border-color);
  background: var(--card-bg);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.list-header-cell {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  user-select: none;
}
.checkbox-cell { width: 24px; }
.title-cell { flex: 1; }
.priority-cell { width: 60px; }
.due-cell { width: 80px; }
.assignee-cell { width: 32px; }
.list-body {
  flex: 1;
  overflow-y: auto;
  background: var(--card-bg);
}
</style>
