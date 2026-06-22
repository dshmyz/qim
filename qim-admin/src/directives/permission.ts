import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import { getCurrentUser } from '@/api/auth'

let isInitializing = false

// 初始化权限 store：拉取当前用户角色并标记为已初始化
// 权限判断基于角色（见 permission store 的 PERMISSION_ROLE_MAP），
// 因此只需设置 roles 即可，无需单独的 permissions 数组
async function ensureInitialized() {
  const store = usePermissionStore()
  if (store.isInitialized || isInitializing) return
  isInitializing = true
  try {
    const { data } = await getCurrentUser()
    if (data.data?.roles?.length) {
      store.setRoles(
        data.data.roles.map((r: string) => ({
          id: 0, name: r, code: r, description: '', permissions: [], userCount: 0, createdAt: '',
        }))
      )
    }
    store.markInitialized()
  } catch {
    // 获取用户信息失败也标记为已初始化，避免后续指令反复触发请求
    // 失败状态下 hasPermission 会因 roles 为空而返回 false，元素保持隐藏
    store.markInitialized()
  } finally {
    isInitializing = false
  }
}

function applyVisibility(el: HTMLElement, binding: DirectiveBinding<string>) {
  const store = usePermissionStore()

  const permission = binding.value
  if (!permission) return
  const [resource, action] = permission.split(':')
  if (!resource || !action) return

  if (!store.isInitialized) {
    // 权限未初始化时先隐藏元素，异步初始化完成后重新判断显隐
    el.style.display = 'none'
    ensureInitialized().then(() => {
      const s = usePermissionStore()
      if (s.hasPermission(resource, action)) {
        el.style.removeProperty('display')
      }
    })
    return
  }

  if (store.hasPermission(resource, action)) {
    el.style.removeProperty('display')
  } else {
    el.style.display = 'none'
  }
}

export const permissionDirective: Directive<HTMLElement, string> = {
  mounted: applyVisibility,
  updated: applyVisibility,
}
