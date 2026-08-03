/**
 * /task 斜杠命令定义：触发后搜索会话关联的任务，选中插入 #T-<id>。
 *
 * 数据源沿用 fetchTasksByConversation（仅会话成员可查），带会话级缓存。
 * 候选项渲染用 TaskCommandItem.vue。
 */

import { markRaw } from 'vue'
import TaskCommandItem from '../../components/chat/TaskCommandItem.vue'
import { fetchTasksByConversation } from '../../api/task'
import type { Task } from '../../types/task'
import type { SlashCommand } from '../slashCommand'

/** 会话任务列表缓存：key=conversationId，避免切换会话时重复拉取。 */
const conversationTaskCache = new Map<string, Task[]>()

async function fetchItems({ conversationId }: { conversationId: string }): Promise<Task[]> {
  if (!conversationId) return []
  const cached = conversationTaskCache.get(conversationId)
  if (cached) return cached
  try {
    const tasks = await fetchTasksByConversation(conversationId)
    conversationTaskCache.set(conversationId, tasks)
    return tasks
  } catch {
    return []
  }
}

function filter(items: Task[], query: string): Task[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  return items.filter(t => t.title.toLowerCase().includes(q))
}

function getInsertText(item: Task): string {
  const id = Number(item.id)
  if (!Number.isFinite(id) || id <= 0) return ''
  return `#T-${id} `
}

/** 清除指定会话的任务缓存（会话切换/任务变更时可调用）。 */
export function clearTaskCache(conversationId?: string): void {
  if (conversationId) {
    conversationTaskCache.delete(conversationId)
  } else {
    conversationTaskCache.clear()
  }
}

/** /task 命令：所有会话类型可用（私人任务可在任意会话引用，会话任务按会话隔离）。 */
export const taskCommand: SlashCommand<Task> = {
  trigger: '/task',
  title: '选择任务',
  icon: 'fas fa-tasks',
  description: '引用任务',
  fetchItems,
  filter,
  getInsertText,
  itemComponent: markRaw(TaskCommandItem),
}
