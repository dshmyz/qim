import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { usePermissionStore } from '@/stores/permission'
import type { Permission, Role } from '@/types'

describe('permission store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should initialize with empty permissions', () => {
    const store = usePermissionStore()
    expect(store.permissions).toEqual([])
    expect(store.roles).toEqual([])
  })

  it('should set permissions and roles', () => {
    const store = usePermissionStore()
    const perm: Permission = { resource: 'user', actions: ['create', 'read', 'update', 'delete'] }
    const role: Role = { id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }
    store.setPermissions([perm])
    store.setRoles([role])
    expect(store.permissions.length).toBe(1)
    expect(store.roles.length).toBe(1)
  })

  it('hasPermission returns true when permission exists', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['create', 'read'] }])
    expect(store.hasPermission('user', 'create')).toBe(true)
    expect(store.hasPermission('user', 'delete')).toBe(false)
  })

  it('hasAnyPermission returns true if any check passes', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    expect(store.hasAnyPermission(['user:delete', 'user:read'])).toBe(true)
    expect(store.hasAnyPermission(['user:delete', 'group:create'])).toBe(false)
  })

  it('hasAnyPermission returns false for malformed permission strings', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    expect(store.hasAnyPermission(['user'])).toBe(false)
    expect(store.hasAnyPermission([''])).toBe(false)
    expect(store.hasAnyPermission([])).toBe(false)
  })

  it('isRole returns true for matching role', () => {
    const store = usePermissionStore()
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    expect(store.isRole('system_admin')).toBe(true)
    expect(store.isRole('system_publisher')).toBe(false)
  })

  it('system_admin has all permissions', () => {
    const store = usePermissionStore()
    store.setPermissions([])
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    expect(store.hasPermission('anything', 'anything')).toBe(true)
  })

  it('reset clears all state', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    store.setRoles([{ id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }])
    store.markInitialized()
    store.reset()
    expect(store.permissions).toEqual([])
    expect(store.roles).toEqual([])
    expect(store.isInitialized).toBe(false)
  })
})
