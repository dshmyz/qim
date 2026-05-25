import { Ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export interface Conversation {
  id: string
  name: string
  avatar: string
  type: string
  lastMessage?: any
  unread_count?: number
  timestamp: number
  members?: any[]
}

export function useMainConversationHandlers(
  currentConversationId: Ref<string | null>,
  messages: Ref<any[]>,
  activeOption: Ref<string>,
  loadMessages: (conversationId: string, reset?: boolean) => Promise<void>,
  loadConversations: () => Promise<void>,
  messagePage: Ref<number>,
  hasMoreMessages: Ref<boolean>
) {
  const chatStore = useChatStore()

  const handleConversationSelect = (conversation: Conversation) => {
    const conversationId = String(conversation.id)
    
    logger.log('[Main.vue] handleConversationSelect 被调用', {
      conversationId,
      currentConversationId: currentConversationId.value,
      isSameConversation: currentConversationId.value === conversationId
    })
    
    if (currentConversationId.value === conversationId) {
      logger.log('[Main.vue] 相同会话，跳过处理')
      return
    }
    
    currentConversationId.value = conversationId
    activeOption.value = 'recent'
    loadMessages(conversation.id)
    chatStore.markConversationRead(conversation.id)
    if ((window as any).electron?.tray) {
      (window as any).electron.tray.stopFlash()
    }
  }

  const handleConversationCreated = (newConversation: any) => {
    loadConversations()
    
    if (newConversation && newConversation.id) {
      const conversationId = String(newConversation.id)
      currentConversationId.value = conversationId
      chatStore.clearMessages(conversationId)
      messagePage.value = 1
      hasMoreMessages.value = true
      
      loadMessages(conversationId)
    }
  }

  const handleConversationUpdated = (data: any) => {
    logger.log('会话更新:', data)
    
    if (!data || !data.id) {
      logger.warn('会话更新数据无效:', data)
      return
    }
    
    try {
      const normalizedData = {
        ...data,
        id: data.id.toString()
      }
      chatStore.patchConversation(normalizedData.id, normalizedData)
    } catch (error) {
      logger.error('处理会话更新失败:', error)
      QMessage.error('处理会话更新失败')
    }
  }

  const handleLoadMore = (conversationId: string) => {
    loadMessages(conversationId, false)
  }

  return {
    handleConversationSelect,
    handleConversationCreated,
    handleConversationUpdated,
    handleLoadMore
  }
}
