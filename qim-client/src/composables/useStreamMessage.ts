import { Ref } from 'vue'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useStreamMessage(
  messages: Ref<any[]>,
  serverUrl: Ref<string>
) {
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
    try {
      const streamMessageId = `stream_${Date.now()}`

      const streamMessage = {
        id: streamMessageId,
        content: '',
        sender: { id: '0', name: 'AI助手', avatar: '' },
        timestamp: new Date().getTime(),
        type: 'streaming',
        isSelf: false,
        isRead: false,
        isStreaming: true,
        conversationId
      }

      messages.value.push(streamMessage)

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

        const messageIndex = messages.value.findIndex((m: any) => m.id === streamMessageId)
        if (messageIndex !== -1) {
          messages.value[messageIndex].content = accumulatedContent
          messages.value[messageIndex].isStreaming = true
        }
      }

      const messageIndex = messages.value.findIndex((m: any) => m.id === streamMessageId)
      if (messageIndex !== -1) {
        messages.value[messageIndex].isStreaming = false
        messages.value[messageIndex].type = 'markdown'
      }

    } catch (error) {
      logger.error('流式消息处理失败:', error)
      showMessage({ message: '消息发送失败: ' + (error as Error).message, type: 'error' })
    }
  }

  return { handleStreamMessage }
}
