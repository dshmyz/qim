import { request, type ApiResponse } from '../composables/useRequest'
import type { Task, Tag, SubTask, TaskUser, CreateTaskData, UpdateTaskData } from '../types/task'

interface RawTask {
  id: number | string
  user_id?: number
  title: string
  description?: string
  due_date?: string | null
  priority?: string
  status?: string
  assignee_id?: string
  tags?: string | Tag[]
  sub_tasks?: string | SubTask[]
  position?: number
  created_at?: string
  updated_at?: string
  assignee?: TaskUser
  creator?: TaskUser
  comment_count?: number
}

function normalizeDate(value: string | null | undefined): string | null {
  if (!value) return null
  return value.split('T')[0]
}

function parseTags(value: string | Tag[] | undefined): Tag[] {
  if (!value) return []
  if (Array.isArray(value)) return value
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function parseSubTasks(value: string | SubTask[] | undefined): SubTask[] {
  if (!value) return []
  if (Array.isArray(value)) return value
  try {
    const parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

function normalizeTask(raw: RawTask): Task {
  const assigneeId = raw.assignee_id ?? ''
  const assignee: Task['assignee'] = assigneeId
    ? { id: assigneeId, name: '', avatar: '' }
    : null
  return {
    id: String(raw.id),
    title: raw.title ?? '',
    description: raw.description ?? '',
    status: (raw.status as Task['status']) ?? 'todo',
    priority: (raw.priority as Task['priority']) ?? 'medium',
    due_date: normalizeDate(raw.due_date),
    tags: parseTags(raw.tags),
    assignee: raw.assignee ?? assignee,
    creator: raw.creator ?? { id: String(raw.user_id ?? ''), name: '', avatar: '' },
    sub_tasks: parseSubTasks(raw.sub_tasks),
    comment_count: raw.comment_count ?? 0,
    position: raw.position ?? 0,
    created_at: raw.created_at ?? '',
    updated_at: raw.updated_at ?? ''
  }
}

function serializeTaskData(data: CreateTaskData | UpdateTaskData): Record<string, any> {
  const body: Record<string, any> = { ...data }
  if (body.tags !== undefined) {
    body.tags = Array.isArray(body.tags) ? JSON.stringify(body.tags) : body.tags
  }
  if (body.sub_tasks !== undefined) {
    body.sub_tasks = Array.isArray(body.sub_tasks) ? JSON.stringify(body.sub_tasks) : body.sub_tasks
  }
  return body
}

export async function fetchTasks(): Promise<Task[]> {
  const response = await request<ApiResponse<RawTask[]>>('/api/v1/tasks')
  return (response.data ?? []).map(normalizeTask)
}

export async function fetchTaskById(id: string): Promise<Task> {
  const response = await request<ApiResponse<RawTask>>(`/api/v1/tasks/${id}`)
  return normalizeTask(response.data)
}

export async function createTask(data: CreateTaskData): Promise<Task> {
  const response = await request<ApiResponse<RawTask>>('/api/v1/tasks', {
    method: 'POST',
    body: JSON.stringify(serializeTaskData(data))
  })
  return normalizeTask(response.data)
}

export async function updateTask(id: string, data: UpdateTaskData): Promise<Task> {
  const response = await request<ApiResponse<RawTask>>(`/api/v1/tasks/${id}`, {
    method: 'PUT',
    body: JSON.stringify(serializeTaskData(data))
  })
  return normalizeTask(response.data)
}

export async function updateTaskStatus(id: string, status: string): Promise<Task> {
  const response = await request<ApiResponse<RawTask>>(`/api/v1/tasks/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status })
  })
  return normalizeTask(response.data)
}

export async function reorderTask(id: string, position: number, status?: string): Promise<void> {
  await request<ApiResponse<void>>(`/api/v1/tasks/${id}/reorder`, {
    method: 'PUT',
    body: JSON.stringify({ position, status })
  })
}

export async function deleteTask(id: string): Promise<void> {
  await request<ApiResponse<void>>(`/api/v1/tasks/${id}`, {
    method: 'DELETE'
  })
}
