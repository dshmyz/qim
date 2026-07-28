import { ref } from 'vue'
import { useRequest } from './useRequest'
import { useAIStream } from './useAIStream'

export function useAIActions() {
  const { post, serverUrl } = useRequest()
  const { stream, abort: abortStream } = useAIStream()
  const isProcessing = ref(false)
  const errorMessage = ref<string | null>(null)

  const translateText = async (text: string, targetLang: string = 'zh') => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/translate',
        { text, target_lang: targetLang },
        { baseUrl: serverUrl.value }
      )
      if (!response) {
        errorMessage.value = '翻译失败'
        throw new Error('翻译失败')
      }
      return response.data.translated_text
    } catch (error: any) {
      errorMessage.value = error.message || '翻译失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const translateImage = async (
    imageUrl: string,
    targetLang: string = 'zh'
  ) => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/translate/image',
        { image_url: imageUrl, target_lang: targetLang },
        { baseUrl: serverUrl.value, timeout: 60000 }
      )
      if (!response) {
        errorMessage.value = '图片翻译失败'
        throw new Error('图片翻译失败')
      }
      return response.data.translated_text
    } catch (error: any) {
      errorMessage.value = error.message || '图片翻译失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const rewriteText = async (
    text: string,
    style: string = 'concise',
    tone: string = 'professional'
  ) => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/rewrite',
        { text, style, tone },
        { baseUrl: serverUrl.value }
      )
      if (!response) {
        errorMessage.value = '改写失败'
        throw new Error('改写失败')
      }
      return response.data.rewritten_text
    } catch (error: any) {
      errorMessage.value = error.message || '改写失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const polishText = async (text: string, language: string = 'zh') => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/polish',
        { text, language },
        { baseUrl: serverUrl.value }
      )
      if (!response) {
        errorMessage.value = '润色失败'
        throw new Error('润色失败')
      }
      return response.data.polished_text
    } catch (error: any) {
      errorMessage.value = error.message || '润色失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const generateSummary = async (
    conversationId: number,
    timeRange: string = 'today'
  ) => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/summary',
        { conversation_id: conversationId, time_range: timeRange },
        { baseUrl: serverUrl.value }
      )
      if (!response) {
        errorMessage.value = '摘要生成失败'
        throw new Error('摘要生成失败')
      }
      return response.data
    } catch (error: any) {
      errorMessage.value = error.message || '摘要生成失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  const searchMessages = async (
    conversationId: number,
    query: string,
    options?: {
      senderId?: number
      startTime?: string
      endTime?: string
    }
  ) => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const payload: Record<string, any> = {
        conversation_id: conversationId,
        query,
      }
      if (options?.senderId !== undefined) {
        payload.sender_id = options.senderId
      }
      if (options?.startTime) {
        payload.start_time = options.startTime
      }
      if (options?.endTime) {
        payload.end_time = options.endTime
      }

      const response = await post<any>(
        '/api/v1/ai/search',
        payload,
        { baseUrl: serverUrl.value }
      )
      if (!response) {
        errorMessage.value = '搜索失败'
        throw new Error('搜索失败')
      }
      return response.data
    } catch (error: any) {
      errorMessage.value = error.message || '搜索失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  // 帮我回复：根据目标消息 + 对话上下文起草回复草稿
  const draftReply = async (conversationId: number, messageId: number) => {
    isProcessing.value = true
    errorMessage.value = null
    try {
      const response = await post<any>(
        '/api/v1/ai/draft-reply',
        { conversation_id: conversationId, message_id: messageId },
        { baseUrl: serverUrl.value }
      )
      if (!response || !response.data) {
        errorMessage.value = '生成回复失败'
        throw new Error('生成回复失败')
      }
      return response.data.reply ?? ''
    } catch (error: any) {
      errorMessage.value = error.message || '生成回复失败'
      throw error
    } finally {
      isProcessing.value = false
    }
  }

  // 帮我回复（流式）：逐字推送给调用方，实现打字机效果
  const draftReplyStream = (
    conversationId: number,
    messageId: number,
    handlers: {
      onChunk: (content: string) => void
      onComplete: () => void
      onError: (error: Error) => void
    }
  ) => {
    isProcessing.value = true
    errorMessage.value = null
    stream({
      url: `${serverUrl.value}/api/v1/ai/draft-reply/stream`,
      body: { conversation_id: conversationId, message_id: messageId },
      onChunk: handlers.onChunk,
      onComplete: () => {
        isProcessing.value = false
        handlers.onComplete()
      },
      onError: (e: Error) => {
        isProcessing.value = false
        errorMessage.value = e.message || '生成回复失败'
        handlers.onError(e)
      },
    })
  }

  // 终止正在进行的流式帮我回复
  const abortDraftReply = () => {
    abortStream()
  }

  return {
    isProcessing,
    errorMessage,
    translateText,
    translateImage,
    rewriteText,
    polishText,
    generateSummary,
    searchMessages,
    draftReply,
    draftReplyStream,
    abortDraftReply,
  }
}
