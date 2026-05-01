<template>
  <div class="task-toolbar">
    <div class="toolbar-left">
      <span class="toolbar-title">{{ viewLabel }}</span>
      <span class="toolbar-count">{{ store.filteredTasks.length }} 个任务</span>
    </div>
    <div class="toolbar-right">
      <div class="toolbar-search">
        <input
          type="text"
          :value="store.filters.search"
          @input="onSearch"
          placeholder="搜索任务..."
          class="search-input"
        />
        <span class="search-shortcut">⌘K</span>
      </div>
      <button class="create-btn" @click="$emit('create')">
        <i class="fas fa-plus"></i> 新建
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import type { TaskView } from '../../types/task'
import { useTaskStore } from '../../stores/task'

const store = useTaskStore()

const viewLabels: Record<TaskView, string> = {
  kanban: '看板视图',
  list: '列表视图',
  calendar: '日历视图',
  workspace: '我的工作台'
}

const viewLabel = computed(() => viewLabels[store.currentView])

function onSearch(event: Event) {
  store.setFilters({ search: (event.target as HTMLInputElement).value })
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    const input = document.querySelector('.search-input') as HTMLInputElement
    input?.focus()
  }
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onUnmounted(() => document.removeEventListener('keydown', onKeydown))

defineEmits<{
  create: []
}>()
</script>

<style scoped>
.task-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-2) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}
.toolbar-left { display: flex; align-items: center; gap: var(--spacing-2); }
.toolbar-title { font-size: 13px; font-weight: 600; color: var(--text-primary); }
.toolbar-count { font-size: 11px; color: var(--text-secondary); }
.toolbar-right { display: flex; align-items: center; gap: var(--spacing-2); }
.toolbar-search { position: relative; }
.search-input {
  padding: 5px 28px 5px 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  width: 160px;
  color: var(--text-primary);
  background: var(--input-bg);
  transition: border-color var(--animation-fast) ease;
}
.search-input:focus { outline: none; border-color: #8b5cf6; }
.search-shortcut {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 10px;
  color: var(--text-secondary);
  pointer-events: none;
}
.create-btn {
  padding: 5px 12px;
  background: #8b5cf6;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: background var(--animation-fast) ease;
}
.create-btn:hover { background: #7c3aed; }
</style>
