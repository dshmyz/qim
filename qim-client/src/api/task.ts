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
