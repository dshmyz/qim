import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { usePermissionStore } from '@/stores/permission'
import { usePermission } from '@/composables/usePermission'

describe('usePermission composable', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('canAccess returns true for permitted action', () => {
    const { canAccess } = usePermission()
    const permStore = usePermissionStore()
    permStore.setPermissions([{ resource: 'user', actions: ['create', 'read'] }])

    expect(canAccess('user:create')).toBe(true)
    expect(canAccess('user:read')).toBe(true)
    expect(canAccess('user:delete')).toBe(false)
  })

  it('canAccessAny returns true if any permission matches', () => {
    const { canAccessAny } = usePermission()
    const permStore = usePermissionStore()
    permStore.setPermissions([{ resource: 'user', actions: ['read'] }])

    expect(canAccessAny(['user:delete', 'user:read'])).toBe(true)
    expect(canAccessAny(['user:delete', 'group:create'])).toBe(false)
  })

  it('isAdmin returns true for system_admin', () => {
    const { isAdmin } = usePermission()
    const permStore = usePermissionStore()
    permStore.setRoles([
      { id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }
    ])

    expect(isAdmin()).toBe(true)
  })

  it('isAdmin returns false for non-admin role', () => {
    const { isAdmin } = usePermission()
    const permStore = usePermissionStore()
    permStore.setRoles([
      { id: 2, name: '编辑', code: 'editor', description: '', permissions: [], userCount: 0, createdAt: '' }
    ])

    expect(isAdmin()).toBe(false)
  })

  it('system_admin can access any resource via canAccess', () => {
    const { canAccess } = usePermission()
    const permStore = usePermissionStore()
    permStore.setPermissions([])
    permStore.setRoles([
      { id: 1, name: '管理员', code: 'system_admin', description: '', permissions: [], userCount: 0, createdAt: '' }
    ])

    expect(canAccess('anything:anything')).toBe(true)
  })
})
