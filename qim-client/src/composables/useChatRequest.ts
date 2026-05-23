import { getToken } from './useRequest'

/**
 * 聊天请求相关 composable
 * 包含获取 token、格式化日期、发起 HTTP 请求等功能
 */
export function useChatRequest(baseUrl: string) {
  // 规范化 baseUrl，确保以 http 开头且不以斜杠结尾
  const normalizedBaseUrl = baseUrl.replace(/\/+$/, '')

  // 格式化日期为 YYYY-MM-DD 格式（本地时间）
  const formatDate = (date: Date): string => {
    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }

  // 通用请求方法
  const request = async (url: string, options?: RequestInit) => {
    const token = getToken()
    const headers = {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    }

    const normalizedPath = url.startsWith('/') ? url : `/${url}`
    const fullUrl = `${normalizedBaseUrl}${normalizedPath}`

    const requestHeaders = {
      ...headers,
      ...options?.headers
    }

    try {
      const response = await fetch(fullUrl, {
        ...options,
        headers: requestHeaders
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        const error = new Error(errorData.message || '请求失败') as any
        error.response = response
        error.data = errorData
        throw error
      }

      const data = await response.json()
      return data
    } catch (error) {
      throw error
    }
  }

  return {
    getToken,
    formatDate,
    request
  }
}
