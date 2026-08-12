import { Ref, ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'
import type { Message } from '../types'

export function useMainMessageLoading(
  conversations: Ref<any[]>,
  processMessage: (msg: any, conversationId?: string) => Message
) {
  const chatStore = useChatStore()
  const messagePage = ref(1)
  const messagePageSize = ref(20)
  const hasMoreMessages = ref(true)

  const markMessagesAsRead = async (convId: string) => {
    try {
      const { request } = await import('./useRequest')
      await request(`/api/v1/conversations/${convId}/read`, { method: 'POST' })
    } catch (error) {
      logger.error('标记消息已读失败:', error)
    }
  }

  const loadMessages = async (conversationId: string, reset: boolean = true) => {
    if (chatStore.isLoadingMessages) return

    if (reset) {
      messagePage.value = 1
      hasMoreMessages.value = true
    } else if (!hasMoreMessages.value) {
      return
    }

    chatStore.setLoading(true)
    try {
      const { request } = await import('./useRequest')

      // 首次加载（切换会话）时先标记已读再加载消息，确保 per-user is_read 已落库；
      // 加载更多时不重复调用，避免多余网络请求。
      if (reset) {
        await markMessagesAsRead(conversationId)
        const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
        if (conversationIndex !== -1) {
          conversations.value[conversationIndex].unread_count = 0
        }
      }

      const response = await request(`/api/v1/conversations/${conversationId}/messages?page=${messagePage.value}&page_size=${messagePageSize.value}`)
      if (response.code === 0) {
        const messagesArray = response.data?.messages || []
        const serverMessages = Array.isArray(messagesArray) ? messagesArray.map((msg: any) => processMessage(msg)) : []

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

  const getMessageReadUsers = async (messageId: string) => {
    try {
      const { request } = await import('./useRequest')
      
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
    loadMessages,
    getMessageReadUsers,
    messagePage,
    messagePageSize,
    hasMoreMessages
  }
}
