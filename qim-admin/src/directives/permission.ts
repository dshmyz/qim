import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import { getCurrentUser } from '@/api/auth'

let isInitializing = false

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
    // ignore
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
    // 权限未初始化时隐藏元素，异步初始化后重新判断
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
