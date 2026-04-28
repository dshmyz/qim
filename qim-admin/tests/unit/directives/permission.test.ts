import { describe, it, expect, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import type { DirectiveBinding } from 'vue'
import { usePermissionStore } from '@/stores/permission'
import { permissionDirective } from '@/directives/permission'

describe('v-permission directive', () => {
  let store: ReturnType<typeof usePermissionStore>

  beforeEach(() => {
    setActivePinia(createPinia())
    store = usePermissionStore()
  })

  it('should remove element when no permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBeNull()
  })

  it('should keep element when has permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['create'] }])
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBe(parent)
  })
})
