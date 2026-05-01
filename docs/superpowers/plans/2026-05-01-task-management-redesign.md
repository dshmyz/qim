# 任务管理模块重新设计 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将任务管理模块从 915 行单体组件重构为多视图、可拖拽、支持标签/子任务/指派的现代任务管理系统

**架构：** 渐进式重构四步走——先拆分组件+类型+API+Store解决技术债，再升级视觉+四种视图，再增强拖拽/右键/快捷键交互，最后接入标签/子任务/聊天联动。每步独立可验证，保持功能可用。

**技术栈：** Vue 3 + TypeScript + Pinia + 原生 HTML5 Drag & Drop + 项目自研 CSS 变量体系

---

## 文件结构

### 新建文件

| 文件 | 职责 |
|------|------|
| `src/types/task.ts` | Task/SubTask/Tag 等类型定义 |
| `src/api/task.ts` | 任务 API 封装（基于 useRequest） |
| `src/stores/task.ts` | 任务 Pinia Store（Setup Store 风格） |
| `src/composables/useTaskDragDrop.ts` | 拖拽逻辑 composable |
| `src/components/apps/task/TaskManagementApp.vue` | 主容器（重构后） |
| `src/components/apps/task/TaskSidebar.vue` | 侧边栏（视图切换+筛选+进度） |
| `src/components/apps/task/TaskToolbar.vue` | 工具栏（搜索+新建+批量操作） |
| `src/components/apps/task/views/KanbanView.vue` | 看板视图 |
| `src/components/apps/task/views/KanbanColumn.vue` | 看板列 |
| `src/components/apps/task/views/ListView.vue` | 列表视图 |
| `src/components/apps/task/views/CalendarView.vue` | 日历视图 |
| `src/components/apps/task/views/MyWorkspace.vue` | 我的工作台 |
| `src/components/apps/task/components/TaskCard.vue` | 任务卡片 |
| `src/components/apps/task/components/TaskRow.vue` | 列表行 |
| `src/components/apps/task/components/TaskDetailPanel.vue` | 任务详情侧滑面板 |
| `src/components/apps/task/components/TaskCreateModal.vue` | 创建/编辑任务弹窗 |
| `src/components/apps/task/components/TagSelector.vue` | 标签选择器 |
| `src/components/apps/task/components/AssigneeSelector.vue` | 指派人选择 |
| `src/components/apps/task/components/SubTaskList.vue` | 子任务列表 |
| `src/components/apps/task/index.ts` | 统一导出 |

### 修改文件

| 文件 | 变更 |
|------|------|
| `src/types/index.ts` | 末尾添加 `export * from './task'` |
| `src/views/Main.vue` | 更新 TaskManagementApp 导入路径 |

### 删除文件

| 文件 | 原因 |
|------|------|
| `src/components/apps/TaskManagementApp.vue` | 旧单体组件，被新目录结构替代 |

---

## 阶段一：基础重构（解决技术债）

### 任务 1：创建类型定义

**文件：**
- 创建：`src/types/task.ts`
- 修改：`src/types/index.ts`

- [ ] **步骤 1：创建 `src/types/task.ts`**

```typescript
export type TaskStatus = 'todo' | 'in_progress' | 'completed'
export type TaskPriority = 'low' | 'medium' | 'high'
export type TaskView = 'kanban' | 'list' | 'calendar' | 'workspace'

export interface Task {
  id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  due_date: string | null
  tags: Tag[]
  assignee: TaskUser | null
  creator: TaskUser
  sub_tasks: SubTask[]
  comment_count: number
  position: number
  created_at: string
  updated_at: string
}

export interface SubTask {
  id: string
  title: string
  completed: boolean
  position: number
}

export interface Tag {
  id: string
  name: string
  color: string
}

export interface TaskUser {
  id: string
  name: string
  avatar: string
}

export interface TaskFilters {
  search: string
  priority: TaskPriority | null
  assignee_id: string | null
  tag_id: string | null
  due_date_range: { start: string; end: string } | null
}

export interface CreateTaskData {
  title: string
  description?: string
  priority?: TaskPriority
  due_date?: string | null
  status?: TaskStatus
  tags?: string[]
  assignee_id?: string | null
}

export interface UpdateTaskData extends Partial<CreateTaskData> {
  position?: number
}
```

- [ ] **步骤 2：修改 `src/types/index.ts`，末尾添加重导出**

在文件末尾已有的 `export * from './xxx'` 行之后添加：

```typescript
export * from './task'
```

- [ ] **步骤 3：Commit**

```bash
git add src/types/task.ts src/types/index.ts
git commit -m "feat(task): add Task type definitions"
```

---

### 任务 2：创建 API 层

**文件：**
- 创建：`src/api/task.ts`

- [ ] **步骤 1：创建 `src/api/task.ts`**

```typescript
import { request } from '../composables/useRequest'
import type { Task, CreateTaskData, UpdateTaskData } from '../types/task'

export async function fetchTasks(): Promise<Task[]> {
  return request<Task[]>('/api/v1/tasks')
}

export async function fetchTaskById(id: string): Promise<Task> {
  return request<Task>(`/api/v1/tasks/${id}`)
}

export async function createTask(data: CreateTaskData): Promise<Task> {
  return request<Task>('/api/v1/tasks', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}

export async function updateTask(id: string, data: UpdateTaskData): Promise<Task> {
  return request<Task>(`/api/v1/tasks/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  })
}

export async function updateTaskStatus(id: string, status: string): Promise<Task> {
  return request<Task>(`/api/v1/tasks/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status })
  })
}

export async function reorderTask(id: string, position: number, status?: string): Promise<void> {
  return request<void>(`/api/v1/tasks/${id}/reorder`, {
    method: 'PUT',
    body: JSON.stringify({ position, status })
  })
}

export async function deleteTask(id: string): Promise<void> {
  return request<void>(`/api/v1/tasks/${id}`, {
    method: 'DELETE'
  })
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/api/task.ts
git commit -m "feat(task): add task API layer with useRequest"
```

---

### 任务 3：创建 Pinia Store

**文件：**
- 创建：`src/stores/task.ts`

- [ ] **步骤 1：创建 `src/stores/task.ts`**

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Task, TaskFilters, TaskView, TaskStatus } from '../types/task'
import { fetchTasks, createTask as apiCreateTask, updateTask as apiUpdateTask, deleteTask as apiDeleteTask, updateTaskStatus as apiUpdateStatus, reorderTask as apiReorderTask } from '../api/task'
import type { CreateTaskData, UpdateTaskData } from '../types/task'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<Task[]>([])
  const currentView = ref<TaskView>('kanban')
  const filters = ref<TaskFilters>({
    search: '',
    priority: null,
    assignee_id: null,
    tag_id: null,
    due_date_range: null
  })
  const selectedTaskId = ref<string | null>(null)
  const loading = ref(false)

  const filteredTasks = computed(() => {
    let result = tasks.value
    if (filters.value.search) {
      const q = filters.value.search.toLowerCase()
      result = result.filter(t =>
        t.title.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q)
      )
    }
    if (filters.value.priority) {
      result = result.filter(t => t.priority === filters.value.priority)
    }
    if (filters.value.assignee_id) {
      result = result.filter(t => t.assignee?.id === filters.value.assignee_id)
    }
    if (filters.value.tag_id) {
      result = result.filter(t => t.tags.some(tag => tag.id === filters.value.tag_id))
    }
    return result
  })

  const todoTasks = computed(() =>
    filteredTasks.value.filter(t => t.status === 'todo').sort((a, b) => a.position - b.position)
  )

  const inProgressTasks = computed(() =>
    filteredTasks.value.filter(t => t.status === 'in_progress').sort((a, b) => a.position - b.position)
  )

  const completedTasks = computed(() =>
    filteredTasks.value.filter(t => t.status === 'completed').sort((a, b) => a.position - b.position)
  )

  const tasksByDate = computed(() => {
    const map = new Map<string, Task[]>()
    filteredTasks.value.forEach(t => {
      if (t.due_date) {
        const date = t.due_date.split('T')[0]
        if (!map.has(date)) map.set(date, [])
        map.get(date)!.push(t)
      }
    })
    return map
  })

  const myTasks = computed(() =>
    filteredTasks.value.filter(t => t.assignee?.id === 'me')
  )

  const selectedTask = computed(() =>
    tasks.value.find(t => t.id === selectedTaskId.value) || null
  )

  async function loadTasks() {
    loading.value = true
    try {
      tasks.value = await fetchTasks()
    } catch (e: any) {
      console.error('Failed to load tasks:', e)
    } finally {
      loading.value = false
    }
  }

  async function createTask(data: CreateTaskData) {
    try {
      const task = await apiCreateTask(data)
      tasks.value.push(task)
      return task
    } catch (e: any) {
      console.error('Failed to create task:', e)
      throw e
    }
  }

  async function updateTask(id: string, data: UpdateTaskData) {
    try {
      const updated = await apiUpdateTask(id, data)
      const index = tasks.value.findIndex(t => t.id === id)
      if (index !== -1) tasks.value[index] = updated
      return updated
    } catch (e: any) {
      console.error('Failed to update task:', e)
      throw e
    }
  }

  async function removeTask(id: string) {
    try {
      await apiDeleteTask(id)
      tasks.value = tasks.value.filter(t => t.id !== id)
    } catch (e: any) {
      console.error('Failed to delete task:', e)
      throw e
    }
  }

  async function changeStatus(id: string, status: TaskStatus) {
    try {
      const updated = await apiUpdateStatus(id, status)
      const index = tasks.value.findIndex(t => t.id === id)
      if (index !== -1) tasks.value[index] = updated
      return updated
    } catch (e: any) {
      console.error('Failed to update task status:', e)
      throw e
    }
  }

  async function reorderTaskItem(id: string, position: number, status?: string) {
    try {
      await apiReorderTask(id, position, status)
      const task = tasks.value.find(t => t.id === id)
      if (!task) return
      if (status) task.status = status as TaskStatus
      task.position = position
    } catch (e: any) {
      console.error('Failed to reorder task:', e)
      throw e
    }
  }

  function setView(view: TaskView) {
    currentView.value = view
  }

  function selectTask(id: string | null) {
    selectedTaskId.value = id
  }

  function setFilters(newFilters: Partial<TaskFilters>) {
    filters.value = { ...filters.value, ...newFilters }
  }

  function resetFilters() {
    filters.value = {
      search: '',
      priority: null,
      assignee_id: null,
      tag_id: null,
      due_date_range: null
    }
  }

  return {
    tasks,
    currentView,
    filters,
    selectedTaskId,
    loading,
    filteredTasks,
    todoTasks,
    inProgressTasks,
    completedTasks,
    tasksByDate,
    myTasks,
    selectedTask,
    loadTasks,
    createTask,
    updateTask,
    removeTask,
    changeStatus,
    reorderTaskItem,
    setView,
    selectTask,
    setFilters,
    resetFilters
  }
})
```

- [ ] **步骤 2：Commit**

```bash
git add src/stores/task.ts
git commit -m "feat(task): add task Pinia store with Setup Store style"
```

---

### 任务 4：创建 TaskCard 组件

**文件：**
- 创建：`src/components/apps/task/components/TaskCard.vue`

- [ ] **步骤 1：创建 TaskCard.vue**

```vue
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
import type { Task } from '../../../types/task'

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

.task-card.priority-high {
  border-left-color: #ef4444;
}

.task-card.priority-medium {
  border-left-color: #fbbf24;
}

.task-card.priority-low {
  border-left-color: #60a5fa;
}

.task-card.is-completed {
  opacity: 0.6;
}

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
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/TaskCard.vue
git commit -m "feat(task): add TaskCard component with priority bar, tags, progress"
```

---

### 任务 5：创建 KanbanColumn 和 KanbanView

**文件：**
- 创建：`src/components/apps/task/views/KanbanColumn.vue`
- 创建：`src/components/apps/task/views/KanbanView.vue`

- [ ] **步骤 1：创建 KanbanColumn.vue**

```vue
<template>
  <div
    class="kanban-column"
    @dragover.prevent="onDragOver"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <div class="kanban-column-header">
      <div class="column-title">
        <span class="column-dot" :style="{ background: dotColor }"></span>
        <span>{{ title }}</span>
        <span class="column-count">{{ tasks.length }}</span>
      </div>
    </div>
    <div class="kanban-column-list">
      <TaskCard
        v-for="task in tasks"
        :key="task.id"
        :task="task"
        @click="$emit('taskClick', $event)"
        @contextmenu="$emit('taskContextmenu', $event)"
        @dragstart="$emit('taskDragstart', $event)"
        @dragend="$emit('taskDragend')"
      />
      <div v-if="!tasks.length" class="kanban-column-empty">
        拖拽任务到此处
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Task, TaskStatus } from '../../../types/task'
import TaskCard from '../components/TaskCard.vue'

const props = defineProps<{
  title: string
  status: TaskStatus
  tasks: Task[]
  dotColor: string
}>()

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
  taskDragstart: [event: DragEvent, task: Task]
  taskDragend: []
  drop: [taskId: string, status: TaskStatus]
}>()

const isDragOver = ref(false)

function onDragOver(event: DragEvent) {
  isDragOver.value = true
  event.dataTransfer!.dropEffect = 'move'
}

function onDragLeave() {
  isDragOver.value = false
}

function onDrop(event: DragEvent) {
  isDragOver.value = false
  const taskId = event.dataTransfer!.getData('text/plain')
  if (taskId) {
    emit('drop', taskId, props.status)
  }
}
</script>

<style scoped>
.kanban-column {
  flex: 1;
  min-width: 240px;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  background: var(--card-bg);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  overflow: hidden;
}

.kanban-column.drag-over {
  background: var(--hover-bg);
  border-color: #a78bfa;
}

.kanban-column-header {
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
}

.column-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.column-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.column-count {
  font-size: 10px;
  background: var(--border-color);
  color: var(--text-secondary);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
}

.kanban-column-list {
  flex: 1;
  padding: var(--spacing-3);
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
  min-height: 60px;
}

.kanban-column-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-4);
  font-size: 12px;
  color: var(--text-secondary);
  border: 1px dashed var(--border-color);
  border-radius: var(--radius-md);
}
</style>
```

- [ ] **步骤 2：创建 KanbanView.vue**

```vue
<template>
  <div class="kanban-view">
    <KanbanColumn
      title="待办"
      status="todo"
      :tasks="todoTasks"
      dot-color="#fbbf24"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="进行中"
      status="in_progress"
      :tasks="inProgressTasks"
      dot-color="#a78bfa"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
    <KanbanColumn
      title="已完成"
      status="completed"
      :tasks="completedTasks"
      dot-color="#34d399"
      @task-click="emit('taskClick', $event)"
      @task-contextmenu="emit('taskContextmenu', $event)"
      @task-dragstart="emit('taskDragstart', $event)"
      @task-dragend="emit('taskDragend')"
      @drop="onColumnDrop"
    />
  </div>
</template>

<script setup lang="ts">
import type { Task, TaskStatus } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import KanbanColumn from './KanbanColumn.vue'

const store = useTaskStore()

const todoTasks = store.todoTasks
const inProgressTasks = store.inProgressTasks
const completedTasks = store.completedTasks

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
  taskDragstart: [event: DragEvent, task: Task]
  taskDragend: []
}>()

async function onColumnDrop(taskId: string, status: TaskStatus) {
  await store.changeStatus(taskId, status)
}
</script>

<style scoped>
.kanban-view {
  flex: 1;
  display: flex;
  gap: var(--spacing-4);
  overflow-x: auto;
  padding-bottom: var(--spacing-4);
}
</style>
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/apps/task/views/KanbanColumn.vue src/components/apps/task/views/KanbanView.vue
git commit -m "feat(task): add KanbanView and KanbanColumn with drag-drop support"
```

---

### 任务 6：创建 ListView

**文件：**
- 创建：`src/components/apps/task/components/TaskRow.vue`
- 创建：`src/components/apps/task/views/ListView.vue`

- [ ] **步骤 1：创建 TaskRow.vue**

```vue
<template>
  <div
    class="task-row"
    :class="{ 'is-completed': task.status === 'completed', 'is-selected': isSelected }"
    @click="$emit('click', task)"
    @contextmenu.prevent="$emit('contextmenu', $event, task)"
  >
    <button class="row-checkbox" @click.stop="$emit('toggleComplete', task)">
      <i v-if="task.status === 'completed'" class="fas fa-check-circle" style="color: #34d399;"></i>
      <i v-else class="far fa-circle" style="color: var(--border-color);"></i>
    </button>
    <span class="row-title">{{ task.title }}</span>
    <span v-if="task.tags.length" class="row-tags">
      <span
        v-for="tag in task.tags.slice(0, 2)"
        :key="tag.id"
        class="task-tag"
        :style="{ background: tag.color + '20', color: tag.color }"
      >{{ tag.name }}</span>
    </span>
    <span class="row-priority" :class="'priority-' + task.priority">
      {{ priorityLabel }}
    </span>
    <span v-if="task.due_date" class="row-due">{{ dueDateLabel }}</span>
    <div v-if="task.assignee" class="row-assignee">
      <div class="assignee-avatar" :style="{ background: avatarColor }">
        {{ task.assignee.name.charAt(0) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '../../../types/task'

const props = defineProps<{
  task: Task
  isSelected?: boolean
}>()

defineEmits<{
  click: [task: Task]
  contextmenu: [event: MouseEvent, task: Task]
  toggleComplete: [task: Task]
}>()

const priorityLabel = computed(() => {
  const map: Record<string, string> = { high: '高', medium: '中', low: '低' }
  return map[props.task.priority] || props.task.priority
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
  return `${date.getMonth() + 1}/${date.getDate()}`
})

const avatarColor = computed(() => {
  const colors = ['#8b5cf6', '#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']
  const index = props.task.assignee ? props.task.assignee.name.charCodeAt(0) % colors.length : 0
  return colors[index]
})
</script>

<style scoped>
.task-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background var(--animation-base) ease;
}

.task-row:hover {
  background: var(--hover-bg);
}

.task-row.is-selected {
  background: var(--active-bg);
}

.task-row.is-completed .row-title {
  text-decoration: line-through;
  color: var(--text-secondary);
}

.row-checkbox {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-size: 14px;
  display: flex;
  align-items: center;
}

.row-title {
  flex: 1;
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.row-tags {
  display: flex;
  gap: 4px;
}

.task-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.row-priority {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-weight: 500;
}

.row-priority.priority-high {
  background: #fef2f2;
  color: #ef4444;
}

.row-priority.priority-medium {
  background: #fffbeb;
  color: #d97706;
}

.row-priority.priority-low {
  background: #eff6ff;
  color: #3b82f6;
}

.row-due {
  font-size: 11px;
  color: var(--text-secondary);
  flex-shrink: 0;
}

.row-assignee {
  flex-shrink: 0;
}

.assignee-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: #fff;
  font-weight: 500;
}
</style>
```

- [ ] **步骤 2：创建 ListView.vue**

```vue
<template>
  <div class="list-view">
    <div class="list-header">
      <div class="list-header-cell checkbox-cell"></div>
      <div class="list-header-cell title-cell" @click="toggleSort('title')">
        任务名称
        <i v-if="sortBy === 'title'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell priority-cell" @click="toggleSort('priority')">
        优先级
        <i v-if="sortBy === 'priority'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell due-cell" @click="toggleSort('due_date')">
        截止日期
        <i v-if="sortBy === 'due_date'" :class="sortDir === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'"></i>
      </div>
      <div class="list-header-cell assignee-cell">指派人</div>
    </div>
    <div class="list-body">
      <TaskRow
        v-for="task in sortedTasks"
        :key="task.id"
        :task="task"
        :is-selected="task.id === selectedTaskId"
        @click="emit('taskClick', $event)"
        @contextmenu="emit('taskContextmenu', $event)"
        @toggle-complete="onToggleComplete"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Task } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
import TaskRow from '../components/TaskRow.vue'

const store = useTaskStore()
const selectedTaskId = computed(() => store.selectedTaskId)

const sortBy = ref<'title' | 'priority' | 'due_date'>('due_date')
const sortDir = ref<'asc' | 'desc'>('asc')

const priorityOrder: Record<string, number> = { high: 0, medium: 1, low: 2 }

const sortedTasks = computed(() => {
  const tasks = [...store.filteredTasks]
  tasks.sort((a, b) => {
    let cmp = 0
    if (sortBy.value === 'title') {
      cmp = a.title.localeCompare(b.title)
    } else if (sortBy.value === 'priority') {
      cmp = (priorityOrder[a.priority] ?? 1) - (priorityOrder[b.priority] ?? 1)
    } else if (sortBy.value === 'due_date') {
      const aDate = a.due_date ? new Date(a.due_date).getTime() : Infinity
      const bDate = b.due_date ? new Date(b.due_date).getTime() : Infinity
      cmp = aDate - bDate
    }
    return sortDir.value === 'asc' ? cmp : -cmp
  })
  return tasks
})

function toggleSort(field: 'title' | 'priority' | 'due_date') {
  if (sortBy.value === field) {
    sortDir.value = sortDir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = field
    sortDir.value = 'asc'
  }
}

async function onToggleComplete(task: Task) {
  const newStatus = task.status === 'completed' ? 'todo' : 'completed'
  await store.changeStatus(task.id, newStatus)
}

const emit = defineEmits<{
  taskClick: [task: Task]
  taskContextmenu: [event: MouseEvent, task: Task]
}>()
</script>

<style scoped>
.list-view {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.list-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 2px solid var(--border-color);
  background: var(--card-bg);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.list-header-cell {
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  user-select: none;
}

.checkbox-cell {
  width: 24px;
}

.title-cell {
  flex: 1;
}

.priority-cell {
  width: 60px;
}

.due-cell {
  width: 80px;
}

.assignee-cell {
  width: 32px;
}

.list-body {
  flex: 1;
  overflow-y: auto;
  background: var(--card-bg);
}
</style>
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/apps/task/components/TaskRow.vue src/components/apps/task/views/ListView.vue
git commit -m "feat(task): add ListView with sortable columns and TaskRow"
```

---

### 任务 7：创建 CalendarView 和 MyWorkspace

**文件：**
- 创建：`src/components/apps/task/views/CalendarView.vue`
- 创建：`src/components/apps/task/views/MyWorkspace.vue`

- [ ] **步骤 1：创建 CalendarView.vue**

```vue
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
  const lastDay = new Date(year, month + 1, 0)
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
    cells.push({
      key: dateStr,
      day: date.getDate(),
      isCurrentMonth,
      isToday,
      tasks
    })
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

.cal-nav-btn:hover {
  background: var(--hover-bg);
}

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

.cal-cell.is-today {
  background: #faf5ff;
}

.cal-cell.is-other-month {
  opacity: 0.4;
}

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

.cal-task.priority-high {
  background: #fef2f2;
  color: #ef4444;
  border-left-color: #ef4444;
}

.cal-task.priority-medium {
  background: #fffbeb;
  color: #d97706;
  border-left-color: #d97706;
}

.cal-task.priority-low {
  background: #eff6ff;
  color: #3b82f6;
  border-left-color: #3b82f6;
}

.cal-more {
  font-size: 9px;
  color: var(--text-secondary);
  padding: 1px 4px;
}
</style>
```

- [ ] **步骤 2：创建 MyWorkspace.vue**

```vue
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
        <div v-if="!todayTasks.length" class="workspace-empty">今天没有待办任务 🎉</div>
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
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/apps/task/views/CalendarView.vue src/components/apps/task/views/MyWorkspace.vue
git commit -m "feat(task): add CalendarView and MyWorkspace views"
```

---

### 任务 8：创建 TaskSidebar

**文件：**
- 创建：`src/components/apps/task/TaskSidebar.vue`

- [ ] **步骤 1：创建 TaskSidebar.vue**

```vue
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
  store.setFilters({
    priority: store.filters.priority === priority ? null : priority
  })
}

function toggleDueSoonFilter() {
  if (store.filters.due_date_range) {
    store.setFilters({ due_date_range: null })
  } else {
    const today = new Date()
    const nextWeek = new Date(today.getTime() + 7 * 24 * 60 * 60 * 1000)
    store.setFilters({
      due_date_range: {
        start: today.toISOString().split('T')[0],
        end: nextWeek.toISOString().split('T')[0]
      }
    })
  }
}

function toggleMyTasksFilter() {
  store.setFilters({
    assignee_id: store.filters.assignee_id ? null : 'me'
  })
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

.sidebar-section {
  margin-bottom: var(--spacing-4);
}

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

.sidebar-item:hover {
  background: var(--hover-bg);
  color: var(--text-primary);
}

.sidebar-item.active {
  background: var(--active-bg);
  color: var(--text-primary);
  font-weight: 500;
}

.filter-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

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

.progress-percent {
  color: #059669;
  font-weight: 500;
}

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
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/TaskSidebar.vue
git commit -m "feat(task): add TaskSidebar with view switcher, filters, progress"
```

---

### 任务 9：创建 TaskToolbar

**文件：**
- 创建：`src/components/apps/task/TaskToolbar.vue`

- [ ] **步骤 1：创建 TaskToolbar.vue**

```vue
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
import type { TaskView } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'

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

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.toolbar-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.toolbar-count {
  font-size: 11px;
  color: var(--text-secondary);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.toolbar-search {
  position: relative;
}

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

.search-input:focus {
  outline: none;
  border-color: #8b5cf6;
}

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

.create-btn:hover {
  background: #7c3aed;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/TaskToolbar.vue
git commit -m "feat(task): add TaskToolbar with search and create button"
```

---

### 任务 10：创建 TaskCreateModal

**文件：**
- 创建：`src/components/apps/task/components/TaskCreateModal.vue`

- [ ] **步骤 1：创建 TaskCreateModal.vue**

```vue
<template>
  <ModalContainer
    :visible="visible"
    :title="task ? '编辑任务' : '创建任务'"
    @close="$emit('close')"
  >
    <div class="task-form-group">
      <label>任务标题</label>
      <input type="text" class="task-form-input" v-model="form.title" placeholder="请输入任务标题">
    </div>
    <div class="task-form-group">
      <label>任务描述</label>
      <textarea class="task-form-textarea" v-model="form.description" placeholder="请输入任务描述"></textarea>
    </div>
    <div class="task-form-row">
      <div class="task-form-group">
        <label>截止日期</label>
        <input type="date" class="task-form-input" v-model="form.due_date">
      </div>
      <div class="task-form-group">
        <label>优先级</label>
        <select class="task-form-select" v-model="form.priority">
          <option value="low">低</option>
          <option value="medium">中</option>
          <option value="high">高</option>
        </select>
      </div>
    </div>
    <div class="task-form-group">
      <label>状态</label>
      <select class="task-form-select" v-model="form.status">
        <option value="todo">待办</option>
        <option value="in_progress">进行中</option>
        <option value="completed">已完成</option>
      </select>
    </div>
    <template #footer>
      <button class="task-modal-btn task-cancel-btn" @click="$emit('close')">取消</button>
      <button class="task-modal-btn task-confirm-btn" @click="onSubmit">
        {{ task ? '更新' : '创建' }}
      </button>
    </template>
  </ModalContainer>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { Task, TaskPriority, TaskStatus } from '../../../types/task'
import ModalContainer from '../../modals/ModalContainer.vue'

const props = defineProps<{
  visible: boolean
  task?: Task | null
}>()

const emit = defineEmits<{
  close: []
  submit: [data: {
    title: string
    description: string
    due_date: string | null
    priority: TaskPriority
    status: TaskStatus
  }]
}>()

const form = reactive({
  title: '',
  description: '',
  due_date: '' as string | null,
  priority: 'medium' as TaskPriority,
  status: 'todo' as TaskStatus
})

watch(() => props.visible, (val) => {
  if (val) {
    if (props.task) {
      form.title = props.task.title
      form.description = props.task.description
      form.due_date = props.task.due_date
      form.priority = props.task.priority
      form.status = props.task.status
    } else {
      form.title = ''
      form.description = ''
      form.due_date = null
      form.priority = 'medium'
      form.status = 'todo'
    }
  }
})

function onSubmit() {
  if (!form.title.trim()) return
  emit('submit', {
    title: form.title.trim(),
    description: form.description.trim(),
    due_date: form.due_date || null,
    priority: form.priority,
    status: form.status
  })
}
</script>

<style scoped>
.task-form-group {
  margin-bottom: var(--spacing-3);
}

.task-form-group label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}

.task-form-input,
.task-form-textarea,
.task-form-select {
  width: 100%;
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--text-primary);
  background: var(--input-bg);
  transition: border-color var(--animation-fast) ease;
  box-sizing: border-box;
}

.task-form-input:focus,
.task-form-textarea:focus,
.task-form-select:focus {
  outline: none;
  border-color: #8b5cf6;
}

.task-form-textarea {
  resize: vertical;
  min-height: 80px;
}

.task-form-row {
  display: flex;
  gap: var(--spacing-3);
}

.task-form-row .task-form-group {
  flex: 1;
}

.task-modal-btn {
  padding: var(--spacing-2) var(--spacing-4);
  border: none;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--animation-fast) ease;
}

.task-cancel-btn {
  background: var(--hover-bg);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.task-cancel-btn:hover {
  background: var(--border-color);
}

.task-confirm-btn {
  background: #8b5cf6;
  color: #fff;
}

.task-confirm-btn:hover {
  background: #7c3aed;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/TaskCreateModal.vue
git commit -m "feat(task): add TaskCreateModal with form validation"
```

---

### 任务 11：创建主容器 TaskManagementApp（重构后）

**文件：**
- 创建：`src/components/apps/task/TaskManagementApp.vue`
- 创建：`src/components/apps/task/index.ts`

- [ ] **步骤 1：创建 TaskManagementApp.vue**

```vue
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
    </div>

    <TaskCreateModal
      :visible="showCreateModal"
      :task="selectedTask"
      @close="showCreateModal = false"
      @submit="onSubmitTask"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Task } from '../../types/task'
import { useTaskStore } from '../../stores/task'
import AppHeader from '../AppHeader.vue'
import TaskSidebar from './TaskSidebar.vue'
import TaskToolbar from './TaskToolbar.vue'
import KanbanView from './views/KanbanView.vue'
import ListView from './views/ListView.vue'
import CalendarView from './views/CalendarView.vue'
import MyWorkspace from './views/MyWorkspace.vue'
import TaskCreateModal from './components/TaskCreateModal.vue'
import QMessage from '../../utils/qmessage'

const store = useTaskStore()
const showCreateModal = ref(false)
const selectedTask = computed(() => store.selectedTask)

defineEmits<{
  back: []
  toggleSidebar: []
}>()

onMounted(() => {
  store.loadTasks()
})

function onTaskClick(task: Task) {
  store.selectTask(task.id)
  showCreateModal.value = true
}

function onTaskContextmenu(event: MouseEvent, task: Task) {
  store.selectTask(task.id)
}

async function onSubmitTask(data: {
  title: string
  description: string
  due_date: string | null
  priority: string
  status: string
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
</style>
```

- [ ] **步骤 2：创建 index.ts**

```typescript
export { default as TaskManagementApp } from './TaskManagementApp.vue'
```

- [ ] **步骤 3：更新 Main.vue 的导入路径**

在 `src/views/Main.vue` 中，将旧的导入：

```typescript
import TaskManagementApp from '../components/apps/TaskManagementApp.vue'
```

替换为：

```typescript
import TaskManagementApp from '../components/apps/task/TaskManagementApp.vue'
```

- [ ] **步骤 4：删除旧的 TaskManagementApp.vue**

删除 `src/components/apps/TaskManagementApp.vue`

- [ ] **步骤 5：验证应用正常运行**

运行开发服务器，进入任务管理应用，确认：
- 看板视图正常显示三列
- 可以创建和编辑任务
- 搜索功能正常
- 视图切换正常（看板/列表/日历/工作台）

- [ ] **步骤 6：Commit**

```bash
git add -A
git commit -m "feat(task): refactor TaskManagementApp into modular component structure"
```

---

## 阶段二：交互增强

### 任务 12：创建 useTaskDragDrop composable

**文件：**
- 创建：`src/composables/useTaskDragDrop.ts`

- [ ] **步骤 1：创建 useTaskDragDrop.ts**

```typescript
import { ref } from 'vue'
import type { TaskStatus } from '../types/task'
import { useTaskStore } from '../stores/task'

export function useTaskDragDrop() {
  const store = useTaskStore()
  const draggedTaskId = ref<string | null>(null)
  const dropTargetStatus = ref<TaskStatus | null>(null)

  function onDragStart(taskId: string) {
    draggedTaskId.value = taskId
  }

  function onDragEnd() {
    draggedTaskId.value = null
    dropTargetStatus.value = null
  }

  function onDragOver(status: TaskStatus) {
    dropTargetStatus.value = status
  }

  function onDragLeave() {
    dropTargetStatus.value = null
  }

  async function onDrop(status: TaskStatus) {
    if (!draggedTaskId.value) return
    const taskId = draggedTaskId.value
    draggedTaskId.value = null
    dropTargetStatus.value = null
    await store.changeStatus(taskId, status)
  }

  return {
    draggedTaskId,
    dropTargetStatus,
    onDragStart,
    onDragEnd,
    onDragOver,
    onDragLeave,
    onDrop
  }
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/composables/useTaskDragDrop.ts
git commit -m "feat(task): add useTaskDragDrop composable"
```

---

### 任务 13：实现右键菜单

**文件：**
- 修改：`src/components/apps/task/TaskManagementApp.vue`

- [ ] **步骤 1：在 TaskManagementApp.vue 中添加右键菜单**

在模板中添加右键菜单组件：

```vue
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
```

在 script 中添加右键菜单逻辑：

```typescript
import { reactive } from 'vue'

const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  taskId: '' as string
})

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

onMounted(() => document.addEventListener('click', onGlobalClick))
```

添加右键菜单样式：

```css
.context-menu {
  position: fixed;
  z-index: var(--z-dropdown);
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

.context-item:hover {
  background: var(--hover-bg);
}

.context-item.danger {
  color: #ef4444;
}

.context-item.danger:hover {
  background: #fef2f2;
}

.context-divider {
  height: 1px;
  background: var(--border-color);
  margin: var(--spacing-1) 0;
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/TaskManagementApp.vue
git commit -m "feat(task): add context menu for task cards"
```

---

### 任务 14：实现快捷键

**文件：**
- 修改：`src/components/apps/task/TaskManagementApp.vue`

- [ ] **步骤 1：在 TaskManagementApp.vue 的 script 中添加快捷键逻辑**

```typescript
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

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/TaskManagementApp.vue
git commit -m "feat(task): add keyboard shortcuts (N, 1-4, Delete)"
```

---

## 阶段三：任务组织与协作联动

### 任务 15：实现 TagSelector 组件

**文件：**
- 创建：`src/components/apps/task/components/TagSelector.vue`

- [ ] **步骤 1：创建 TagSelector.vue**

```vue
<template>
  <div class="tag-selector">
    <label class="tag-label">标签</label>
    <div class="tag-list">
      <span
        v-for="tag in availableTags"
        :key="tag.id"
        class="tag-item"
        :class="{ selected: isSelected(tag.id) }"
        :style="isSelected(tag.id) ? { background: tag.color + '20', color: tag.color, borderColor: tag.color } : {}"
        @click="toggleTag(tag)"
      >
        {{ tag.name }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Tag } from '../../../types/task'

const props = defineProps<{
  availableTags: Tag[]
  selectedTagIds: string[]
}>()

const emit = defineEmits<{
  update: [tagIds: string[]]
}>()

function isSelected(id: string) {
  return props.selectedTagIds.includes(id)
}

function toggleTag(tag: Tag) {
  const ids = [...props.selectedTagIds]
  const index = ids.indexOf(tag.id)
  if (index === -1) {
    ids.push(tag.id)
  } else {
    ids.splice(index, 1)
  }
  emit('update', ids)
}
</script>

<style scoped>
.tag-selector {
  margin-bottom: var(--spacing-3);
}

.tag-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-1);
}

.tag-item {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-color);
  cursor: pointer;
  color: var(--text-secondary);
  transition: all var(--animation-fast) ease;
}

.tag-item:hover {
  border-color: var(--text-secondary);
}

.tag-item.selected {
  font-weight: 500;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/TagSelector.vue
git commit -m "feat(task): add TagSelector component"
```

---

### 任务 16：实现 AssigneeSelector 组件

**文件：**
- 创建：`src/components/apps/task/components/AssigneeSelector.vue`

- [ ] **步骤 1：创建 AssigneeSelector.vue**

```vue
<template>
  <div class="assignee-selector">
    <label class="assignee-label">指派人</label>
    <div class="assignee-dropdown" v-if="showDropdown">
      <input
        class="assignee-search"
        v-model="searchQuery"
        placeholder="搜索联系人..."
        ref="searchInput"
      />
      <div class="assignee-options">
        <button
          class="assignee-option"
          @click="selectAssignee(null)"
        >
          <span class="no-assignee">未指派</span>
        </button>
        <button
          v-for="user in filteredUsers"
          :key="user.id"
          class="assignee-option"
          :class="{ selected: user.id === modelValue }"
          @click="selectAssignee(user.id)"
        >
          <div class="user-avatar" :style="{ background: getAvatarColor(user.name) }">
            {{ user.name.charAt(0) }}
          </div>
          <span>{{ user.name }}</span>
        </button>
      </div>
    </div>
    <button class="assignee-trigger" @click="showDropdown = !showDropdown">
      <template v-if="currentAssignee">
        <div class="user-avatar" :style="{ background: getAvatarColor(currentAssignee.name) }">
          {{ currentAssignee.name.charAt(0) }}
        </div>
        <span>{{ currentAssignee.name }}</span>
      </template>
      <template v-else>
        <i class="fas fa-user-plus"></i>
        <span>指派</span>
      </template>
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import type { TaskUser } from '../../../types/task'

const props = defineProps<{
  modelValue: string | null
  contacts: TaskUser[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
}>()

const showDropdown = ref(false)
const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)

const currentAssignee = computed(() =>
  props.contacts.find(u => u.id === props.modelValue) || null
)

const filteredUsers = computed(() => {
  if (!searchQuery.value) return props.contacts
  const q = searchQuery.value.toLowerCase()
  return props.contacts.filter(u => u.name.toLowerCase().includes(q))
})

function selectAssignee(id: string | null) {
  emit('update:modelValue', id)
  showDropdown.value = false
  searchQuery.value = ''
}

function getAvatarColor(name: string) {
  const colors = ['#8b5cf6', '#6366f1', '#ec4899', '#f59e0b', '#10b981', '#3b82f6']
  const index = name.charCodeAt(0) % colors.length
  return colors[index]
}
</script>

<style scoped>
.assignee-selector {
  margin-bottom: var(--spacing-3);
  position: relative;
}

.assignee-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}

.assignee-trigger {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--input-bg);
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
}

.assignee-trigger:hover {
  border-color: #8b5cf6;
}

.assignee-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: var(--z-dropdown);
  background: var(--card-bg);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--spacing-2);
  min-width: 200px;
  margin-top: var(--spacing-1);
}

.assignee-search {
  width: 100%;
  padding: var(--spacing-1) var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin-bottom: var(--spacing-2);
  color: var(--text-primary);
  background: var(--input-bg);
}

.assignee-search:focus {
  outline: none;
  border-color: #8b5cf6;
}

.assignee-options {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 200px;
  overflow-y: auto;
}

.assignee-option {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-1) var(--spacing-2);
  border: none;
  background: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  color: var(--text-primary);
  text-align: left;
}

.assignee-option:hover {
  background: var(--hover-bg);
}

.assignee-option.selected {
  background: #f5f3ff;
}

.no-assignee {
  color: var(--text-secondary);
  font-style: italic;
}

.user-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  color: #fff;
  font-weight: 500;
  flex-shrink: 0;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/AssigneeSelector.vue
git commit -m "feat(task): add AssigneeSelector component"
```

---

### 任务 17：实现 SubTaskList 组件

**文件：**
- 创建：`src/components/apps/task/components/SubTaskList.vue`

- [ ] **步骤 1：创建 SubTaskList.vue**

```vue
<template>
  <div class="subtask-list">
    <label class="subtask-label">子任务 ({{ completedCount }}/{{ subTasks.length }})</label>
    <div class="subtask-items">
      <div v-for="st in subTasks" :key="st.id" class="subtask-item">
        <button class="subtask-check" @click="$emit('toggle', st.id)">
          <i v-if="st.completed" class="fas fa-check-square" style="color:#34d399;"></i>
          <i v-else class="far fa-square" style="color:var(--border-color);"></i>
        </button>
        <span class="subtask-title" :class="{ completed: st.completed }">{{ st.title }}</span>
        <button class="subtask-delete" @click="$emit('remove', st.id)">
          <i class="fas fa-times"></i>
        </button>
      </div>
    </div>
    <div class="subtask-add">
      <input
        v-model="newSubTask"
        class="subtask-input"
        placeholder="添加子任务..."
        @keydown.enter="addSubTask"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { SubTask } from '../../../types/task'

const props = defineProps<{
  subTasks: SubTask[]
}>()

const emit = defineEmits<{
  toggle: [id: string]
  remove: [id: string]
  add: [title: string]
}>()

const newSubTask = ref('')

const completedCount = computed(() =>
  props.subTasks.filter(st => st.completed).length
)

function addSubTask() {
  const title = newSubTask.value.trim()
  if (!title) return
  emit('add', title)
  newSubTask.value = ''
}
</script>

<style scoped>
.subtask-list {
  margin-bottom: var(--spacing-3);
}

.subtask-label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: var(--spacing-1);
}

.subtask-items {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-1);
}

.subtask-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-1) 0;
}

.subtask-check {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-size: 14px;
}

.subtask-title {
  flex: 1;
  font-size: 12px;
  color: var(--text-primary);
}

.subtask-title.completed {
  text-decoration: line-through;
  color: var(--text-secondary);
}

.subtask-delete {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  font-size: 10px;
  color: var(--text-secondary);
  opacity: 0;
  transition: opacity var(--animation-fast) ease;
}

.subtask-item:hover .subtask-delete {
  opacity: 1;
}

.subtask-delete:hover {
  color: #ef4444;
}

.subtask-add {
  margin-top: var(--spacing-2);
}

.subtask-input {
  width: 100%;
  padding: var(--spacing-1) var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--text-primary);
  background: var(--input-bg);
}

.subtask-input:focus {
  outline: none;
  border-color: #8b5cf6;
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/SubTaskList.vue
git commit -m "feat(task): add SubTaskList component"
```

---

### 任务 18：实现 TaskDetailPanel 侧滑面板

**文件：**
- 创建：`src/components/apps/task/components/TaskDetailPanel.vue`

- [ ] **步骤 1：创建 TaskDetailPanel.vue**

```vue
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
import type { Task, Tag, TaskUser, TaskStatus, TaskPriority } from '../../../types/task'
import { useTaskStore } from '../../../stores/task'
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
  await store.updateTask(props.task.id, { sub_tasks: updated as any })
}

async function onSubTaskRemove(subTaskId: string) {
  if (!props.task) return
  const updated = props.task.sub_tasks.filter(st => st.id !== subTaskId)
  await store.updateTask(props.task.id, { sub_tasks: updated as any })
}

async function onSubTaskAdd(title: string) {
  if (!props.task) return
  const newSubTask = {
    id: `st-${Date.now()}`,
    title,
    completed: false,
    position: props.task.sub_tasks.length
  }
  await store.updateTask(props.task.id, { sub_tasks: [...props.task.sub_tasks, newSubTask] as any })
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

.detail-close:hover {
  background: var(--hover-bg);
}

.detail-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-4);
}

.detail-field {
  margin-bottom: var(--spacing-3);
}

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

.detail-field-row {
  display: flex;
  gap: var(--spacing-3);
}

.detail-field-row .detail-field {
  flex: 1;
}

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
.detail-input:focus {
  outline: none;
  border-color: #8b5cf6;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform var(--animation-base) ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(100%);
}
</style>
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/components/TaskDetailPanel.vue
git commit -m "feat(task): add TaskDetailPanel with tags, assignee, subtasks"
```

---

### 任务 19：集成 TaskDetailPanel 到主容器

**文件：**
- 修改：`src/components/apps/task/TaskManagementApp.vue`

- [ ] **步骤 1：在 TaskManagementApp.vue 中集成 TaskDetailPanel**

在模板中 `TaskCreateModal` 之后添加：

```vue
<TaskDetailPanel
  :task="store.selectedTask"
  :available-tags="availableTags"
  :contacts="contacts"
  @close="store.selectTask(null)"
/>
```

在 script 中添加导入和数据：

```typescript
import TaskDetailPanel from './components/TaskDetailPanel.vue'
import type { Tag, TaskUser } from '../../types/task'

const availableTags = ref<Tag[]>([
  { id: '1', name: '设计', color: '#ec4899' },
  { id: '2', name: '后端', color: '#6366f1' },
  { id: '3', name: '前端', color: '#3b82f6' },
  { id: '4', name: '重构', color: '#8b5cf6' },
  { id: '5', name: '文档', color: '#10b981' },
  { id: '6', name: 'Bug', color: '#ef4444' }
])

const contacts = ref<TaskUser[]>([])
```

修改 `onTaskClick` 方法，改为打开详情面板而非编辑弹窗：

```typescript
function onTaskClick(task: Task) {
  store.selectTask(task.id)
}
```

- [ ] **步骤 2：Commit**

```bash
git add src/components/apps/task/TaskManagementApp.vue
git commit -m "feat(task): integrate TaskDetailPanel into main container"
```

---

### 任务 20：从消息创建任务（聊天联动）

**文件：**
- 修改：`src/components/chat/ChatWindow.vue`（或消息右键菜单相关组件）

- [ ] **步骤 1：在消息右键菜单中添加"创建为任务"选项**

找到消息右键菜单的定义位置，添加一个菜单项：

```typescript
{
  label: '创建为任务',
  icon: 'fas fa-check-square',
  action: 'create-task'
}
```

- [ ] **步骤 2：实现创建任务的处理逻辑**

在右键菜单的 action handler 中添加：

```typescript
case 'create-task': {
  const taskStore = useTaskStore()
  const messageText = selectedMessage.value?.content || ''
  await taskStore.createTask({
    title: messageText.slice(0, 50) + (messageText.length > 50 ? '...' : ''),
    description: messageText,
    priority: 'medium',
    status: 'todo'
  })
  QMessage.success('已创建为任务')
  break
}
```

- [ ] **步骤 3：Commit**

```bash
git add src/components/chat/ChatWindow.vue
git commit -m "feat(task): add 'create as task' option in message context menu"
```

---

### 任务 21：最终验证与清理

**文件：**
- 检查所有新建文件

- [ ] **步骤 1：运行 TypeScript 类型检查**

```bash
cd qim-client && npx vue-tsc --noEmit
```

修复所有类型错误。

- [ ] **步骤 2：运行 lint 检查**

```bash
cd qim-client && npm run lint
```

修复所有 lint 错误。

- [ ] **步骤 3：手动验证所有功能**

- 看板视图：三列显示、拖拽切换状态
- 列表视图：排序、点击切换完成
- 日历视图：按日期展示任务
- 我的工作台：今日待办、进行中、已指派
- 侧边栏：视图切换、筛选、进度
- 搜索功能
- 创建/编辑任务
- 右键菜单
- 快捷键（N、1-4、Delete）
- 任务详情面板（标签、指派人、子任务）
- 从消息创建任务

- [ ] **步骤 4：最终 Commit**

```bash
git add -A
git commit -m "feat(task): complete task management module redesign"
```
