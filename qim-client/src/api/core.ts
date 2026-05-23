import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, AxiosError } from 'axios'
import { getStoredServerUrl } from '../composables/useServerUrl'

// API 错误类型
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

// 请求配置类型
export interface RequestConfig extends AxiosRequestConfig {
  retries?: number
  retryDelay?: number
  skipAuth?: boolean
}

// API 响应类型
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
    if (!config.headers || config.headers['skipAuth'] !== true) {
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

    // 处理 401 未授权
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.reload()
      throw new ApiError('登录已过期，请重新登录', 401, 401)
    }

    // 处理 403 权限不足
    if (error.response?.status === 403) {
      const message = error.response.data?.message || '权限不足'
      throw new ApiError(message, 403, 403)
    }

    // 处理 429 请求过于频繁
    if (error.response?.status === 429) {
      const message = error.response.data?.message || '请求过于频繁，请稍后再试'
      throw new ApiError(message, 429, 429)
    }

    // 处理网络错误
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
  const { retries, retryDelay, skipAuth, ...axiosConfig } = config || {}

  // 设置 skipAuth header
  if (skipAuth) {
    axiosConfig.headers = {
      ...axiosConfig.headers,
      'skipAuth': 'true'
    }
  }

  const requestFn = () => api.request<T>({
    url,
    ...axiosConfig
  })

  // 如果配置了重试，使用重试逻辑
  if (retries && retries > 0) {
    const response = await retryRequest(requestFn, retries, retryDelay)
    return (response as any).data
  }

  const response = await requestFn()
  return response.data
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
