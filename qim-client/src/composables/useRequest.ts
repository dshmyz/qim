import { ref, computed } from 'vue'
import { useServerUrl } from './useServerUrl'
import QMessage from '../utils/qmessage'

const { serverUrl, setServerUrl } = useServerUrl()

export interface RequestOptions extends RequestInit {
  baseUrl?: string
  timeout?: number
  params?: Record<string, string | number | boolean>
}

export interface ApiResponse<T = any> {
  code: number
  data: T
  message?: string
}

/**
 * 获取认证 token
 */
export const getToken = (): string | null => {
  return localStorage.getItem('token')
}

/**
 * 通用 HTTP 请求封装
 * @param url 请求路径
 * @param options 请求配置
 * @returns API 响应数据
 */
let isHandling401 = false

export function onUnauthorized() {
  if (isHandling401) return
  isHandling401 = true
  QMessage.error('登录已过期，请重新登录', 5000)
  localStorage.removeItem('token')
  setTimeout(() => {
    window.location.reload()
    isHandling401 = false
  }, 1500)
}

export async function request<T = any>(
  url: string,
  options?: RequestOptions
): Promise<T> {
  const token = getToken()

  const headers: Record<string, string> = {}

  if (!options?.body || !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const baseUrl = options?.baseUrl || serverUrl.value
  let fullUrl = baseUrl.startsWith('http')
    ? `${baseUrl}${url}`
    : `${serverUrl.value}${url}`

  if (options?.params) {
    const searchParams = new URLSearchParams()
    Object.entries(options.params).forEach(([key, value]) => {
      searchParams.append(key, String(value))
    })
    const queryString = searchParams.toString()
    fullUrl += queryString ? `?${queryString}` : ''
  }

  const requestOptions: RequestInit = {
    ...options,
    headers: {
      ...headers,
      ...options?.headers
    }
  }

  const timeout = options?.timeout || 30000
  const controller = new AbortController()
  requestOptions.signal = controller.signal

  const timeoutId = setTimeout(() => controller.abort(), timeout)

  try {
    const response = await fetch(fullUrl, requestOptions)
    clearTimeout(timeoutId)

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))

      if (response.status === 401) {
        onUnauthorized()
        throw new Error('UNAUTHORIZED')
      }
      if (response.status === 403) {
        throw new Error(errorData.message || '权限不足，请检查您的权限')
      }
      if (response.status === 429) {
        const message = errorData.message || '请求过于频繁，请稍后再试'
        QMessage.warning(message, 5000)
        throw new Error(message)
      }
      throw new Error(errorData.message || `请求失败 (${response.status})`)
    }

    const data = await response.json()
    return data as T
  } catch (error) {
    clearTimeout(timeoutId)

    if (error instanceof Error && error.name === 'AbortError') {
      throw new Error('请求超时，请稍后重试')
    }

    throw error
  }
}

/**
 * 更新服务器地址
 */
export const updateServerUrl = (url: string) => {
  setServerUrl(url)
}

/**
 * useRequest composable - 提供请求相关的状态和方法
 */
export function useRequest() {
  const isRequesting = ref(false)
  const lastError = ref<string | null>(null)

  /**
   * 带错误处理的请求封装
   */
  const safeRequest = async <T = any>(
    url: string,
    options?: RequestOptions
  ): Promise<T | null> => {
    isRequesting.value = true
    lastError.value = null

    try {
      const result = await request<T>(url, options)
      return result
    } catch (error) {
      lastError.value = error instanceof Error ? error.message : '请求失败'
      console.error('请求失败:', error)
      return null
    } finally {
      isRequesting.value = false
    }
  }

  /**
   * GET 请求
   */
  const get = <T = any>(url: string, options?: RequestOptions): Promise<T | null> => {
    return safeRequest<T>(url, { ...options, method: 'GET' })
  }

  /**
   * POST 请求
   */
  const post = <T = any>(url: string, data?: any, options?: RequestOptions): Promise<T | null> => {
    return safeRequest<T>(url, {
      ...options,
      method: 'POST',
      body: data instanceof FormData ? data : JSON.stringify(data)
    })
  }

  /**
   * PUT 请求
   */
  const put = <T = any>(url: string, data?: any, options?: RequestOptions): Promise<T | null> => {
    return safeRequest<T>(url, {
      ...options,
      method: 'PUT',
      body: data instanceof FormData ? data : JSON.stringify(data)
    })
  }

  /**
   * DELETE 请求
   */
  const del = <T = any>(url: string, options?: RequestOptions): Promise<T | null> => {
    return safeRequest<T>(url, { ...options, method: 'DELETE' })
  }

  const hasError = computed(() => lastError.value !== null)

  return {
    // 状态
    serverUrl,
    isRequesting,
    lastError,
    hasError,

    // 方法
    request,
    safeRequest,
    get,
    post,
    put,
    delete: del,
    getToken,
    updateServerUrl
  }
}
