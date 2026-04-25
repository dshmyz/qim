import { ref } from 'vue'
import { getToken as unifiedGetToken, type ApiRequestConfig } from '@/utils/request'
import { request as unifiedRequest } from '@/utils/request'

/**
 * 聊天请求相关 composable（委托给统一 request 客户端）
 */
export function useChatRequest(baseUrl: string) {
  // 获取 token
  const getToken = unifiedGetToken

  // 格式化日期为 YYYY-MM-DD 格式（本地时间）
  const formatDate = (date: Date): string => {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }

  // 通用请求方法（委托给统一 request）
  const request = async (url: string, options?: RequestInit) => {
    const config: ApiRequestConfig = {
      ...options,
      customBaseUrl: baseUrl,
      data: (options as any)?.body,
      method: options?.method?.toUpperCase() as ApiRequestConfig['method'],
      headers: options?.headers as Record<string, string>
    }

    return unifiedRequest(url, config)
  }

  return {
    getToken,
    formatDate,
    request
  }
}
