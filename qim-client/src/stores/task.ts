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
