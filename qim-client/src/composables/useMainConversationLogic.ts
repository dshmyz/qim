import { ref } from 'vue'
import type { Conversation } from '../types'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useMainConversationLogic(
  updateConversations: (conversations: Conversation[]) => void,
  processConversation: (conv: any) => Conversation,
  conversations: { value: Conversation[] }
) {
  const conversationPage = ref(1)
  const conversationPageSize = ref(20)
  const hasMoreConversations = ref(true)
  const isLoadingConversations = ref(false)

  const loadConversations = async (page: number = 1, append: boolean = false) => {
    if (isLoadingConversations.value) return
    
    isLoadingConversations.value = true
    try {
      const response = await request(`/api/v1/conversations?page=${page}&page_size=${conversationPageSize.value}`)
      if (response.code === 0 && response.data) {
        const serverConversations = response.data.list.map((conv: any) => processConversation(conv))
        
        if (append) {
          // 追加模式：滚动加载更多
          updateConversations([...conversations.value, ...serverConversations])
        } else {
          // 替换模式：首次加载或刷新
          updateConversations(serverConversations)
        }
        
        conversationPage.value = page
        hasMoreConversations.value = response.data.has_more
      } else {
        if (!append) {
          updateConversations([])
        }
        hasMoreConversations.value = false
      }
    } catch (error) {
      logger.error('加载会话失败:', error)
      QMessage.error('加载会话失败')
      if (!append) {
        updateConversations([])
      }
      hasMoreConversations.value = false
    } finally {
      isLoadingConversations.value = false
    }
  }

  const loadMoreConversations = async () => {
    if (!hasMoreConversations.value || isLoadingConversations.value) return
    await loadConversations(conversationPage.value + 1, true)
  }

  return {
    loadConversations,
    loadMoreConversations,
    conversationPage,
    conversationPageSize,
    hasMoreConversations,
    isLoadingConversations
  }
}
