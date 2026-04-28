import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'
import { usePermissionStore } from '@/stores/permission'

export function setupPermissionGuard() {
  return (
    to: RouteLocationNormalized,
    _from: RouteLocationNormalized,
    next: NavigationGuardNext
  ) => {
    const token = localStorage.getItem('token')
    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !token) {
      next('/login')
      return
    }

    if (to.path === '/login' && token) {
      next('/')
      return
    }

    if (requiresAuth && token) {
      const permissionStore = usePermissionStore()
      const requiredPermission = to.meta.permission as string | undefined

      // 权限系统未初始化时的处理策略：
      // 1. 如果路由不需要特定权限（大部分页面），直接放行
      // 2. 如果需要权限但未初始化，避免重定向循环，改为放行（由指令控制显示）
      // 3. 登录页不检查权限
      if (requiredPermission && !permissionStore.isInitialized) {
        // 允许访问，但通过权限指令控制具体元素的显示
        // 避免因为权限未加载而反复重定向到登录页
      }

      if (requiredPermission && permissionStore.isInitialized) {
        const [resource, action] = requiredPermission.split(':')
        if (!permissionStore.hasPermission(resource, action)) {
          next('/403')
          return
        }
      }
    }

    next()
  }
}
