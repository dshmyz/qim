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

  return {
    isProcessing,
    errorMessage,
    translateText,
    rewriteText,
    polishText,
    generateSummary,
    searchMessages,
  }
}
