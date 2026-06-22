import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Permission, Role } from '@/types'

// 权限码到角色的映射表
// 当前后端权限模型是基于角色的（system_admin / system_publisher 等），
// 前端 v-permission="'user:read'" 这类细粒度权限码在后端无对应实现。
// 这里维护一份权限码 → 所需角色的映射，使现有 v-permission 调用保持工作，
// 同时支持 v-role="'system_admin'" 这种更直白的角色判断。
const PERMISSION_ROLE_MAP: Record<string, string[]> = {
  // 管理类操作统一要求 system_admin
  'user:read': ['system_admin'],
  'user:create': ['system_admin'],
  'user:update': ['system_admin'],
  'user:delete': ['system_admin'],
  'organization:read': ['system_admin'],
  'role:read': ['system_admin'],
  'group:read': ['system_admin'],
  'conversation:read': ['system_admin'],
  'channel:read': ['system_admin'],
  'app:read': ['system_admin'],
  'miniapp:read': ['system_admin'],
  'message:read': ['system_admin'],
  'notification:read': ['system_admin'],
  'blacklist:read': ['system_admin'],
  'sensitive:read': ['system_admin'],
  'log:read': ['system_admin'],
  'config:read': ['system_admin'],
  'auth:read': ['system_admin'],
  'org:read': ['system_admin'],
  'version:read': ['system_admin'],
  'file:read': ['system_admin'],
  'monitor:read': ['system_admin'],
  'feedback:read': ['system_admin'],
  'approval:read': ['system_admin'],
  // AI 相关：system_admin 或 system_publisher
  'ai:read': ['system_admin', 'system_publisher'],
}

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
    const code = `${resource}:${action}`
    const requiredRoles = PERMISSION_ROLE_MAP[code]
    if (!requiredRoles || requiredRoles.length === 0) {
      // 未配置映射的权限码，默认仅 system_admin 可访问
      return isRole('system_admin')
    }
    return requiredRoles.some(role => isRole(role))
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

  function hasAnyRole(roleCodes: string[]): boolean {
    return roleCodes.some(code => isRole(code))
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
    hasAnyRole,
    markInitialized,
    reset,
  }
})
