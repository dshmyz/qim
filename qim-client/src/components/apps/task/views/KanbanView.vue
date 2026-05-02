<template>
  <div class="kanban-view">
    <div v-if="!todoTasks.length && !inProgressTasks.length && !completedTasks.length" class="kanban-empty">
      <i class="fas fa-tasks"></i>
      <p>暂无任务</p>
      <span>点击「新建」按钮或按 N 键创建任务</span>
    </div>
    <template v-else>
      <KanbanColumn
      title="待办"
      status="todo"
      :tasks="todoTasks"
      dot-color="#fbbf24"
      @task-click="(task) => emit('taskClick', task)"
      @task-contextmenu="(event, task) => emit('taskContextmenu', event, task)"
      @task-dragstart="(event, task) => emit('taskDragstart', event, task)"
      @task-dragend="() => emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="进行中"
      status="in_progress"
      :tasks="inProgressTasks"
      dot-color="#a78bfa"
      @task-click="(task) => emit('taskClick', task)"
      @task-contextmenu="(event, task) => emit('taskContextmenu', event, task)"
      @task-dragstart="(event, task) => emit('taskDragstart', event, task)"
      @task-dragend="() => emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="已完成"
      status="completed"
      :tasks="completedTasks"
      dot-color="#34d399"
      @task-click="(task) => emit('taskClick', task)"
      @task-contextmenu="(event, task) => emit('taskContextmenu', event, task)"
      @task-dragstart="(event, task) => emit('taskDragstart', event, task)"
      @task-dragend="() => emit('taskDragend')"
      @drop="onColumnDrop"
    />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task, TaskStatus } from '../../../../types/task'
import { useTaskStore } from '../../../../stores/task'
import KanbanColumn from './KanbanColumn.vue'

const store = useTaskStore()

const todoTasks = computed(() => store.todoTasks)
const inProgressTasks = computed(() => store.inProgressTasks)
const completedTasks = computed(() => store.completedTasks)

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
  overflow-y: hidden;
  min-height: 0;
  padding-bottom: var(--spacing-4);
}
.kanban-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-2);
  color: var(--text-secondary);
}
.kanban-empty i { font-size: 32px; opacity: 0.4; }
.kanban-empty p { font-size: 14px; margin: 0; color: var(--text-primary); }
.kanban-empty span { font-size: 12px; }
</style>
