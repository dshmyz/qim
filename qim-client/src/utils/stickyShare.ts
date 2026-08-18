// utils/stickyShare.ts
// 便签分享消息的解析工具：share 消息 content 为 JSON 字符串，此处统一解析。
// 供 ShareMessage（消息本体/预览弹窗）与引用预览等场景复用，避免各组件重复 JSON.parse。

export const STICKY_COLORS = ['yellow', 'blue', 'green', 'red', 'purple', 'pink'] as const
export const STICKY_PAPERS = ['plain', 'lined', 'grid', 'dotted'] as const

export type StickyColor = typeof STICKY_COLORS[number]
export type StickyPaper = typeof STICKY_PAPERS[number]

export interface StickySharePayload {
  type: string
  id?: string | number
  name?: string
  content?: string
  originalContent?: string
  style?: string | Record<string, unknown>
  tags?: string[] | string
  created_at?: string
}

export interface StickyStyle {
  color: StickyColor
  paperStyle: StickyPaper
  fontFamily: string
}

// 解析 share 消息 content JSON；非 sticky 类型或无法解析时返回 null
export function parseStickyShare(content: string): StickySharePayload | null {
  try {
    const data = JSON.parse(content)
    if (!data || typeof data !== 'object' || data.type !== 'sticky') return null
    return data as StickySharePayload
  } catch {
    return null
  }
}

// 解析便签样式：旧消息无 style 字段或脏数据时回退默认黄色，保证任何历史消息都能渲染
export function parseStickyStyle(raw: unknown): StickyStyle {
  let style: Record<string, unknown> | null = null
  if (typeof raw === 'string' && raw !== '{}') {
    try {
      style = JSON.parse(raw)
    } catch {
      style = null
    }
  } else if (raw && typeof raw === 'object') {
    style = raw as Record<string, unknown>
  }
  const color = style?.color
  const paperStyle = style?.paperStyle
  return {
    color: typeof color === 'string' && (STICKY_COLORS as readonly string[]).includes(color)
      ? color as StickyColor
      : 'yellow',
    paperStyle: typeof paperStyle === 'string' && (STICKY_PAPERS as readonly string[]).includes(paperStyle)
      ? paperStyle as StickyPaper
      : 'plain',
    fontFamily: typeof style?.fontFamily === 'string' ? style.fontFamily : ''
  }
}

// 便签标签：兼容数组 / JSON 字符串（旧数据），统一返回字符串数组
export function parseStickyTags(tags: unknown): string[] {
  if (!tags) return []
  if (Array.isArray(tags)) return tags.filter((t): t is string => typeof t === 'string')
  if (typeof tags === 'string') {
    try {
      const parsed = JSON.parse(tags)
      return Array.isArray(parsed) ? parsed.filter((t): t is string => typeof t === 'string') : []
    } catch {
      return []
    }
  }
  return []
}

// 便签创建日期（如 "8月18日"）；缺失或无效返回空串
export function formatStickyDate(raw: unknown): string {
  if (typeof raw !== 'string' || !raw) return ''
  const d = new Date(raw)
  if (isNaN(d.getTime())) return ''
  return d.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
