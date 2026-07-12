import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsPanel from '@/components/settings/SettingsPanel.vue'

// mock APP_CONFIG 以验证版本号动态读取（A4）
vi.mock('@/config/appConfig', () => ({
  APP_CONFIG: { version: '9.9.9' },
}))

const invokeMock = vi.fn()

function mountSettingsPanel(overrides: { messageSettings?: any; appearanceSettings?: any } = {}) {
  return mount(SettingsPanel, {
    props: {
      visible: true,
      currentUser: { username: 'alice' },
      serverUrl: 'http://localhost:8080',
      profile: {},
      messageSettings: {},
      appearanceSettings: {},
      advancedSettings: {},
      ...overrides,
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
  expect(tab, `应找到 Tab：${tabName}`).toBeTruthy()
  await tab!.trigger('click')
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

  it('版本号应动态读取而非硬编码（A4）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    const versionBadge = wrapper.find('.version-badge')
    expect(versionBadge.exists()).toBe(true)
    // APP_CONFIG 被 mock 为 9.9.9，如果版本号动态读取则显示 v9.9.9
    // 如果仍为硬编码则显示 v1.0.0，测试会失败
    expect(versionBadge.text()).toBe('v9.9.9')
  })

  it('消息设置 Tab 包含 C1 发送方式切换', async () => {
    const wrapper = mountSettingsPanel()
    // 切换到消息设置 Tab
    await clickTab(wrapper, '消息设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('发送方式')
    const sendShortcutSelect = wrapper.find('[data-testid="send-shortcut-select"]')
    expect(sendShortcutSelect.exists()).toBe(true)
  })

  it('消息设置 Tab 包含 C2 通知细化项', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '消息设置')
    await wrapper.vm.$nextTick()
    // @提及强提醒
    expect(wrapper.text()).toContain('@提及强提醒')
    // 通知内容预览
    expect(wrapper.text()).toContain('通知内容预览')
    // 通知声音
    expect(wrapper.text()).toContain('通知声音')
    // 夜间自动免打扰
    expect(wrapper.text()).toContain('夜间免打扰')
  })

  it('勿扰例外名单输入应更新 localMessageSettings', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '消息设置')
    await wrapper.vm.$nextTick()
    const exceptionInput = wrapper.find('[data-testid="dnd-exception-input"]')
    expect(exceptionInput.exists()).toBe(true)
    await exceptionInput.setValue('user123')
    await exceptionInput.trigger('keyup.enter')
    // 验证输入值被添加到例外名单
    expect((wrapper.vm as any).localMessageSettings.dndExceptions).toContain('user123')
  })

  it('DataStorageSettings 缓存清理后 SettingsPanel 应向上 emit cacheCleared（A2）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '数据与存储')
    await wrapper.vm.$nextTick()
    // 找到清除全部缓存按钮
    const clearAllBtn = wrapper.find('[data-testid="clear-all-btn"]')
    expect(clearAllBtn.exists()).toBe(true)
    // mock confirm 返回 true
    window.confirm = vi.fn().mockReturnValue(true)
    await clearAllBtn.trigger('click')
    // SettingsPanel 应向上 emit cacheCleared
    expect(wrapper.emitted('cacheCleared')).toBeTruthy()
  })

  it('清缓存后 SettingsPanel 的 local 状态应同步刷新（端到端）', async () => {
    const wrapper = mountSettingsPanel({
      messageSettings: { sendShortcut: 'ctrl_enter', mentionAlert: false }
    })
    await clickTab(wrapper, '数据与存储')
    await wrapper.vm.$nextTick()
    // 模拟父组件 cacheCleared 后重新加载设置（messageSettings 被清空）
    await wrapper.setProps({ messageSettings: { sendShortcut: 'enter', mentionAlert: true } })
    await wrapper.vm.$nextTick()
    // localMessageSettings 应同步为新的 props 值
    expect((wrapper.vm as any).localMessageSettings.sendShortcut).toBe('enter')
    expect((wrapper.vm as any).localMessageSettings.mentionAlert).toBe(true)
  })
})
