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

  it('should hide element when no permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['read'] }])
    store.markInitialized()
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.style.display).toBe('none')
  })

  it('should keep element when has permission', async () => {
    store.setPermissions([{ resource: 'user', actions: ['create'] }])
    store.markInitialized()
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:create' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBe(parent)
  })

  it('should keep element when permission system not initialized', async () => {
    const parent = document.createElement('div')
    const el = document.createElement('button')
    parent.appendChild(el)
    const binding = { value: 'user:read' } as DirectiveBinding
    permissionDirective.mounted!(el, binding)
    await nextTick()
    expect(el.parentNode).toBe(parent)
  })
})
