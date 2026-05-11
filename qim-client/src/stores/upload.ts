import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UploadTask {
  uploadId: string
  file: File
  folderId: number | null
  status: 'pending' | 'uploading' | 'completed' | 'failed' | 'cancelled'
  progress: number
  uploadedSize: number
  totalSize: number
  uploadedChunks: number[]
  totalChunks: number
  error?: string
  retryCount: number
  fileId?: number
}

export const useUploadStore = defineStore('upload', () => {
  // 状态
  const tasks = ref<Map<string, UploadTask>>(new Map())
  const isExpanded = ref(true)

  // 计算属性
  const activeTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(
      task => task.status === 'pending' || task.status === 'uploading'
    )
  })

  const completedTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(
      task => task.status === 'completed'
    )
  })

  const failedTasks = computed(() => {
    return Array.from(tasks.value.values()).filter(
      task => task.status === 'failed'
    )
  })

  const totalProgress = computed(() => {
    const allTasks = Array.from(tasks.value.values())
    if (allTasks.length === 0) return 0

    const totalSize = allTasks.reduce((sum, task) => sum + task.totalSize, 0)
    if (totalSize === 0) return 0

    const uploadedSize = allTasks.reduce((sum, task) => sum + task.uploadedSize, 0)
    return Math.round((uploadedSize / totalSize) * 100)
  })

  // 方法
  function addTask(file: File, folderId?: number): string {
    const uploadId = generateUploadId()
    const task: UploadTask = {
      uploadId,
      file,
      folderId: folderId ?? null,
      status: 'pending',
      progress: 0,
      uploadedSize: 0,
      totalSize: file.size,
      uploadedChunks: [],
      totalChunks: 0,
      retryCount: 0
    }
    tasks.value.set(uploadId, task)
    return uploadId
  }

  function updateTask(uploadId: string, updates: Partial<UploadTask>) {
    const task = tasks.value.get(uploadId)
    if (task) {
      tasks.value.set(uploadId, { ...task, ...updates })
    }
  }

  function updateProgress(uploadId: string, progress: number, uploadedSize: number) {
    const task = tasks.value.get(uploadId)
    if (task) {
      tasks.value.set(uploadId, {
        ...task,
        progress,
        uploadedSize
      })
    }
  }

  function updateChunkProgress(uploadId: string, chunkIndex: number) {
    const task = tasks.value.get(uploadId)
    if (task) {
      const uploadedChunks = [...task.uploadedChunks]
      if (!uploadedChunks.includes(chunkIndex)) {
        uploadedChunks.push(chunkIndex)
      }
      tasks.value.set(uploadId, {
        ...task,
        uploadedChunks
      })
    }
  }

  function markCompleted(uploadId: string, fileId: number) {
    const task = tasks.value.get(uploadId)
    if (task) {
      tasks.value.set(uploadId, {
        ...task,
        status: 'completed',
        progress: 100,
        uploadedSize: task.totalSize,
        fileId
      })
    }
  }

  function markFailed(uploadId: string, error: string) {
    const task = tasks.value.get(uploadId)
    if (task) {
      tasks.value.set(uploadId, {
        ...task,
        status: 'failed',
        error
      })
    }
  }

  function cancelTask(uploadId: string) {
    const task = tasks.value.get(uploadId)
    if (task && (task.status === 'pending' || task.status === 'uploading')) {
      tasks.value.set(uploadId, {
        ...task,
        status: 'cancelled'
      })
    }
  }

  function removeTask(uploadId: string) {
    tasks.value.delete(uploadId)
  }

  function clearCompleted() {
    const completedIds = Array.from(tasks.value.entries())
      .filter(([_, task]) => task.status === 'completed' || task.status === 'cancelled')
      .map(([id]) => id)

    completedIds.forEach(id => tasks.value.delete(id))
  }

  function toggleExpanded() {
    isExpanded.value = !isExpanded.value
  }

  // 辅助函数
  function generateUploadId(): string {
    return `upload_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`
  }

  return {
    // 状态
    tasks,
    isExpanded,
    // 计算属性
    activeTasks,
    completedTasks,
    failedTasks,
    totalProgress,
    // 方法
    addTask,
    updateTask,
    updateProgress,
    updateChunkProgress,
    markCompleted,
    markFailed,
    cancelTask,
    removeTask,
    clearCompleted,
    toggleExpanded
  }
})
