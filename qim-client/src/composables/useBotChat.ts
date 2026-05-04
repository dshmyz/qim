// src/composables/useBotChat.ts

import { ref, computed, type Ref } from 'vue'
import { request, getToken } from './useRequest'
import type { BotMessage } from '../types/bot'

/**
 * Bot 对话管理 composable
 * 负责 Bot 会话的初始化、消息加载、发送和流式处理
 */
export function useBotChat(botId: Ref<number | null>) {
  // 会话状态
  const conversationId = ref<number | null>(null)
  const messages = ref<BotMessage[]>([])
  const isLoading = ref(false)
  const isSending = ref(false)
  const error = ref<string | null>(null)

  // 流式消息状态
  const streamingMessageId = ref<string | null>(null)
  const abortController = ref<AbortController | null>(null)

  // 分页状态
  const currentPage = ref(1)
  const pageSize = ref(20)
  const hasMoreMessages = ref(false)

  /**
   * 处理消息数据
   */
  const processMessage = (msg: any): BotMessage => {
    return {
      id: msg.id,
      conversationId: msg.conversation_id,
      senderId: msg.sender_id,
      senderType: msg.sender_type,
      sender: msg.sender ? {
        id: msg.sender.id,
        nickname: msg.sender.nickname || msg.sender.username || '未知用户',
        avatar: msg.sender.avatar,
        type: msg.sender.type
      } : undefined,
      type: msg.type || 'text',
      content: msg.content,
      timestamp: new Date(msg.created_at || msg.timestamp || Date.now()),
      isStreaming: false
    }
  }

  /**
   * 初始化 Bot 会话
   * 创建或获取与指定 Bot 的会话
   */
  const initConversation = async (): Promise<boolean> => {
    if (!botId.value) {
      error.value = 'Bot ID 不能为空'
      return false
    }

    isLoading.value = true
    error.value = null

    try {
      const response: any = await request('/api/v1/conversations/bot', {
        method: 'POST',
        body: JSON.stringify({ botId: botId.value })
      })

      if (response.code === 0 && response.data) {
        conversationId.value = response.data.id || response.data.conversationId
        return true
      } else {
        error.value = response.message || '初始化会话失败'
        return false
      }
    } catch (e: any) {
      error.value = e.message || '初始化会话失败'
      console.error('初始化 Bot 会话失败:', e)
      return false
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 加载历史消息
   * @param reset 是否重置分页
   */
  const loadMessages = async (reset: boolean = true): Promise<void> => {
    if (!conversationId.value) {
      // 如果没有会话 ID，先初始化会话
      const success = await initConversation()
      if (!success) return
    }

    if (isLoading.value) return

    if (reset) {
      currentPage.value = 1
      hasMoreMessages.value = true
    }

    if (!hasMoreMessages.value) return

    isLoading.value = true
    error.value = null

    try {
      const response: any = await request(
        `/api/v1/conversations/${conversationId.value}/messages?page=${currentPage.value}&page_size=${pageSize.value}`
      )

      if (response.code === 0) {
        const serverMessages = Array.isArray(response.data)
          ? response.data.map((msg: any) => processMessage(msg))
          : []

        if (reset) {
          messages.value = serverMessages.reverse()
        } else {
          messages.value = [...serverMessages.reverse(), ...messages.value]
        }

        // 处理分页信息
        if (response.pagination) {
          const { current_page, total_pages } = response.pagination
          hasMoreMessages.value = current_page < total_pages
          currentPage.value = current_page + 1
        } else {
          hasMoreMessages.value = serverMessages.length === pageSize.value
          currentPage.value++
        }
      } else {
        if (reset) {
          messages.value = []
        }
        hasMoreMessages.value = false
      }
    } catch (e: any) {
      error.value = e.message || '加载消息失败'
      console.error('加载消息失败:', e)
      if (reset) {
        messages.value = []
      }
      hasMoreMessages.value = false
    } finally {
      isLoading.value = false
    }
  }

  /**
   * 加载更多消息
   */
  const loadMoreMessages = async (): Promise<void> => {
    await loadMessages(false)
  }

  /**
   * 发送消息（流式）
   * @param content 消息内容
   */
  const sendMessage = async (content: string): Promise<void> => {
    if (!conversationId.value) {
      error.value = '会话未初始化'
      return
    }

    if (!content.trim()) {
      error.value = '消息内容不能为空'
      return
    }

    isSending.value = true
    error.value = null

    // 添加用户消息
    const userMessage: BotMessage = {
      id: Date.now(),
      conversationId: conversationId.value,
      senderId: 0,
      senderType: 'user',
      type: 'text',
      content: content.trim(),
      timestamp: new Date(),
      isStreaming: false
    }
    messages.value.push(userMessage)

    // 创建流式消息占位符
    const streamMessageId = `stream_${Date.now()}`
    const streamMessage: BotMessage = {
      id: streamMessageId as any,
      conversationId: conversationId.value,
      senderId: botId.value || 0,
      senderType: 'bot',
      type: 'text',
      content: '',
      timestamp: new Date(),
      isStreaming: true
    }
    messages.value.push(streamMessage)
    streamingMessageId.value = streamMessageId

    try {
      abortController.value = new AbortController()

      const token = getToken()
      const serverUrl = localStorage.getItem('serverUrl') || import.meta.env.VITE_API_URL || 'http://localhost:8080'

      const response = await fetch(
        `${serverUrl}/api/v1/conversations/${conversationId.value}/messages/stream`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { 'Authorization': `Bearer ${token}` } : {})
          },
          body: JSON.stringify({ content: content.trim() }),
          signal: abortController.value.signal
        }
      )

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
              const parsedChunk = JSON.parse(data)

              if (parsedChunk.content) {
                accumulatedContent += parsedChunk.content
              }

              if (parsedChunk.finish === 'stop') {
                break
              }
            } catch {
              // 如果不是 JSON，直接追加内容
              accumulatedContent += data
            }
          }
        }

        // 更新流式消息内容
        const messageIndex = messages.value.findIndex(m => String(m.id) === streamMessageId)
        if (messageIndex !== -1) {
          messages.value[messageIndex].content = accumulatedContent
        }
      }

      // 完成流式传输
      const messageIndex = messages.value.findIndex(m => String(m.id) === streamMessageId)
      if (messageIndex !== -1) {
        messages.value[messageIndex].isStreaming = false
        messages.value[messageIndex].type = 'markdown'
      }
    } catch (e: any) {
      if (e.name === 'AbortError') {
        console.log('消息发送已取消')
      } else {
        error.value = e.message || '发送消息失败'
        console.error('发送消息失败:', e)

        // 移除失败的流式消息
        const messageIndex = messages.value.findIndex(m => String(m.id) === streamMessageId)
        if (messageIndex !== -1) {
          messages.value.splice(messageIndex, 1)
        }
      }
    } finally {
      isSending.value = false
      streamingMessageId.value = null
      abortController.value = null
    }
  }

  /**
   * 取消当前流式消息
   */
  const cancelStream = (): void => {
    if (abortController.value) {
      abortController.value.abort()
    }
  }

  /**
   * 清空消息
   */
  const clearMessages = (): void => {
    messages.value = []
    currentPage.value = 1
    hasMoreMessages.value = false
  }

  /**
   * 重置会话
   */
  const reset = (): void => {
    conversationId.value = null
    clearMessages()
    error.value = null
    isLoading.value = false
    isSending.value = false
    streamingMessageId.value = null
    if (abortController.value) {
      abortController.value.abort()
      abortController.value = null
    }
  }

  /**
   * 是否正在流式传输
   */
  const isStreaming = computed(() => streamingMessageId.value !== null)

  return {
    // 状态
    conversationId,
    messages,
    isLoading,
    isSending,
    isStreaming,
    error,
    hasMoreMessages,

    // 方法
    initConversation,
    loadMessages,
    loadMoreMessages,
    sendMessage,
    cancelStream,
    clearMessages,
    reset
  }
}
