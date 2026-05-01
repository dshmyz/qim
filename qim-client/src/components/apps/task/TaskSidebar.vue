<template>
  <div class="task-sidebar">
    <div class="sidebar-title">
      <span class="sidebar-icon">✓</span>
      任务管理
    </div>
    <div class="sidebar-section">
      <div class="sidebar-label">视图</div>
      <button
        v-for="view in views"
        :key="view.id"
        class="sidebar-item"
        :class="{ active: currentView === view.id }"
        @click="store.setView(view.id)"
      >
        <i :class="view.icon"></i>
        {{ view.label }}
      </button>
    </div>
    <div class="sidebar-section">
      <div class="sidebar-label">筛选</div>
      <button
        class="sidebar-item"
        :class="{ active: store.filters.priority === 'high' }"
        @click="togglePriorityFilter('high')"
      >
        <span class="filter-dot" style="background:#ef4444;"></span>
        高优先级
      </button>
      <button
        class="sidebar-item"
        :class="{ active: !!store.filters.due_date_range }"
        @click="toggleDueSoonFilter"
      >
        <i class="fas fa-clock"></i>
        即将到期
      </button>
      <button
        class="sidebar-item"
        :class="{ active: !!store.filters.assignee_id }"
        @click="toggleMyTasksFilter"
      >
        <i class="fas fa-user"></i>
        指派给我
      </button>
    </div>
    <div class="sidebar-section" v-if="store.tasks.length">
      <div class="sidebar-label">进度</div>
      <div class="progress-summary">
        <div class="progress-header">
          <span>本周进度</span>
          <span class="progress-percent">{{ completionPercent }}%</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: completionPercent + '%' }"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TaskView } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'

const store = useTaskStore()
const currentView = computed(() => store.currentView)

const views: { id: TaskView; label: string; icon: string }[] = [
  { id: 'kanban', label: '看板', icon: 'fas fa-th-large' },
  { id: 'list', label: '列表', icon: 'fas fa-list' },
  { id: 'calendar', label: '日历', icon: 'fas fa-calendar-alt' },
  { id: 'workspace', label: '我的工作台', icon: 'fas fa-user-circle' }
]

const completionPercent = computed(() => {
  const total = store.tasks.length
  if (!total) return 0
  const completed = store.tasks.filter(t => t.status === 'completed').length
  return Math.round((completed / total) * 100)
})

function togglePriorityFilter(priority: 'high') {
  store.setFilters({ priority: store.filters.priority === priority ? null : priority })
}

function toggleDueSoonFilter() {
  if (store.filters.due_date_range) {
    store.setFilters({ due_date_range: null })
  } else {
    const today = new Date()
    const nextWeek = new Date(today.getTime() + 7 * 24 * 60 * 60 * 1000)
    store.setFilters({
      due_date_range: { start: today.toISOString().split('T')[0], end: nextWeek.toISOString().split('T')[0] }
    })
  }
}

function toggleMyTasksFilter() {
  store.setFilters({ assignee_id: store.filters.assignee_id ? null : 'me' })
}
</script>

<style scoped>
.task-sidebar {
  width: 200px;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  padding: var(--spacing-4);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-1);
  overflow-y: auto;
  flex-shrink: 0;
}
.sidebar-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: var(--spacing-4);
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}
.sidebar-icon {
  width: 20px;
  height: 20px;
  background: #8b5cf6;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 10px;
}
.sidebar-section { margin-bottom: var(--spacing-4); }
.sidebar-label {
  font-size: 10px;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: var(--spacing-1);
  padding: 0 var(--spacing-2);
}
.sidebar-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  width: 100%;
  padding: var(--spacing-1) var(--spacing-2);
  border: none;
  background: none;
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  text-align: left;
  transition: all var(--animation-fast) ease;
}
.sidebar-item:hover { background: var(--hover-bg); color: var(--text-primary); }
.sidebar-item.active { background: var(--active-bg); color: var(--text-primary); font-weight: 500; }
.filter-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.progress-summary {
  padding: var(--spacing-2);
  background: var(--hover-bg);
  border-radius: var(--radius-md);
}
.progress-header {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}
.progress-percent { color: #059669; font-weight: 500; }
.progress-bar {
  height: 4px;
  background: var(--border-color);
  border-radius: 2px;
  overflow: hidden;
}
.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #a78bfa, #8b5cf6);
  border-radius: 2px;
  transition: width var(--animation-slow) ease;
}
</style>
