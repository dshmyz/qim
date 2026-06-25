import { type Ref } from 'vue'
import type { Message } from '../types'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'

export function useMainWebSocketHandlers(
  currentConversationId: Ref<string | null>,
  messages: Ref<Message[]>
) {
  const chatStore = useChatStore()

  const handleReadReceipt = (data: any) => {
    const { conversation_id, user_id, last_read_message_id } = data
    const convIdStr = conversation_id.toString()

    if (currentConversationId.value === convIdStr) {
      const lastReadId = last_read_message_id?.toString()
      if (lastReadId) {
        // 按消息顺序标记到 last_read_message_id 为止（只标记自己发送的消息）
        for (const msg of messages.value) {
          if (msg.isSelf && !msg.isRead) {
            chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
          }
          if (msg.id.toString() === lastReadId) {
            break
          }
        }
      } else {
        // 没有 last_read_message_id 时，标记所有自发送消息为已读
        messages.value.forEach(msg => {
          if (msg.isSelf && !msg.isRead) {
            chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
          }
        })
      }
    }

    chatStore.markConversationRead(convIdStr)

    logger.log('处理已读回执，会话:', convIdStr, '用户:', user_id, 'last_read_message_id:', last_read_message_id)
  }

  const handleMessageRecalled = (data: any) => {
    const messageId = data.id.toString()
    const conversationId = data.conversation_id.toString()
    
    logger.log('收到消息撤回通知:', data)
    
    chatStore.recallMessage(conversationId, messageId)
  }

  return {
    handleReadReceipt,
    handleMessageRecalled
  }
}
