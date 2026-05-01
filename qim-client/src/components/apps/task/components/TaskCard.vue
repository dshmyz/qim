<template>
  <div
    class="task-card"
    :class="{
      'priority-high': task.priority === 'high',
      'priority-medium': task.priority === 'medium',
      'priority-low': task.priority === 'low',
      'is-completed': task.status === 'completed'
    }"
    draggable="true"
    @dragstart="onDragStart"
    @dragend="onDragEnd"
    @click="$emit('click', task)"
    @contextmenu.prevent="$emit('contextmenu', $event, task)"
  >
    <div class="task-card-title">{{ task.title }}</div>
    <div v-if="task.tags.length" class="task-card-tags">
      <span
        v-for="tag in task.tags"
        :key="tag.id"
        class="task-tag"
        :style="{ background: tag.color + '20', color: tag.color }"
      >{{ tag.name }}</span>
    </div>
    <div v-if="task.status === 'in_progress' && task.sub_tasks.length" class="task-card-progress">
      <div class="progress-bar">
        <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
      </div>
      <span class="progress-text">{{ progressPercent }}%</span>
    </div>
    <div class="task-card-footer">
      <div class="task-card-left">
        <div v-if="task.assignee" class="task-assignee">
          <div class="assignee-avatar" :style="{ background: avatarColor }">
            {{ task.assignee.name.charAt(0) }}
          </div>
          <span v-if="task.due_date" class="task-due">{{ dueDateLabel }}</span>
        </div>
        <span v-else-if="task.due_date" class="task-due">{{ dueDateLabel }}</span>
      </div>
      <div class="task-card-right">
        <span v-if="task.sub_tasks.length" class="task-meta-item">📋 {{ completedSubTasks }}/{{ task.sub_tasks.length }}</span>
        <span v-if="task.comment_count" class="task-meta-item">💬 {{ task.comment_count }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../../../../types/task'

const props = defineProps<{
  task: Task
}>()

defineEmits<{
  click: [task: Task]
  contextmenu: [event: MouseEvent, task: Task]
  dragstart: [event: DragEvent, task: Task]
  dragend: [event: DragEvent]
}>()

const completedSubTasks = computed(() =>
  props.task.sub_tasks.filter(st => st.completed).length
)

const progressPercent = computed(() => {
  if (!props.task.sub_tasks.length) return 0
  return Math.round((completedSubTasks.value / props.task.sub_tasks.length) * 100)
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
  if (diff <= 7) return `${diff}天后`
  return `${date.getMonth() + 1}月${date.getDate()}日`
})

const avatarColor = computed(() => {
  const colors = ['#8b5cf6', '#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']
  const index = props.task.assignee ? props.task.assignee.name.charCodeAt(0) % colors.length : 0
  return colors[index]
})

function onDragStart(event: DragEvent) {
  event.dataTransfer!.setData('text/plain', props.task.id)
  event.dataTransfer!.effectAllowed = 'move'
  ;(event.target as HTMLElement).classList.add('dragging')
}

function onDragEnd(event: DragEvent) {
  ;(event.target as HTMLElement).classList.remove('dragging')
}
</script>

<style scoped>
.task-card {
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: var(--spacing-3);
  cursor: pointer;
  transition: all var(--animation-base) ease;
  border-left: 3px solid transparent;
}
.task-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}
.task-card.priority-high { border-left-color: #ef4444; }
.task-card.priority-medium { border-left-color: #fbbf24; }
.task-card.priority-low { border-left-color: #60a5fa; }
.task-card.is-completed { opacity: 0.6; }
.task-card.is-completed .task-card-title {
  text-decoration: line-through;
  color: var(--text-secondary);
}
.task-card.dragging {
  opacity: 0.5;
  transform: rotate(2deg);
}
.task-card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
  line-height: 1.4;
  word-break: break-word;
}
.task-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.task-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}
.task-card-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}
.progress-bar {
  flex: 1;
  height: 3px;
  background: var(--border-color);
  border-radius: 2px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: #a78bfa;
  border-radius: 2px;
  transition: width var(--animation-base) ease;
}
.progress-text {
  font-size: 10px;
  color: #7c3aed;
  flex-shrink: 0;
}
.task-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}
.task-card-left {
  display: flex;
  align-items: center;
  gap: 6px;
}
.task-assignee {
  display: flex;
  align-items: center;
  gap: 4px;
}
.assignee-avatar {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: #fff;
  font-weight: 500;
}
.task-due {
  font-size: 10px;
  color: var(--text-secondary);
}
.task-card-right {
  display: flex;
  align-items: center;
  gap: 6px;
}
.task-meta-item {
  font-size: 10px;
  color: var(--text-secondary);
}
</style>
