<template>
  <div class="workspace-view">
    <div class="workspace-section">
      <div class="workspace-section-header">
        <span class="section-dot" style="background:#ef4444;"></span>
        <h3>今日待办</h3>
        <span class="section-count">{{ todayTasks.length }}</span>
      </div>
      <div class="workspace-task-list">
        <TaskCard
          v-for="task in todayTasks"
          :key="task.id"
          :task="task"
          @click="emit('taskClick', $event)"
        />
        <div v-if="!todayTasks.length" class="workspace-empty">今天没有待办任务</div>
      </div>
    </div>
    <div class="workspace-section">
      <div class="workspace-section-header">
        <span class="section-dot" style="background:#a78bfa;"></span>
        <h3>进行中</h3>
        <span class="section-count">{{ inProgressTasks.length }}</span>
      </div>
      <div class="workspace-task-list">
        <TaskCard
          v-for="task in inProgressTasks"
          :key="task.id"
          :task="task"
          @click="emit('taskClick', $event)"
        />
      </div>
    </div>
    <div class="workspace-section">
      <div class="workspace-section-header">
        <span class="section-dot" style="background:#f59e0b;"></span>
        <h3>已指派给我</h3>
        <span class="section-count">{{ myTasks.length }}</span>
      </div>
      <div class="workspace-task-list">
        <TaskCard
          v-for="task in myTasks"
          :key="task.id"
          :task="task"
          @click="emit('taskClick', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import TaskCard from '../components/TaskCard.vue'

const store = useTaskStore()

const todayTasks = computed(() => {
  const today = new Date().toISOString().split('T')[0]
  return store.filteredTasks.filter(t => t.due_date?.split('T')[0] === today && t.status !== 'completed')
})

const inProgressTasks = computed(() => store.inProgressTasks)
const myTasks = computed(() => store.myTasks)

const emit = defineEmits<{
  taskClick: [task: Task]
}>()
</script>

<style scoped>
.workspace-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-6);
  overflow-y: auto;
  padding: var(--spacing-4);
}
.workspace-section-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  margin-bottom: var(--spacing-3);
}
.workspace-section-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}
.section-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.section-count {
  font-size: 11px;
  background: var(--border-color);
  color: var(--text-secondary);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}
.workspace-task-list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}
.workspace-empty {
  padding: var(--spacing-4);
  text-align: center;
  font-size: 13px;
  color: var(--text-secondary);
}
</style>
