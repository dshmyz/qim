import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from 'axios'
import { getStoredServerUrl } from '../composables/useServerUrl'
import type { ApiResponse } from '../composables/useRequest'
import { onUnauthorized } from '../composables/useRequest'
import { requestInterceptor } from '../utils/requestInterceptor'

export class ApiError extends Error {
  constructor(
    message: string,
    public code: number,
    public statusCode: number,
    public data?: any
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export interface RequestConfig extends AxiosRequestConfig {
  retries?: number
  retryDelay?: number
  skipAuth?: boolean
  cache?: boolean
  cacheTTL?: number
}

interface RetryConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const DEFAULT_RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 10000,
}

declare module 'axios' {
  interface AxiosRequestConfig {
    __retryCount?: number
  }
}

function calculateRetryDelay(retryCount: number): number {
  const exponentialDelay = DEFAULT_RETRY_CONFIG.baseDelay * Math.pow(2, retryCount)
  const cappedDelay = Math.min(exponentialDelay, DEFAULT_RETRY_CONFIG.maxDelay)
  const jitter = Math.random() * 1000
  return cappedDelay + jitter
}

function shouldRetry(error: any): boolean {
  if (!error.response) {
    return true
  }
  const status = error.response.status
  return status >= 500 || status === 429
}

// 创建 axios 实例
const api: AxiosInstance = axios.create({
  baseURL: getStoredServerUrl(),
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器 - 动态设置 baseURL 和添加 token
api.interceptors.request.use(
  (config) => {
    config.baseURL = getStoredServerUrl()
    if (!config.headers || config.headers['skipAuth'] !== 'true') {
      const token = localStorage.getItem('token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 统一错误处理
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = localStorage.getItem('refresh_token')
  if (!refreshToken) return null

  try {
    const baseURL = getStoredServerUrl()
    const response = await axios.post(`${baseURL}/api/v1/auth/refresh`, {}, {
      headers: { 'Authorization': `Bearer ${refreshToken}` }
    })

    if (response.data?.code === 0 && response.data?.data?.token) {
      const newToken = response.data.data.token
      const newRefreshToken = response.data.data.refresh_token

      localStorage.setItem('token', newToken)
      if (newRefreshToken) {
        localStorage.setItem('refresh_token', newRefreshToken)
      }

      return newToken
    }
    return null
  } catch {
    return null
  }
}

api.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { code, data, message } = response.data

    if (code !== 0) {
      throw new ApiError(
        message || '请求失败',
        code,
        response.status,
        data
      )
    }

    return response
  },
  async (error: AxiosError<ApiResponse>) => {
    const config = error.config as RequestConfig

    if (error.response?.status === 401) {
      // 如果是 refresh 请求失败，直接登出
      if (config?.url?.includes('/auth/refresh')) {
        onUnauthorized()
        throw new ApiError('登录已过期，请重新登录', 401, 401)
      }

      // 尝试用 refresh_token 刷新
      if (!isRefreshing) {
        isRefreshing = true
        try {
          const newToken = await refreshAccessToken()
          if (newToken) {
            onTokenRefreshed(newToken)
            // 用新 token 重试原请求
            if (config) {
              config.headers = config.headers || {}
              config.headers.Authorization = `Bearer ${newToken}`
              return api.request(config)
            }
          } else {
            // 刷新失败，清理排队请求并登出
            refreshSubscribers = []
            onUnauthorized()
            throw new ApiError('登录已过期，请重新登录', 401, 401)
          }
        } catch (err) {
          // 异常时清理排队请求
          refreshSubscribers = []
          if (err instanceof ApiError) throw err
          onUnauthorized()
          throw new ApiError('登录已过期，请重新登录', 401, 401)
        } finally {
          isRefreshing = false
        }
      } else {
        // 正在刷新中，排队等待
        return new Promise((resolve) => {
          addRefreshSubscriber((newToken: string) => {
            if (config) {
              config.headers = config.headers || {}
              config.headers.Authorization = `Bearer ${newToken}`
              resolve(api.request(config))
            }
          })
        })
      }
    }

    if (error.response?.status === 403) {
      const message = error.response.data?.message || '权限不足'
      throw new ApiError(message, 403, 403)
    }

    const retryCount = config?.__retryCount || 0
    if (shouldRetry(error) && retryCount < DEFAULT_RETRY_CONFIG.maxRetries) {
      config.__retryCount = retryCount + 1
      const delay = calculateRetryDelay(retryCount)
      
      console.log(`[API] 第 ${config.__retryCount} 次重试，延迟 ${delay}ms`)
      
      await new Promise(resolve => setTimeout(resolve, delay))
      return api.request(config)
    }

    if (error.response?.status === 429) {
      const message = error.response.data?.message || '请求过于频繁，请稍后再试'
      throw new ApiError(message, 429, 429)
    }

    if (error.code === 'ECONNABORTED') {
      throw new ApiError('请求超时，请稍后重试', -1, 0)
    }

    if (!error.response) {
      throw new ApiError('网络连接失败，请检查网络', -1, 0)
    }

    // 处理业务错误
    const responseData = error.response.data
    if (responseData?.message) {
      throw new ApiError(
        responseData.message,
        responseData.code || error.response.status,
        error.response.status
      )
    }

    throw new ApiError(
      error.message || '请求失败',
      error.response?.status || -1,
      error.response?.status || 0
    )
  }
)

// 重试逻辑
async function retryRequest<T>(
  requestFn: () => Promise<AxiosResponse<T>>,
  retries: number = 3,
  retryDelay: number = 1000
): Promise<T> {
  let lastError: Error | null = null

  for (let i = 0; i < retries; i++) {
    try {
      const response = await requestFn()
      return response.data
    } catch (error) {
      lastError = error as Error

      // 如果是 ApiError 且不是网络错误或超时错误，不重试
      if (error instanceof ApiError && error.statusCode !== 0) {
        throw error
      }

      // 如果还有重试次数，等待后重试
      if (i < retries - 1) {
        await new Promise(resolve => setTimeout(resolve, retryDelay * (i + 1)))
      }
    }
  }

  throw lastError || new Error('请求失败')
}

// 通用请求方法
export async function request<T = any>(url: string, config?: RequestConfig): Promise<T> {
  const { retries, retryDelay, skipAuth, cache, cacheTTL, ...axiosConfig } = config || {}

  if (skipAuth) {
    axiosConfig.headers = {
      ...axiosConfig.headers,
      'skipAuth': 'true'
    }
  }

  const requestFn = async () => {
    const response = await api.request<T>({
      url,
      ...axiosConfig
    })
    return response.data
  }

  const method = axiosConfig.method || 'GET'
  const shouldUseCache = method === 'GET' && cache !== false

  if (shouldUseCache) {
    return requestInterceptor.request<T>(
      requestFn,
      url,
      { ...axiosConfig, cache, cacheTTL }
    )
  }

  if (retries && retries > 0) {
    const retryRequestFn = () => api.request<T>({ url, ...axiosConfig })
    const response = await retryRequest(retryRequestFn, retries, retryDelay)
    return (response as any).data
  }

  return requestFn()
}

// 便捷的 HTTP 方法
export const http = {
  async get<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return request<T>(url, { ...config, method: 'GET' })
  },

  async post<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return request<T>(url, { ...config, method: 'POST', data })
  },

  async put<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return request<T>(url, { ...config, method: 'PUT', data })
  },

  async delete<T = any>(url: string, config?: RequestConfig): Promise<T> {
    return request<T>(url, { ...config, method: 'DELETE' })
  },

  async patch<T = any>(url: string, data?: any, config?: RequestConfig): Promise<T> {
    return request<T>(url, { ...config, method: 'PATCH', data })
  }
}

// 文件上传（使用 FormData）
export async function uploadFile<T = any>(
  url: string,
  file: File | Blob,
  additionalData?: Record<string, any>,
  config?: Omit<RequestConfig, 'headers' | 'Content-Type'>
): Promise<T> {
  const formData = new FormData()
  formData.append('file', file)

  if (additionalData) {
    Object.entries(additionalData).forEach(([key, value]) => {
      formData.append(key, value as string | Blob)
    })
  }

  const response = await request<T>(url, {
    ...config,
    method: 'POST',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  })

  return response
}

// 分片上传
export async function* chunkedUpload(
  url: string,
  file: File,
  chunkSize: number = 2 * 1024 * 1024, // 2MB
  additionalData?: Record<string, any>
) {
  const totalChunks = Math.ceil(file.size / chunkSize)
  const fileHash = await calculateFileHash(file)

  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize
    const end = Math.min(start + chunkSize, file.size)
    const chunk = file.slice(start, end)

    const formData = new FormData()
    formData.append('chunk', chunk)
    formData.append('chunkIndex', String(i))
    formData.append('totalChunks', String(totalChunks))
    formData.append('fileHash', fileHash)
    formData.append('filename', file.name)

    if (additionalData) {
      Object.entries(additionalData).forEach(([key, value]) => {
        formData.append(key, value as string)
      })
    }

    const progress = Math.round(((i + 1) / totalChunks) * 100)
    yield { progress, chunkIndex: i, totalChunks }

    await request(url, {
      method: 'POST',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
  }
}

// 计算文件 hash（使用 Web Workers）
async function calculateFileHash(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const buffer = e.target?.result as ArrayBuffer
        const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
        const hashArray = Array.from(new Uint8Array(hashBuffer))
        const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('')
        resolve(hashHex)
      } catch (error) {
        reject(error)
      }
    }
    reader.onerror = () => reject(reader.error)
    reader.readAsArrayBuffer(file)
  })
}

export default api
export { api }
