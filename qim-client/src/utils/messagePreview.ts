export type MessagePreview = {
  kind: 'text' | 'image' | 'file' | 'share' | 'miniApp' | 'news' | 'unknown'
  label: string
}

type MessagePreviewInput = {
  type: string
  content: string
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

const isFilePayload = (data: MessagePayload | null): boolean => {
  if (!data) return false

  const hasName = typeof data.name === 'string' || typeof data.fileName === 'string'
  return hasName && (typeof data.size === 'number' || typeof data.fileSize === 'number' || typeof data.mimeType === 'string')
}

const getString = (data: MessagePayload | null, key: string): string | null => {
  const value = data?.[key]
  return typeof value === 'string' && value.trim() ? value : null
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

const formatFile = (data: MessagePayload | null, content: string): string => {
  const name = getString(data, 'name') || getString(data, 'fileName') || fileNameFromContent(content)
  const size = formatFileSize(data?.size ?? data?.fileSize)
  return size ? `${name} · ${size}` : name
}

const stripMarkdown = (content: string): string => content
  .replace(/```(?:\w+)?\n?([\s\S]*?)```/g, '$1')
  .replace(/`([^`]+)`/g, '$1')
  .replace(/^\s{0,3}(?:#{1,6}\s+|[-*+]\s+|\d+\.\s+)/gm, '')
  .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
  .replace(/[>*_~]/g, '')
  .replace(/\s+/g, ' ')
  .trim()
  .slice(0, 120)

export const getMessagePreview = ({ type, content }: MessagePreviewInput): MessagePreview => {
  const data = safeParse(content)
  if (type === 'image') return { kind: 'image', label: '图片' }
  if (type === 'file' || isFilePayload(data)) return { kind: 'file', label: formatFile(data, content) }
  if (type === 'share') return { kind: 'share', label: `分享：${getString(data, 'name') || '内容'}` }
  if (type === 'miniApp' || type === 'mini-app') return { kind: 'miniApp', label: `小程序：${getString(data, 'name') || '未命名'}` }
  if (type === 'news') return { kind: 'news', label: getString(data, 'title') || '资讯' }
  if (type === 'markdown') return { kind: 'text', label: stripMarkdown(content) || '无内容' }
  return data ? { kind: 'unknown', label: '未知消息' } : { kind: 'text', label: content || '无内容' }
}
