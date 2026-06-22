import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'
import { usePermissionStore } from '@/stores/permission'
import { useAuthStore } from '@/stores/auth'
import { getCurrentUser } from '@/api/auth'

let isInitializing = false

async function ensurePermissionsInitialized(): Promise<void> {
  const permissionStore = usePermissionStore()
  if (permissionStore.isInitialized || isInitializing) return

  isInitializing = true
  try {
    const { data } = await getCurrentUser()
    if (data.data?.roles?.length) {
      permissionStore.setRoles(
        data.data.roles.map((r: string) => ({
          id: 0, name: r, code: r, description: '', permissions: [], userCount: 0, createdAt: '',
        }))
      )
    }
    permissionStore.markInitialized()
  } catch {
    // 获取用户信息失败，权限守卫会在下次检查时重试
  } finally {
    isInitializing = false
  }
}

export function setupPermissionGuard() {
  return async (
    to: RouteLocationNormalized,
    _from: RouteLocationNormalized,
    next: NavigationGuardNext
  ) => {
    const authStore = useAuthStore()
    const requiresAuth = to.meta.requiresAuth !== false

    if (requiresAuth && !authStore.isAuthenticated) {
      next('/login')
      return
    }

    if (to.path === '/login' && authStore.isAuthenticated) {
      next('/')
      return
    }

    if (requiresAuth && authStore.isAuthenticated) {
      const permissionStore = usePermissionStore()

      // 页面刷新后权限未初始化时，先获取用户信息
      if (!permissionStore.isInitialized) {
        await ensurePermissionsInitialized()
      }

      // 路由级角色校验：meta.roles 配置允许访问的角色码
      const requiredRoles = to.meta.roles as string[] | undefined
      if (requiredRoles && requiredRoles.length > 0 && permissionStore.isInitialized) {
        const hasAccess = requiredRoles.some(role => permissionStore.isRole(role))
        if (!hasAccess) {
          next('/403')
          return
        }
      }
    }

    next()
  }
}
