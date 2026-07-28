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
  drop: [taskId: string, status: TaskStatus, index: number]
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
  if (!taskId) return
  // 计算 drop 插入位置（基于鼠标 Y 相对各卡片中点）
  const list = (event.currentTarget as HTMLElement).querySelector('.kanban-column-list')
  const cards = list
    ? (Array.from(list.children) as HTMLElement[]).filter(c => !c.classList.contains('kanban-column-empty'))
    : []
  let index = cards.length
  for (let i = 0; i < cards.length; i++) {
    const rect = cards[i].getBoundingClientRect()
    if (event.clientY < rect.top + rect.height / 2) {
      index = i
      break
    }
  }
  emit('drop', taskId, props.status, index)
}
</script>

<style scoped>
.kanban-column {
  flex: 1 1 0%;
  min-width: 220px;
  max-width: 340px;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  overflow: hidden;
  min-height: 0;
}
.kanban-column.drag-over {
  background: var(--hover-bg);
  border-color: #a78bfa;
}
.kanban-column-header {
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
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
  min-height: 0;
  padding: var(--spacing-3);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
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
