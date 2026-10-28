import type { Message } from '../types'

/**
 * 从 message_updated 载荷构造需要合并到本地消息的字段。
 *
 * 返回 null 表示该事件无需处理（既没有正文变更，也没有结构变更）。
 * 注意：核验存疑标注是**事后增量推送**（只带 id + quality，没有正文），
 * 不能按"必须有 content"丢弃，否则徽章永远不会实时出现。
 */
export function buildMessageUpdate(data: any): Partial<Message> | null {
  const hasQuality = data?.quality !== undefined
  if (!data?.content && !hasQuality) return null

  const updates: Partial<Message> = {}
  if (data.content) updates.content = data.content
  if (data.type) updates.type = data.type
  if (data.is_streaming !== undefined) updates.isStreaming = data.is_streaming
  if (hasQuality) updates.quality = data.quality
  return updates
}
