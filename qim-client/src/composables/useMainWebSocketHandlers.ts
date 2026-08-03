import { type Ref } from 'vue'
import type { Message } from '../types'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'

const READ_RECEIPT_UPDATED_EVENT = 'message-read-receipt-updated'

export function useMainWebSocketHandlers(
  currentConversationId: Ref<string | null>,
  messages: Ref<Message[]>
) {
  const chatStore = useChatStore()

  const notifyReadUsersRefresh = (conversationId: string, messageIds: string[], readerUserId: any) => {
    if (typeof window === 'undefined' || messageIds.length === 0) return

    window.dispatchEvent(new CustomEvent(READ_RECEIPT_UPDATED_EVENT, {
      detail: {
        conversationId,
        messageIds,
        readerUserId: readerUserId == null ? undefined : readerUserId.toString()
      }
    }))
  }

  const handleReadReceipt = (data: any) => {
    const { conversation_id, user_id, message_ids, last_read_message_id } = data
    const convIdStr = conversation_id.toString()

    if (currentConversationId.value === convIdStr) {
      // 新协议：后端推送 message_ids（本次新写入回执的消息 ID 列表）
      // 精确标记这些消息为已读，避免误标未读消息
      if (Array.isArray(message_ids) && message_ids.length > 0) {
        const idSet = new Set(message_ids.map((id: any) => id.toString()))
        const matchedSelfMessageIds: string[] = []
        messages.value.forEach(msg => {
          const messageId = msg.id.toString()
          if (msg.isSelf && idSet.has(messageId)) {
            matchedSelfMessageIds.push(messageId)
            if (!msg.isRead) {
              chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
            }
          }
        })
        notifyReadUsersRefresh(convIdStr, matchedSelfMessageIds, user_id)
      } else if (last_read_message_id) {
        // 兼容旧协议：按消息顺序标记到 last_read_message_id 为止
        const lastReadId = last_read_message_id.toString()
        for (const msg of messages.value) {
          if (msg.isSelf && !msg.isRead) {
            chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
          }
          if (msg.id.toString() === lastReadId) {
            break
          }
        }
      } else {
        // 兼容旧协议：无 message_ids 时标记所有自发送消息为已读
        messages.value.forEach(msg => {
          if (msg.isSelf && !msg.isRead) {
            chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
          }
        })
      }
    }

    chatStore.markConversationRead(convIdStr)

    logger.log('处理已读回执，会话:', convIdStr, '用户:', user_id, 'message_ids:', message_ids, 'last_read_message_id:', last_read_message_id)
  }

  const handleMessageRecalled = (data: any) => {
    const messageId = data.id.toString()
    const conversationId = data.conversation_id.toString()
    const extra = data.extra || ''
    
    logger.log('收到消息撤回通知:', data)
    
    chatStore.recallMessage(conversationId, messageId, extra)
  }

  return {
    handleReadReceipt,
    handleMessageRecalled
  }
}
