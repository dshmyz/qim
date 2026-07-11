import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export type DownloadStatus = 'pending' | 'downloading' | 'completed' | 'failed' | 'cancelled'

export interface DownloadTask {
  downloadId: string
  fileName: string
  size: number
  status: DownloadStatus
  percent: number
  receivedBytes: number
  totalBytes: number
  error?: string
  filePath?: string
}

export const useDownloadStore = defineStore('download', () => {
  const tasks = ref<Map<string, DownloadTask>>(new Map())
  const isExpanded = ref(true)

  const activeTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(
      task => task.status === 'pending' || task.status === 'downloading'
    )
  })

  const completedTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(task => task.status === 'completed')
  })

  const failedTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(task => task.status === 'failed')
  })

  const totalProgress = computed(() => {
    const allTasks = Array.from(tasks.value.values())
    if (allTasks.length === 0) return 0

    const totalSize = allTasks.reduce((sum, task) => sum + task.totalBytes, 0)
    if (totalSize === 0) return 0

    const receivedSize = allTasks.reduce((sum, task) => sum + task.receivedBytes, 0)
    return Math.round((receivedSize / totalSize) * 100)
  })

  function addTask({ fileName, size }: { fileName: string; size: number }): string {
    const downloadId = generateDownloadId()
    const task: DownloadTask = {
      downloadId,
      fileName,
      size,
      status: 'pending',
      percent: 0,
      receivedBytes: 0,
      totalBytes: size
    }
    tasks.value.set(downloadId, task)
    return downloadId
  }

  function updateProgress(downloadId: string, percent: number, receivedBytes: number, totalBytes: number) {
    const task = tasks.value.get(downloadId)
    if (task) {
      tasks.value.set(downloadId, {
        ...task,
        status: task.status === 'pending' ? 'downloading' : task.status,
        percent,
        receivedBytes,
        totalBytes: totalBytes || task.totalBytes
      })
    }
  }

  function markCompleted(downloadId: string, filePath: string) {
    const task = tasks.value.get(downloadId)
    if (task) {
      tasks.value.set(downloadId, {
        ...task,
        status: 'completed',
        percent: 100,
        receivedBytes: task.totalBytes || task.receivedBytes,
        filePath
      })
    }
  }

  function markFailed(downloadId: string, error: string) {
    const task = tasks.value.get(downloadId)
    if (task) {
      tasks.value.set(downloadId, {
        ...task,
        status: 'failed',
        error
      })
    }
  }

  function removeTask(downloadId: string) {
    tasks.value.delete(downloadId)
  }

  function clearCompleted() {
    const ids = Array.from(tasks.value.entries())
      .filter(([, task]) => task.status === 'completed' || task.status === 'failed' || task.status === 'cancelled')
      .map(([id]) => id)

    ids.forEach(id => tasks.value.delete(id))
  }

  function toggleExpanded() {
    isExpanded.value = !isExpanded.value
  }

  function generateDownloadId(): string {
    return `download_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`
  }

  return {
    tasks,
    isExpanded,
    activeTasks,
    completedTasks,
    failedTasks,
    totalProgress,
    addTask,
    updateProgress,
    markCompleted,
    markFailed,
    removeTask,
    clearCompleted,
    toggleExpanded
  }
})
