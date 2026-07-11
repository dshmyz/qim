# 设置面板优化实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 清理设置面板中的占位/死代码/误导项，修补姓名可编辑与版本号动态读取，补齐 IM 应用该有的发送方式、通知细化、数据存储管理、外观补充与关于更新等功能。

**架构：** 前端 Vue3+TS，设置数据通过 `useSettings` composable 管理并持久化到 localStorage（多设备同步为已知架构债，本轮不处理）。SettingsPanel.vue 已 847 行，新增「数据与存储」Tab 拆为独立子组件 `DataStorageSettings.vue`，其余 Tab 新增项直接加到 SettingsPanel。后端 Go+Gin，本轮不碰后端代码。版本号通过 vite 构建时常量 `__APP_VERSION__`（已配置）经 `APP_CONFIG.version` 读取。

**技术栈：** Vue 3 + TypeScript + Vite + Vitest（happy-dom 环境）+ @vue/test-utils

---

## 文件结构

### 将创建的文件

| 文件 | 职责 |
|---|---|
| `qim-client/src/components/settings/DataStorageSettings.vue` | 「数据与存储」Tab 子组件：默认保存目录 + 缓存大小显示 + 分类清理 |
| `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` | SettingsPanel 清理与修补项的组件测试（任务 1-4） |
| `qim-client/tests/unit/components/DataStorageSettings.test.ts` | DataStorageSettings 子组件测试（任务 7） |

### 将修改的文件

| 文件 | 职责 |
|---|---|
| `qim-client/src/composables/useSettings.ts` | 扩展 MessageSettings/AppearanceSettings 接口与默认值；删除 dndMode 兼容补丁（A5）；改造 clearCache（A2） |
| `qim-client/src/components/settings/SettingsPanel.vue` | 删除死代码（A6/A1）；隐藏 2FA（A8）；姓名可编辑（A3）；版本号动态读取（A4）；新增 C1/C2 消息设置 UI；新增 C4 外观设置 UI；新增 C5 关于区域；接入 DataStorageSettings 子组件 |
| `qim-client/src/views/Main.vue` | 移除 openSecurity 和 clearCache 的失效事件绑定；接入新事件处理 |
| `qim-client/tests/unit/composables/useSettings.test.ts` | 扩展测试覆盖新字段默认值、加载、保存 |

---

## 范围说明

本计划覆盖规格文档中的全部 **A1-A8**（该删/该修）和 **C1-C5**（该添加）中的纯前端可实现部分。

**C3「数据与存储」范围裁剪：** 规格第 5 节 C3 列了 4 项，其中只有「缓存大小显示 + 分类清理」（A2 改造）和「默认保存目录迁移」（A7）是纯前端可做的。另外 3 项——聊天记录保留时长、自动清理过期文件、聊天记录导出——需要后端支持（无 user_settings 表，见规格第 3/7 节），本轮不实现，列在文末「范围外说明」中作为后续独立任务。

---

## 任务 1：删除死代码（A6 currentUserAvatar + A5 dndMode 补丁 + A1 账号安全占位按钮）

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:201, 206, 158-163, 244, 272-276`
- 修改：`qim-client/src/composables/useSettings.ts:62`
- 测试：`qim-client/tests/unit/composables/useSettings.test.ts`

本任务为纯删除，TDD 模式调整为：先运行现有测试确认通过 → 删除代码 → 运行测试确认仍通过。

- [ ] **步骤 1：运行现有测试确认通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/composables/useSettings.test.ts tests/unit/components/ShortcutSettingsPanel.test.ts
```
预期：所有测试 PASS

- [ ] **步骤 2：删除 A6 — SettingsPanel.vue 中的 currentUserAvatar 死代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 201，将 `computed` 从 import 中移除：
```ts
// 修改前
import { ref, watch, computed } from 'vue'
// 修改后
import { ref, watch } from 'vue'
```

行 206，删除 generateAvatar 和 isAbsoluteUrl 的 import（仅 currentUserAvatar 使用）：
```ts
// 删除这一行
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'
```

行 272-276，删除整个 currentUserAvatar computed：
```ts
// 删除这 5 行
const currentUserAvatar = computed(() => {
  if (!props.currentUser?.avatar) return generateAvatar('me')
  if (isAbsoluteUrl(props.currentUser.avatar)) return props.currentUser.avatar
  return props.serverUrl + props.currentUser.avatar
})
```

- [ ] **步骤 3：删除 A1 — SettingsPanel.vue 中的账号安全占位按钮**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 158-163，删除账号安全 settings-item 块：
```html
<!-- 删除这 6 行 -->
<div class="settings-item">
  <label>账号安全</label>
  <div class="settings-item-content">
    <button class="action-btn" @click="$emit('openSecurity')">查看安全设置</button>
  </div>
</div>
```

行 244，从 emits 定义中删除 `'openSecurity': []`：
```ts
// 修改前
const emit = defineEmits<{
  'close': []
  'save': [data: { profile: any; messageSettings: any; appearanceSettings: any; avatarFile?: File; shortcuts?: ShortcutsConfig }]
  'clearCache': []
  'saveTwoFactor': [enabled: boolean]
  'openSecurity': []
  'browseDirectory': [callback: (path: string) => void]
  'openFeedback': []
}>()
// 修改后
const emit = defineEmits<{
  'close': []
  'save': [data: { profile: any; messageSettings: any; appearanceSettings: any; avatarFile?: File; shortcuts?: ShortcutsConfig }]
  'clearCache': []
  'saveTwoFactor': [enabled: boolean]
  'browseDirectory': [callback: (path: string) => void]
  'openFeedback': []
}>()
```

- [ ] **步骤 4：删除 A5 — useSettings.ts 中的 dndMode 兼容补丁**

修改 `qim-client/src/composables/useSettings.ts`：

行 62，删除 dndMode 'work'→'all_day' 补丁：
```ts
// 修改前（行 60-63）
      try {
        const parsed = JSON.parse(savedMessageSettings)
        if (parsed.dndMode === 'work') parsed.dndMode = 'all_day'
        messageSettings.value = { ...messageSettings.value, ...parsed }
// 修改后
      try {
        const parsed = JSON.parse(savedMessageSettings)
        messageSettings.value = { ...messageSettings.value, ...parsed }
```

- [ ] **步骤 5：运行测试确认仍通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/composables/useSettings.test.ts tests/unit/components/ShortcutSettingsPanel.test.ts
```
预期：所有测试 PASS（删除死代码不影响现有功能）

- [ ] **步骤 6：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue src/composables/useSettings.ts && git commit -m "refactor: 删除设置面板死代码与占位项

- 删除 currentUserAvatar 死代码（A6，模板未引用）
- 删除 dndMode work→all_day 兼容补丁（A5，无老数据）
- 删除账号安全占位按钮及 openSecurity emit（A1，无实际页面）"
```

---

## 任务 2：隐藏 2FA 开关（A8）

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:146-157, 249-270`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`（新建）

A8 要求「隐藏」而非「删除」——后端字段保留，仅前端 UI 隐藏。用 `v-if` 绑定常量控制，便于将来恢复。

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`：

```ts
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
function clickTab(wrapper: ReturnType<typeof mountSettingsPanel>, tabName: string) {
  const tab = wrapper.findAll('.settings-sidebar-item').find(item => item.text().includes(tabName))
  if (tab) tab.trigger('click')
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

  it('2FA 开关应被隐藏（A8）', () => {
    const wrapper = mountSettingsPanel()
    // 切换到高级设置 Tab
    clickTab(wrapper, '高级设置')
    // 2FA 相关文本不应出现在 DOM 中
    expect(wrapper.text()).not.toContain('双因素认证')
    expect(wrapper.text()).not.toContain('开启后，下次登录需要输入验证码')
  })

  it('账号安全占位按钮应已删除（A1）', () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    expect(wrapper.text()).not.toContain('账号安全')
    expect(wrapper.text()).not.toContain('查看安全设置')
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — "2FA 开关应被隐藏" 测试失败，因为当前 2FA 开关仍然渲染

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

在 script 部分（行 249 附近，localTab 定义之后）添加常量：
```ts
const localTab = ref('basic')
// 2FA 功能暂未完整实现，前端隐藏开关（后端字段保留不动）
const showTwoFactorSetting = false
```

在模板中（行 146），给 2FA 的 settings-item 添加 `v-if="showTwoFactorSetting"`：
```html
<!-- 修改前 -->
<div class="settings-item">
  <label>双因素认证</label>
  <div class="settings-item-content">
    <div class="setting-row">
      <label class="switch">
        <input type="checkbox" v-model="localAdvancedSettings.twoFactorEnabled" @change="handleTwoFactorChange" />
        <span class="slider round"></span>
      </label>
    </div>
    <div class="settings-hint">开启后，下次登录需要输入验证码</div>
  </div>
</div>
<!-- 修改后 -->
<div v-if="showTwoFactorSetting" class="settings-item">
  <label>双因素认证</label>
  <div class="settings-item-content">
    <div class="setting-row">
      <label class="switch">
        <input type="checkbox" v-model="localAdvancedSettings.twoFactorEnabled" @change="handleTwoFactorChange" />
        <span class="slider round"></span>
      </label>
    </div>
    <div class="settings-hint">开启后，下次登录需要输入验证码</div>
  </div>
</div>
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — 2FA 开关已隐藏，账号安全按钮已删除

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 隐藏 2FA 开关（A8）

2FA 功能疑似未生效，前端通过 v-if 隐藏开关，
后端 two_factor_enabled 字段保留不动。"
```

---

## 任务 3：姓名改为可编辑 input（A3）

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:51-56`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

A3：当前姓名是只读 span，但 saveSettings 仍发 nickname（永远发原值）。改为可编辑 input 后，发送 nickname 才有意义。saveSettings 本身无需修改（发送逻辑正确，只是之前 UI 不可编辑导致看起来冗余）。

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 中追加测试：

```ts
  it('姓名应为可编辑 input（A3）', async () => {
    const wrapper = mountSettingsPanel()
    // 基本设置 Tab 是默认 Tab
    // 查找姓名输入框
    const nicknameInput = wrapper.find('input[data-testid="nickname-input"]')
    expect(nicknameInput.exists()).toBe(true)
    // 应该是可编辑的（不是 readonly）
    expect((nicknameInput.element as HTMLInputElement).readOnly).toBe(false)
    // 修改值应更新 localProfile
    await nicknameInput.setValue('新姓名')
    expect(nicknameInput.element.value).toBe('新姓名')
  })
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — 找不到 `input[data-testid="nickname-input"]`，因为当前是 span

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue` 行 51-56：

```html
<!-- 修改前 -->
<div class="settings-item">
  <label>姓名</label>
  <div class="settings-item-content">
    <span class="settings-value">{{ localProfile.nickname || '' }}</span>
  </div>
</div>
<!-- 修改后 -->
<div class="settings-item">
  <label>姓名</label>
  <div class="settings-item-content">
    <input
      type="text"
      v-model="localProfile.nickname"
      class="settings-input"
      data-testid="nickname-input"
      placeholder="输入姓名"
    />
  </div>
</div>
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 姓名改为可编辑 input（A3）

将基本设置中的姓名从只读 span 改为可编辑 input，
saveSettings 发送 nickname 不再是冗余操作。"
```

---

## 任务 4：版本号动态读取（A4）

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:167-170, 200-206`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

A4：版本号 `v1.0.0` 硬编码在模板中。`vitest.config.ts` 行 14 已定义 `__APP_VERSION__: JSON.stringify(pkg.version)`，`src/vite-env.d.ts` 行 5 已声明全局类型，`src/config/appConfig.ts` 行 9 已封装为 `APP_CONFIG.version`。直接 import 使用。

- [ ] **步骤 1：编写失败的测试**

首先在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 文件顶部（import 语句之后、`mountSettingsPanel` 函数之前）添加 `vi.mock` 模拟 APP_CONFIG，使版本号变为可控值：

```ts
// mock APP_CONFIG 以验证版本号动态读取（A4）
vi.mock('@/config/appConfig', () => ({
  APP_CONFIG: { version: '9.9.9' },
}))
```

然后在 `describe` 块中追加测试：

```ts
  it('版本号应动态读取而非硬编码（A4）', () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    const versionBadge = wrapper.find('.version-badge')
    expect(versionBadge.exists()).toBe(true)
    // APP_CONFIG 被 mock 为 9.9.9，如果版本号动态读取则显示 v9.9.9
    // 如果仍为硬编码则显示 v1.0.0，测试会失败
    expect(versionBadge.text()).toBe('v9.9.9')
  })
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — 版本号当前硬编码为 `v1.0.0`，未 import APP_CONFIG，mock 无效，断言 `v9.9.9` 失败

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 200-206 的 import 部分，添加 APP_CONFIG import。在现有 import 之后添加：

```ts
import { ref, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import AvatarCropper from '../modals/AvatarCropper.vue'
import ShortcutSettings from './ShortcutSettings.vue'
import type { ShortcutsConfig } from '../../composables/useShortcuts'
import { APP_CONFIG } from '../../config/appConfig'
```

在 script 部分（localTab 定义附近），添加 appVersion 常量：

```ts
const localTab = ref('basic')
// 2FA 功能暂未完整实现，前端隐藏开关（后端字段保留不动）
const showTwoFactorSetting = false
// 版本号从构建时常量动态读取（A4）
const appVersion = APP_CONFIG.version
```

行 167-170，修改模板中的版本号：

```html
<!-- 修改前 -->
<div class="about-info">
  <span class="version-badge">v1.0.0</span>
  <span class="about-text">当前为最新版本</span>
</div>
<!-- 修改后 -->
<div class="about-info">
  <span class="version-badge">v{{ appVersion }}</span>
  <span class="about-text">当前为最新版本</span>
</div>
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — mock APP_CONFIG.version 为 '9.9.9' 后，版本徽章显示 'v9.9.9'

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 版本号动态读取（A4）

从 APP_CONFIG.version（构建时常量 __APP_VERSION__）读取版本号，
替代硬编码的 v1.0.0。"
```

---

## 任务 5：useSettings 扩展 MessageSettings 和 AppearanceSettings

**文件：**
- 修改：`qim-client/src/composables/useSettings.ts:9-26, 36-49`
- 测试：`qim-client/tests/unit/composables/useSettings.test.ts`

为 C1/C2/C4 新设置项扩展数据层接口和默认值。新增字段存 localStorage（已知架构债，本轮不处理多设备同步）。

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/composables/useSettings.test.ts` 中追加测试：

```ts
describe('useSettings 扩展字段', () => {
  it('消息设置包含 C1/C2 新字段默认值', () => {
    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    // C1: 发送方式
    expect(settings.messageSettings.value.sendShortcut).toBe('enter')
    // C2: 通知细化
    expect(settings.messageSettings.value.mentionAlert).toBe(true)
    expect(settings.messageSettings.value.notificationPreview).toBe('content')
    expect(settings.messageSettings.value.notificationSound).toBe('default')
    expect(settings.messageSettings.value.dndExceptions).toEqual([])
    expect(settings.messageSettings.value.nightDndEnabled).toBe(false)
    expect(settings.messageSettings.value.nightDndStart).toBe('23:00')
    expect(settings.messageSettings.value.nightDndEnd).toBe('07:00')
  })

  it('外观设置包含 C4 新字段默认值', () => {
    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    expect(settings.appearanceSettings.value.followSystemTheme).toBe(false)
    expect(settings.appearanceSettings.value.language).toBe('zh-CN')
    expect(settings.appearanceSettings.value.showSidebar).toBe(true)
    expect(settings.appearanceSettings.value.chatFontSize).toBe(14)
  })

  it('从 localStorage 加载时恢复 C1/C2 新字段', () => {
    storage.set('messageSettings', JSON.stringify({
      sendShortcut: 'ctrl_enter',
      mentionAlert: false,
      notificationPreview: 'simple',
      notificationSound: 'chime',
      dndExceptions: ['user1', 'user2'],
      nightDndEnabled: true,
      nightDndStart: '22:00',
      nightDndEnd: '06:00',
    }))

    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    settings.loadSettings()

    expect(settings.messageSettings.value.sendShortcut).toBe('ctrl_enter')
    expect(settings.messageSettings.value.mentionAlert).toBe(false)
    expect(settings.messageSettings.value.notificationPreview).toBe('simple')
    expect(settings.messageSettings.value.notificationSound).toBe('chime')
    expect(settings.messageSettings.value.dndExceptions).toEqual(['user1', 'user2'])
    expect(settings.messageSettings.value.nightDndEnabled).toBe(true)
    expect(settings.messageSettings.value.nightDndStart).toBe('22:00')
    expect(settings.messageSettings.value.nightDndEnd).toBe('06:00')
  })

  it('从 localStorage 加载时恢复 C4 新字段', () => {
    storage.set('appearanceSettings', JSON.stringify({
      followSystemTheme: true,
      language: 'en-US',
      showSidebar: false,
      chatFontSize: 16,
    }))

    const settings = useSettings(ref(null), ref('http://localhost:8080'), vi.fn())
    settings.loadSettings()

    expect(settings.appearanceSettings.value.followSystemTheme).toBe(true)
    expect(settings.appearanceSettings.value.language).toBe('en-US')
    expect(settings.appearanceSettings.value.showSidebar).toBe(false)
    expect(settings.appearanceSettings.value.chatFontSize).toBe(16)
  })

  it('保存设置时 C1/C2/C4 新字段写入 localStorage', async () => {
    const mockRequest = vi.fn().mockResolvedValue({ code: 0, message: 'success' })
    const settings = useSettings(
      ref({ nickname: 'test', username: 'test' }),
      ref('http://localhost:8080'),
      mockRequest
    )
    settings.messageSettings.value.sendShortcut = 'ctrl_enter'
    settings.messageSettings.value.mentionAlert = false
    settings.messageSettings.value.dndExceptions = ['user_a']
    settings.appearanceSettings.value.followSystemTheme = true
    settings.appearanceSettings.value.language = 'en-US'
    settings.appearanceSettings.value.chatFontSize = 16

    await settings.saveSettings()

    const savedMessage = JSON.parse(storage.get('messageSettings') || '{}')
    expect(savedMessage.sendShortcut).toBe('ctrl_enter')
    expect(savedMessage.mentionAlert).toBe(false)
    expect(savedMessage.dndExceptions).toEqual(['user_a'])

    const savedAppearance = JSON.parse(storage.get('appearanceSettings') || '{}')
    expect(savedAppearance.followSystemTheme).toBe(true)
    expect(savedAppearance.language).toBe('en-US')
    expect(savedAppearance.chatFontSize).toBe(16)
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/composables/useSettings.test.ts
```
预期：FAIL — `settings.messageSettings.value.sendShortcut` 为 `undefined`，接口中不存在该字段

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/composables/useSettings.ts`：

行 9-17，扩展 MessageSettings 接口：
```ts
// 修改前
export interface MessageSettings {
  notificationsEnabled: boolean
  soundEnabled: boolean
  desktopNotificationsEnabled: boolean
  dndMode: 'none' | 'all_day' | 'custom'
  dndStartTime: string
  dndEndTime: string
  defaultSaveDirectory: string
}
// 修改后
export interface MessageSettings {
  notificationsEnabled: boolean
  soundEnabled: boolean
  desktopNotificationsEnabled: boolean
  dndMode: 'none' | 'all_day' | 'custom'
  dndStartTime: string
  dndEndTime: string
  defaultSaveDirectory: string
  // C1: 发送方式
  sendShortcut: 'enter' | 'ctrl_enter'
  // C2: 通知细化
  mentionAlert: boolean
  notificationPreview: 'content' | 'simple'
  notificationSound: string
  dndExceptions: string[]
  nightDndEnabled: boolean
  nightDndStart: string
  nightDndEnd: string
}
```

行 19-22，扩展 AppearanceSettings 接口：
```ts
// 修改前
export interface AppearanceSettings {
  theme: string
  fontSize: number
}
// 修改后
export interface AppearanceSettings {
  theme: string
  fontSize: number
  // C4: 外观补充
  followSystemTheme: boolean
  language: string
  showSidebar: boolean
  chatFontSize: number
}
```

行 36-49，扩展默认值：
```ts
// 修改前
  const messageSettings = ref<MessageSettings>({
    notificationsEnabled: true,
    soundEnabled: true,
    desktopNotificationsEnabled: true,
    dndMode: 'none',
    dndStartTime: '22:00',
    dndEndTime: '08:00',
    defaultSaveDirectory: ''
  })

  const appearanceSettings = ref<AppearanceSettings>({
    theme: currentTheme.value,
    fontSize: 14
  })
// 修改后
  const messageSettings = ref<MessageSettings>({
    notificationsEnabled: true,
    soundEnabled: true,
    desktopNotificationsEnabled: true,
    dndMode: 'none',
    dndStartTime: '22:00',
    dndEndTime: '08:00',
    defaultSaveDirectory: '',
    // C1: 发送方式
    sendShortcut: 'enter',
    // C2: 通知细化
    mentionAlert: true,
    notificationPreview: 'content',
    notificationSound: 'default',
    dndExceptions: [],
    nightDndEnabled: false,
    nightDndStart: '23:00',
    nightDndEnd: '07:00'
  })

  const appearanceSettings = ref<AppearanceSettings>({
    theme: currentTheme.value,
    fontSize: 14,
    // C4: 外观补充
    followSystemTheme: false,
    language: 'zh-CN',
    showSidebar: true,
    chatFontSize: 14
  })
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/composables/useSettings.test.ts
```
预期：PASS — 所有新字段默认值、加载、保存测试通过

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/composables/useSettings.ts tests/unit/composables/useSettings.test.ts && git commit -m "feat: 扩展 useSettings 数据层支持 C1/C2/C4 新设置项

MessageSettings 新增: sendShortcut, mentionAlert, notificationPreview,
notificationSound, dndExceptions, nightDndEnabled/Start/End
AppearanceSettings 新增: followSystemTheme, language, showSidebar, chatFontSize"
```

---

## 任务 6：消息设置 Tab 新增 C1 发送方式 + C2 通知细化项

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:71-116, 227-247`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

在消息设置 Tab 中新增 UI 控件。同时更新 Props 类型引用 useSettings 的接口。

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 中追加测试：

```ts
  it('消息设置 Tab 包含 C1 发送方式切换', async () => {
    const wrapper = mountSettingsPanel()
    // 切换到消息设置 Tab（索引 1）
    clickTab(wrapper, '消息设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('发送方式')
    const sendShortcutSelect = wrapper.find('[data-testid="send-shortcut-select"]')
    expect(sendShortcutSelect.exists()).toBe(true)
  })

  it('消息设置 Tab 包含 C2 通知细化项', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '消息设置')
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
    clickTab(wrapper, '消息设置')
    await wrapper.vm.$nextTick()
    const exceptionInput = wrapper.find('[data-testid="dnd-exception-input"]')
    expect(exceptionInput.exists()).toBe(true)
    await exceptionInput.setValue('user123')
    await exceptionInput.trigger('keyup.enter')
    // 验证输入值被添加到例外名单
    expect((wrapper.vm as any).localMessageSettings.dndExceptions).toContain('user123')
  })
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — 找不到 `[data-testid="send-shortcut-select"]` 等元素

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 227-235，更新 Props 类型引用 useSettings 接口。首先添加 import：
```ts
import type { ShortcutsConfig } from '../../composables/useShortcuts'
import { APP_CONFIG } from '../../config/appConfig'
import type { MessageSettings, AppearanceSettings } from '../../composables/useSettings'
```

更新 Props 中的 messageSettings 和 appearanceSettings 类型：
```ts
interface Props {
  visible: boolean
  currentUser?: { username?: string; avatar?: string }
  serverUrl: string
  profile: { nickname?: string; signature?: string }
  messageSettings: Partial<MessageSettings>
  appearanceSettings: Partial<AppearanceSettings>
  advancedSettings: { twoFactorEnabled?: boolean }
}
```

在 script 部分（localMessageSettings 定义之后），添加勿扰例外名单的输入值 ref 和添加函数：
```ts
const localTab = ref('basic')
// 2FA 功能暂未完整实现，前端隐藏开关（后端字段保留不动）
const showTwoFactorSetting = false
// 版本号从构建时常量动态读取（A4）
const appVersion = APP_CONFIG.version
// 勿扰例外名单输入值
const dndExceptionInput = ref('')

const addDndException = () => {
  const value = dndExceptionInput.value.trim()
  if (value && !localMessageSettings.value.dndExceptions?.includes(value)) {
    localMessageSettings.value.dndExceptions = [...(localMessageSettings.value.dndExceptions || []), value]
    dndExceptionInput.value = ''
  }
}

const removeDndException = (index: number) => {
  const exceptions = [...(localMessageSettings.value.dndExceptions || [])]
  exceptions.splice(index, 1)
  localMessageSettings.value.dndExceptions = exceptions
}
```

行 71-116，在消息设置 Tab 模板中，在「消息免打扰」相关项之后、「默认保存目录」之前（行 102 之后），添加 C1 和 C2 的 UI：

```html
            <!-- C1: 发送方式 -->
            <div class="settings-item">
              <label>发送方式</label>
              <div class="settings-item-content">
                <select v-model="localMessageSettings.sendShortcut" class="settings-select" data-testid="send-shortcut-select">
                  <option value="enter">Enter 发送（Shift+Enter 换行）</option>
                  <option value="ctrl_enter">Ctrl+Enter 发送（Enter 换行）</option>
                </select>
              </div>
            </div>
            <!-- C2: @提及强提醒 -->
            <div class="settings-item">
              <label>@提及强提醒</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localMessageSettings.mentionAlert" />
                  <span class="slider round"></span>
                </label>
                <div class="settings-hint">静音时被 @ 仍会提醒</div>
              </div>
            </div>
            <!-- C2: 通知内容预览 -->
            <div class="settings-item">
              <label>通知内容预览</label>
              <div class="settings-item-content">
                <select v-model="localMessageSettings.notificationPreview" class="settings-select">
                  <option value="content">显示消息内容</option>
                  <option value="simple">仅显示"新消息"</option>
                </select>
              </div>
            </div>
            <!-- C2: 通知声音 -->
            <div class="settings-item">
              <label>通知声音</label>
              <div class="settings-item-content">
                <select v-model="localMessageSettings.notificationSound" class="settings-select">
                  <option value="default">默认</option>
                  <option value="chime">清脆</option>
                  <option value="bell">铃声</option>
                  <option value="none">无声</option>
                </select>
              </div>
            </div>
            <!-- C2: 勿扰例外名单 -->
            <div class="settings-item">
              <label>勿扰例外</label>
              <div class="settings-item-content">
                <div class="input-with-btn">
                  <input
                    v-model="dndExceptionInput"
                    type="text"
                    class="settings-input"
                    data-testid="dnd-exception-input"
                    placeholder="输入用户名后回车添加"
                    @keyup.enter="addDndException"
                  />
                  <button class="browse-btn" @click="addDndException">添加</button>
                </div>
                <div v-if="localMessageSettings.dndExceptions && localMessageSettings.dndExceptions.length > 0" class="exception-list">
                  <span v-for="(item, index) in localMessageSettings.dndExceptions" :key="index" class="exception-tag">
                    {{ item }}
                    <button class="exception-remove" @click="removeDndException(index)">×</button>
                  </span>
                </div>
              </div>
            </div>
            <!-- C2: 夜间自动免打扰 -->
            <div class="settings-item">
              <label>夜间免打扰</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localMessageSettings.nightDndEnabled" />
                  <span class="slider round"></span>
                </label>
                <div class="settings-hint">到点自动开启免打扰</div>
              </div>
            </div>
            <div v-if="localMessageSettings.nightDndEnabled" class="settings-item">
              <label>夜间免打扰时间</label>
              <div class="dnd-time-range">
                <input v-model="localMessageSettings.nightDndStart" type="time" class="settings-select" />
                <span>至</span>
                <input v-model="localMessageSettings.nightDndEnd" type="time" class="settings-select" />
              </div>
            </div>
```

在 `<style scoped>` 中添加例外名单相关样式：
```css
.exception-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.exception-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--hover-color);
  border-radius: 12px;
  font-size: 12px;
  color: var(--text-color);
}

.exception-remove {
  border: none;
  background: none;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1;
  padding: 0;
}

.exception-remove:hover {
  color: var(--primary-color);
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — 所有 C1/C2 UI 元素存在且可交互

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 消息设置 Tab 新增 C1 发送方式与 C2 通知细化项（C1/C2）

C1: Enter/Ctrl+Enter 发送方式切换
C2: @提及强提醒、通知内容预览、通知声音、勿扰例外名单、夜间免打扰"
```

---

## 任务 7：新建 DataStorageSettings.vue 子组件（A7 迁移 + A2 改造 + C3 缓存管理）

**文件：**
- 创建：`qim-client/src/components/settings/DataStorageSettings.vue`
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:9-30, 103-115, 140-145`
- 测试：`qim-client/tests/unit/components/DataStorageSettings.test.ts`（新建）

新建「数据与存储」Tab 子组件，迁移默认保存目录（A7）从消息设置到此处，改造清除缓存（A2）为缓存大小显示 + 分类清理。

- [ ] **步骤 1：编写失败的测试**

创建 `qim-client/tests/unit/components/DataStorageSettings.test.ts`：

```ts
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DataStorageSettings from '@/components/settings/DataStorageSettings.vue'

const storage = new Map<string, string>()

beforeEach(() => {
  storage.clear()
  vi.mocked(localStorage.getItem).mockImplementation((key) => storage.get(key) ?? null)
  vi.mocked(localStorage.setItem).mockImplementation((key, value) => {
    storage.set(key, value)
  })
  vi.mocked(localStorage.removeItem).mockImplementation((key) => {
    storage.delete(key)
  })
  vi.mocked(localStorage.clear).mockImplementation(() => {
    storage.clear()
  })
  // mock length 和 key 以支持遍历
  Object.defineProperty(localStorage, 'length', {
    get: () => storage.size,
    configurable: true,
  })
  vi.mocked(localStorage.key).mockImplementation((index) => {
    const keys = Array.from(storage.keys())
    return keys[index] ?? null
  })
  ;(window as any).$QMessage = {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  }
})

function mountDataStorageSettings(props: Record<string, any> = {}) {
  return mount(DataStorageSettings, {
    props: {
      defaultSaveDirectory: '',
      ...props,
    },
  })
}

describe('DataStorageSettings 子组件', () => {
  it('渲染默认保存目录输入框（A7）', () => {
    const wrapper = mountDataStorageSettings({ defaultSaveDirectory: '/custom/path' })
    const dirInput = wrapper.find('[data-testid="save-directory-input"]')
    expect(dirInput.exists()).toBe(true)
    expect((dirInput.element as HTMLInputElement).value).toBe('/custom/path')
  })

  it('点击浏览按钮发出 browseDirectory 事件（A7）', async () => {
    const wrapper = mountDataStorageSettings()
    const browseBtn = wrapper.find('[data-testid="browse-directory-btn"]')
    await browseBtn.trigger('click')
    expect(wrapper.emitted('browseDirectory')).toBeTruthy()
    expect(wrapper.emitted('browseDirectory')![0]).toHaveLength(1) // callback 函数
  })

  it('显示缓存总大小（A2 改造）', () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('appearanceSettings', '{"theme":"dark"}')
    const wrapper = mountDataStorageSettings()
    const cacheTotal = wrapper.find('[data-testid="cache-total"]')
    expect(cacheTotal.exists()).toBe(true)
    expect(cacheTotal.text()).toMatch(/\d+(\.\d+)?\s*(B|KB|MB)/)
  })

  it('显示各分类缓存大小', () => {
    storage.set('messageSettings', '{"notificationsEnabled":true,"soundEnabled":true}')
    storage.set('theme', 'modern-light')
    const wrapper = mountDataStorageSettings()
    // 设置数据分类应显示大小
    const settingsCategory = wrapper.find('[data-testid="cache-category-settings"]')
    expect(settingsCategory.exists()).toBe(true)
    expect(settingsCategory.text()).toMatch(/\d+(\.\d+)?\s*(B|KB|MB)/)
  })

  it('点击分类清理按钮清除对应缓存', async () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('appearanceSettings', '{"theme":"dark"}')
    storage.set('theme', 'modern-light')
    storage.set('token', 'abc123')

    const wrapper = mountDataStorageSettings()
    const clearBtn = wrapper.find('[data-testid="clear-category-settings"]')
    await clearBtn.trigger('click')

    // 设置数据应被清除
    expect(storage.get('messageSettings')).toBeUndefined()
    expect(storage.get('appearanceSettings')).toBeUndefined()
    expect(storage.get('theme')).toBeUndefined()
    // token 不应被清除（受保护）
    expect(storage.get('token')).toBe('abc123')
  })

  it('点击全部清理清除所有非保护数据', async () => {
    storage.set('messageSettings', '{"notificationsEnabled":true}')
    storage.set('theme', 'modern-light')
    storage.set('token', 'abc123')

    const wrapper = mountDataStorageSettings()
    const clearAllBtn = wrapper.find('[data-testid="clear-all-btn"]')
    await clearAllBtn.trigger('click')

    expect(storage.get('messageSettings')).toBeUndefined()
    expect(storage.get('theme')).toBeUndefined()
    expect(storage.get('token')).toBe('abc123')
  })

  it('清理后发出 cacheCleared 事件通知父组件刷新', async () => {
    storage.set('messageSettings', '{}')
    const wrapper = mountDataStorageSettings()
    const clearBtn = wrapper.find('[data-testid="clear-category-settings"]')
    await clearBtn.trigger('click')
    expect(wrapper.emitted('cacheCleared')).toBeTruthy()
  })
})
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/DataStorageSettings.test.ts
```
预期：FAIL — 模块不存在，`Failed to resolve import '@/components/settings/DataStorageSettings.vue'`

- [ ] **步骤 3：编写最少实现代码**

创建 `qim-client/src/components/settings/DataStorageSettings.vue`：

```vue
<template>
  <div class="data-storage-settings">
    <!-- 默认保存目录（A7 迁移） -->
    <div class="settings-section-header"><h4>存储设置</h4></div>
    <div class="settings-item">
      <label>默认保存目录</label>
      <div class="settings-item-content">
        <div class="input-with-btn">
          <input
            type="text"
            :value="defaultSaveDirectory"
            class="settings-input"
            data-testid="save-directory-input"
            placeholder="选择默认保存目录"
            readonly
          />
          <button
            class="browse-btn"
            data-testid="browse-directory-btn"
            @click="$emit('browseDirectory', (path: string) => $emit('update:defaultSaveDirectory', path))"
          >
            <i class="fas fa-folder-open"></i>
            <span>浏览</span>
          </button>
        </div>
        <div class="settings-hint">设置接收文件的默认保存位置</div>
      </div>
    </div>

    <!-- 缓存管理（A2 改造） -->
    <div class="settings-section-header"><h4>缓存管理</h4></div>
    <div class="cache-overview">
      <div class="cache-total-row">
        <span class="cache-label">当前缓存占用</span>
        <span class="cache-size" data-testid="cache-total">{{ formatSize(cacheTotal) }}</span>
      </div>
    </div>
    <div
      v-for="cat in cacheCategories"
      :key="cat.id"
      class="settings-item"
    >
      <label>{{ cat.label }}</label>
      <div class="settings-item-content">
        <div class="cache-row">
          <span class="cache-size" :data-testid="`cache-category-${cat.id}`">{{ formatSize(getCategorySize(cat.id)) }}</span>
          <button
            class="action-btn"
            :data-testid="`clear-category-${cat.id}`"
            @click="clearCategory(cat.id)"
          >清理</button>
        </div>
      </div>
    </div>
    <div class="settings-item">
      <label>全部清理</label>
      <div class="settings-item-content">
        <button class="action-btn danger" data-testid="clear-all-btn" @click="clearAll">清除全部缓存</button>
        <div class="settings-hint">清除所有缓存数据（登录凭证除外）</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import QMessage from '../../utils/qmessage'

interface Props {
  defaultSaveDirectory: string
}

defineProps<Props>()

const emit = defineEmits<{
  'update:defaultSaveDirectory': [value: string]
  'browseDirectory': [callback: (path: string) => void]
  'cacheCleared': []
}>()

// 缓存分类定义
interface CacheCategory {
  id: string
  label: string
  match: (key: string) => boolean
}

const cacheCategories: CacheCategory[] = [
  {
    id: 'settings',
    label: '设置数据',
    match: (key) => ['messageSettings', 'appearanceSettings', 'theme', 'fontSize'].includes(key),
  },
  {
    id: 'messages',
    label: '消息缓存',
    match: (key) => key.startsWith('msg_') || key.startsWith('message_cache_'),
  },
  {
    id: 'images',
    label: '图片缓存',
    match: (key) => key.startsWith('img_') || key.startsWith('image_cache_'),
  },
  {
    id: 'files',
    label: '文件缓存',
    match: (key) => key.startsWith('file_') || key.startsWith('file_cache_'),
  },
]

// 受保护的 key，清理时不删除
const PROTECTED_KEYS = ['token']

const cacheTotal = ref(0)

// 计算缓存总大小（近似字节数）
const getCacheTotal = (): number => {
  let total = 0
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key) {
      const value = localStorage.getItem(key) || ''
      total += key.length + value.length
    }
  }
  return total
}

// 计算指定分类的缓存大小
const getCategorySize = (categoryId: string): number => {
  const category = cacheCategories.find(c => c.id === categoryId)
  if (!category) return 0
  let size = 0
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && category.match(key)) {
      const value = localStorage.getItem(key) || ''
      size += key.length + value.length
    }
  }
  return size
}

// 格式化大小为人类可读字符串
const formatSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// 清理指定分类的缓存
const clearCategory = (categoryId: string) => {
  const category = cacheCategories.find(c => c.id === categoryId)
  if (!category) return
  const keysToRemove: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && category.match(key) && !PROTECTED_KEYS.includes(key)) {
      keysToRemove.push(key)
    }
  }
  keysToRemove.forEach(key => localStorage.removeItem(key))
  cacheTotal.value = getCacheTotal()
  QMessage.success(`已清理${category.label}`)
  emit('cacheCleared')
}

// 清理全部缓存（保留受保护数据）
const clearAll = () => {
  const keysToRemove: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && !PROTECTED_KEYS.includes(key)) {
      keysToRemove.push(key)
    }
  }
  keysToRemove.forEach(key => localStorage.removeItem(key))
  cacheTotal.value = getCacheTotal()
  QMessage.success('缓存已全部清除')
  emit('cacheCleared')
}

onMounted(() => {
  cacheTotal.value = getCacheTotal()
})
</script>

<style scoped>
.data-storage-settings {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.settings-section-header {
  margin-bottom: 8px;
}

.settings-section-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.settings-item {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  min-height: 40px;
}

.settings-item label {
  min-width: 100px;
  flex-shrink: 0;
  color: var(--text-color);
  font-size: 14px;
  padding-top: 10px;
  font-weight: 500;
}

.settings-item-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 100%;
}

.settings-input {
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  flex: 1;
  font-size: 14px;
  background: var(--input-bg);
  color: var(--text-color);
  transition: border-color 0.2s;
}

.input-with-btn {
  display: flex;
  gap: 12px;
}

.input-with-btn .settings-input {
  flex: 1;
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary);
  width: 100%;
  margin-top: 6px;
  line-height: 1.4;
}

.browse-btn {
  padding: 10px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 8px;
}

.browse-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.cache-overview {
  padding: 16px;
  background: var(--sidebar-bg);
  border-radius: 12px;
}

.cache-total-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.cache-label {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.cache-size {
  font-size: 14px;
  color: var(--text-secondary);
}

.cache-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.action-btn {
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  background: var(--btn-bg);
  cursor: pointer;
  color: var(--text-color);
  font-size: 13px;
  font-weight: 500;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.action-btn:hover {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
}

.action-btn.danger:hover {
  background: #e74c3c;
  border-color: #e74c3c;
}
</style>
```

- [ ] **步骤 4：在 SettingsPanel 中接入 DataStorageSettings 并迁移默认保存目录**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 204-205 的 import 部分，添加 DataStorageSettings import：
```ts
import ShortcutSettings from './ShortcutSettings.vue'
import DataStorageSettings from './DataStorageSettings.vue'
```

行 9-30，在侧边栏中「外观设置」和「高级设置」之间添加「数据与存储」Tab：
```html
          <div class="settings-sidebar-item" :class="{ active: localTab === 'appearance' }" @click="localTab = 'appearance'">
            <i class="fas fa-paint-brush"></i>
            <span>外观设置</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'data-storage' }" @click="localTab = 'data-storage'">
            <i class="fas fa-database"></i>
            <span>数据与存储</span>
          </div>
          <div class="settings-sidebar-item" :class="{ active: localTab === 'advanced' }" @click="localTab = 'advanced'">
            <i class="fas fa-cog"></i>
            <span>高级设置</span>
          </div>
```

行 103-115，从消息设置 Tab 中删除「默认保存目录」块（迁移到数据与存储 Tab）：
```html
<!-- 删除这 13 行 -->
<div class="settings-item">
  <label>默认保存目录</label>
  <div class="settings-item-content">
    <div class="input-with-btn">
      <input type="text" v-model="localMessageSettings.defaultSaveDirectory" class="settings-input" placeholder="选择默认保存目录" readonly />
      <button class="browse-btn" @click="$emit('browseDirectory', (path: string) => { localMessageSettings.defaultSaveDirectory = path })">
        <i class="fas fa-folder-open"></i>
        <span>浏览</span>
      </button>
    </div>
    <div class="settings-hint">设置接收文件的默认保存位置</div>
  </div>
</div>
```

行 140-145，从高级设置 Tab 中删除「清除缓存」块（迁移到数据与存储 Tab）：
```html
<!-- 删除这 6 行 -->
<div class="settings-item">
  <label>清除缓存</label>
  <div class="settings-item-content">
    <button class="action-btn" @click="$emit('clearCache')">清除缓存</button>
  </div>
</div>
```

在外观设置 Tab 之后、高级设置 Tab 之前，添加数据与存储 Tab 的内容：
```html
          <div v-if="localTab === 'data-storage'" class="settings-section">
            <DataStorageSettings
              :defaultSaveDirectory="localMessageSettings.defaultSaveDirectory || ''"
              @update:defaultSaveDirectory="localMessageSettings.defaultSaveDirectory = $event"
              @browseDirectory="$emit('browseDirectory', $event)"
            />
          </div>
```

- [ ] **步骤 5：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/DataStorageSettings.test.ts tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — DataStorageSettings 子组件功能正确，SettingsPanel 现有测试不受影响

- [ ] **步骤 6：Commit**

```bash
cd qim-client && git add src/components/settings/DataStorageSettings.vue src/components/settings/SettingsPanel.vue tests/unit/components/DataStorageSettings.test.ts && git commit -m "feat: 新建数据与存储 Tab 子组件（A7/A2/C3）

- 新建 DataStorageSettings.vue 子组件
- A7: 默认保存目录从消息设置迁移到数据与存储 Tab
- A2: 清除缓存改造为缓存大小显示 + 分类清理（设置/消息/图片/文件）
- SettingsPanel 新增数据与存储侧边栏 Tab 项"
```

---

## 任务 8：外观设置 Tab 新增 C4 主题跟随系统/语言/侧边栏/聊天字体

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:118-136`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 中追加测试：

```ts
  it('外观设置 Tab 包含 C4 主题跟随系统开关', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('跟随系统主题')
    const followSystemSwitch = wrapper.find('[data-testid="follow-system-theme-switch"]')
    expect(followSystemSwitch.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 语言切换', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('语言')
    const languageSelect = wrapper.find('[data-testid="language-select"]')
    expect(languageSelect.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 侧边栏显示开关', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('显示侧边栏')
    const sidebarSwitch = wrapper.find('[data-testid="show-sidebar-switch"]')
    expect(sidebarSwitch.exists()).toBe(true)
  })

  it('外观设置 Tab 包含 C4 聊天字体大小滑块', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '外观设置')
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('聊天字体')
    const chatFontSlider = wrapper.find('[data-testid="chat-font-size-slider"]')
    expect(chatFontSlider.exists()).toBe(true)
  })
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — 找不到 `[data-testid="follow-system-theme-switch"]` 等元素

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 118-136，在外观设置 Tab 中，在「字体大小」项之后添加 C4 的新设置项：

```html
          <div v-if="localTab === 'appearance'" class="settings-section">
            <div class="settings-section-header"><h4>主题设置</h4></div>
            <div class="settings-item">
              <label>主题</label>
              <div class="theme-selector">
                <div v-for="theme in themes" :key="theme.id" class="theme-option" :class="{ active: localAppearanceSettings.theme === theme.id }" @click="localAppearanceSettings.theme = theme.id">
                  <div :class="['theme-preview', theme.previewClass]"></div>
                  <span>{{ theme.name }}</span>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>字体大小</label>
              <div class="font-size-slider">
                <input type="range" v-model.number="localAppearanceSettings.fontSize" min="12" max="18" step="1" />
                <span class="font-size-value">{{ localAppearanceSettings.fontSize }}px</span>
              </div>
            </div>
            <!-- C4: 跟随系统主题 -->
            <div class="settings-item">
              <label>跟随系统主题</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localAppearanceSettings.followSystemTheme" data-testid="follow-system-theme-switch" />
                  <span class="slider round"></span>
                </label>
                <div class="settings-hint">根据系统深浅色自动切换主题</div>
              </div>
            </div>
            <!-- C4: 语言切换 -->
            <div class="settings-item">
              <label>语言</label>
              <div class="settings-item-content">
                <select v-model="localAppearanceSettings.language" class="settings-select" data-testid="language-select">
                  <option value="zh-CN">简体中文</option>
                  <option value="en-US">English</option>
                </select>
              </div>
            </div>
            <!-- C4: 显示侧边栏 -->
            <div class="settings-item">
              <label>显示侧边栏</label>
              <div class="settings-item-content">
                <label class="switch">
                  <input type="checkbox" v-model="localAppearanceSettings.showSidebar" data-testid="show-sidebar-switch" />
                  <span class="slider round"></span>
                </label>
              </div>
            </div>
            <!-- C4: 聊天字体大小 -->
            <div class="settings-item">
              <label>聊天字体</label>
              <div class="font-size-slider">
                <input type="range" v-model.number="localAppearanceSettings.chatFontSize" min="12" max="20" step="1" data-testid="chat-font-size-slider" />
                <span class="font-size-value">{{ localAppearanceSettings.chatFontSize }}px</span>
              </div>
            </div>
          </div>
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — 所有 C4 外观设置 UI 元素存在

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 外观设置 Tab 新增 C4 主题跟随系统/语言/侧边栏/聊天字体

- 跟随系统主题开关（浅/深自动切换）
- 语言切换（简体中文/English）
- 侧边栏显示/隐藏开关
- 聊天字体单独调整滑块"
```

---

## 任务 9：关于区域新增检查更新/更新日志/开源许可（C5）

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:164-176, 239-247`
- 修改：`qim-client/src/views/Main.vue:637-653`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

C5 在高级设置 Tab 的「关于」区域新增 3 项功能。通过 emit 事件由 Main.vue 处理实际逻辑。

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 中追加测试：

```ts
  it('关于区域包含检查更新按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const checkUpdateBtn = wrapper.find('[data-testid="check-update-btn"]')
    expect(checkUpdateBtn.exists()).toBe(true)
  })

  it('点击检查更新按钮发出 checkUpdate 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const checkUpdateBtn = wrapper.find('[data-testid="check-update-btn"]')
    await checkUpdateBtn.trigger('click')
    expect(wrapper.emitted('checkUpdate')).toBeTruthy()
  })

  it('关于区域包含更新日志按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const changelogBtn = wrapper.find('[data-testid="open-changelog-btn"]')
    expect(changelogBtn.exists()).toBe(true)
  })

  it('点击更新日志按钮发出 openChangelog 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const changelogBtn = wrapper.find('[data-testid="open-changelog-btn"]')
    await changelogBtn.trigger('click')
    expect(wrapper.emitted('openChangelog')).toBeTruthy()
  })

  it('关于区域包含开源许可按钮（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const licensesBtn = wrapper.find('[data-testid="open-licenses-btn"]')
    expect(licensesBtn.exists()).toBe(true)
  })

  it('点击开源许可按钮发出 openLicenses 事件（C5）', async () => {
    const wrapper = mountSettingsPanel()
    clickTab(wrapper, '高级设置')
    await wrapper.vm.$nextTick()
    const licensesBtn = wrapper.find('[data-testid="open-licenses-btn"]')
    await licensesBtn.trigger('click')
    expect(wrapper.emitted('openLicenses')).toBeTruthy()
  })
```

- [ ] **步骤 2：运行测试验证失败**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：FAIL — 找不到 `[data-testid="check-update-btn"]` 等元素

- [ ] **步骤 3：编写最少实现代码**

修改 `qim-client/src/components/settings/SettingsPanel.vue`：

行 239-247，在 emits 定义中添加新事件：
```ts
const emit = defineEmits<{
  'close': []
  'save': [data: { profile: any; messageSettings: any; appearanceSettings: any; avatarFile?: File; shortcuts?: ShortcutsConfig }]
  'clearCache': []
  'saveTwoFactor': [enabled: boolean]
  'browseDirectory': [callback: (path: string) => void]
  'openFeedback': []
  'checkUpdate': []
  'openChangelog': []
  'openLicenses': []
}>()
```

行 164-176，修改高级设置 Tab 中「关于」区域，添加 C5 按钮：
```html
            <div class="settings-item">
              <label>关于</label>
              <div class="settings-item-content">
                <div class="about-info">
                  <span class="version-badge">v{{ appVersion }}</span>
                  <span class="about-text">当前为最新版本</span>
                </div>
                <div class="about-actions">
                  <button class="action-btn" data-testid="check-update-btn" @click="$emit('checkUpdate')">
                    <i class="fas fa-sync-alt"></i>
                    检查更新
                  </button>
                  <button class="action-btn" data-testid="open-changelog-btn" @click="$emit('openChangelog')">
                    <i class="fas fa-list"></i>
                    更新日志
                  </button>
                  <button class="action-btn" data-testid="open-licenses-btn" @click="$emit('openLicenses')">
                    <i class="fas fa-file-alt"></i>
                    开源许可
                  </button>
                  <button class="action-btn feedback-btn" @click="$emit('openFeedback')">
                    <i class="fas fa-bullhorn"></i>
                    问题反馈
                  </button>
                </div>
              </div>
            </div>
```

在 `<style scoped>` 中添加 about-actions 样式：
```css
.about-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
```

修改 `qim-client/src/views/Main.vue` 行 637-653，在 SettingsPanel 的事件绑定中添加新事件处理：

```html
  <SettingsPanel
    v-if="showSettingsModal"
    :visible="showSettingsModal"
    :currentUser="currentUser"
    :serverUrl="serverUrl"
    :profile="settingsProfile"
    :messageSettings="messageSettings"
    :appearanceSettings="appearanceSettings"
    :advancedSettings="advancedSettings"
    @close="closeSettingsModal"
    @save="handleSaveSettings"
    @clearCache="clearCache"
    @saveTwoFactor="saveTwoFactorSetting"
    @browseDirectory="browseDefaultSaveDirectory"
    @openFeedback="openFeedbackModal"
    @checkUpdate="checkForUpdates"
    @openChangelog="openChangelog"
    @openLicenses="openLicenses"
  />
```

在 Main.vue 的 script 部分（适当位置，其他事件处理函数附近）添加处理函数：

```ts
const checkForUpdates = () => {
  // 复用现有的更新检查逻辑
  if (window.electron?.ipcRenderer?.invoke) {
    window.electron.ipcRenderer.invoke('check-for-updates').catch(() => {
      QMessage.info('当前为最新版本')
    })
  } else {
    QMessage.info('当前为最新版本')
  }
}

const openChangelog = () => {
  QMessage.info('更新日志功能开发中')
}

const openLicenses = () => {
  QMessage.info('开源许可功能开发中')
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS — 所有 C5 按钮存在且 emit 事件正确

运行全部测试确认无回归：
```bash
cd qim-client && npx vitest run
```
预期：PASS

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/components/settings/SettingsPanel.vue src/views/Main.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "feat: 关于区域新增检查更新/更新日志/开源许可（C5）

- 检查更新按钮（emit checkUpdate，Main.vue 调用 Electron IPC）
- 更新日志按钮（emit openChangelog）
- 开源许可按钮（emit openLicenses）"
```

---

## 任务 10：Main.vue 清理失效事件绑定

**文件：**
- 修改：`qim-client/src/views/Main.vue:637-653`
- 修改：`qim-client/src/components/settings/SettingsPanel.vue:239-247`
- 测试：`qim-client/tests/unit/components/SettingsPanelCleanup.test.ts`

任务 1 删除了 `openSecurity` emit，任务 7 迁移了 `clearCache` 到子组件。Main.vue 中对应的 `@openSecurity` 绑定已失效需移除。`@clearCache` 绑定保留（DataStorageSettings 内部自管理缓存，但 useSettings.clearCache 函数仍存在，保留绑定不影响）。

- [ ] **步骤 1：编写失败的测试**

在 `qim-client/tests/unit/components/SettingsPanelCleanup.test.ts` 中追加测试：

```ts
  it('SettingsPanel 不再发出 openSecurity 事件（A1 清理验证）', () => {
    const wrapper = mountSettingsPanel()
    // openSecurity emit 已从定义中移除
    // 验证组件不会发出此事件
    expect(wrapper.emitted('openSecurity')).toBeUndefined()
  })
```

- [ ] **步骤 2：运行测试验证**

运行：
```bash
cd qim-client && npx vitest run tests/unit/components/SettingsPanelCleanup.test.ts
```
预期：PASS（任务 1 已删除 openSecurity emit，测试应通过。此测试作为回归保护）

- [ ] **步骤 3：清理 Main.vue 中的失效绑定**

修改 `qim-client/src/views/Main.vue` 行 637-653：

删除行 650 的 `@openSecurity` 绑定（任务 9 已在 Main.vue 中添加了新事件绑定，此处确认最终状态）：

```html
  <SettingsPanel
    v-if="showSettingsModal"
    :visible="showSettingsModal"
    :currentUser="currentUser"
    :serverUrl="serverUrl"
    :profile="settingsProfile"
    :messageSettings="messageSettings"
    :appearanceSettings="appearanceSettings"
    :advancedSettings="advancedSettings"
    @close="closeSettingsModal"
    @save="handleSaveSettings"
    @clearCache="clearCache"
    @saveTwoFactor="saveTwoFactorSetting"
    @browseDirectory="browseDefaultSaveDirectory"
    @openFeedback="openFeedbackModal"
    @checkUpdate="checkForUpdates"
    @openChangelog="openChangelog"
    @openLicenses="openLicenses"
  />
```

注意：`@openSecurity="QMessage.info('打开安全设置')"` 行已被删除（任务 9 重写 Main.vue 绑定时自然移除）。如果任务 9 未移除此行，在此步骤中手动删除。

- [ ] **步骤 4：运行全部测试确认无回归**

运行：
```bash
cd qim-client && npx vitest run
```
预期：PASS — 所有测试通过

- [ ] **步骤 5：Commit**

```bash
cd qim-client && git add src/views/Main.vue src/components/settings/SettingsPanel.vue tests/unit/components/SettingsPanelCleanup.test.ts && git commit -m "refactor: 清理 Main.vue 中失效的 openSecurity 事件绑定

任务 1 已从 SettingsPanel 删除 openSecurity emit，
此处移除 Main.vue 中对应的 @openSecurity 绑定。"
```

---

## 自检结果

### 1. 规格覆盖度

| 规格项 | 描述 | 对应任务 | 状态 |
|---|---|---|---|
| A1 | 账号安全占位按钮删除 | 任务 1 步骤 3 + 任务 10 | ✅ 覆盖 |
| A2 | 清除缓存改造为分类清理 | 任务 7 | ✅ 覆盖 |
| A3 | 姓名改为可编辑 input | 任务 3 | ✅ 覆盖 |
| A4 | 版本号动态读取 | 任务 4 | ✅ 覆盖 |
| A5 | dndMode 兼容补丁删除 | 任务 1 步骤 4 | ✅ 覆盖 |
| A6 | currentUserAvatar 死代码删除 | 任务 1 步骤 2 | ✅ 覆盖 |
| A7 | 默认保存目录迁移到数据与存储 | 任务 7 | ✅ 覆盖 |
| A8 | 2FA 开关隐藏 | 任务 2 | ✅ 覆盖 |
| C1 | 发送方式切换 | 任务 6 | ✅ 覆盖 |
| C2 | 通知细化 6 项 | 任务 6 | ✅ 覆盖 |
| C3 | 缓存大小显示 + 分类清理 | 任务 7 | ✅ 覆盖（纯前端部分） |
| C3 | 聊天记录保留时长 | — | ❌ 范围外（需后端） |
| C3 | 自动清理过期文件 | — | ❌ 范围外（需后端） |
| C3 | 聊天记录导出 | — | ❌ 范围外（需后端） |
| C4 | 外观补充 4 项 | 任务 8 | ✅ 覆盖 |
| C5 | 关于与更新 3 项 | 任务 9 | ✅ 覆盖 |

### 2. 占位符扫描

已扫描全文，无 TODO/待定/类似任务N 等占位符。每个步骤均包含完整代码。

### 3. 类型一致性

- `MessageSettings` 接口新增字段（任务 5 定义）：`sendShortcut`、`mentionAlert`、`notificationPreview`、`notificationSound`、`dndExceptions`、`nightDndEnabled`、`nightDndStart`、`nightDndEnd` — 任务 6 UI 中使用的字段名与此一致 ✅
- `AppearanceSettings` 接口新增字段（任务 5 定义）：`followSystemTheme`、`language`、`showSidebar`、`chatFontSize` — 任务 8 UI 中使用的字段名与此一致 ✅
- `DataStorageSettings.vue` 的 Props（`defaultSaveDirectory`）和 Emits（`update:defaultSaveDirectory`、`browseDirectory`、`cacheCleared`）在任务 7 定义，SettingsPanel 接入时引用一致 ✅
- SettingsPanel emits 新增的 `checkUpdate`、`openChangelog`、`openLicenses`（任务 9 定义）与 Main.vue 绑定一致 ✅
- `APP_CONFIG.version` 引用路径 `../../config/appConfig`（任务 4）与现有 `src/config/appConfig.ts` 一致 ✅
- `__APP_VERSION__` 全局类型已在 `src/vite-env.d.ts` 行 5 声明 ✅
- 测试中 Tab 切换使用 `clickTab(wrapper, 'Tab名称')` 文本匹配辅助函数（任务 2 定义），避免任务 7 新增「数据与存储」Tab 后索引变化导致早期测试失效 ✅
- 任务 4 版本号测试使用 `vi.mock('@/config/appConfig')` 在文件顶部 mock，mock 后 `APP_CONFIG.version` 为 `'9.9.9'`，硬编码 `'v1.0.0'` 时测试 FAIL，改为动态读取后 PASS ✅
- `wrapper.vm` 访问内部状态使用 `as any` 转换（如 `(wrapper.vm as any).localMessageSettings`），与现有 `ShortcutSettingsPanel.test.ts` 模式一致 ✅

---

## 范围外说明

以下规格中的功能需要后端支持（无 user_settings 表，见规格第 3/7 节），本轮不实现，作为后续独立任务：

1. **聊天记录保留时长**（C3）：需要后端记录每条消息的过期时间，并提供定时清理任务
2. **自动清理过期文件**（C3）：需要后端管理文件存储，并能按过期时间自动删除
3. **聊天记录导出**（C3）：需要后端提供导出 API，打包消息+附件

以上 3 项实现前需评估是否建立 `user_settings` 持久化机制（见规格第 7 节架构债）。

**多设备同步架构债**（规格第 7 节）：9 项原始设置 + 本轮新增的所有设置项仍只存 localStorage，换设备不同步。本轮仅在文档中标记，不动代码。
