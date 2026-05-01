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
