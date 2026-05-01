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
      @close="onCloseModal"
      @submit="onSubmitTask"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Task, TaskStatus } from '../../types/task'
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

function onTaskContextmenu(_event: MouseEvent, _task: Task) {
  // 右键菜单将在后续任务中实现
}

function onCloseModal() {
  showCreateModal.value = false
  store.selectTask(null)
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
</style>
