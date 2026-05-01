<template>
  <div class="task-app">
    <AppHeader title="任务管理" @back="$emit('back')">
      <template #extra-buttons>
        <button class="icon-btn" @click="$emit('toggleSidebar')">
          <i class="fas fa-compress"></i>
        </button>
      </template>
    </AppHeader>
    <div class="task-app-body">
      <TaskSidebar />
      <div class="task-app-main">
        <TaskToolbar @create="showCreateModal = true" />
        <div class="task-app-content">
          <KanbanView
            v-if="store.currentView === 'kanban'"
            @task-click="onTaskClick"
            @task-contextmenu="onTaskContextmenu"
          />
          <ListView
            v-else-if="store.currentView === 'list'"
            @task-click="onTaskClick"
            @task-contextmenu="onTaskContextmenu"
          />
          <CalendarView
            v-else-if="store.currentView === 'calendar'"
            @task-click="onTaskClick"
          />
          <MyWorkspace
            v-else-if="store.currentView === 'workspace'"
            @task-click="onTaskClick"
          />
        </div>
      </div>
      <TaskDetailPanel
        :task="store.selectedTask"
        :available-tags="availableTags"
        :contacts="contacts"
        @close="store.selectTask(null)"
      />
    </div>

    <TaskCreateModal
      :visible="showCreateModal"
      :task="selectedTask"
      @close="onCloseModal"
      @submit="onSubmitTask"
    />

    <div
      v-if="contextMenu.visible"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
    >
      <button class="context-item" @click="onContextEdit">
        <i class="fas fa-edit"></i> 编辑
      </button>
      <button class="context-item" @click="onContextStatusChange('todo')">
        <i class="fas fa-circle" style="color:#fbbf24;font-size:8px;"></i> 移到待办
      </button>
      <button class="context-item" @click="onContextStatusChange('in_progress')">
        <i class="fas fa-circle" style="color:#a78bfa;font-size:8px;"></i> 移到进行中
      </button>
      <button class="context-item" @click="onContextStatusChange('completed')">
        <i class="fas fa-circle" style="color:#34d399;font-size:8px;"></i> 移到已完成
      </button>
      <div class="context-divider"></div>
      <button class="context-item danger" @click="onContextDelete">
        <i class="fas fa-trash"></i> 删除
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import type { Task, TaskStatus, TaskPriority, Tag, TaskUser } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import AppHeader from '../AppHeader.vue'
import TaskSidebar from './TaskSidebar.vue'
import TaskToolbar from './TaskToolbar.vue'
import KanbanView from './views/KanbanView.vue'
import ListView from './views/ListView.vue'
import CalendarView from './views/CalendarView.vue'
import MyWorkspace from './views/MyWorkspace.vue'
import TaskCreateModal from './components/TaskCreateModal.vue'
import TaskDetailPanel from './components/TaskDetailPanel.vue'
import QMessage from '../../../utils/qmessage'

const store = useTaskStore()
const showCreateModal = ref(false)
const selectedTask = computed(() => store.selectedTask)
const availableTags = ref<Tag[]>([
  { id: '1', name: '设计', color: '#ec4899' },
  { id: '2', name: '后端', color: '#6366f1' },
  { id: '3', name: '前端', color: '#3b82f6' },
  { id: '4', name: '重构', color: '#8b5cf6' },
  { id: '5', name: '文档', color: '#10b981' },
  { id: '6', name: 'Bug', color: '#ef4444' }
])
const contacts = ref<TaskUser[]>([])
const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  taskId: '' as string
})

defineEmits<{
  back: []
  toggleSidebar: []
}>()

onMounted(() => {
  store.loadTasks()
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('click', onGlobalClick)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('click', onGlobalClick)
})

function onTaskClick(task: Task) {
  store.selectTask(task.id)
}

function onTaskContextmenu(event: MouseEvent, task: Task) {
  contextMenu.visible = true
  contextMenu.x = event.clientX
  contextMenu.y = event.clientY
  contextMenu.taskId = task.id
  store.selectTask(task.id)
}

function closeContextMenu() {
  contextMenu.visible = false
}

function onContextEdit() {
  showCreateModal.value = true
  closeContextMenu()
}

async function onContextStatusChange(status: TaskStatus) {
  await store.changeStatus(contextMenu.taskId, status)
  closeContextMenu()
}

async function onContextDelete() {
  await store.removeTask(contextMenu.taskId)
  closeContextMenu()
  QMessage.success('任务已删除')
}

function onGlobalClick() {
  if (contextMenu.visible) closeContextMenu()
}

function onKeydown(e: KeyboardEvent) {
  if (showCreateModal.value) return
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return

  switch (e.key) {
    case 'n':
    case 'N':
      e.preventDefault()
      showCreateModal.value = true
      break
    case '1':
      e.preventDefault()
      store.setView('kanban')
      break
    case '2':
      e.preventDefault()
      store.setView('list')
      break
    case '3':
      e.preventDefault()
      store.setView('calendar')
      break
    case '4':
      e.preventDefault()
      store.setView('workspace')
      break
    case 'Delete':
    case 'Backspace':
      if (store.selectedTaskId) {
        e.preventDefault()
        store.removeTask(store.selectedTaskId)
        QMessage.success('任务已删除')
      }
      break
  }
}

function onCloseModal() {
  showCreateModal.value = false
  store.selectTask(null)
}

async function onSubmitTask(data: {
  title: string
  description: string
  due_date: string | null
  priority: TaskPriority
  status: TaskStatus
}) {
  try {
    if (selectedTask.value) {
      await store.updateTask(selectedTask.value.id, data)
      QMessage.success('任务已更新')
    } else {
      await store.createTask(data)
      QMessage.success('任务已创建')
    }
    showCreateModal.value = false
    store.selectTask(null)
  } catch {
    QMessage.error('操作失败，请重试')
  }
}
</script>

<style scoped>
.task-app {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  min-width: 0;
}
.task-app-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}
.task-app-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}
.task-app-content {
  flex: 1;
  display: flex;
  overflow: hidden;
  padding: var(--spacing-3);
  background: var(--content-bg);
}
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: var(--spacing-1) var(--spacing-2);
  border-radius: var(--radius-sm);
  font-size: 14px;
}
.icon-btn:hover {
  background: var(--hover-bg);
}
.context-menu {
  position: fixed;
  z-index: var(--z-dropdown, 1000);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--spacing-1);
  min-width: 160px;
}
.context-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  border: none;
  background: none;
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-primary);
  cursor: pointer;
  text-align: left;
}
.context-item:hover { background: var(--hover-bg); }
.context-item.danger { color: #ef4444; }
.context-item.danger:hover { background: #fef2f2; }
.context-divider {
  height: 1px;
  background: var(--border-color);
  margin: var(--spacing-1) 0;
}
</style>
