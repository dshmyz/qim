<template>
  <Transition name="slide">
    <div v-if="task" class="detail-panel">
      <div class="detail-header">
        <h3 class="detail-title">{{ task.title }}</h3>
        <button class="detail-close" @click="$emit('close')">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="detail-body">
        <div class="detail-field">
          <label>描述</label>
          <p class="detail-description">{{ task.description || '暂无描述' }}</p>
        </div>
        <div class="detail-field-row">
          <div class="detail-field">
            <label>状态</label>
            <select class="detail-select" :value="task.status" @change="onStatusChange">
              <option value="todo">待办</option>
              <option value="in_progress">进行中</option>
              <option value="completed">已完成</option>
            </select>
          </div>
          <div class="detail-field">
            <label>优先级</label>
            <select class="detail-select" :value="task.priority" @change="onPriorityChange">
              <option value="low">低</option>
              <option value="medium">中</option>
              <option value="high">高</option>
            </select>
          </div>
        </div>
        <div class="detail-field">
          <label>截止日期</label>
          <input type="date" class="detail-input" :value="task.due_date" @change="onDueDateChange">
        </div>
        <TagSelector
          :available-tags="availableTags"
          :selected-tag-ids="task.tags.map(t => t.id)"
          @update="onTagsUpdate"
        />
        <AssigneeSelector
          :model-value="task.assignee?.id || null"
          :contacts="contacts"
          @update:model-value="onAssigneeUpdate"
        />
        <SubTaskList
          :sub-tasks="task.sub_tasks"
          @toggle="onSubTaskToggle"
          @remove="onSubTaskRemove"
          @add="onSubTaskAdd"
        />
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import type { Task, Tag, TaskUser, TaskStatus, TaskPriority } from '../../../../types/task'
import { useTaskStore } from '../../../../stores/task'
import TagSelector from './TagSelector.vue'
import AssigneeSelector from './AssigneeSelector.vue'
import SubTaskList from './SubTaskList.vue'

const props = defineProps<{
  task: Task | null
  availableTags: Tag[]
  contacts: TaskUser[]
}>()

const store = useTaskStore()

defineEmits<{
  close: []
}>()

async function onStatusChange(e: Event) {
  if (!props.task) return
  await store.changeStatus(props.task.id, (e.target as HTMLSelectElement).value as TaskStatus)
}

async function onPriorityChange(e: Event) {
  if (!props.task) return
  await store.updateTask(props.task.id, { priority: (e.target as HTMLSelectElement).value as TaskPriority })
}

async function onDueDateChange(e: Event) {
  if (!props.task) return
  await store.updateTask(props.task.id, { due_date: (e.target as HTMLInputElement).value })
}

async function onTagsUpdate(tagIds: string[]) {
  if (!props.task) return
  await store.updateTask(props.task.id, { tags: tagIds })
}

async function onAssigneeUpdate(assigneeId: string | null) {
  if (!props.task) return
  await store.updateTask(props.task.id, { assignee_id: assigneeId })
}

async function onSubTaskToggle(subTaskId: string) {
  if (!props.task) return
  const subTask = props.task.sub_tasks.find(st => st.id === subTaskId)
  if (!subTask) return
  const updated = props.task.sub_tasks.map(st =>
    st.id === subTaskId ? { ...st, completed: !st.completed } : st
  )
  await store.updateTask(props.task.id, { sub_tasks: updated })
}

async function onSubTaskRemove(subTaskId: string) {
  if (!props.task) return
  const updated = props.task.sub_tasks.filter(st => st.id !== subTaskId)
  await store.updateTask(props.task.id, { sub_tasks: updated })
}

async function onSubTaskAdd(title: string) {
  if (!props.task) return
  const newSubTask = {
    id: `st-${Date.now()}`,
    title,
    completed: false,
    position: props.task.sub_tasks.length
  }
  await store.updateTask(props.task.id, { sub_tasks: [...props.task.sub_tasks, newSubTask] })
}
</script>

<style scoped>
.detail-panel {
  width: 320px;
  background: var(--card-bg);
  border-left: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  flex-shrink: 0;
}
.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
}
.detail-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  flex: 1;
  word-break: break-word;
}
.detail-close {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: var(--spacing-1);
  border-radius: var(--radius-sm);
}
.detail-close:hover { background: var(--hover-bg); }
.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-4);
}
.detail-field { margin-bottom: var(--spacing-3); }
.detail-field label {
  display: block;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: var(--spacing-1);
}
.detail-description {
  font-size: 13px;
  color: var(--text-primary);
  line-height: 1.5;
  margin: 0;
}
.detail-field-row { display: flex; gap: var(--spacing-3); }
.detail-field-row .detail-field { flex: 1; }
.detail-select,
.detail-input {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-primary);
  background: var(--input-bg);
}
.detail-select:focus,
.detail-input:focus { outline: none; border-color: #8b5cf6; }
.slide-enter-active,
.slide-leave-active { transition: transform var(--animation-base) ease; }
.slide-enter-from,
.slide-leave-to { transform: translateX(100%); }
</style>
