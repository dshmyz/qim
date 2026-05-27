import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

function applyVisibility(el: HTMLElement, binding: DirectiveBinding<string>) {
  const store = usePermissionStore()

  // 权限系统未初始化时放行（降级模式）
  if (!store.isInitialized) {
    el.style.removeProperty('display')
    return
  }

  const permission = binding.value
  if (!permission) {
    return
  }
  const [resource, action] = permission.split(':')
  if (!resource || !action) {
    return
  }

  if (store.hasPermission(resource, action)) {
    el.style.removeProperty('display')
  } else {
    // 不直接 removeChild，避免破坏 Vue 的 vnode patch
    el.style.display = 'none'
  }
}

export const permissionDirective: Directive<HTMLElement, string> = {
  mounted: applyVisibility,
  updated: applyVisibility,
}
