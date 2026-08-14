<template>
  <div class="task-app">
    <AppHeader title="任务管理" @back="$emit('back')">
      <template #actions>
        <div class="header-search">
          <i class="fas fa-search"></i>
          <input
            type="text"
            :value="store.filters.search"
            @input="onSearch"
            placeholder="搜索任务..."
            class="header-search-input"
          />
        </div>
        <button class="create-task-btn" @click="showCreateModal = true">
          <i class="fas fa-plus"></i>
          新建任务
        </button>
      </template>
    </AppHeader>
    <div class="task-app-body">
      <TaskSidebar />
      <div class="task-app-main">
        <div class="task-app-content">
          <div v-if="store.loading" class="task-loading">
            <i class="fas fa-spinner fa-spin"></i>
            <span>加载中...</span>
          </div>
          <template v-else>
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
              @task-contextmenu="onTaskContextmenu"
              @create-on-date="createTaskOnDate"
            />
            <MyWorkspace
              v-else-if="store.currentView === 'workspace'"
              @task-click="onTaskClick"
              @task-contextmenu="onTaskContextmenu"
            />
          </template>
        </div>
      </div>
    </div>
    <TaskDetailPanel
      :task="store.selectedTask"
      :available-tags="availableTags"
      :contacts="contacts"
      @close="store.selectTask(null)"
    />

    <TaskCreateModal
      :visible="showCreateModal"
      :task="editingTask"
      :defaultDueDate="defaultDueDate || undefined"
      @close="onCloseModal"
      @submit="onSubmitTask"
    />

    <UniversalContextMenu menuId="task" :items="contextMenuItems" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import type { Task, TaskStatus, TaskPriority, Tag, TaskUser } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import { request } from '../../../composables/useRequest'
import AppHeader from '../AppHeader.vue'
import TaskSidebar from './TaskSidebar.vue'
import KanbanView from './views/KanbanView.vue'
import ListView from './views/ListView.vue'
import CalendarView from './views/CalendarView.vue'
import MyWorkspace from './views/MyWorkspace.vue'
import UniversalContextMenu from '../../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../../shared/context-menu-types'
import TaskCreateModal from './components/TaskCreateModal.vue'
import TaskDetailPanel from './components/TaskDetailPanel.vue'
import QMessage from '../../../utils/qmessage'
import { openMenu, closeMenu } from '../../../composables/useUI'

const store = useTaskStore()
const showCreateModal = ref(false)
const editingTask = ref<Task | null>(null)
const defaultDueDate = ref<string | null>(null)
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
}>()

onMounted(async () => {
  // 先等任务加载完成，保证待办深链/聚焦时 store.tasks 已就绪（否则 find 落空导致深链静默失效）
  await store.loadTasks()
  loadContacts()
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('click', onGlobalClick)
  // 日历右键"新建任务"跨应用跳转：预填截止日打开创建弹窗
  if (store.pendingCreateOnDate) {
    createTaskOnDate(new Date(store.pendingCreateOnDate))
    store.pendingCreateOnDate = null
  }
  // 通知中心"待办指派"点击后：聚焦打开对应任务详情
  if (store.pendingFocusTaskId) {
    const taskId = store.pendingFocusTaskId
    store.pendingFocusTaskId = null
    const task = store.tasks.find(t => t.id === taskId)
    if (task) {
      store.selectTask(taskId)
    }
  }
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('click', onGlobalClick)
})

async function loadContacts() {
  try {
    const response: any = await request('/api/v1/organization/tree')
    if (response.code === 0 && response.data) {
      const users: TaskUser[] = []
      const collectUsers = (departments: any[]) => {
        for (const dept of departments) {
          if (dept.employees) {
            for (const emp of dept.employees) {
              users.push({
                id: String(emp.id),
                name: emp.nickname || emp.username || '',
                avatar: emp.avatar || ''
              })
            }
          }
          if (dept.subDepartments) {
            collectUsers(dept.subDepartments)
          }
        }
      }
      collectUsers(response.data.departments)
      contacts.value = users
      store.setContacts(users)
    }
  } catch (e) {
    console.error('Failed to load contacts:', e)
  }
}

function onTaskClick(task: Task) {
  store.selectTask(task.id)
}

function onSearch(event: Event) {
  store.setFilters({ search: (event.target as HTMLInputElement).value })
}

function onTaskContextmenu(event: MouseEvent, task: Task) {
  contextMenu.taskId = task.id
  openMenu('task', event.clientX, event.clientY)
}

function closeContextMenu() {
  closeMenu()
}

const contextMenuItems = computed<ContextMenuItem[]>(() => [
  { label: '编辑', icon: 'fas fa-edit', action: onContextEdit },
  { label: '移到待办', icon: 'fas fa-circle', iconColor: '#fbbf24', action: () => onContextStatusChange('todo') },
  { label: '移到进行中', icon: 'fas fa-circle', iconColor: '#a78bfa', action: () => onContextStatusChange('in_progress') },
  { label: '移到已完成', icon: 'fas fa-circle', iconColor: '#34d399', action: () => onContextStatusChange('completed') },
  { divider: true },
  { label: '删除', icon: 'fas fa-trash', danger: true, action: onContextDelete }
])

function onContextEdit() {
  const task = store.tasks.find(t => t.id === contextMenu.taskId)
  if (task) editingTask.value = task
  showCreateModal.value = true
  closeContextMenu()
}

// 日历视图双击日期：新建任务并预填截止日
function createTaskOnDate(date: Date) {
  editingTask.value = null
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  defaultDueDate.value = `${y}-${m}-${d}`
  showCreateModal.value = true
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
  closeContextMenu()
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
      store.setView('workspace')
      break
    case '2':
      e.preventDefault()
      store.setView('kanban')
      break
    case '3':
      e.preventDefault()
      store.setView('list')
      break
    case '4':
      e.preventDefault()
      store.setView('calendar')
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
  editingTask.value = null
  defaultDueDate.value = null
}

async function onSubmitTask(data: {
  title: string
  description: string
  due_date: string | null
  priority: TaskPriority
  status: TaskStatus
}) {
  try {
    if (editingTask.value) {
      await store.updateTask(editingTask.value.id, data)
      QMessage.success('任务已更新')
    } else {
      await store.createTask(data)
      QMessage.success('任务已创建')
    }
    showCreateModal.value = false
    editingTask.value = null
    defaultDueDate.value = null
    store.refreshTasks()
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
  box-shadow: var(--shadow-md);
  min-width: 0;
}
.header-search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 32px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
}
.header-search i {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}
.header-search-input {
  width: 140px;
  height: 100%;
  border: none;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  background: transparent;
  outline: none;
}
.header-search-input::placeholder {
  color: var(--text-secondary);
}
.create-task-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 14px;
  height: 32px;
  background: #8b5cf6;
  color: #fff;
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
}
.create-task-btn:hover {
  background: #7c3aed;
}
.task-app-body {
  flex: 1;
  display: flex;
  flex-direction: column;
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
  min-height: 0;
  padding: var(--spacing-3);
  background: var(--card-bg, #fff);
}
.task-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-3);
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
}
.task-loading i { font-size: var(--font-size-2xl); color: #8b5cf6; }
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-secondary);
  padding: var(--spacing-1) var(--spacing-2);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
}
.icon-btn:hover {
  background: var(--hover-bg);
}
</style>
