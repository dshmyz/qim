import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { API_BASE_URL } from '@/config'

/**
 * API 响应数据结构
 */
export interface ApiResponse<T = any> {
  code: number
  data: T
  message?: string
}

/**
 * 请求配置扩展
 */
export interface ApiRequestConfig extends AxiosRequestConfig {
  /** 自定义 base URL（覆盖默认配置） */
  customBaseUrl?: string
  /** URL 查询参数 */
  params?: Record<string, string | number | boolean>
}

/**
 * 创建 axios 实例
 */
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

/**
 * 请求拦截器：自动附加 token
 */
apiClient.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

/**
 * 响应拦截器：统一处理错误
 */
apiClient.interceptors.response.use(
  (response: AxiosResponse) => response,
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

/**
 * 通用请求函数（兼容原有 useRequest 签名）
 * 支持自定义 base URL、URL 参数、FormData 等
 */
export async function request<T = any>(
  url: string,
  options?: ApiRequestConfig
): Promise<T> {
  const { customBaseUrl, params, ...axiosConfig } = options || {}

  // 合并配置
  const config: AxiosRequestConfig = {
    ...axiosConfig,
    url,
    params: { ...axiosConfig.params, ...params }
  }

  // 支持自定义 base URL
  if (customBaseUrl) {
    config.baseURL = customBaseUrl
  }

  // FormData 时删除 Content-Type 让浏览器自动设置 boundary
  if (config.data instanceof FormData) {
    delete config.headers?.['Content-Type']
  }

  const response: AxiosResponse<T> = await apiClient.request(config)
  return response.data
}

/**
 * GET 请求
 */
export function get<T = any>(url: string, config?: ApiRequestConfig): Promise<T> {
  return request<T>(url, { ...config, method: 'GET' })
}

/**
 * POST 请求
 */
export function post<T = any>(url: string, data?: any, config?: ApiRequestConfig): Promise<T> {
  return request<T>(url, { ...config, method: 'POST', data })
}

/**
 * PUT 请求
 */
export function put<T = any>(url: string, data?: any, config?: ApiRequestConfig): Promise<T> {
  return request<T>(url, { ...config, method: 'PUT', data })
}

/**
 * DELETE 请求
 */
export function del<T = any>(url: string, config?: ApiRequestConfig): Promise<T> {
  return request<T>(url, { ...config, method: 'DELETE' })
}

/**
 * 获取认证 token
 */
export function getToken(): string | null {
  return localStorage.getItem('token')
}

export default apiClient
