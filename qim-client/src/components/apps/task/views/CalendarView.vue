<template>
  <div class="calendar-view">
    <div class="calendar-nav">
      <button class="cal-nav-btn" @click="prevMonth"><i class="fas fa-chevron-left"></i></button>
      <span class="cal-nav-title">{{ monthLabel }}</span>
      <button class="cal-nav-btn" @click="nextMonth"><i class="fas fa-chevron-right"></i></button>
    </div>
    <div class="calendar-grid">
      <div v-for="day in weekDays" :key="day" class="cal-header-cell">{{ day }}</div>
      <div
        v-for="cell in calendarCells"
        :key="cell.key"
        class="cal-cell"
        :class="{ 'is-today': cell.isToday, 'is-other-month': !cell.isCurrentMonth }"
      >
        <div class="cal-date">{{ cell.day }}</div>
        <div class="cal-tasks">
          <div
            v-for="task in cell.tasks.slice(0, 3)"
            :key="task.id"
            class="cal-task"
            :class="'priority-' + task.priority"
            @click="emit('taskClick', task)"
          >
            {{ task.title }}
          </div>
          <div v-if="cell.tasks.length > 3" class="cal-more">+{{ cell.tasks.length - 3 }} 更多</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Task } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'

const store = useTaskStore()

const currentDate = ref(new Date())
const weekDays = ['一', '二', '三', '四', '五', '六', '日']

const monthLabel = computed(() => {
  const y = currentDate.value.getFullYear()
  const m = currentDate.value.getMonth() + 1
  return `${y}年${m}月`
})

interface CalendarCell {
  key: string
  day: number
  isCurrentMonth: boolean
  isToday: boolean
  tasks: Task[]
}

const calendarCells = computed(() => {
  const year = currentDate.value.getFullYear()
  const month = currentDate.value.getMonth()
  const firstDay = new Date(year, month, 1)
  const startOffset = (firstDay.getDay() + 6) % 7
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const cells: CalendarCell[] = []

  for (let i = -startOffset; i < 42 - startOffset; i++) {
    const date = new Date(year, month, 1 + i)
    const dateStr = date.toISOString().split('T')[0]
    const isCurrentMonth = date.getMonth() === month
    const isToday = date.getTime() === today.getTime()
    const tasks = store.tasksByDate.get(dateStr) || []
    cells.push({ key: dateStr, day: date.getDate(), isCurrentMonth, isToday, tasks })
  }
  return cells
})

function prevMonth() {
  currentDate.value = new Date(currentDate.value.getFullYear(), currentDate.value.getMonth() - 1, 1)
}

function nextMonth() {
  currentDate.value = new Date(currentDate.value.getFullYear(), currentDate.value.getMonth() + 1, 1)
}

const emit = defineEmits<{
  taskClick: [task: Task]
}>()
</script>

<style scoped>
.calendar-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.calendar-nav {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-4);
  padding: var(--spacing-3);
}
.cal-nav-btn {
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: var(--spacing-1) var(--spacing-2);
  cursor: pointer;
  color: var(--text-secondary);
}
.cal-nav-btn:hover { background: var(--hover-bg); }
.cal-nav-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  min-width: 100px;
  text-align: center;
}
.calendar-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 1px;
  background: var(--border-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.cal-header-cell {
  padding: var(--spacing-2);
  text-align: center;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  background: var(--card-bg);
}
.cal-cell {
  background: var(--card-bg);
  padding: var(--spacing-1) var(--spacing-2);
  min-height: 80px;
  overflow: hidden;
}
.cal-cell.is-today { background: #faf5ff; }
.cal-cell.is-other-month { opacity: 0.4; }
.cal-date {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 2px;
}
.cal-tasks {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.cal-task {
  font-size: 9px;
  padding: 1px 4px;
  border-radius: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
  border-left: 2px solid transparent;
}
.cal-task.priority-high { background: #fef2f2; color: #ef4444; border-left-color: #ef4444; }
.cal-task.priority-medium { background: #fffbeb; color: #d97706; border-left-color: #d97706; }
.cal-task.priority-low { background: #eff6ff; color: #3b82f6; border-left-color: #3b82f6; }
.cal-more {
  font-size: 9px;
  color: var(--text-secondary);
  padding: 1px 4px;
}
</style>
