import type { RouteLocationNormalized, NavigationGuardNext } from 'vue-router'
import { usePermissionStore } from '@/stores/permission'
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
    const token = sessionStorage.getItem('token')
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

      // 页面刷新后权限未初始化时，先获取用户信息
      if (!permissionStore.isInitialized) {
        await ensurePermissionsInitialized()
      }

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
