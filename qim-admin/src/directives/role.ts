import type { Directive, DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'

// v-role 指令：基于角色码控制元素显隐
// 用法：v-role="'system_admin'" 或 v-role="['system_admin', 'system_publisher']"
// 与 v-permission 的区别：v-role 直接判断角色码，语义更直白；
// v-permission 内部通过权限码→角色映射表间接判断，保留是为了兼容现有模板。
function applyVisibility(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
  const store = usePermissionStore()
  const value = binding.value
  if (!value) return

  const roleCodes = Array.isArray(value) ? value : [value]
  if (roleCodes.length === 0) return

  if (!store.isInitialized) {
    // 权限未初始化时隐藏元素，等待初始化完成后再判断
    el.style.display = 'none'
    return
  }

  if (store.hasAnyRole(roleCodes)) {
    el.style.removeProperty('display')
  } else {
    el.style.display = 'none'
  }
}

export const roleDirective: Directive<HTMLElement, string | string[]> = {
  mounted: applyVisibility,
  updated: applyVisibility,
}
