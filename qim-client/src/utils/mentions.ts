/**
 * @ 提及工具：span 跟踪、序列化、解析。
 *
 * content 格式（与后端 pkg/mention 对齐）：
 *   @{mention:<id>|<urlencoded name>}   @ 单人
 *   @{mention:all|所有人}               @ 所有人
 *
 * 输入侧：用户在 textarea 看到的是纯文本 "@张三"，mentionSpans 记录其位置与 userId。
 * 发送时 serializeToContent 把 span 替换为 token。
 *
 * 展示侧：后端返回的 content 是 token 文本，parseContent 解析为纯文本 + mention 位置用于 chip 渲染。
 */

export interface MentionSpan {
  start: number
  end: number
  text: string // "@张三"（用户所见，含 @ 前缀）
  userId: number | 'all'
}

export interface ParsedMention {
  start: number // 在 decode 后纯文本中的偏移
  end: number
  text: string // "@张三"
  userId: number | 'all'
}

const TOKEN_REGEX = /@\{mention:(all|[1-9]\d*)(?:\|([^}]*))?\}/g

/** 生成 @ 单人的 token。 */
export function encodeMentionToken(userId: number, name: string): string {
  return `@{mention:${userId}|${encodeURIComponent(name)}}`
}

/** 生成 @ 所有人 的 token。 */
export function encodeAllMentionToken(): string {
  return `@{mention:all|${encodeURIComponent('所有人')}}`
}

/**
 * 文本编辑后按公共前后缀平移/裁剪 span。
 * span 被删除部分覆盖时 span 失效（丢弃）。
 */
export function reconcileMentionSpans(
  spans: MentionSpan[],
  prevText: string,
  nextText: string
): MentionSpan[] {
  if (prevText === nextText) return spans
  if (spans.length === 0) return []

  // 计算公共前缀长度
  let prefixLen = 0
  const sharedLen = Math.min(prevText.length, nextText.length)
  while (prefixLen < sharedLen && prevText[prefixLen] === nextText[prefixLen]) {
    prefixLen++
  }

  // 计算公共后缀长度（不与前缀重叠）
  let suffixLen = 0
  while (
    suffixLen < prevText.length - prefixLen &&
    suffixLen < nextText.length - prefixLen &&
    prevText[prevText.length - 1 - suffixLen] === nextText[nextText.length - 1 - suffixLen]
  ) {
    suffixLen++
  }

  const removedEnd = prevText.length - suffixLen
  const delta = nextText.length - prevText.length

  const result: MentionSpan[] = []
  for (const span of spans) {
    if (span.end <= prefixLen) {
      // span 完全在前缀保留区，不变
      result.push(span)
    } else if (span.start >= removedEnd) {
      // span 完全在后缀保留区，平移 delta
      result.push({ ...span, start: span.start + delta, end: span.end + delta })
    }
    // span 跨越编辑区或被编辑区覆盖 → 失效，丢弃
  }
  return result
}

/**
 * 将纯文本 + spans 序列化为带 token 的 content。
 * 仅对 text.slice(start,end) === span.text 的 span 生效（防止 span 漂移后误替换）。
 */
export function serializeToContent(text: string, spans: MentionSpan[]): string {
  const validSpans = spans
    .filter((span) => text.slice(span.start, span.end) === span.text)
    .sort((a, b) => b.start - a.start) // 从后往前替换，避免偏移

  let content = text
  for (const span of validSpans) {
    const token =
      span.userId === 'all'
        ? encodeAllMentionToken()
        : encodeMentionToken(span.userId, span.text.slice(1)) // 去掉 @ 前缀
    content = content.slice(0, span.start) + token + content.slice(span.end)
  }
  return content
}

/**
 * 解析 content（含 token）为纯文本 + mention 位置（用于 chip 渲染）。
 * token 被替换为 "@姓名" 纯文本，mention 记录其在纯文本中的偏移。
 */
export function parseContent(content: string): { text: string; mentions: ParsedMention[] } {
  const mentions: ParsedMention[] = []
  let text = ''
  let lastIdx = 0
  let match: RegExpExecArray | null

  TOKEN_REGEX.lastIndex = 0
  while ((match = TOKEN_REGEX.exec(content)) !== null) {
    const start = match.index
    const end = start + match[0].length
    // 追加 token 之前的纯文本
    text += content.slice(lastIdx, start)

    const target = match[1]
    const encodedName = match[2]
    let name: string
    if (target === 'all') {
      name = '所有人'
    } else if (encodedName) {
      try {
        name = decodeURIComponent(encodedName)
      } catch {
        name = encodedName
      }
    } else {
      name = `用户${target}`
    }

    const atText = `@${name}`
    const mentionStart = text.length
    text += atText
    mentions.push({
      start: mentionStart,
      end: mentionStart + atText.length,
      text: atText,
      userId: target === 'all' ? 'all' : Number(target),
    })

    lastIdx = end
  }
  // 追加剩余纯文本
  text += content.slice(lastIdx)
  return { text, mentions }
}

/** 将 content 中的 token 替换为 "@姓名" 纯文本（用于搜索、复制、通知）。 */
export function decodeToPlainText(content: string): string {
  return content.replace(TOKEN_REGEX, (token, target: string, encodedName?: string) => {
    if (target === 'all') return '@所有人'
    if (!encodedName) return `@用户${target}`
    try {
      return `@${decodeURIComponent(encodedName)}`
    } catch {
      return `@${encodedName}`
    }
  })
}

/** 判断 content 是否包含任何 mention token。 */
export function hasAnyMention(content: string): boolean {
  return TOKEN_REGEX.test(content)
}
