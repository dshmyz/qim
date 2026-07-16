import { decodeToPlainText } from './mentions'
import { parseMergedForwardPayload, type MergedForwardPayload } from './mergedForward'

export type MessageDisplayKind =
  | 'text'
  | 'image'
  | 'file'
  | 'share'
  | 'miniApp'
  | 'news'
  | 'video'
  | 'audio'
  | 'system'
  | 'streaming'
  | 'mergedForward'
  | 'unknown'

export type MessageDisplayInput = {
  type?: string
  content?: string
  file_name?: string
  fileName?: string
  file_size?: number
  miniAppData?: Record<string, unknown>
  shareData?: Record<string, unknown>
  newsData?: Record<string, unknown>
  title?: string
  description?: string
}

export type MessageDisplay = {
  kind: MessageDisplayKind
  label: string
  summary: string
  title: string
  description: string
  data?: Record<string, unknown> | MergedForwardPayload
}

type MessagePayload = Record<string, unknown>

const safeParse = (content: string): MessagePayload | null => {
  try {
    const value: unknown = JSON.parse(content)
    return value && typeof value === 'object' && !Array.isArray(value) ? value as MessagePayload : null
  } catch {
    return null
  }
}

const getString = (data: MessagePayload | null | undefined, key: string): string | null => {
  const value = data?.[key]
  return typeof value === 'string' && value.trim() ? value : null
}

const isFilePayload = (data: MessagePayload | null): boolean => {
  if (!data) return false
  const hasName = typeof data.name === 'string' || typeof data.fileName === 'string'
  return hasName && (typeof data.size === 'number' || typeof data.fileSize === 'number' || typeof data.mimeType === 'string')
}

const formatFileSize = (size: unknown): string | null => {
  if (typeof size !== 'number' || !Number.isFinite(size) || size < 0) return null
  const units = ['B', 'KB', 'MB']
  let value = size
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const rounded = Math.round(value * 10) / 10
  return `${Number.isInteger(rounded) ? rounded : rounded.toFixed(1)} ${units[unitIndex]}`
}

const fileNameFromContent = (content: string): string => {
  const path = content.split('?')[0]
  return path.split('/').pop() || content || '文件'
}

const getFileName = (data: MessagePayload | null, input: MessageDisplayInput): string =>
  getString(data, 'name') || getString(data, 'fileName') || input.file_name || input.fileName || fileNameFromContent(input.content || '')

const stripMarkdown = (content: string): string => content
  .replace(/```(?:\w+)?\n?([\s\S]*?)```/g, '$1')
  .replace(/`([^`]+)`/g, '$1')
  .replace(/^\s{0,3}(?:#{1,6}\s+|[-*+]\s+|\d+\.\s+)/gm, '')
  .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
  .replace(/[>*_~]/g, '')
  .replace(/\s+/g, ' ')
  .trim()
  .slice(0, 120)

const unwrapPayload = (value: MessagePayload | null | undefined): MessagePayload | null => {
  if (!value) return null
  const nested = value.data
  return nested && typeof nested === 'object' && !Array.isArray(nested) ? nested as MessagePayload : value
}

const asPayload = (value: Record<string, unknown> | undefined, fallback: MessagePayload | null): MessagePayload | null => {
  const supplied = unwrapPayload(value)
  const parsed = unwrapPayload(fallback)
  if (!supplied) return parsed
  if (!parsed) return supplied
  return { ...supplied, ...parsed }
}

const display = (
  kind: MessageDisplayKind,
  label: string,
  summary: string,
  title = label,
  description = '',
  data?: MessageDisplay['data'],
): MessageDisplay => ({ kind, label, summary, title, description, data })

export const resolveMessageDisplay = (input: MessageDisplayInput): MessageDisplay => {
  const type = input.type || 'text'
  const content = input.content || ''
  const parsed = safeParse(content)

  if (type === 'merged_forward') {
    const payload = parseMergedForwardPayload(content)
    const label = payload ? `[聊天记录] ${payload.messages.length} 条消息` : '[聊天记录]'
    return display('mergedForward', label, label, '聊天记录', payload ? `${payload.messages.length} 条消息` : '聊天记录无法加载', payload || undefined)
  }
  if (type === 'image') return display('image', '图片', `[图片] ${getFileName(parsed, input)}`, getFileName(parsed, input))
  if (type === 'file' || isFilePayload(parsed)) {
    const name = getFileName(parsed, input)
    const size = formatFileSize(parsed?.size ?? parsed?.fileSize ?? input.file_size)
    return display('file', size ? `${name} · ${size}` : name, `[文件] ${name}`, name, size || '')
  }
  if (type === 'share') {
    const data = asPayload(input.shareData, parsed)
    const name = getString(data, 'name') || input.title || '内容'
    return display('share', `分享：${name}`, `[分享] ${name}`, name, getString(data, 'description') || input.description || '点击查看分享', data || undefined)
  }
  if (type === 'miniApp' || type === 'mini-app' || type === 'mini_app') {
    const data = asPayload(input.miniAppData, parsed)
    const name = getString(data, 'name') || input.title || '未命名'
    return display('miniApp', `小程序：${name}`, `[小程序] ${name}`, name, getString(data, 'description') || input.description || '点击打开小程序', data || undefined)
  }
  if (type === 'news') {
    const data = asPayload(input.newsData, parsed)
    const title = getString(data, 'title') || input.title || '资讯'
    return display('news', title, `[资讯] ${title}`, title, getString(data, 'summary') || getString(data, 'description') || input.description || '点击查看资讯', data || undefined)
  }
  if (type === 'video') return display('video', '视频', '[视频]', '视频')
  if (type === 'audio') return display('audio', '语音', '[语音]', '语音')
  if (type === 'markdown') {
    const text = stripMarkdown(content) || '无内容'
    return display('text', text, text, text)
  }
  if (type === 'streaming') {
    const text = decodeToPlainText(content) || '无内容'
    return display('streaming', text, text, text)
  }
  if (type === 'system') {
    const text = decodeToPlainText(content) || '系统消息'
    return display('system', text, text, text)
  }
  if (parsed) return display('unknown', '未知消息', '[未知消息]', '未知消息')

  const text = decodeToPlainText(content) || '无内容'
  return display('text', text, text, text)
}
