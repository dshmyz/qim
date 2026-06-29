import { Ref } from 'vue'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'

export function useStreamMessage(
  serverUrl: Ref<string>
) {
  const chatStore = useChatStore()
  const { currentUser } = useCurrentUser()
  // 简单序列计数器，避免同一毫秒内 ID 冲突
  let streamSeq = 0

  const showMessage = (options: { message: string, type?: string, duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else QMessage.info(message, duration)
  }

  const handleStreamMessage = async (
    conversationId: string,
    requestData: any,
    _messageData: any,
    _miniAppData: any,
    _newsData: any
  ) => {
    let streamMessageId: string | null = null

    try {
      // 1. 先将用户自己的消息加入消息列表
      const userMessage = {
        id: `user_${Date.now()}_${streamSeq++}`,
        content: requestData.content || '',
        sender: {
          id: currentUser.value?.id?.toString() || '',
          name: currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: requestData.type || 'text',
        isSelf: true,
        isRead: false,
        conversationId
      }
      chatStore.receiveMessage(conversationId, userMessage as any, true)

      // 2. 创建 bot 流式响应占位符
      streamMessageId = `stream_${Date.now()}_${streamSeq++}`
      const currentConversation = chatStore.currentConversation
      const assistantName = currentConversation?.name || 'AI助手'
      const assistantAvatar = currentConversation?.avatar || ''

      const streamMessage = {
        id: streamMessageId,
        content: '',
        sender: { id: '0', name: assistantName, avatar: assistantAvatar, type: 'bot' },
        timestamp: new Date().getTime(),
        type: 'streaming',
        isSelf: false,
        isRead: false,
        isStreaming: true,
        origin: 'assistant',
        isAIMessage: true,
        is_ai_message: true,
        ai_assistant_name: assistantName,
        conversationId
      }

      // 通过 store 方法添加，确保响应式更新
      chatStore.appendMessage(conversationId, streamMessage as any)

      const token = localStorage.getItem('token')
      const response = await fetch(`${serverUrl.value}/api/v1/conversations/${conversationId}/messages/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {})
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

      try {
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
              } catch {
                accumulatedContent += data
              }
            }
          }

          // 通过 store 方法更新，确保响应式
          chatStore.updateMessage(conversationId, streamMessageId, {
            content: accumulatedContent,
            isStreaming: true
          })
        }

        // 流式结束，标记完成
        chatStore.updateMessage(conversationId, streamMessageId, {
          isStreaming: false,
          type: 'markdown'
        })
      } finally {
        reader.releaseLock()
      }

    } catch (error) {
      logger.error('流式消息处理失败:', error)
      // 异常时标记失败状态，避免一直显示"加载中"
      if (streamMessageId) {
        chatStore.updateMessage(conversationId, streamMessageId, {
          isStreaming: false,
          type: 'text',
          content: '⚠️ 消息发送失败，请重试'
        })
      }
      showMessage({ message: '消息发送失败: ' + (error as Error).message, type: 'error' })
    }
  }

  return { handleStreamMessage }
}
