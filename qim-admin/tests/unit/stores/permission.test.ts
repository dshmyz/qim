import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { usePermissionStore } from '@/stores/permission'
import type { Permission, Role } from '@/types'

const adminRole: Role = { id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }
const publisherRole: Role = { id: 2, name: '发布者', code: 'system_publisher', description: '', permissions: [], userCount: 0, createdAt: '' }

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
    store.setPermissions([perm])
    store.setRoles([adminRole])
    expect(store.permissions.length).toBe(1)
    expect(store.roles.length).toBe(1)
  })

  it('hasPermission returns true when user has required role', () => {
    const store = usePermissionStore()
    // user:read 映射到 system_admin 角色
    store.setRoles([adminRole])
    expect(store.hasPermission('user', 'read')).toBe(true)
    expect(store.hasPermission('user', 'create')).toBe(true)
  })

  it('hasPermission returns false when user lacks required role', () => {
    const store = usePermissionStore()
    // 未设置任何角色
    expect(store.hasPermission('user', 'read')).toBe(false)
  })

  it('hasPermission maps ai:read to system_admin or system_publisher', () => {
    const store = usePermissionStore()
    store.setRoles([publisherRole])
    expect(store.hasPermission('ai', 'read')).toBe(true)
    expect(store.hasPermission('user', 'read')).toBe(false) // publisher 无权管理用户
  })

  it('hasAnyPermission returns true if any check passes', () => {
    const store = usePermissionStore()
    store.setRoles([adminRole])
    // user:read 映射到 system_admin，user:delete 同样映射到 system_admin
    expect(store.hasAnyPermission(['user:delete', 'user:read'])).toBe(true)
  })

  it('hasAnyPermission returns false when no role matches', () => {
    const store = usePermissionStore()
    store.setRoles([publisherRole])
    // publisher 不能管理用户和群组
    expect(store.hasAnyPermission(['user:delete', 'group:create'])).toBe(false)
  })

  it('hasAnyPermission returns false for malformed permission strings', () => {
    const store = usePermissionStore()
    store.setRoles([adminRole])
    expect(store.hasAnyPermission(['user'])).toBe(false)
    expect(store.hasAnyPermission([''])).toBe(false)
    expect(store.hasAnyPermission([])).toBe(false)
  })

  it('isRole returns true for matching role', () => {
    const store = usePermissionStore()
    store.setRoles([adminRole])
    expect(store.isRole('system_admin')).toBe(true)
    expect(store.isRole('system_publisher')).toBe(false)
  })

  it('hasAnyRole returns true if user has any of the specified roles', () => {
    const store = usePermissionStore()
    store.setRoles([publisherRole])
    expect(store.hasAnyRole(['system_admin', 'system_publisher'])).toBe(true)
    expect(store.hasAnyRole(['system_admin'])).toBe(false)
  })

  it('system_admin has all permissions', () => {
    const store = usePermissionStore()
    store.setRoles([adminRole])
    // 未配置映射的权限码默认仅 system_admin 可访问
    expect(store.hasPermission('anything', 'anything')).toBe(true)
  })

  it('reset clears all state', () => {
    const store = usePermissionStore()
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    store.setRoles([adminRole])
    store.markInitialized()
    store.reset()
    expect(store.permissions).toEqual([])
    expect(store.roles).toEqual([])
    expect(store.isInitialized).toBe(false)
  })
})
