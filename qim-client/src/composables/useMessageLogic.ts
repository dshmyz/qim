import { ref } from 'vue'
import type { Message } from '../types'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { request } from '../composables/useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useMessageLogic() {
  const chatStore = useChatStore()
  const { currentUser } = useCurrentUser()
  
  const messagePage = ref(1)
  const messagePageSize = ref(20)
  const hasMoreMessages = ref(true)

  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const loadMessages = async (
    conversationId: string,
    processMessage: (msg: any) => Message,
    markMessagesAsRead: (id: string) => Promise<void>,
    reset: boolean = true
  ) => {
    if (chatStore.isLoadingMessages) return
    
    if (reset) {
      messagePage.value = 1
      hasMoreMessages.value = true
    } else if (!hasMoreMessages.value) {
      return
    }
    
    chatStore.setLoading(true)
    try {
      const response = await request(`/api/v1/conversations/${conversationId}/messages?page=${messagePage.value}&page_size=${messagePageSize.value}`)
      if (response.code === 0) {
        const messagesArray = response.data?.messages || []
        const serverMessages = Array.isArray(messagesArray) ? messagesArray.map(processMessage) : []
        
        const messageListElement = document.querySelector('.message-list')
        let scrollTop = 0
        let initialHeight = 0
        if (messageListElement) {
          scrollTop = messageListElement.scrollTop
          initialHeight = messageListElement.scrollHeight
        }
        
        if (reset) {
          chatStore.setMessages(conversationId, serverMessages)
        } else {
          chatStore.prependMessages(conversationId, serverMessages)
        }
        
        setTimeout(() => {
          if (messageListElement) {
            const newHeight = messageListElement.scrollHeight
            const heightDiff = newHeight - initialHeight
            messageListElement.scrollTop = scrollTop + heightDiff
          }
        }, 0)
        
        if (response.pagination) {
          const { current_page, total_pages } = response.pagination
          hasMoreMessages.value = current_page < total_pages
          messagePage.value = current_page + 1
        } else {
          hasMoreMessages.value = serverMessages.length === messagePageSize.value
          messagePage.value++
        }
        
        markMessagesAsRead(conversationId).catch((error: any) => {
          logger.error('标记消息已读失败:', error)
        })
      } else {
        if (reset) {
          chatStore.clearMessages(conversationId)
        }
        hasMoreMessages.value = false
      }
    } catch (error) {
      logger.error('加载消息失败:', error)
      QMessage.error('加载消息失败')
      if (reset) {
        chatStore.clearMessages(conversationId)
      }
      hasMoreMessages.value = false
    } finally {
      chatStore.setLoading(false)
    }
  }

  const handleLoadMore = (conversationId: string) => {
    loadMessages(conversationId, () => ({} as Message), async () => {}, false)
  }

  const getMessageReadUsers = async (messageId: string) => {
    try {
      const response = await request(`/api/v1/messages/${messageId}/read-users`)
      if (response.code === 0) {
        return response.data
      }
      return { read_users: [], total_members: 0 }
    } catch (error) {
      logger.error('获取已读用户列表失败:', error)
      QMessage.error('获取已读用户列表失败')
      return { read_users: [], total_members: 0 }
    }
  }

  return {
    messagePage,
    messagePageSize,
    hasMoreMessages,
    loadMessages,
    handleLoadMore,
    getMessageReadUsers,
    showMessage
  }
}
