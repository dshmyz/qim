import { ref, computed } from 'vue'
import { useServerUrl } from './useServerUrl'
import QMessage from '../utils/qmessage'
import { APP_CONFIG } from '../config/appConfig'
import { detectPlatform } from '../utils/platform'

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
  pagination?: {
    current_page: number
    total_pages: number
    total: number
  }
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
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('user')
  setTimeout(() => {
    window.location.reload()
  }, 1500)
}

/**
 * 用 refresh_token 换取新的 access token（refresh_token 滑动续期）。
 * 成功返回新 access token 并写入 localStorage；失败返回 null。
 * 直接走裸 fetch，不经 request()，避免刷新请求自身再触发 401 处理。
 */
export async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) return null

  try {
    const response = await fetch(`${serverUrl.value}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${refreshToken}`
      }
    })
    if (!response.ok) return null
    const data = await response.json()
    if (data?.code === 0 && data?.data?.token) {
      localStorage.setItem('token', data.data.token)
      if (data.data.refresh_token) {
        localStorage.setItem('refresh_token', data.data.refresh_token)
      }
      return data.data.token
    }
    return null
  } catch {
    return null
  }
}

// 单飞：并发 401 只发起一次 refresh，其余排队等待结果，避免刷新风暴
let isRefreshingToken = false
let refreshWaiters: Array<(token: string | null) => void> = []

export async function refreshWithSingleFlight(): Promise<string | null> {
  if (isRefreshingToken) {
    return new Promise(resolve => refreshWaiters.push(resolve))
  }
  isRefreshingToken = true
  try {
    const token = await refreshAccessToken()
    refreshWaiters.forEach(waiter => waiter(token))
    refreshWaiters = []
    return token
  } finally {
    isRefreshingToken = false
  }
}

/** 401 内部标记：区分"token 失效"与其他业务错误，供刷新重试逻辑识别 */
class UnauthorizedError extends Error {
  constructor() {
    super('UNAUTHORIZED')
    this.name = 'UnauthorizedError'
  }
}

export async function request<T = any>(
  url: string,
  options?: RequestOptions
): Promise<T> {
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

  // 认证类接口本身返回 401 属于业务结果（如密码错误），不做刷新重试
  const isAuthEndpoint = /\/auth\/(login|register|refresh)/.test(url)

  const attempt = async (authToken: string | null): Promise<T> => {
    const headers: Record<string, string> = {}

    if (!options?.body || !(options.body instanceof FormData)) {
      headers['Content-Type'] = 'application/json'
    }

    if (authToken) {
      headers['Authorization'] = `Bearer ${authToken}`
    }
    // 上报客户端版本与平台，供服务端消息发送版本门槛校验（真实发送路径走 fetch，不经 axios 拦截器）
    headers['X-App-Version'] = APP_CONFIG.version
    headers['X-App-Platform'] = detectPlatform()

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

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))

        if (response.status === 401) {
          throw new UnauthorizedError()
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
      if (error instanceof Error && error.name === 'AbortError') {
        throw new Error('请求超时，请稍后重试')
      }

      throw error
    } finally {
      clearTimeout(timeoutId)
    }
  }

  try {
    return await attempt(getToken())
  } catch (error) {
    if (error instanceof UnauthorizedError) {
      // 认证类接口的 401 维持原有语义：直接登出
      if (isAuthEndpoint) {
        onUnauthorized()
        throw new Error('UNAUTHORIZED')
      }
      // access token 过期：先用 refresh_token 静默续期，成功则用新 token 重试一次原请求；
      // 重试仍 401 或刷新失败才登出，避免误杀仍可续期的会话
      const newToken = await refreshWithSingleFlight()
      if (newToken) {
        try {
          return await attempt(newToken)
        } catch (retryError) {
          if (retryError instanceof UnauthorizedError) {
            onUnauthorized()
            throw new Error('UNAUTHORIZED')
          }
          throw retryError
        }
      }
      onUnauthorized()
      throw new Error('UNAUTHORIZED')
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
