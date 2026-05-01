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
