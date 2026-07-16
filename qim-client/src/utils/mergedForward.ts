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
  title: '聊天记录'
  messages: MergedForwardItem[]
}

export const createMergedForwardPayload = (messages: Message[]): MergedForwardPayload => ({
  version: 1,
  title: '聊天记录',
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
    return value?.version === 1 && value?.title === '聊天记录' && Array.isArray(value?.messages) ? value : null
  } catch {
    return null
  }
}
