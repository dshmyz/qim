import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import ChatToolbar from '@/components/chat/ChatToolbar.vue'
import { DEFAULT_SHORTCUTS, type ShortcutsConfig } from '@/composables/useShortcuts'

const mountToolbar = (props: Record<string, unknown> = {}) =>
  mount(ChatToolbar, {
    props: { isElectron: false, showAiActions: false, ...props },
    global: {
      stubs: { ChatToolbarButton: { template: '<button @click="$emit(\'click\')"><slot/></button>', emits: ['click'] } },
    },
  })

describe('ChatToolbar code block button', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    ;(window as any).electron = undefined
  })

  it('renders a code block button', () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    expect(btn.exists()).toBe(true)
  })

  it('emits open-code-block when code block button is clicked', async () => {
    const wrapper = mountToolbar()
    const btn = wrapper.find('[title="代码块"]')
    await btn.trigger('click')
    expect(wrapper.emitted('open-code-block')).toBeTruthy()
    expect(wrapper.emitted('open-code-block')).toHaveLength(1)
  })

  it('shows the screenshot shortcut in the screenshot button title', async () => {
    const shortcuts = cloneShortcuts()
    ;(window as any).electron = {
      ipcRenderer: {
        invoke: vi.fn((channel: string) => {
          if (channel === 'get-shortcuts') return Promise.resolve(shortcuts)
          return Promise.resolve()
        }),
        on: vi.fn(),
        removeListener: vi.fn(),
      },
    }

    const wrapper = mountToolbar({ isElectron: true })
    await flushPromises()

    expect(wrapper.find('.screenshot-btn').attributes('title')).toBe('截图（Ctrl+Shift+A）')
  })

  it('refreshes the screenshot shortcut title when hovering the screenshot button', async () => {
    const shortcuts = cloneShortcuts()
    const invoke = vi.fn((channel: string) => {
      if (channel === 'get-shortcuts') return Promise.resolve(shortcuts)
      return Promise.resolve()
    })
    ;(window as any).electron = {
      ipcRenderer: {
        invoke,
        on: vi.fn(),
        removeListener: vi.fn(),
      },
    }

    const wrapper = mountToolbar({ isElectron: true })
    await flushPromises()
    shortcuts.global.screenshot.accelerator = 'CommandOrControl+Alt+S'

    await wrapper.find('.screenshot-btn').trigger('mouseenter')
    await flushPromises()

    expect(wrapper.find('.screenshot-btn').attributes('title')).toBe('截图（Ctrl+Alt+S）')
  })

  it('shows a visible screenshot shortcut tooltip on hover', async () => {
    const shortcuts = cloneShortcuts()
    ;(window as any).electron = {
      ipcRenderer: {
        invoke: vi.fn((channel: string) => {
          if (channel === 'get-shortcuts') return Promise.resolve(shortcuts)
          return Promise.resolve()
        }),
        on: vi.fn(),
        removeListener: vi.fn(),
      },
    }

    const wrapper = mountToolbar({ isElectron: true })
    await flushPromises()

    await wrapper.find('.screenshot-btn').trigger('mouseenter')
    await flushPromises()

    const tooltip = wrapper.find('[data-testid="screenshot-shortcut-tooltip"]')
    expect(tooltip.exists()).toBe(true)
    expect(tooltip.text()).toBe('截图（Ctrl+Shift+A）')
  })

  it('uses the same shortcut text for the region screenshot menu item', async () => {
    const shortcuts = cloneShortcuts()
    ;(window as any).electron = {
      ipcRenderer: {
        invoke: vi.fn((channel: string) => {
          if (channel === 'get-shortcuts') return Promise.resolve(shortcuts)
          return Promise.resolve()
        }),
        on: vi.fn(),
        removeListener: vi.fn(),
      },
    }

    const wrapper = mountToolbar({ isElectron: true })
    await flushPromises()

    // 菜单由 UniversalContextMenu 渲染，经 Teleport 挂到 body，且仅在打开时渲染
    await wrapper.find('.screenshot-dropdown-trigger').trigger('click')
    await flushPromises()

    const firstItem = document.body.querySelector('.ucm-item')
    expect(firstItem).not.toBeNull()
    expect(firstItem!.querySelector('.ucm-label')?.textContent).toBe('截图（Ctrl+Shift+A）')
  })
})

function cloneShortcuts(): ShortcutsConfig {
  return JSON.parse(JSON.stringify(DEFAULT_SHORTCUTS))
}
