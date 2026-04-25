import { ref, computed } from 'vue'
import { API_BASE_URL } from '../config'
import {
  request as unifiedRequest,
  getToken as unifiedGetToken,
  type ApiRequestConfig
} from '@/utils/request'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

export interface RequestOptions extends ApiRequestConfig {
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
export const getToken = unifiedGetToken

/**
 * 通用 HTTP 请求封装（委托给统一 request 客户端）
 */
export async function request<T = any>(
  url: string,
  options?: RequestOptions
): Promise<T> {
  const baseUrl = options?.baseUrl || serverUrl.value

  const config: ApiRequestConfig = {
    ...options,
    customBaseUrl: baseUrl,
    // 将 Fetch API 的 body 映射为 axios 的 data
    data: options?.body,
    // 将 Fetch API 的 method 传递（axios 不区分大小写，但规范化更安全）
    method: options?.method?.toUpperCase() as ApiRequestConfig['method'],
    // 将 Fetch API 的 headers 合并
    headers: options?.headers as Record<string, string>
  }

  return unifiedRequest<T>(url, config)
}

/**
 * 更新服务器地址
 */
export const updateServerUrl = (url: string) => {
  serverUrl.value = url
  localStorage.setItem('serverUrl', url)
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
