import { ref } from 'vue'
import type { Conversation } from '../types'
import { useChatStore } from '../stores/chat'
import { useConversation } from './useConversation'
import { useProcessConversation } from './useProcessConversation'
import { request } from '../composables/useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useConversationLogic() {
  const chatStore = useChatStore()
  const { 
    currentConversationId, 
    setCurrentConversationId, 
    _handleConversationSelect 
  } = useConversation()
  const { processConversation } = useProcessConversation()
  
  const activeOption = ref('recent')
  const messagePage = ref(1)
  const hasMoreMessages = ref(true)

  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const loadConversations = async () => {
    try {
      const response = await request('/api/v1/conversations')
      if (response.code === 0 && response.data) {
        const serverConversations = response.data.list.map((conv: any) => processConversation(conv))
        chatStore.updateConversations(serverConversations)
      } else {
        chatStore.updateConversations([])
      }
    } catch (error) {
      logger.error('加载会话失败:', error)
      QMessage.error('加载会话失败')
      chatStore.updateConversations([])
    }
  }

  const handleConversationSelect = (
    conversation: Conversation,
    loadMessages: (id: string | number) => void
  ) => {
    const conversationId = String(conversation.id)
    
    logger.log('[ConversationLogic] handleConversationSelect', {
      conversationId,
      currentConversationId: currentConversationId.value,
      isSameConversation: currentConversationId.value === conversationId
    })
    
    if (currentConversationId.value === conversationId) {
      logger.log('[ConversationLogic] 相同会话，跳过处理')
      return
    }
    
    _handleConversationSelect(conversation)
    activeOption.value = 'recent'
    loadMessages(conversation.id)
    chatStore.markConversationRead(conversation.id)
    
    if (window.electron?.tray) {
      window.electron.tray.stopFlash()
    }
  }

  const handleConversationCreated = (
    newConversation: any,
    loadMessages: (id: string | number) => void
  ) => {
    loadConversations()
    
    if (newConversation && newConversation.id) {
      const conversationId = String(newConversation.id)
      setCurrentConversationId(conversationId)
      chatStore.clearMessages(conversationId)
      messagePage.value = 1
      hasMoreMessages.value = true
      loadMessages(conversationId)
    }
  }

  return {
    activeOption,
    messagePage,
    hasMoreMessages,
    loadConversations,
    handleConversationSelect,
    handleConversationCreated,
    showMessage
  }
}
