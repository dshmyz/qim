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
      next('/admin')
      return
    }

    if (requiresAuth && token) {
      const permissionStore = usePermissionStore()
      const requiredPermission = to.meta.permission as string | undefined
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
