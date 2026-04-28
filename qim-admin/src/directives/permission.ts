import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

export const permissionDirective: Directive = {
  mounted(el: HTMLElement, binding: DirectiveBinding<string>) {
    const store = usePermissionStore()

    // 权限系统未初始化时放行（降级模式）
    if (!store.isInitialized) {
      return
    }

    const permission = binding.value
    const [resource, action] = permission.split(':')

    if (!store.hasPermission(resource, action)) {
      el.parentNode?.removeChild(el)
    }
  }
}
