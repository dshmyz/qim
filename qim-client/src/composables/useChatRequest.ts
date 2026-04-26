import { ref } from 'vue'

/**
 * 聊天请求相关 composable
 * 包含获取 token、格式化日期、发起 HTTP 请求等功能
 */
export function useChatRequest(baseUrl: string) {
  // 获取 token
  const getToken = (): string | null => {
    return localStorage.getItem('token')
  }

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

    const fullUrl = `${baseUrl}${url}`
    const requestHeaders = {
      ...headers,
      ...options?.headers
    }
    console.log('发送请求:', fullUrl, options)
    console.log('请求头:', requestHeaders)
    console.log('Token:', token)

    try {
      const response = await fetch(fullUrl, {
        ...options,
        headers: requestHeaders
      })

      console.log('响应状态:', response.status, response.statusText)

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        console.error('请求失败:', errorData)
        if (response.status === 403) {
          throw new Error(errorData.message || '权限不足，请检查您的权限')
        }
        throw new Error(errorData.message || '请求失败')
      }

      const data = await response.json()
      console.log('响应数据:', data)
      return data
    } catch (error) {
      console.error('网络错误:', error)
      throw error
    }
  }

  return {
    getToken,
    formatDate,
    request
  }
}
