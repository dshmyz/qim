import { Ref } from 'vue'
import { useCurrentUser } from './useCurrentUser'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export interface Message {
  id: string
  content: string
  sender: {
    id: string
    name: string
    avatar: string
    user?: any
  }
  timestamp: number
  type: string
  isSelf: boolean
  isRead: boolean
  isRecalled?: boolean
  isFailed?: boolean
  isStreaming?: boolean
  isAtMention?: boolean
  isAvatarReply?: boolean
  is_avatar_reply?: boolean
  ai_type?: string
  isAIMessage?: boolean
  is_ai_message?: boolean
  ai_assistant_name?: string
  avatar_name?: string
  conversationId: string
  file_name?: string
  file_size?: number
  shareData?: any
  miniAppData?: any
  newsData?: any
  quotedMessage?: any
}

export function useMainMessageHandlers() {
  const { currentUser } = useCurrentUser()

  const processMessage = (msg: any, conversationId?: string): Message => {
    const messageObj: any = {
      id: msg.id ? msg.id.toString() : '',
      content: msg.content || '',
      file_name: msg.file_name,
      file_size: msg.file_size,
      sender: msg.sender ? {
        id: msg.sender.id ? msg.sender.id.toString() : '',
        name: msg.sender.nickname || msg.sender.username || msg.sender.name || msg.sender.user?.nickname || msg.sender.user?.username || '',
        avatar: msg.sender.avatar || '',
        user: msg.sender
      } : {
        id: '',
        name: '',
        avatar: ''
      },
      timestamp: msg.created_at ? new Date(msg.created_at).getTime() : Date.now(),
      type: msg.type || 'text',
      isSelf: (msg.sender && msg.sender.id ? msg.sender.id.toString() === currentUser.value?.id?.toString() : false) || (msg.ai_type === 'avatar' && msg.sender_id?.toString() === currentUser.value?.id?.toString()),
      isRead: msg.is_read || false,
      isRecalled: msg.is_recalled || false,
      isFailed: msg.is_failed || false,
      isStreaming: msg.is_streaming || false,
      isAtMention: Array.isArray(msg.mention_user_ids)
        ? msg.mention_user_ids.some((uid: number) => uid.toString() === currentUser.value?.id?.toString())
          && msg.sender_id?.toString() !== currentUser.value?.id?.toString()
        : (msg.is_at_mention === true),
      isAvatarReply: msg.ai_type === 'avatar',
      is_avatar_reply: msg.ai_type === 'avatar',
      ai_type: msg.ai_type || '',
      isAIMessage: msg.ai_type === 'assistant' || msg.ai_type === 'avatar' || msg.sender?.type === 'bot' || msg.sender?.type === 'system' || msg.is_ai_message === true || msg.isAIMessage === true,
      is_ai_message: msg.is_ai_message || false,
      ai_assistant_name: msg.ai_assistant_name || '',
      avatar_name: msg.avatar_name || '',
      conversationId: msg.conversation_id?.toString() || msg.conversationId || conversationId || '',
      quotedMessage: msg.quoted_message ? {
        id: msg.quoted_message.id?.toString() || '',
        content: msg.quoted_message.content || '',
        file_name: msg.quoted_message.file_name,
        file_size: msg.quoted_message.file_size,
        sender: msg.quoted_message.sender ? {
          id: msg.quoted_message.sender.id?.toString() || '',
          name: msg.quoted_message.sender?.nickname || msg.quoted_message.sender?.username || msg.quoted_message.sender?.name || msg.quoted_message.sender?.user?.nickname || msg.quoted_message.sender?.user?.username || '未知用户',
          avatar: msg.quoted_message.sender.avatar || ''
        } : {
          id: '',
          name: '未知用户',
          avatar: ''
        },
        timestamp: msg.quoted_message.created_at ? new Date(msg.quoted_message.created_at).getTime() : Date.now(),
        type: msg.quoted_message.type || 'text',
        isSelf: msg.quoted_message.sender?.id?.toString() === currentUser.value?.id?.toString()
      } : undefined,
    }
    
    if (msg.type === 'share' && msg.content) {
      try {
        const shareData = JSON.parse(msg.content)
        messageObj.shareData = shareData
      } catch (e) {
        messageObj.shareData = {
          type: 'text',
          content: msg.content
        }
      }
    }
    
    if (msg.type === 'miniApp' && msg.content) {
      try {
        messageObj.miniAppData = JSON.parse(msg.content)
      } catch (e) {
        logger.error('解析小程序数据失败:', e)
      }
    }
    
    if (msg.type === 'news' && msg.content) {
      try {
        messageObj.newsData = JSON.parse(msg.content)
      } catch (e) {
        logger.error('解析资讯数据失败:', e)
      }
    }
    
    return messageObj
  }

  const handleMessageLike = async (message: any) => {
    try {
      const { request } = await import('./useRequest')
      
      const response = await request(`/api/v1/channels/messages/${message.id}/like`, {
        method: 'POST'
      })
      if (response.code === 0) {
        QMessage.success('点赞成功')
      }
    } catch (error) {
      logger.error('点赞失败:', error)
      QMessage.error('点赞失败')
    }
  }

  const handleMessageUnlike = async (message: any) => {
    try {
      const { request } = await import('./useRequest')
      
      const response = await request(`/api/v1/channels/messages/${message.id}/like`, { method: 'DELETE'
      })
      if (response.code === 0) {
        QMessage.success('取消点赞')
      }
    } catch (error) {
      logger.error('取消点赞失败:', error)
      QMessage.error('取消点赞失败')
    }
  }

  const handleMessageComment = async (message: any, content: string) => {
    if (!content.trim()) {
      QMessage.warning('评论内容不能为空')
      return null
    }
    
    try {
      const { request } = await import('./useRequest')
      
      const response = await request(`/api/v1/channels/messages/${message.id}/comments`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content: content.trim() })
      })
      if (response.code === 0) {
        QMessage.success('评论成功')
        return response.data
      }
    } catch (error) {
      logger.error('评论失败:', error)
      QMessage.error('评论失败')
    }
    return null
  }

  const getMessageComments = async (messageId: number | string) => {
    try {
      const { request } = await import('./useRequest')
      
      const response = await request(`/api/v1/channels/messages/${messageId}/comments`)
      if (response.code === 0) {
        return response.data
      }
    } catch (error) {
      logger.error('获取评论失败:', error)
    }
    return []
  }

  return {
    processMessage,
    handleMessageLike,
    handleMessageUnlike,
    handleMessageComment,
    getMessageComments
  }
}
