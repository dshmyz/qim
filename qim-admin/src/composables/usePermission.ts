import { usePermissionStore } from '@/stores/permission'

/**
 * 权限组合式函数
 * 提供便捷的权限检查方法，封装 permission store 的底层逻辑
 */
export function usePermission() {
  const store = usePermissionStore()

  /**
   * 检查是否拥有指定权限
   * @param permission 权限字符串，格式为 resource:action（如 user:create）
   */
  function canAccess(permission: string): boolean {
    const [resource, action] = permission.split(':')
    return store.hasPermission(resource, action)
  }

  /**
   * 检查是否拥有任一指定权限
   * @param permissions 权限字符串数组
   */
  function canAccessAny(permissions: string[]): boolean {
    return store.hasAnyPermission(permissions)
  }

  /**
   * 检查是否为系统管理员
   */
  function isAdmin(): boolean {
    return store.isRole('system_admin')
  }

  return { canAccess, canAccessAny, isAdmin }
}
