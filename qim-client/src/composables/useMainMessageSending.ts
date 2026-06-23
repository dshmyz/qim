import { Ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

function parseMessageData(messageData: any): {
  messageType: string
  messageContent: string
  miniAppData: any
  newsData: any
} {
  let messageType = 'text'
  let messageContent = ''
  let miniAppData: any = null
  let newsData: any = null

  if (typeof messageData === 'string') {
    try {
      const parsedData = JSON.parse(messageData)
      if (parsedData.type === 'miniApp' && parsedData.data) {
        messageType = 'miniApp'
        messageContent = JSON.stringify(parsedData.data)
        miniAppData = parsedData.data
      } else if (parsedData.type === 'news' && parsedData.data) {
        messageType = 'news'
        messageContent = JSON.stringify(parsedData.data)
        newsData = parsedData.data
      } else {
        messageType = parsedData.type || 'text'
        messageContent = parsedData.content || messageData
      }
    } catch {
      messageType = 'text'
      messageContent = messageData
    }
  } else {
    messageType = messageData.type || 'text'
    messageContent = messageData.content
    miniAppData = messageData.miniAppData
    newsData = messageData.newsData
  }

  return { messageType, messageContent, miniAppData, newsData }
}

function createFailedMessage(
  messageData: any,
  messageType: string,
  messageContent: string,
  miniAppData: any,
  newsData: any,
  conversationId: string,
  currentUser: Ref<any>
) {
  return {
    id: Date.now().toString(),
    content: messageContent,
    file_name: messageData.fileName,
    file_size: messageData.fileSize,
    sender: {
      id: currentUser.value?.id?.toString() || '',
      name: currentUser.value?.nickname || currentUser.value?.username || '',
      avatar: currentUser.value?.avatar || ''
    },
    timestamp: new Date().getTime(),
    type: messageType,
    isSelf: true,
    isRead: false,
    isFailed: true,
    conversationId,
    quotedMessage: messageData.quotedMessage,
    miniAppData,
    newsData,
    originalData: messageData
  }
}

export function useMainMessageSending(
  currentConversationId: Ref<string | null>,
  messages: Ref<any[]>,
  currentConversation: Ref<any>,
  isConnected: Ref<boolean>,
  sessionExpired: Ref<boolean>,
  handleStreamMessage: (conversationId: string, requestData: any, messageData: any, miniAppData: any, newsData: any) => Promise<void>,
  onMessageSent?: () => void,
  onConversationMissing?: () => void
) {
  const chatStore = useChatStore()
  const { currentUser } = useCurrentUser()

  // 检查会话是否在列表中，不在则触发重新加载（处理移除后发消息恢复显示的场景）
  const ensureConversationInList = (conversationId: string) => {
    const exists = chatStore.conversations.some(c => c.id === conversationId)
    if (!exists && onConversationMissing) {
      onConversationMissing()
    }
  }

  const showMessage = (options: { message: string, type?: string, duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const handleSendMessage = async (messageData: any) => {
    if (!currentConversationId.value) return

    const conversationId = String(currentConversationId.value)

    if (conversationId.startsWith('conv_')) {
      QMessage.error('会话创建失败，请重试')
      return
    }

    const { messageType, messageContent, miniAppData, newsData } = parseMessageData(messageData)

    const requestData: any = {
      type: messageType,
      content: messageContent
    }

    if (messageData.quotedMessage && messageData.quotedMessage.id) {
      requestData.quoted_message_id = parseInt(messageData.quotedMessage.id)
    }

    if (messageType === 'file' || messageType === 'image') {
      requestData.file_size = messageData.fileSize
      requestData.file_name = messageData.fileName
    }

    if (!isConnected.value) {
      QMessage.error('网络连接已断开，消息发送失败')
      const failedMessage = createFailedMessage(messageData, messageType, messageContent, miniAppData, newsData, conversationId, currentUser)
      chatStore.receiveMessage(conversationId, failedMessage as any, true)
      return
    }

    const currentConv = currentConversation.value
    const isBotConversation = currentConv && (currentConv.type === 'bot' || currentConv.isBot)

    if (isBotConversation) {
      ensureConversationInList(conversationId)
      await handleStreamMessage(conversationId, requestData, messageData, miniAppData, newsData)
      return
    }

    try {
      const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
        method: 'POST',
        body: JSON.stringify(requestData)
      })

      if (response.code === 0) {
        const newMessage = {
          id: response.data.id?.toString() || Date.now().toString(),
          content: response.data.content || '',
          file_name: response.data.file_name || messageData.fileName,
          file_size: response.data.file_size || messageData.fileSize,
          sender: {
            id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
            name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
            avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
          },
          timestamp: new Date().getTime(),
          type: response.data.type || messageType,
          isSelf: true,
          isRead: false,
          conversationId,
          quotedMessage: messageData.quotedMessage,
          miniAppData,
          newsData
        }

        ensureConversationInList(conversationId)
        chatStore.receiveMessage(conversationId, newMessage as any, true)
        onMessageSent?.()
      } else {
        let errorMessage = '消息发送失败'
        if (response.code === 401) {
          errorMessage = '登录已过期，请重新登录'
          sessionExpired.value = true
        } else if (response.code === 403) {
          errorMessage = '没有发送消息的权限'
        } else if (response.code === 404) {
          errorMessage = '会话不存在或已被解散'
        } else if (response.message) {
          errorMessage = response.message
        }

        QMessage.error(errorMessage)

        const failedMessage = createFailedMessage(messageData, messageType, messageContent, miniAppData, newsData, conversationId, currentUser)
        chatStore.receiveMessage(conversationId, failedMessage as any, true)
      }
    } catch (error: any) {
      logger.error('发送消息失败:', error)

      let errorMessage = '消息发送失败'
      if (error?.response?.status === 401) {
        errorMessage = '登录已过期，请重新登录'
        sessionExpired.value = true
      } else if (error?.message?.includes('Network') || error?.message?.includes('network')) {
        errorMessage = '网络连接失败，请检查网络'
      } else if (error?.response?.data?.message) {
        errorMessage = error.response.data.message
      } else if (error.message) {
        const msg = error.message.toLowerCase()
        if (msg.includes('unauthorized')) {
          errorMessage = '登录已过期，请重新登录'
          sessionExpired.value = true
        } else if (msg.includes('forbidden')) {
          errorMessage = '没有发送消息的权限'
        } else if (msg.includes('not found')) {
          errorMessage = '会话不存在或已被解散'
        } else {
          errorMessage = error.message
        }
      }

      QMessage.error(errorMessage)

      const failedMessage = createFailedMessage(messageData, messageType, messageContent, miniAppData, newsData, conversationId, currentUser)
      chatStore.receiveMessage(conversationId, failedMessage as any, true)
    }
  }

  const handleRecallMessage = async (messageId: number) => {
    try {
      if (currentConversationId.value) {
        chatStore.recallMessage(currentConversationId.value, messageId.toString())
      }
    } catch (error) {
      logger.error('撤回消息失败:', error)
      QMessage.error('撤回消息失败')
    }
  }

  const handleRetrySendMessage = async (failedMessage: any) => {
    const messageIndex = messages.value.findIndex((msg: any) => msg.id === failedMessage.id)
    if (messageIndex !== -1) {
      messages.value.splice(messageIndex, 1)
    }

    if (failedMessage.originalData) {
      await handleSendMessage(failedMessage.originalData)
    } else {
      await handleSendMessage(failedMessage.content)
    }
  }

  return {
    handleSendMessage,
    handleRecallMessage,
    handleRetrySendMessage
  }
}
