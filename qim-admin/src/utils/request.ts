import axios from 'axios'
import type { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types'
import { usePermissionStore } from '@/stores/permission'

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
  (error) => {
    console.error('[Response] error:', error)
    const status = error.response?.status

    if (status === 401) {
      const permStore = usePermissionStore()
      permStore.reset()
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else if (status === 403) {
      ElMessage.error('权限不足，无法执行此操作')
    } else {
      const message = error.response?.data?.message || error.message || '网络异常'
      ElMessage.error(message)
    }

    return Promise.reject(error)
  }
)

export default service

export const request = <T = unknown>(config: AxiosRequestConfig): Promise<AxiosResponse<ApiResponse<T>>> => {
  return service(config)
}
