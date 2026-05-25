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
    const { conversation_id, user_id } = data
    const convIdStr = conversation_id.toString()
    
    if (currentConversationId.value === convIdStr) {
      messages.value.forEach(msg => {
        if (msg.isSelf && !msg.isRead) {
          chatStore.updateMessage(convIdStr, msg.id, { isRead: true })
        }
      })
    }
    
    chatStore.markConversationRead(convIdStr)
    
    logger.log('处理已读回执，会话:', convIdStr, '用户:', user_id)
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
