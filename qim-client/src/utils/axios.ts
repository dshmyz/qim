import axios from 'axios'
import { API_BASE_URL } from '../config'
import QMessage from '../utils/qmessage'

const axiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000
})

axiosInstance.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

axiosInstance.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response

      if (status === 401) {
        localStorage.removeItem('token')
        window.location.href = '/login'
        return Promise.reject(new Error('UNAUTHORIZED'))
      }

      if (status === 429) {
        const message = data?.message || '请求过于频繁，请稍后再试'
        QMessage.warning(message, 5000)
        return Promise.reject(new Error(message))
      }

      if (status === 403) {
        const message = data?.message || '权限不足，请检查您的权限'
        return Promise.reject(new Error(message))
      }

      const message = data?.message || `请求失败 (${status})`
      return Promise.reject(new Error(message))
    }

    if (error.code === 'ECONNABORTED') {
      return Promise.reject(new Error('请求超时，请稍后重试'))
    }

    return Promise.reject(error)
  }
)

export default axiosInstance
