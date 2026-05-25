import { ref } from 'vue'
import { useRequest } from './useRequest'

export function useAIActions() {
  const { post, serverUrl } = useRequest()
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

  const generateSmartReply = async (messageContent: string) => {
    isProcessing.value = true
    errorMessage.value = null

    try {
      const response = await post<any>(
        '/api/v1/ai/completion',
        {
          messages: [
            { role: 'system', content: '你是一个友好的智能回复助手。请根据对方的最后一条消息生成一条简短、自然的中文回复。直接返回回复内容，不要加任何前缀或解释。' },
            { role: 'user', content: messageContent }
          ]
        },
        { baseUrl: serverUrl.value }
      )
      if (!response || !response.data) {
        errorMessage.value = '生成回复失败'
        throw new Error('生成回复失败')
      }
      // The completion endpoint returns data as the reply text directly
      return typeof response.data === 'string' ? response.data : response.data.reply || response.data.content || response.data
    } catch (error: any) {
      errorMessage.value = error.message || '生成回复失败'
      throw error
    } finally {
      isProcessing.value = false
    }
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
    generateSmartReply,
  }
}
