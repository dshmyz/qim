<template>
  <div class="kanban-view">
    <KanbanColumn
      title="待办"
      status="todo"
      :tasks="todoTasks"
      dot-color="#fbbf24"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="进行中"
      status="in_progress"
      :tasks="inProgressTasks"
      dot-color="#a78bfa"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="已完成"
      status="completed"
      :tasks="completedTasks"
      dot-color="#34d399"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
  </div>
</template>

<script setup lang="ts">
import type { Task, TaskStatus } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import KanbanColumn from './KanbanColumn.vue'

const store = useTaskStore()

const todoTasks = store.todoTasks
const inProgressTasks = store.inProgressTasks
const completedTasks = store.completedTasks

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
  taskDragstart: [event: DragEvent, task: Task]
  taskDragend: []
}>()

async function onColumnDrop(taskId: string, status: TaskStatus) {
  await store.changeStatus(taskId, status)
}
</script>

<style scoped>
.kanban-view {
  flex: 1;
  display: flex;
  gap: var(--spacing-4);
  overflow-x: auto;
  padding-bottom: var(--spacing-4);
}
</style>
