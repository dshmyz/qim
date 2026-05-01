<template>
  <div
    class="kanban-column"
    :class="{ 'drag-over': isDragOver }"
    @dragover.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <div class="kanban-column-header">
      <div class="column-title">
        <span class="column-dot" :style="{ background: dotColor }"></span>
        <span>{{ title }}</span>
        <span class="column-count">{{ tasks.length }}</span>
      </div>
    </div>
    <div class="kanban-column-list">
      <TaskCard
        v-for="task in tasks"
        :key="task.id"
        :task="task"
        @click="(task) => emit('taskClick', task)"
        @contextmenu="(event, task) => emit('taskContextmenu', event, task)"
        @dragstart="(event, task) => emit('taskDragstart', event, task)"
        @dragend="() => emit('taskDragend')"
      />
      <div v-if="!tasks.length" class="kanban-column-empty">
        拖拽任务到此处
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Task, TaskStatus } from '../../../../types/task'
import TaskCard from '../components/TaskCard.vue'

const props = defineProps<{
  title: string
  status: TaskStatus
  tasks: Task[]
  dotColor: string
}>()

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
  taskDragstart: [event: DragEvent, task: Task]
  taskDragend: []
  drop: [taskId: string, status: TaskStatus]
}>()

const isDragOver = ref(false)

function onDragOver(event: DragEvent) {
  isDragOver.value = true
  event.dataTransfer!.dropEffect = 'move'
}

function onDragLeave() {
  isDragOver.value = false
}

function onDrop(event: DragEvent) {
  isDragOver.value = false
  const taskId = event.dataTransfer!.getData('text/plain')
  if (taskId) {
    emit('drop', taskId, props.status)
  }
}
</script>

<style scoped>
.kanban-column {
  flex: 1;
  min-width: 240px;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  overflow: hidden;
}
.kanban-column.drag-over {
  background: var(--hover-bg);
  border-color: #a78bfa;
}
.kanban-column-header {
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
}
.column-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.column-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}
.column-count {
  font-size: 10px;
  background: var(--border-color);
  color: var(--text-secondary);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}
.kanban-column-list {
  flex: 1;
  padding: var(--spacing-3);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
  min-height: 60px;
}
.kanban-column-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-4);
  font-size: 12px;
  color: var(--text-secondary);
  border: 1px dashed var(--border-color);
  border-radius: var(--radius-md);
}
</style>
