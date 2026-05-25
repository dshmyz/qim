import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types'
import { usePermissionStore } from '@/stores/permission'

declare module 'axios' {
  interface AxiosRequestConfig {
    __retryCount?: number
  }
}

interface RetryConfig {
  maxRetries: number
  baseDelay: number
  maxDelay: number
}

const RETRY_CONFIG: RetryConfig = {
  maxRetries: 3,
  baseDelay: 1000,
  maxDelay: 10000,
}

function calculateRetryDelay(retryCount: number): number {
  const exponentialDelay = RETRY_CONFIG.baseDelay * Math.pow(2, retryCount)
  const cappedDelay = Math.min(exponentialDelay, RETRY_CONFIG.maxDelay)
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

const service: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

service.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    console.error('[Request] error:', error)
    return Promise.reject(error)
  }
)

service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.message || '请求失败')
      if (res.code === 401) {
        const permStore = usePermissionStore()
        permStore.reset()
        localStorage.removeItem('token')
        window.location.href = '/login'
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return response
  },
  async (error) => {
    console.error('[Response] error:', error)
    const status = error.response?.status
    const config = error.config

    if (status === 401) {
      const permStore = usePermissionStore()
      permStore.reset()
      localStorage.removeItem('token')
      window.location.href = '/login'
      return Promise.reject(error)
    }

    if (status === 403) {
      ElMessage.error('权限不足，无法执行此操作')
      return Promise.reject(error)
    }

    const retryCount = config?.__retryCount || 0
    if (shouldRetry(error) && retryCount < RETRY_CONFIG.maxRetries) {
      config.__retryCount = retryCount + 1
      const delay = calculateRetryDelay(retryCount)
      
      console.log(`[Request] 第 ${config.__retryCount} 次重试，延迟 ${delay}ms`)
      
      await new Promise(resolve => setTimeout(resolve, delay))
      return service(config)
    }

    const message = error.response?.data?.message || error.message || '网络异常'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default service

export const request = <T = unknown>(config: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return service(config)
}
