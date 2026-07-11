import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'

const invokeMock = vi.fn()

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
    },
    global: {
      stubs: {
        Avatar: { template: '<div />' },
        AvatarCropper: { template: '<div />' },
        ShortcutSettings: { template: '<div />' },
      },
    },
  })
}

// 通过文本匹配点击侧边栏 Tab，避免 Tab 增减后索引变化导致测试失效
async function clickTab(wrapper: ReturnType<typeof mountSettingsPanel>, tabName: string) {
  const tab = wrapper.findAll('.settings-sidebar-item').find(item => item.text().includes(tabName))
  if (tab) await tab.trigger('click')
}

describe('SettingsPanel 清理与修补', () => {
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
      info: vi.fn(),
    }
    invokeMock.mockImplementation((channel: string) => {
      if (channel === 'get-shortcuts') return Promise.resolve({})
      return Promise.resolve()
    })
  })

  it('2FA 开关应被隐藏（A8）', async () => {
    const wrapper = mountSettingsPanel()
    // 切换到高级设置 Tab
    await clickTab(wrapper, '高级设置')
    // 2FA 相关文本不应出现在 DOM 中
    expect(wrapper.text()).not.toContain('双因素认证')
    expect(wrapper.text()).not.toContain('开启后，下次登录需要输入验证码')
  })

  it('账号安全占位按钮应已删除（A1）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    expect(wrapper.text()).not.toContain('账号安全')
    expect(wrapper.text()).not.toContain('查看安全设置')
  })
})
