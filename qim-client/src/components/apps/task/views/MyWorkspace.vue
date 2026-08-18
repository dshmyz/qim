<template>
  <div class="workspace-view">
    <div v-if="!visibleSections.length" class="workspace-empty-global">
      <i class="fas fa-tasks"></i>
      <p>暂无任务</p>
      <span>点击「新建任务」或按 N 键创建任务</span>
    </div>
    <div v-if="todayTasksEmpty" class="workspace-hint">
      <i class="fas fa-sun"></i>
      <span>今天没有待办任务</span>
    </div>
    <section
      v-for="section in visibleSections"
      :key="section.title"
      class="workspace-section"
    >
      <div class="workspace-section-header">
        <span class="section-dot" :style="{ background: section.color }"></span>
        <h3>{{ section.title }}</h3>
        <span class="section-count">{{ section.tasks.length }}</span>
      </div>
      <div class="workspace-task-list">
        <TaskCard
          v-for="task in section.tasks"
          :key="task.id"
          :task="task"
          @click="emit('taskClick', $event)"
          @contextmenu="(event, task) => emit('taskContextmenu', event, task)"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task, TaskPriority } from '../../../../types/task'
import { useTaskStore } from '../../../../stores/task'
import TaskCard from '../components/TaskCard.vue'

const store = useTaskStore()

const PRIORITY_RANK: Record<TaskPriority, number> = { high: 0, medium: 1, low: 2 }

// 工作台统一排序：优先级高→低，其次截止日期近→远
function byPriorityAndDue(a: Task, b: Task) {
  const rankDiff = (PRIORITY_RANK[a.priority] ?? 3) - (PRIORITY_RANK[b.priority] ?? 3)
  if (rankDiff !== 0) return rankDiff
  const da = a.due_date ? a.due_date.split('T')[0] : '9999-99-99'
  const db = b.due_date ? b.due_date.split('T')[0] : '9999-99-99'
  return da < db ? -1 : da > db ? 1 : 0
}

// 今日待办/待办/进行中 三区互不重叠：今日待办只收「今天到期且未开始」的任务，
// 其余按状态归属（进行中不受今天到期影响），保证工作台能看到全部非已完成任务；
// 已指派给我保留为“我的任务”汇总。
// today 用本地日期拼 YYYY-MM-DD（toISOString 取 UTC 日期，UTC+8 凌晨 0-8 点会错位一天）
const today = computed(() => {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
})
const dueToday = (t: Task) => t.due_date?.split('T')[0] === today.value
const notDueToday = (t: Task) => !dueToday(t)

const sections = computed(() => {
  const tasks = store.filteredTasks
  const todayTasks = tasks.filter(t => dueToday(t) && t.status === 'todo').sort(byPriorityAndDue)
  const todoTasks = tasks.filter(t => t.status === 'todo' && notDueToday(t)).sort(byPriorityAndDue)
  const inProgressTasks = tasks.filter(t => t.status === 'in_progress').sort(byPriorityAndDue)
  const myTasks = store.myTasks.sort(byPriorityAndDue)

  return [
    { title: '今日待办', color: '#ef4444', tasks: todayTasks },
    { title: '待办', color: '#fbbf24', tasks: todoTasks },
    { title: '进行中', color: '#a78bfa', tasks: inProgressTasks },
    { title: '已指派给我', color: '#f59e0b', tasks: myTasks }
  ]
})

// 空分区自动隐藏：仅渲染有任务的分区，避免空壳占位造成界面杂乱
const visibleSections = computed(() => sections.value.filter(s => s.tasks.length))

// 今日待办为空但工作台仍有其他任务时，给一行轻提示，避免用户疑惑“今日待办去哪了”
const todayTasksEmpty = computed(
  () => visibleSections.value.length > 0 && !store.filteredTasks.some(t => dueToday(t) && t.status === 'todo')
)

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
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
  min-height: 0;
}
.workspace-section-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  margin-bottom: var(--spacing-3);
}
.workspace-section-header h3 {
  font-size: var(--font-size-sm);
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
  font-size: var(--font-size-xxxs);
  background: var(--border-color);
  color: var(--text-secondary);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}
.workspace-task-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--spacing-2);
}
.workspace-empty-global {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-2);
  color: var(--text-secondary);
}
.workspace-empty-global i {
  font-size: var(--font-size-3xl);
  opacity: 0.4;
}
.workspace-empty-global p {
  font-size: var(--font-size-sm);
  margin: 0;
  color: var(--text-primary);
}
.workspace-empty-global span {
  font-size: var(--font-size-xxs);
}
.workspace-hint {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}
.workspace-hint i {
  font-size: var(--font-size-sm);
  opacity: 0.6;
}
</style>
