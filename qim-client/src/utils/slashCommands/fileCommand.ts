/**
 * /file 斜杠命令定义：触发后搜索当前用户的个人文件，选中后直接发送文件消息。
 *
 * 数据源：/api/v1/files（仅当前用户的个人文件，scope_type='user'）。
 * 选中后通过 onSelect 构造 file 类型消息发送，复用现成的 FileMessage 渲染。
 * 接收方通过 storage_path 经 /static/ 路由下载，与聊天直接上传文件的下载链路一致。
 *
 * 与 /note 的区别：/note 发 share 消息（卡片），/file 发 file 消息（文件卡片+下载）。
 *
 * 搜索方式：后端搜索（backendSearch=true）。每次 query 变化都调用 fetchItems(ctx, query)，
 * 由 ChatWindow 做 debounce。无 query 时返回最新 100 条；有 query 时按 name/original_name 搜索。
 */

import { markRaw } from 'vue'
import { fileApi, type FileItem } from '../../api/file'
import FileCommandItem from '../../components/chat/FileCommandItem.vue'
import type { SlashCommand, SlashCommandContext, SlashCommandSelectResult } from '../slashCommand'

/** 单次拉取上限：文件可能较多，无 query 时取最新 100 条。 */
const FETCH_PAGE_SIZE = 100

/**
 * 后端搜索：query 为空返回最新文件，非空按 name/original_name 模糊匹配。
 * 不做前端缓存——后端搜索场景下每次 query 变化都要重新请求。
 */
async function fetchItems(_ctx: SlashCommandContext, query?: string): Promise<FileItem[]> {
  try {
    const res = await fileApi.getFiles({
      page: 1,
      page_size: FETCH_PAGE_SIZE,
      search: query?.trim() || undefined,
    })
    return res.data?.data?.files || []
  } catch {
    return []
  }
}

/**
 * backendSearch=true 时框架不会调用 filter，直接展示 fetchItems 返回的结果。
 * 这里保留兜底实现（返回原数组），以防被外部直接调用。
 */
function filter(items: FileItem[]): FileItem[] {
  return items
}

/**
 * 选中文件后构造 file 消息，直接发送。
 * content 结构与 ChatWindow.uploadAndSendFile 保持一致：
 *   { url: storage_path, id, name, size, mimeType }
 * 接收方通过 serverUrl + url（即 /static/<storage_path>）下载。
 */
function onSelect(item: FileItem): SlashCommandSelectResult {
  const fileDataObj = {
    url: item.storage_path,
    id: item.id,
    name: item.name || item.original_name || '未命名文件',
    size: item.size,
    mimeType: item.mime_type || '',
  }
  return {
    action: 'send',
    messageType: 'file',
    messageContent: JSON.stringify(fileDataObj),
  }
}

/** getInsertText 在 /file 下不会被调用（onSelect 已返回 send），但接口要求必填，返回空字符串兜底。 */
function getInsertText(): string {
  return ''
}

/** /file 命令：所有会话类型可用（个人文件可分享到任何会话）。 */
export const fileCommand: SlashCommand<FileItem> = {
  trigger: '/file',
  title: '选择文件',
  icon: 'fas fa-file',
  description: '分享我的文件',
  backendSearch: true,
  fetchItems,
  filter,
  getInsertText,
  onSelect,
  itemComponent: markRaw(FileCommandItem),
}
