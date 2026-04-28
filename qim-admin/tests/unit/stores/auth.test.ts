import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAuthStore } from '@/stores/auth'
import { createPinia, setActivePinia } from 'pinia'

describe('auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  describe('初始化状态', () => {
    it('应该在没有 token 时初始化空状态', () => {
      const store = useAuthStore()
      expect(store.token).toBe('')
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('应该从 localStorage 加载已有 token', () => {
      localStorage.setItem('token', 'existing-token')
      const store = useAuthStore()
      expect(store.token).toBe('existing-token')
      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('setToken', () => {
    it('应该设置 token 并保存到 localStorage', () => {
      const store = useAuthStore()
      store.setToken('new-token-123')
      expect(store.token).toBe('new-token-123')
      expect(localStorage.getItem('token')).toBe('new-token-123')
      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('setUser', () => {
    it('应该设置用户信息', () => {
      const store = useAuthStore()
      const mockUser = {
        id: 1, username: 'admin', email: 'admin@example.com',
        avatar: 'https://example.com/avatar.png', role: 'admin',
        createdAt: '2024-01-01T00:00:00Z',
      }
      store.setUser(mockUser)
      expect(store.user).toEqual(mockUser)
    })
  })

  describe('logout', () => {
    it('应该清除认证状态', () => {
      const store = useAuthStore()
      store.setToken('test-token')
      store.setUser({
        id: 1, username: 'admin', email: 'admin@example.com',
        avatar: '', role: 'admin', createdAt: '2024-01-01T00:00:00Z',
      })

      store.logout()

      expect(store.token).toBe('')
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('token')).toBeNull()
    })
  })
})
