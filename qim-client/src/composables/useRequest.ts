import { ref } from 'vue'
import { API_BASE_URL } from '../config'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

const getToken = (): string | null => {
  return localStorage.getItem('token')
}

export interface RequestOptions extends RequestInit {
  baseUrl?: string
}

export async function request<T = any>(
  url: string,
  options?: RequestOptions
): Promise<T> {
  const token = getToken()
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options?.headers
  }

  const baseUrl = options?.baseUrl || serverUrl.value
  const fullUrl = `${baseUrl}${url}`

  const response = await fetch(fullUrl, {
    ...options,
    headers
  })

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}))
    if (response.status === 403) {
      throw new Error(errorData.message || '权限不足，请检查您的权限')
    }
    throw new Error(errorData.message || '请求失败')
  }

  return response.json()
}

export function useRequest() {
  return {
    serverUrl,
    request,
    getToken
  }
}
