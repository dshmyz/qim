import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'
import ShortcutSettings from '@/components/settings/ShortcutSettings.vue'
import { DEFAULT_SHORTCUTS, type ShortcutsConfig } from '@/composables/useShortcuts'

const invokeMock = vi.fn()

function cloneShortcuts(shortcuts: ShortcutsConfig = DEFAULT_SHORTCUTS): ShortcutsConfig {
  return JSON.parse(JSON.stringify(shortcuts))
}

function mountSettingsPanel() {
  return mount(SettingsPanel, {
    props: {
      visible: true,
      currentUser: { username: 'alice' },
      serverUrl: 'http://localhost:8080',
      profile: {},
      messageSettings: {},
      appearanceSettings: {},
      advancedSettings: {},
      fileSettings: {},
    },
    global: {
      stubs: {
        Avatar: { template: '<div />' },
        ImageCropper: { template: '<div />' },
      },
    },
  })
}

describe('快捷键设置弹窗', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(window as any).electron = {
      ipcRenderer: {
        invoke: invokeMock,
      },
    }
    ;(window as any).$QMessage = {
      warning: vi.fn(),
      success: vi.fn(),
      error: vi.fn(),
    }
    invokeMock.mockImplementation((channel: string) => {
      if (channel === 'get-shortcuts') return Promise.resolve(cloneShortcuts())
      if (channel === 'reset-shortcuts') return Promise.resolve(cloneShortcuts())
      return Promise.resolve()
    })
  })

  it('保存时发现快捷键冲突应提示并阻止提交', async () => {
    const wrapper = mountSettingsPanel()
    ;(wrapper.vm as any).shortcutSettingsRef = {
      checkConflicts: () => [{
        a: { scope: 'global', name: 'maximize', label: '最大化/还原' },
        b: { scope: 'editor', name: 'link', label: '插入链接' },
      }],
    }

    await (wrapper.vm as any).save()

    expect((window as any).$QMessage.warning).toHaveBeenCalledWith(
      '快捷键冲突：「最大化/还原」与「插入链接」使用了相同的组合',
      5000,
    )
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('恢复默认只更新本地配置，等待设置弹窗保存后再持久化', async () => {
    const savedShortcuts = cloneShortcuts()
    savedShortcuts.editor.link.accelerator = 'Mod+Shift+K'
    invokeMock.mockImplementation((channel: string) => {
      if (channel === 'get-shortcuts') return Promise.resolve(savedShortcuts)
      if (channel === 'reset-shortcuts') return Promise.resolve(cloneShortcuts())
      return Promise.resolve()
    })

    const wrapper = mount(ShortcutSettings, {
      props: { modelValue: savedShortcuts },
      global: {
        stubs: {
          ShortcutInput: { template: '<div />' },
        },
      },
    })
    await flushPromises()

    await wrapper.find('.shortcut-reset-btn').trigger('click')

    expect(invokeMock).not.toHaveBeenCalledWith('reset-shortcuts')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual(cloneShortcuts())
    expect((window as any).$QMessage.success).toHaveBeenCalledWith('已恢复默认快捷键')
  })
})
