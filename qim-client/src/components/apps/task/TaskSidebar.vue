<template>
  <div class="task-topbar">
    <div class="topbar-nav">
      <button
        v-for="view in views"
        :key="view.id"
        class="nav-item"
        :class="{ active: currentView === view.id }"
        :title="view.label"
        @click="store.setView(view.id)"
      >
        <i :class="view.icon"></i>
        <span class="nav-label">{{ view.label }}</span>
      </button>
    </div>
    <div class="topbar-actions">
      <div class="filter-group">
        <button
          class="filter-tag"
          :class="{ on: store.filters.priority === 'high' }"
          title="高优先级"
          @click="togglePriorityFilter('high')"
        >
          <span class="tag-dot" style="background:#ef4444;"></span>
          <span class="tag-label">高优先级</span>
        </button>
        <button
          class="filter-tag"
          :class="{ on: !!store.filters.due_date_range }"
          title="即将到期"
          @click="toggleDueSoonFilter"
        >
          <i class="fas fa-clock"></i>
          <span class="tag-label">即将到期</span>
        </button>
        <button
          class="filter-tag"
          :class="{ on: !!store.filters.assignee_id }"
          title="我的任务"
          @click="toggleMyTasksFilter"
        >
          <i class="fas fa-user"></i>
          <span class="tag-label">我的任务</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import type { TaskView } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'

const store = useTaskStore()
const currentView = computed(() => store.currentView)

const views: { id: TaskView; label: string; icon: string }[] = [
  { id: 'kanban', label: '看板', icon: 'fas fa-th-large' },
  { id: 'list', label: '列表', icon: 'fas fa-list' },
  { id: 'calendar', label: '日历', icon: 'fas fa-calendar-alt' },
  { id: 'workspace', label: '工作台', icon: 'fas fa-user-circle' }
]

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

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    document.querySelector<HTMLElement>('.search-input')?.focus()
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineEmits<{
}>()
</script>

<style scoped>
.task-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
  flex-shrink: 0;
  height: 44px;
  gap: var(--spacing-3);
}
.topbar-nav {
  display: flex;
  align-items: stretch;
  height: 100%;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 14px;
  border: none;
  background: none;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  position: relative;
  transition: color var(--animation-fast) ease;
  white-space: nowrap;
}
.nav-item:hover {
  color: var(--text-primary);
  background: var(--hover-bg);
}
.nav-item.active {
  color: var(--text-primary);
  font-weight: 500;
}
.nav-item.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 14px;
  right: 14px;
  height: 2px;
  background: #8b5cf6;
  border-radius: 1px;
}
.topbar-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  flex-shrink: 0;
}
.filter-group {
  display: flex;
  gap: 4px;
}
.filter-tag {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px;
  border: 1px solid var(--border-color);
  background: none;
  border-radius: 100px;
  font-size: 11px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--animation-fast) ease;
  white-space: nowrap;
}
.filter-tag:hover {
  border-color: var(--text-secondary);
  color: var(--text-primary);
}
.filter-tag.on {
  background: #8b5cf6;
  border-color: #8b5cf6;
  color: #fff;
}
.tag-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 8px;
  height: 28px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  transition: border-color var(--animation-fast) ease;
}
.search-box:focus-within {
  border-color: #8b5cf6;
}
.search-box i {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}
.search-input {
  width: 100px;
  border: none;
  font-size: 12px;
  color: var(--text-primary);
  background: transparent;
  outline: none;
  padding: 0;
  transition: width 0.2s ease;
}
.search-input::placeholder {
  color: var(--text-secondary);
}
.search-input:focus {
  width: 160px;
}
.create-btn {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 5px 14px;
  background: #8b5cf6;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--animation-fast) ease;
  white-space: nowrap;
  flex-shrink: 0;
}
.create-btn:hover { background: #7c3aed; }

@media (max-width: 720px) {
  .nav-label {
    display: none;
  }
  .nav-item {
    padding: 0 10px;
  }
  .tag-label {
    display: none;
  }
  .filter-tag {
    padding: 3px 6px;
  }
  .create-label {
    display: none;
  }
  .create-btn {
    padding: 5px 8px;
  }
  .search-input {
    width: 70px;
  }
  .search-input:focus {
    width: 120px;
  }
}
</style>
