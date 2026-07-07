# 快捷键统一管理与自定义

## 目标

将当前散落在 `main.js` 硬编码注册的 5 个全局快捷键和 `NoteEditor.vue` 硬编码的 4 个编辑器快捷键，统一收敛为可配置项。用户可在设置面板中修改快捷键组合、单独开关某项、或恢复默认值。保存时进行应用内冲突检测，防止快捷键之间互相覆盖或与编辑器按键冲突。

## 背景与问题

当前快捷键分布：

- 全局快捷键（`main.js` 的 `registerGlobalShortcuts()` 与 `initScreenshot()`）：`Cmd+M` 最小化、`Cmd+K` 最大化/还原、`Cmd+W` 隐藏、`Cmd+Q` 退出、`Cmd+Shift+A` 截图。
- 编辑器快捷键（`NoteEditor.vue` 的 `keymap.of([...])`）：`Mod-b` 加粗、`Mod-i` 斜体、`Mod-k` 链接、`Mod-s` 保存。

问题：

1. 无统一管理入口，用户无法修改或关闭快捷键。
2. 已知冲突：全局 `Cmd+K`（最大化）会拦截编辑器 `Mod-k`（插入链接），导致笔记内插入链接失效。
3. 与其他应用的全局快捷键冲突时，用户只能改源码。

## 不做（YAGNI）

- 不做系统级快捷键占用检测（Electron 无法可靠实现）。
- 不做快捷键导入/导出。
- 不做按设备记忆不同快捷键集。
- 不做快捷键分组/分类的二级 UI（当前 9 项足够，一个分页即可）。

## 架构

### 配置存储

存放在主进程 `config.json`（`app.getPath('userData')/config.json`），扩展现有 `loadServerConfig` / `saveServerConfig` 为通用 `loadConfig` / `saveConfig`。

```json
{
  "serverUrl": "...",
  "shortcuts": {
    "global": {
      "minimize":   { "accelerator": "CommandOrControl+M", "enabled": true },
      "maximize":   { "accelerator": "CommandOrControl+K", "enabled": true },
      "hide":       { "accelerator": "CommandOrControl+W", "enabled": true },
      "quit":       { "accelerator": "CommandOrControl+Q", "enabled": true },
      "screenshot": { "accelerator": "CommandOrControl+Shift+A", "enabled": true }
    },
    "editor": {
      "bold":   { "accelerator": "Mod-b", "enabled": true },
      "italic": { "accelerator": "Mod-i", "enabled": true },
      "link":   { "accelerator": "Mod-k", "enabled": true },
      "save":   { "accelerator": "Mod-s", "enabled": true }
    }
  }
}
```

**两个作用域的区别**：

- `global`：Electron accelerator 格式（`CommandOrControl+X`），由主进程 `globalShortcut.register` 注册。
- `editor`：CodeMirror `Mod-x` 格式，由渲染进程通过 `keymapCompartment` 配置。

**为什么不用 localStorage**：快捷键在主进程注册，主进程启动时直接读 config.json 最直接，避免渲染进程未启动时主进程拿不到配置的时序问题。

### 组件结构

```
SettingsPanel.vue
  └─ tab: shortcut
       └─ ShortcutSettings.vue        (新增，分页容器)
            ├─ ShortcutInput.vue      (新增，按键捕获输入框 + 开关)
            │    × N 项
            ├─ "恢复默认"按钮
            └─ "保存"按钮
```

遵循前端规则：抽成子组件，不往一个文件里堆。

### 数据流

```
[启动]
  main.js → loadConfig() → registerGlobalShortcuts(config.shortcuts.global)
         → mainWindow 加载完毕 → 渲染进程通过 IPC get-shortcuts 读取 → 初始化编辑器 keymap

[用户修改]
  ShortcutInput 捕获按键 → ShortcutSettings 收集 → useShortcuts.checkConflict()
    冲突 → 复用通用消息组件提示，阻止保存
    无冲突 → IPC set-shortcuts → 主进程 saveConfig + unregisterAll + 重新注册
           → 主进程 webContents.send('shortcuts-updated', config)
           → NoteEditor 监听到 → keymapCompartment.reconfigure()
```

## 详细设计

### 1. 主进程（main.js）

#### 1.1 配置读写

将现有 `loadServerConfig` / `saveServerConfig` 重构为：

```javascript
function loadConfig() {
  const configPath = getConfigPath()
  try {
    const raw = fs.readFileSync(configPath, 'utf-8')
    return JSON.parse(raw)
  } catch {
    return { serverUrl: '', shortcuts: getDefaultShortcuts() }
  }
}

function saveConfig(config) {
  fs.writeFileSync(getConfigPath(), JSON.stringify(config, null, 2))
}
```

原有读取 `serverUrl` 的位置改为 `loadConfig().serverUrl`。`getDefaultShortcuts()` 返回上面的默认结构。

#### 1.2 registerGlobalShortcuts 改造

```javascript
function registerGlobalShortcuts() {
  const { shortcuts } = loadConfig()
  const handlers = {
    minimize:   () => mainWindow?.minimize(),
    maximize:   () => mainWindow?.isMaximized() ? mainWindow.unmaximize() : mainWindow.maximize(),
    hide:       () => mainWindow?.hide(),
    quit:       () => app.quit(),
    screenshot: () => screenshotInstance?.startCapture?.()
  }
  for (const [name, conf] of Object.entries(shortcuts.global)) {
    if (!conf.enabled || !conf.accelerator) continue
    globalShortcut.register(conf.accelerator, handlers[name])
  }
}
```

`initScreenshot` 中的 `CommandOrControl+Shift+A` 注册移除，并入上面的 `screenshot` 项（需要保证 `screenshotInstance` 已初始化，见 §1.4）。

#### 1.3 IPC 接口

`set-shortcuts` 用 `ipcMain.handle`（而非 `on`），返回保存后的配置，便于前端确认保存成功并同步状态。

```javascript
ipcMain.handle('get-shortcuts', () => loadConfig().shortcuts)

ipcMain.handle('set-shortcuts', (event, shortcuts) => {
  const config = loadConfig()
  config.shortcuts = shortcuts
  saveConfig(config)
  globalShortcut.unregisterAll()
  registerGlobalShortcuts()
  mainWindow?.webContents.send('shortcuts-updated', shortcuts)
  return shortcuts
})

ipcMain.handle('reset-shortcuts', () => {
  const defaults = getDefaultShortcuts()
  const config = loadConfig()
  config.shortcuts = defaults
  saveConfig(config)
  globalShortcut.unregisterAll()
  registerGlobalShortcuts()
  mainWindow?.webContents.send('shortcuts-updated', defaults)
  return defaults
})
```

#### 1.4 截图快捷键的初始化顺序

当前 `initScreenshot` 在 `registerGlobalShortcuts` 之后调用（main.js:388-389），screenshot 的 handler 中用可选链 `screenshotInstance?.startCapture?.()` 兜底，即使顺序不调整也不会崩溃。但为语义清晰，调整为先 `initScreenshot()` 再 `registerGlobalShortcuts()`，保证注册时 screenshotInstance 已就绪。

### 2. 前端

#### 2.1 useShortcuts.ts（新 composable）

位置：`qim-client/src/composables/useShortcuts.ts`

职责：
- `loadShortcuts()`：`ipcRenderer.invoke('get-shortcuts')`
- `saveShortcuts(config)`：`ipcRenderer.invoke('set-shortcuts', config)`
- `resetShortcuts()`：`ipcRenderer.invoke('reset-shortcuts')`
- `normalizeAccelerator(accel)`：将 `CommandOrControl+X` 与 `Mod-x` 互转，用于跨域冲突检测
- `checkConflicts(shortcuts)`：返回冲突列表

```typescript
// 冲突检测核心逻辑
function checkConflicts(shortcuts) {
  const conflicts = []
  // 全局内部冲突
  // 编辑器内部冲突
  // 跨域冲突：Mod-x 与 CommandOrControl+x 视为相同
  return conflicts
}
```

跨域冲突判定：把 `Mod` 展开为 `CommandOrControl`，比较规范化后的组合。`Mod-b` 与 `CommandOrControl+B`（大小写不敏感）视为冲突。

#### 2.2 ShortcutInput.vue（新组件）

位置：`qim-client/src/components/settings/ShortcutInput.vue`

Props：
- `modelValue: { accelerator: string, enabled: boolean }`
- `label: string`

Emits：
- `update:modelValue`

行为：
- 右侧开关：切换 `enabled`，复用现有通用 `Switch.vue`
- 左侧输入框：显示友好格式（`CommandOrControl+Shift+A` → `⌘+Shift+A`），聚焦后监听 `keydown` 捕获组合
- 捕获规则：记录按下的修饰键 + 主键，组装为 accelerator 字符串；按 Esc 取消编辑，按 Backspace/Delete 清空
- 清空 accelerator 与 `enabled: false` 都导致该项不注册，两者独立：清空只是清掉组合键但保留 enabled 状态（用户可重新捕获），关闭开关则显式禁用该项。保存时两者任一满足即不注册

**friendly format 映射**：
- `CommandOrControl` / `Mod` → `⌘`
- `Shift` → `Shift`
- `Alt` / `Option` → `⌥`
- `Control` → `⌃`
- 字母键大写显示

#### 2.3 ShortcutSettings.vue（新组件）

位置：`qim-client/src/components/settings/ShortcutSettings.vue`

作为 SettingsPanel 的子组件，包含：
- 「全局快捷键」分组：5 项 ShortcutInput
- 「编辑器快捷键」分组：4 项 ShortcutInput
- 底部「恢复默认」按钮 + 「保存」按钮

加载时调用 `loadShortcuts()` 填充本地响应式副本，保存时先 `checkConflicts()`，有冲突则用现有通用消息组件提示并阻止保存，无冲突则 `saveShortcuts()`。

#### 2.4 SettingsPanel.vue 改动

在侧边栏新增 `shortcut` tab，内容区条件渲染 `<ShortcutSettings />`。

#### 2.5 NoteEditor.vue 改动

- 新增 `const keymapCompartment = new Compartment()`，参照现有 `themeCompartment`（L59）模式
- `createEditor()` 中 `keymap.of([...])` 改为 `keymapCompartment.of(keymap.of([...]))`
- 初始化时通过 `ipcRenderer.invoke('get-shortcuts')` 拉取 editor 配置，构建 keymap
- 监听 `shortcuts-updated` 事件，调用 `editorView.dispatch({ effects: keymapCompartment.reconfigure(...) })`
- 构建编辑器 keymap 时，`enabled: false` 的项不加入数组

### 3. 冲突检测规则

保存前检查三类冲突：

1. **全局快捷键之间**：两个全局项 accelerator 相同（大小写不敏感）。
2. **编辑器快捷键之间**：两个编辑器项 accelerator 相同。
3. **跨域冲突**：全局项的 `CommandOrControl+X` 与编辑器项的 `Mod-x` 冲突（`Mod` 在 macOS 上映射到 Cmd，等价于 `CommandOrControl`）。

冲突时通过通用消息组件提示「快捷键 X 与 Y 冲突，请修改」，阻止保存。

**不检测**系统级占用：`globalShortcut.register` 返回 false 仅说明注册失败，但全局快捷键仅在窗口失焦时触发，检测时机不可靠，不作为 UI 判定依据。若注册失败，控制台打印 warning。

## 测试要点

- 启动时 config.json 无 shortcuts 字段 → 使用默认值正常注册
- config.json 不存在 → 创建并写入默认值
- 修改全局快捷键 → 旧快捷键注销、新快捷键生效
- 关闭某项 → 该快捷键不响应
- 编辑器快捷键修改 → 不重启即可生效（reconfigure）
- 跨域冲突检测：设置全局 `Cmd+K` 与编辑器 `Mod-k` → 提示冲突
- 恢复默认 → 所有快捷键回到初始值
- 跨域冲突特例：默认配置中 `Cmd+K`（最大化）与 `Mod-k`（链接）冲突，默认配置本身就需要用户决策——见下方「默认配置冲突处理」

## 默认配置冲突处理

默认配置中存在 `Cmd+K`（全局最大化）与 `Mod-k`（编辑器链接）的跨域冲突。这是历史遗留，不是新引入的问题。处理方式：

- 默认值保持现状（不改用户已有行为）
- 首次进入快捷键设置页时，`checkConflicts` 会标记此冲突，UI 上以提示样式标注这两项
- 用户可自行修改其中一项来解决

## 涉及文件清单

| 文件 | 改动类型 |
|------|---------|
| `qim-client/electron/main.js` | 重构配置读写、改造 registerGlobalShortcuts、新增 IPC、调整初始化顺序 |
| `qim-client/electron/preload.cjs` | 无需改动（已有通用 ipcRenderer） |
| `qim-client/src/composables/useShortcuts.ts` | 新建 |
| `qim-client/src/components/settings/ShortcutInput.vue` | 新建 |
| `qim-client/src/components/settings/ShortcutSettings.vue` | 新建 |
| `qim-client/src/components/settings/SettingsPanel.vue` | 新增 shortcut tab |
| `qim-client/src/components/apps/notes/NoteEditor.vue` | keymapCompartment + IPC 监听 |
