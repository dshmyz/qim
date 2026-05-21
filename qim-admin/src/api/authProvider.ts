import client from './client'
import type { AuthProvider } from '@/types/auth'

export const getAuthProviders = () => {
  return client.get<{ data: AuthProvider[] }>('/admin/auth/providers')
}

export const createAuthProvider = (data: Partial<AuthProvider>) => {
  return client.post('/admin/auth/providers', data)
}

export const updateAuthProvider = (id: number, data: Partial<AuthProvider>) => {
  return client.put(`/admin/auth/providers/${id}`, data)
}

export const testAuthProvider = (id: number, testData: { test_username: string; test_password: string }) => {
  return client.post(`/admin/auth/providers/${id}/test`, testData)
}
