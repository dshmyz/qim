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

  it('外观设置 Tab 包含 C4 主题跟随系统开关', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('跟随系统主题')
    const followSystemSwitch = wrapper.find('[data-testid="follow-system-theme-switch"]')
    expect(followSystemSwitch.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 语言切换', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('语言')
    const languageSelect = wrapper.find('[data-testid="language-select"]')
    expect(languageSelect.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 侧边栏显示开关', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('显示侧边栏')
    const sidebarSwitch = wrapper.find('[data-testid="show-sidebar-switch"]')
    expect(sidebarSwitch.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 聊天字体大小滑块', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('聊天字体')
    const chatFontSlider = wrapper.find('[data-testid="chat-font-size-slider"]')
    expect(chatFontSlider.exists()).toBe(true)
  })

  it('C4 跟随系统主题开关切换应更新 localAppearanceSettings', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    const followSystemSwitch = wrapper.find('[data-testid="follow-system-theme-switch"]')
    const before = (wrapper.vm as any).localAppearanceSettings.followSystemTheme
    await followSystemSwitch.setValue(!before)
    expect((wrapper.vm as any).localAppearanceSettings.followSystemTheme).toBe(!before)
  })

  it('C4 语言切换应更新 localAppearanceSettings', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    const languageSelect = wrapper.find('[data-testid="language-select"]')
    await languageSelect.setValue('en-US')
    expect((wrapper.vm as any).localAppearanceSettings.language).toBe('en-US')
  })

  it('C4 聊天字体滑块应更新 localAppearanceSettings', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    const chatFontSlider = wrapper.find('[data-testid="chat-font-size-slider"]')
    await chatFontSlider.setValue('18')
    expect((wrapper.vm as any).localAppearanceSettings.chatFontSize).toBe(18)
  })

  it('关于区域包含检查更新按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const checkUpdateBtn = wrapper.find('[data-testid="check-update-btn"]')
    expect(checkUpdateBtn.exists()).toBe(true)
  })

  it('点击检查更新按钮发出 checkUpdate 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const checkUpdateBtn = wrapper.find('[data-testid="check-update-btn"]')
    await checkUpdateBtn.trigger('click')
    expect(wrapper.emitted('checkUpdate')).toBeTruthy()
  })

  it('关于区域包含更新日志按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const changelogBtn = wrapper.find('[data-testid="open-changelog-btn"]')
    expect(changelogBtn.exists()).toBe(true)
  })

  it('点击更新日志按钮发出 openChangelog 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const changelogBtn = wrapper.find('[data-testid="open-changelog-btn"]')
    await changelogBtn.trigger('click')
    expect(wrapper.emitted('openChangelog')).toBeTruthy()
  })

  it('关于区域包含开源许可按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const licensesBtn = wrapper.find('[data-testid="open-licenses-btn"]')
    expect(licensesBtn.exists()).toBe(true)
  })

  it('点击开源许可按钮发出 openLicenses 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    await clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const licensesBtn = wrapper.find('[data-testid="open-licenses-btn"]')
    await licensesBtn.trigger('click')
    expect(wrapper.emitted('openLicenses')).toBeTruthy()
  })
})
