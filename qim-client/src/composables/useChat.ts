import { ref, type Ref } from 'vue'
import type { Message } from '../types'

/**
 * 聊天操作 composable
 * 负责消息的加载、发送、撤回、重试和流式处理等核心聊天操作
 */
export function useChat() {
  // 分页状态
  const messagePage = ref(1)
  const messagePageSize = ref(20)
  const isLoadingMessages = ref(false)

  /**
   * 处理消息数据
   */
  const processMessage = (msg: any): Message => {
    return {
      ...msg,
      id: msg.id?.toString() || Date.now().toString(),
      timestamp: msg.created_at ? new Date(msg.created_at).getTime() : (msg.timestamp || Date.now()),
      isRecalled: msg.is_recalled || false,
      sender: msg.sender ? {
        ...msg.sender,
        name: msg.sender.name || msg.sender.nickname || msg.sender.username || msg.sender.user?.nickname || msg.sender.user?.username || '未知用户'
      } : null
    }
  }

  /**
   * 加载消息
   * @param conversationId - 会话 ID
   * @param messages - 消息列表 ref（由 useConversation 提供）
   * @param hasMoreMessages - 是否有更多消息 ref（由 useConversation 提供）
   * @param request - 请求函数
   * @param reset - 是否重置分页
   */
  const loadMessages = async (
    conversationId: string,
    messages: Ref<Message[]>,
    hasMoreMessages: Ref<boolean>,
    request: (url: string, options?: any) => Promise<any>,
    reset: boolean = true
  ) => {
    if (isLoadingMessages.value) return
    
    if (reset) {
      messagePage.value = 1
      hasMoreMessages.value = true
    }
    
    if (!hasMoreMessages.value) return
    
    isLoadingMessages.value = true
    try {
      const response = await request(
        `/api/v1/conversations/${conversationId}/messages?page=${messagePage.value}&page_size=${messagePageSize.value}`
      )
      
      if (response.code === 0) {
        const serverMessages = Array.isArray(response.data)
          ? response.data.map((msg: any) => processMessage(msg))
          : []
        
        // 保存滚动位置
        const messageListElement = document.querySelector('.message-list')
        let scrollTop = 0
        let initialHeight = 0
        if (messageListElement) {
          scrollTop = messageListElement.scrollTop
          initialHeight = messageListElement.scrollHeight
        }
        
        if (reset) {
          messages.value = serverMessages
        } else {
          messages.value = [...serverMessages, ...messages.value]
        }
        
        // 恢复滚动位置
        setTimeout(() => {
          if (messageListElement) {
            const newHeight = messageListElement.scrollHeight
            const heightDiff = newHeight - initialHeight
            messageListElement.scrollTop = scrollTop + heightDiff
          }
        }, 0)
        
        // 处理分页信息
        if (response.pagination) {
          const { current_page, total_pages } = response.pagination
          hasMoreMessages.value = current_page < total_pages
          messagePage.value = current_page + 1
        } else {
          hasMoreMessages.value = serverMessages.length === messagePageSize.value
          messagePage.value++
        }
        
        // 标记消息为已读
        try {
          await markMessagesAsRead(conversationId, request)
        } catch (error) {
          console.error('标记消息已读失败:', error)
        }
      } else {
        if (reset) {
          messages.value = []
        }
        hasMoreMessages.value = false
      }
    } catch (error) {
      console.error('加载消息失败:', error)
      if (reset) {
        messages.value = []
      }
      hasMoreMessages.value = false
    } finally {
      isLoadingMessages.value = false
    }
  }

  /**
   * 加载更多消息
   */
  const handleLoadMore = (
    conversationId: string,
    messages: Ref<Message[]>,
    hasMoreMessages: Ref<boolean>,
    request: (url: string, options?: any) => Promise<any>
  ) => {
    loadMessages(conversationId, messages, hasMoreMessages, request, false)
  }

  /**
   * 获取消息的已读用户列表
   */
  const getMessageReadUsers = async (
    messageId: string,
    request: (url: string, options?: any) => Promise<any>
  ) => {
    try {
      const response = await request(`/api/v1/messages/${messageId}/read-users`)
      if (response.code === 0) {
        return response.data
      }
      return { read_users: [], total_members: 0 }
    } catch (error) {
      console.error('获取已读用户列表失败:', error)
      return { read_users: [], total_members: 0 }
    }
  }

  /**
   * 标记消息为已读
   */
  const markMessagesAsRead = async (
    conversationId: string,
    request: (url: string, options?: any) => Promise<any>
  ) => {
    try {
      await request(`/api/v1/conversations/${conversationId}/read`, {
        method: 'POST'
      })
    } catch (error) {
      console.error('标记消息已读失败:', error)
    }
  }

  /**
   * 发送消息
   */
  const handleSendMessage = async (
    messageData: any,
    currentConversationId: Ref<string | null>,
    currentUser: Ref<any>,
    messages: Ref<Message[]>,
    conversations: Ref<any[]>,
    currentConversation: Ref<any>,
    request: (url: string, options?: any) => Promise<any>,
    ws: WebSocket | null,
    serverUrl: Ref<string>,
    getToken: () => string | null,
    storage: any,
    showMessage: (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => void
  ) => {
    if (!currentConversationId.value) return
    
    const conversationId = String(currentConversationId.value)
    
    if (conversationId.startsWith('conv_')) {
      showMessage({ message: '会话创建失败，请重试', type: 'error' })
      return
    }
    
    const isWebSocketConnected = ws && ws.readyState === WebSocket.OPEN
    
    try {
      let requestData: any = {}
      let messageType = 'text'
      let messageContent = ''
      let miniAppData = null
      let newsData = null
      
      // 处理JSON字符串格式的消息数据
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
        } catch (e) {
          messageType = 'text'
          messageContent = messageData
        }
      } else {
        messageType = messageData.type || 'text'
        messageContent = messageData.content
        miniAppData = messageData.miniAppData
        newsData = messageData.newsData
      }
      
      requestData = {
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
      
      // WebSocket断开时标记发送失败
      if (!isWebSocketConnected) {
        const failedMessage = {
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
          quotedMessage: messageData.quotedMessage,
          miniAppData,
          newsData,
          originalData: messageData
        }
        
        messages.value.push(failedMessage)
        
        const conversationIndex = conversations.value.findIndex(
          c => c.id.toString() === conversationId
        )
        if (conversationIndex !== -1) {
          conversations.value[conversationIndex].lastMessage = failedMessage
          conversations.value[conversationIndex].timestamp = failedMessage.timestamp
          storage.saveConversations(conversations.value)
        }
        
        return
      }
      
      // 检查是否为机器人会话
      const currentConv = currentConversation.value
      const isBotConversation = currentConv && (currentConv.type === 'bot' || currentConv.isBot)
      
      if (isBotConversation) {
        await handleStreamMessage(
          conversationId,
          requestData,
          messageData,
          miniAppData,
          newsData,
          messages,
          storage,
          serverUrl,
          getToken,
          showMessage
        )
      } else {
        const response = await request(
          `/api/v1/conversations/${conversationId}/messages`,
          {
            method: 'POST',
            body: JSON.stringify(requestData)
          }
        )
        
        if (response.code === 0) {
          const newMessage = {
            id: response.data.id?.toString() || Date.now().toString(),
            content: response.data.content,
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
            quotedMessage: messageData.quotedMessage,
            miniAppData,
            newsData
          }
          
          messages.value.push(newMessage)
          
          const conversationIndex = conversations.value.findIndex(
            c => c.id.toString() === conversationId
          )
          if (conversationIndex !== -1) {
            conversations.value[conversationIndex].lastMessage = newMessage
            conversations.value[conversationIndex].timestamp = newMessage.timestamp
            storage.saveConversations(conversations.value)
          }
        } else {
          showMessage({ message: '消息发送失败: ' + response.message, type: 'error' })
          
          const failedMessage = {
            id: Date.now().toString(),
            content: messageContent,
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
            quotedMessage: messageData.quotedMessage,
            originalData: messageData
          }
          
          messages.value.push(failedMessage)
        }
      }
    } catch (error) {
      console.error('发送消息失败:', error)
      showMessage({ message: '消息发送失败，请重试', type: 'error' })
    }
  }

  /**
   * 处理流式消息（机器人会话）
   */
  const handleStreamMessage = async (
    conversationId: string,
    requestData: any,
    messageData: any,
    miniAppData: any,
    newsData: any,
    messages: Ref<Message[]>,
    storage: any,
    serverUrl: Ref<string>,
    getToken: () => string | null,
    showMessage: (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => void
  ) => {
    try {
      const streamMessageId = `stream_${Date.now()}`
      
      const streamMessage = {
        id: streamMessageId,
        content: '',
        sender: {
          id: '0',
          name: 'AI助手',
          avatar: ''
        },
        timestamp: new Date().getTime(),
        type: 'streaming',
        isSelf: false,
        isRead: false,
        isStreaming: true
      }
      
      messages.value.push(streamMessage)
      
      const response = await fetch(`${serverUrl.value}/api/v1/conversations/${conversationId}/messages/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
        },
        body: JSON.stringify(requestData)
      })
      
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      
      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('No response body')
      }
      
      let accumulatedContent = ''
      let buffer = ''
      
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        
        const chunk = new TextDecoder('utf-8').decode(value)
        buffer += chunk
        
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''
        
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6).trim()
            if (!data) continue
            
            try {
              const chunk = JSON.parse(data)
              
              if (chunk.content) {
                accumulatedContent += chunk.content
              }
              
              if (chunk.finish === 'stop') {
                break
              }
            } catch (e) {
              accumulatedContent += data
            }
          }
        }
        
        const messageIndex = messages.value.findIndex(m => m.id === streamMessageId)
        if (messageIndex !== -1) {
          messages.value[messageIndex].content = accumulatedContent
          messages.value[messageIndex].isStreaming = true
        }
      }
      
      const messageIndex = messages.value.findIndex(m => m.id === streamMessageId)
      if (messageIndex !== -1) {
        messages.value[messageIndex].isStreaming = false
        messages.value[messageIndex].type = 'markdown'
      }
    } catch (error) {
      console.error('流式消息处理失败:', error)
      showMessage({ message: '消息发送失败: ' + (error as Error).message, type: 'error' })
    }
  }

  /**
   * 重新发送失败的消息
   */
  const handleRetrySendMessage = (
    failedMessage: any,
    messages: Ref<Message[]>,
    currentConversationId: Ref<string | null>,
    handleSendMessage: (...args: any[]) => void
  ) => {
    const messageIndex = messages.value.findIndex(msg => msg.id === failedMessage.id)
    if (messageIndex !== -1) {
      messages.value.splice(messageIndex, 1)
    }
    
    if (failedMessage.originalData) {
      handleSendMessage(failedMessage.originalData)
    } else {
      handleSendMessage(failedMessage.content)
    }
  }

  /**
   * 撤回消息
   */
  const handleRecallMessage = async (
    messageId: number,
    messages: Ref<Message[]>,
    request: (url: string, options?: any) => Promise<any>
  ) => {
    try {
      const response = await request(`/api/v1/messages/${messageId}/recall`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        const index = messages.value.findIndex(m => m.id === messageId.toString())
        if (index !== -1) {
          messages.value[index].content = '[消息已撤回]'
          messages.value[index].isRecalled = true
        }
      } else {
        console.error('撤回消息失败:', response.message)
      }
    } catch (error) {
      console.error('撤回消息失败:', error)
    }
  }

  return {
    // 状态
    messagePage,
    messagePageSize,
    isLoadingMessages,
    
    // 操作方法
    loadMessages,
    handleLoadMore,
    getMessageReadUsers,
    markMessagesAsRead,
    handleSendMessage,
    handleStreamMessage,
    handleRetrySendMessage,
    handleRecallMessage,
    processMessage
  }
}
