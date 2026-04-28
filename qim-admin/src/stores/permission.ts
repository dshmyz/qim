import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Permission, Role } from '@/types'

export const usePermissionStore = defineStore('permission', () => {
  const permissions = ref<Permission[]>([])
  const roles = ref<Role[]>([])
  const isInitialized = ref(false)

  function setPermissions(newPermissions: Permission[]) {
    permissions.value = newPermissions
  }

  function setRoles(newRoles: Role[]) {
    roles.value = newRoles
  }

  function hasPermission(resource: string, action: string): boolean {
    // system_admin has all permissions
    if (roles.value.some(r => r.code === 'system_admin')) {
      return true
    }
    return permissions.value.some(
      p => p.resource === resource && p.actions.includes(action)
    )
  }

  function hasAnyPermission(checks: string[]): boolean {
    return checks.some(check => {
      const parts = check.split(':')
      if (parts.length !== 2) return false
      return hasPermission(parts[0], parts[1])
    })
  }

  function isRole(roleCode: string): boolean {
    return roles.value.some(r => r.code === roleCode)
  }

  function markInitialized() {
    isInitialized.value = true
  }

  function reset() {
    permissions.value = []
    roles.value = []
    isInitialized.value = false
  }

  return {
    permissions,
    roles,
    isInitialized,
    setPermissions,
    setRoles,
    hasPermission,
    hasAnyPermission,
    isRole,
    markInitialized,
    reset,
  }
})
