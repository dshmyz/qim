import type { Message } from '@/types'

export type MergedForwardItem = {
  id: string
  type: string
  content: string
  senderName: string
  timestamp: number
}

export type MergedForwardPayload = {
  version: 1
  title: string
  messages: MergedForwardItem[]
}

export const createMergedForwardPayload = (messages: Message[], title?: string): MergedForwardPayload => ({
  version: 1,
  title: title?.trim() || '聊天记录',
  messages: messages.map(message => ({
    id: String(message.id),
    type: message.type,
    content: message.content,
    senderName: message.sender?.name || '未知用户',
    timestamp: Number(message.timestamp),
  })),
})

export const parseMergedForwardPayload = (content: string): MergedForwardPayload | null => {
  try {
    const value = JSON.parse(content)
    const hasValidMessages = Array.isArray(value?.messages) && value.messages.every((message: unknown) => {
      if (!message || typeof message !== 'object') return false

      const item = message as Record<string, unknown>
      return typeof item.id === 'string'
        && typeof item.type === 'string'
        && typeof item.content === 'string'
        && typeof item.senderName === 'string'
        && typeof item.timestamp === 'number'
        && Number.isFinite(item.timestamp)
    })

    return value?.version === 1 && typeof value?.title === 'string' && value.title.trim().length > 0 && hasValidMessages ? value : null
  } catch {
    return null
  }
}
