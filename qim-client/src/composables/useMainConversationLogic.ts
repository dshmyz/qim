import type { Conversation } from '../types'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useMainConversationLogic(
  updateConversations: (conversations: Conversation[]) => void,
  processConversation: (conv: any) => Conversation
) {
  const loadConversations = async () => {
    try {
      const response = await request('/api/v1/conversations')
      if (response.code === 0 && response.data) {
        const serverConversations = response.data.map((conv: any) => processConversation(conv))
        updateConversations(serverConversations)
      } else {
        updateConversations([])
      }
    } catch (error) {
      logger.error('加载会话失败:', error)
      QMessage.error('加载会话失败')
      updateConversations([])
    }
  }

  return {
    loadConversations
  }
}
