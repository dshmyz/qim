/**
 * /note 斜杠命令定义：触发后搜索当前用户的笔记，选中后直接发送分享消息。
 *
 * 数据源：/api/v1/notes（仅当前用户的笔记，排除便签），与 useNotes.fetchNotes 对齐。
 * 选中后通过 onSelect 直接构造 share 消息发送，复用现成的 ShareMessage 渲染。
 * 与 /task 的区别：/task 插入 #T-<id> 文本引用，/note 直接发分享消息卡片。
 */

import { markRaw } from 'vue'
import { request } from '../../composables/useRequest'
import type { Note } from '../../types/note'
import NoteCommandItem from '../../components/chat/NoteCommandItem.vue'
import type { SlashCommand, SlashCommandSelectResult } from '../slashCommand'

/** 笔记列表缓存：进程内只拉一次，避免频繁触发。笔记增删改后调用 clearNoteCache 失效。 */
let cachedNotes: Note[] | null = null

/** 解析笔记的 tags 字段（后端可能返回 JSON 字符串或数组）。 */
function parseTags(raw: any): string[] {
  if (Array.isArray(raw)) return raw
  if (typeof raw !== 'string' || !raw) return []
  try {
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

async function fetchItems(): Promise<Note[]> {
  if (cachedNotes) return cachedNotes
  try {
    const response = await request<any>('/api/v1/notes')
    const notes = response?.data || []
    const result: Note[] = notes
      .filter((n: any) => n.type !== 'sticky')
      .map((n: any) => ({ ...n, tags: parseTags(n.tags) }))
    cachedNotes = result
    return result
  } catch {
    return []
  }
}

function filter(items: Note[], query: string): Note[] {
  const q = query.trim().toLowerCase()
  if (!q) return items
  return items.filter(n =>
    (n.title || '').toLowerCase().includes(q) ||
    (n.summary || '').toLowerCase().includes(q) ||
    (n.content || '').toLowerCase().includes(q),
  )
}

/** 选中笔记后构造 share 消息，直接发送。 */
function onSelect(item: Note): SlashCommandSelectResult {
  const shareDataObj = {
    type: 'note',
    id: item.id,
    name: item.title || '无标题笔记',
    content: `分享了笔记: ${item.title || '无标题笔记'}`,
    originalContent: item.content || '',
  }
  return {
    action: 'send',
    messageType: 'share',
    messageContent: JSON.stringify(shareDataObj),
  }
}

/** getInsertText 在 /note 下不会被调用（onSelect 已返回 send），但接口要求必填，返回空字符串兜底。 */
function getInsertText(): string {
  return ''
}

/** 清除笔记缓存（笔记增删改后可调用）。 */
export function clearNoteCache(): void {
  cachedNotes = null
}

/** /note 命令：所有会话类型可用（笔记是用户个人的，可分享到任何会话）。 */
export const noteCommand: SlashCommand<Note> = {
  trigger: '/note',
  title: '选择笔记',
  icon: 'fas fa-file-alt',
  description: '分享我的笔记',
  fetchItems,
  filter,
  getInsertText,
  onSelect,
  itemComponent: markRaw(NoteCommandItem),
}

