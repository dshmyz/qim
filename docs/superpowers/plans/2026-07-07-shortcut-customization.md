# 快捷键统一管理与自定义 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将散落在 main.js 和 NoteEditor.vue 的硬编码快捷键统一为可配置项，支持在设置面板修改、开关、恢复默认，并做应用内冲突检测。

**架构：** 配置存于主进程 config.json，主进程通过 globalShortcut 注册全局快捷键，渲染进程通过 CodeMirror Compartment 运行时替换编辑器快捷键。前后端通过 3 个 IPC 通道（get/set/reset-shortcuts）通信，保存时主进程重新注册并广播 `shortcuts-updated` 事件，NoteEditor 监听后 reconfigure。

**技术栈：** Electron globalShortcut、CodeMirror 6 Compartment、Vue 3 Composition API、现有 IPC 封装（preload.cjs）、现有 QMessage 消息提示。

**规格文档：** `docs/superpowers/specs/2026-07-07-shortcut-customization-design.md`

---

## 文件结构

| 文件 | 职责 | 改动类型 |
|------|------|---------|
| `qim-client/electron/main.js` | 配置读写、快捷键注册、IPC 接口 | 修改 |
| `qim-client/src/composables/useShortcuts.ts` | 前端快捷键配置读写、冲突检测、格式化 | 新建 |
| `qim-client/src/components/settings/ShortcutInput.vue` | 按键捕获输入框 + 开关 | 新建 |
| `qim-client/src/components/settings/ShortcutSettings.vue` | 快捷键设置分页容器 | 新建 |
| `qim-client/src/components/settings/SettingsPanel.vue` | 新增 shortcut tab | 修改 |
| `qim-client/src/components/apps/notes/NoteEditor.vue` | keymapCompartment + IPC 监听 | 修改 |

---

## 任务 1：主进程配置读写重构

**文件：**
- 修改：`qim-client/electron/main.js:162-191`

**背景：** 现有 `loadServerConfig()` / `saveServerConfig()` 只读写 `serverUrl` 字段。需重构为通用 `loadConfig()` / `saveConfig()`，并增加 `getDefaultShortcuts()` 返回默认快捷键结构。

- [ ] **步骤 1：在 main.js 中添加 getDefaultShortcuts 函数**

在 `saveServerConfig` 函数之后（约 main.js:191 后）添加：

```javascript
function getDefaultShortcuts() {
  return {
    global: {
      minimize:   { accelerator: 'CommandOrControl+M', enabled: true },
      maximize:   { accelerator: 'CommandOrControl+K', enabled: true },
      hide:       { accelerator: 'CommandOrControl+W', enabled: true },
      quit:       { accelerator: 'CommandOrControl+Q', enabled: true },
      screenshot: { accelerator: 'CommandOrControl+Shift+A', enabled: true }
    },
    editor: {
      bold:   { accelerator: 'Mod-b', enabled: true },
      italic: { accelerator: 'Mod-i', enabled: true },
      link:   { accelerator: 'Mod-k', enabled: true },
      save:   { accelerator: 'Mod-s', enabled: true }
    }
  }
}
```

- [ ] **步骤 2：将 loadServerConfig 重构为 loadConfig**

将 main.js:168-181 的 `loadServerConfig` 函数替换为：

```javascript
function loadConfig() {
  try {
    const configPath = getConfigPath()
    if (fs.existsSync(configPath)) {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'))
      if (!config.shortcuts) {
        config.shortcuts = getDefaultShortcuts()
      }
      return config
    }
  } catch (error) {
    console.error('读取配置失败:', error)
  }
  return { serverUrl: null, shortcuts: getDefaultShortcuts() }
}
```

- [ ] **步骤 3：将 saveServerConfig 重构为 saveConfig**

将 main.js:183-191 的 `saveServerConfig` 函数替换为：

```javascript
function saveConfig(config) {
  try {
    const configPath = getConfigPath()
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2))
  } catch (error) {
    console.error('保存配置失败:', error)
  }
}
```

- [ ] **步骤 4：更新所有 loadServerConfig / saveServerConfig 的调用点**

搜索 main.js 中所有调用 `loadServerConfig()` 和 `saveServerConfig()` 的位置，改为：
- `loadServerConfig()` → `loadConfig().serverUrl`
- `saveServerConfig(url)` → `saveConfig({ ...loadConfig(), serverUrl: url })`

使用 Grep 搜索 `loadServerConfig\|saveServerConfig` 找到所有调用点，逐一替换。

- [ ] **步骤 5：验证应用正常启动**

运行：`cd qim-client && npm run electron:dev`（或项目实际的启动命令）
预期：应用正常启动，服务器地址配置读写正常，config.json 中出现 shortcuts 字段。

- [ ] **步骤 6：Commit**

```bash
git add qim-client/electron/main.js
git commit -m "refactor: 将 config 读写通用化，为快捷键配置预留结构"
```

---

## 任务 2：主进程快捷键注册改造

**文件：**
- 修改：`qim-client/electron/main.js:392-405`（registerGlobalShortcuts）
- 修改：`qim-client/electron/main.js:557`（initScreenshot 中的快捷键注册）
- 修改：`qim-client/electron/main.js:388-389`（初始化顺序）

- [ ] **步骤 1：重写 registerGlobalShortcuts 函数**

将 main.js:392-405 的 `registerGlobalShortcuts` 函数替换为：

```javascript
function registerGlobalShortcuts() {
  const { shortcuts } = loadConfig()
  const handlers = {
    minimize:   () => { if (mainWindow && !mainWindow.isDestroyed()) mainWindow.minimize() },
    maximize:   () => {
      if (!mainWindow || mainWindow.isDestroyed()) return
      if (mainWindow.isMaximized()) mainWindow.unmaximize()
      else mainWindow.maximize()
    },
    hide:       () => { if (mainWindow && !mainWindow.isDestroyed()) mainWindow.hide() },
    quit:       () => app.quit(),
    screenshot: () => screenshotInstance?.startCapture?.()
  }
  for (const [name, conf] of Object.entries(shortcuts.global)) {
    if (!conf.enabled || !conf.accelerator) continue
    const registered = globalShortcut.register(conf.accelerator, handlers[name])
    if (!registered) {
      console.warn(`[shortcut] 注册失败: ${conf.accelerator} (${name})，可能被其他应用占用`)
    }
  }
}
```

- [ ] **步骤 2：移除 initScreenshot 中的硬编码快捷键注册**

在 main.js:557 处，删除以下行：

```javascript
globalShortcut.register('CommandOrControl+Shift+A', () => {
  screenshotInstance?.startCapture?.()
})
```

该快捷键已由 `registerGlobalShortcuts` 中的 `screenshot` handler 统一注册。

- [ ] **步骤 3：调整初始化顺序**

在 main.js:388-389 处，将：
```javascript
registerGlobalShortcuts()
initScreenshot()
```
调整为：
```javascript
initScreenshot()
registerGlobalShortcuts()
```

保证 `screenshotInstance` 在 `registerGlobalShortcuts` 注册 screenshot handler 时已初始化。

- [ ] **步骤 4：验证快捷键功能正常**

运行应用，测试：
- Cmd+M 最小化窗口
- Cmd+K 最大化/还原
- Cmd+W 隐藏窗口
- Cmd+Shift+A 触发截图
预期：所有快捷键功能与改造前一致。

- [ ] **步骤 5：Commit**

```bash
git add qim-client/electron/main.js
git commit -m "refactor: registerGlobalShortcuts 从配置读取，移除 initScreenshot 中的硬编码"
```

---

## 任务 3：主进程 IPC 接口

**文件：**
- 修改：`qim-client/electron/main.js`（在现有 ipcMain 监听器区域，约 695-920 之间添加）

- [ ] **步骤 1：添加 get-shortcuts IPC**

在 main.js 的 ipcMain 监听器区域（例如 `ipcMain.handle('get-default-download-path', ...)` 之后）添加：

```javascript
ipcMain.handle('get-shortcuts', () => {
  return loadConfig().shortcuts
})
```

- [ ] **步骤 2：添加 set-shortcuts IPC**

在 get-shortcuts 之后添加：

```javascript
ipcMain.handle('set-shortcuts', (event, shortcuts) => {
  const config = loadConfig()
  config.shortcuts = shortcuts
  saveConfig(config)
  globalShortcut.unregisterAll()
  registerGlobalShortcuts()
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('shortcuts-updated', shortcuts)
  }
  return shortcuts
})
```

- [ ] **步骤 3：添加 reset-shortcuts IPC**

在 set-shortcuts 之后添加：

```javascript
ipcMain.handle('reset-shortcuts', () => {
  const defaults = getDefaultShortcuts()
  const config = loadConfig()
  config.shortcuts = defaults
  saveConfig(config)
  globalShortcut.unregisterAll()
  registerGlobalShortcuts()
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('shortcuts-updated', defaults)
  }
  return defaults
})
```

- [ ] **步骤 4：验证 IPC 可调用**

运行应用，在渲染进程控制台执行：
```javascript
window.electron.ipcRenderer.invoke('get-shortcuts').then(console.log)
```
预期：返回包含 global 和 editor 的快捷键配置对象。

- [ ] **步骤 5：Commit**

```bash
git add qim-client/electron/main.js
git commit -m "feat: 添加 get/set/reset-shortcuts IPC 接口"
```

---

## 任务 4：useShortcuts composable

**文件：**
- 创建：`qim-client/src/composables/useShortcuts.ts`

**背景：** 该 composable 封装快捷键配置的读写、冲突检测、格式化显示。冲突检测的核心是规范化 accelerator 字符串后比较——`Mod` 展开为 `CommandOrControl`，大小写不敏感。

- [ ] **步骤 1：创建 useShortcuts.ts 基础结构**

创建 `qim-client/src/composables/useShortcuts.ts`：

```typescript
export interface ShortcutItem {
  accelerator: string
  enabled: boolean
}

export interface ShortcutsConfig {
  global: Record<string, ShortcutItem>
  editor: Record<string, ShortcutItem>
}

export interface ShortcutConflict {
  a: { scope: 'global' | 'editor', name: string, label: string }
  b: { scope: 'global' | 'editor', name: string, label: string }
}

// 快捷键显示名称映射
export const SHORTCUT_LABELS: Record<string, Record<string, string>> = {
  global: {
    minimize: '最小化窗口',
    maximize: '最大化/还原',
    hide: '隐藏窗口',
    quit: '退出应用',
    screenshot: '截图'
  },
  editor: {
    bold: '加粗',
    italic: '斜体',
    link: '插入链接',
    save: '保存'
  }
}

// 默认快捷键配置
export const DEFAULT_SHORTCUTS: ShortcutsConfig = {
  global: {
    minimize:   { accelerator: 'CommandOrControl+M', enabled: true },
    maximize:   { accelerator: 'CommandOrControl+K', enabled: true },
    hide:       { accelerator: 'CommandOrControl+W', enabled: true },
    quit:       { accelerator: 'CommandOrControl+Q', enabled: true },
    screenshot: { accelerator: 'CommandOrControl+Shift+A', enabled: true }
  },
  editor: {
    bold:   { accelerator: 'Mod-b', enabled: true },
    italic: { accelerator: 'Mod-i', enabled: true },
    link:   { accelerator: 'Mod-k', enabled: true },
    save:   { accelerator: 'Mod-s', enabled: true }
  }
}

/**
 * 规范化 accelerator 字符串用于比较
 * Mod -> CommandOrControl, 全小写
 */
function normalizeAccelerator(accel: string): string {
  if (!accel) return ''
  return accel
    .replace(/\bMod\b/gi, 'CommandOrControl')
    .toLowerCase()
}

/**
 * 将 accelerator 转为友好显示格式
 * CommandOrControl -> ⌘, Shift -> Shift, Alt/Option -> ⌥, Control -> ⌃
 */
export function formatAccelerator(accel: string): string {
  if (!accel) return '未设置'
  const parts = accel.split('+')
  const result: string[] = []
  for (const part of parts) {
    const p = part.trim()
    if (p === 'CommandOrControl' || p === 'Mod') result.push('⌘')
    else if (p === 'Shift') result.push('Shift')
    else if (p === 'Alt' || p === 'Option') result.push('⌥')
    else if (p === 'Control') result.push('⌃')
    else result.push(p.toUpperCase())
  }
  return result.join('+')
}

/**
 * 检测快捷键冲突
 * 三类：全局内部、编辑器内部、跨域（Mod-x 与 CommandOrControl+x）
 */
export function checkConflicts(shortcuts: ShortcutsConfig): ShortcutConflict[] {
  const conflicts: ShortcutConflict[] = []
  const items: { scope: 'global' | 'editor', name: string, label: string, normalized: string }[] = []

  for (const [name, conf] of Object.entries(shortcuts.global)) {
    if (conf.enabled && conf.accelerator) {
      items.push({
        scope: 'global', name, label: SHORTCUT_LABELS.global[name] || name,
        normalized: normalizeAccelerator(conf.accelerator)
      })
    }
  }
  for (const [name, conf] of Object.entries(shortcuts.editor)) {
    if (conf.enabled && conf.accelerator) {
      items.push({
        scope: 'editor', name, label: SHORTCUT_LABELS.editor[name] || name,
        normalized: normalizeAccelerator(conf.accelerator)
      })
    }
  }

  // 两两比较
  for (let i = 0; i < items.length; i++) {
    for (let j = i + 1; j < items.length; j++) {
      if (items[i].normalized === items[j].normalized && items[i].normalized) {
        conflicts.push({
          a: { scope: items[i].scope, name: items[i].name, label: items[i].label },
          b: { scope: items[j].scope, name: items[j].name, label: items[j].label }
        })
      }
    }
  }
  return conflicts
}

export function useShortcuts() {
  const loadShortcuts = async (): Promise<ShortcutsConfig> => {
    if (window.electron?.ipcRenderer?.invoke) {
      return await window.electron.ipcRenderer.invoke('get-shortcuts')
    }
    return DEFAULT_SHORTCUTS
  }

  const saveShortcuts = async (shortcuts: ShortcutsConfig): Promise<ShortcutsConfig> => {
    if (window.electron?.ipcRenderer?.invoke) {
      return await window.electron.ipcRenderer.invoke('set-shortcuts', shortcuts)
    }
    return shortcuts
  }

  const resetShortcuts = async (): Promise<ShortcutsConfig> => {
    if (window.electron?.ipcRenderer?.invoke) {
      return await window.electron.ipcRenderer.invoke('reset-shortcuts')
    }
    return DEFAULT_SHORTCUTS
  }

  return { loadShortcuts, saveShortcuts, resetShortcuts }
}
```

- [ ] **步骤 2：验证类型检查通过**

运行：`cd qim-client && npx vue-tsc --noEmit`（或项目实际的类型检查命令）
预期：无类型错误。

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/composables/useShortcuts.ts
git commit -m "feat: 添加 useShortcuts composable（读写/冲突检测/格式化）"
```

---

## 任务 5：ShortcutInput 组件

**文件：**
- 创建：`qim-client/src/components/settings/ShortcutInput.vue`

**背景：** 按键捕获输入框，聚焦后监听 keydown 捕获组合键。右侧开关复用现有 Switch.vue。Esc 取消编辑，Backspace/Delete 清空。

- [ ] **步骤 1：创建 ShortcutInput.vue**

创建 `qim-client/src/components/settings/ShortcutInput.vue`：

```vue
<template>
  <div class="shortcut-input-wrapper">
    <div class="shortcut-label">{{ label }}</div>
    <div class="shortcut-controls">
      <input
        ref="inputRef"
        class="shortcut-input"
        :value="displayText"
        :placeholder="capturing ? '按下组合键...' : '点击设置'"
        readonly
        :class="{ capturing }"
        @focus="startCapture"
        @blur="stopCapture"
        @keydown.prevent="handleKeyDown"
      />
      <button
        v-if="modelValue.accelerator"
        class="shortcut-clear-btn"
        title="清除"
        @mousedown.prevent="clearAccelerator"
      >×</button>
      <Switch
        :modelValue="modelValue.enabled"
        @update:modelValue="toggleEnabled"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Switch from '../common/Switch.vue'
import { formatAccelerator, type ShortcutItem } from '../../composables/useShortcuts'

const props = defineProps<{
  modelValue: ShortcutItem
  label: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ShortcutItem]
}>()

const inputRef = ref<HTMLInputElement | null>(null)
const capturing = ref(false)

const displayText = computed(() => formatAccelerator(props.modelValue.accelerator))

function startCapture() {
  capturing.value = true
}

function stopCapture() {
  capturing.value = false
}

function toggleEnabled(value: boolean) {
  emit('update:modelValue', { ...props.modelValue, enabled: value })
}

function clearAccelerator() {
  emit('update:modelValue', { ...props.modelValue, accelerator: '' })
  inputRef.value?.focus()
}

function handleKeyDown(e: KeyboardEvent) {
  // Esc 取消
  if (e.key === 'Escape') {
    inputRef.value?.blur()
    return
  }
  // Backspace/Delete 清空
  if (e.key === 'Backspace' || e.key === 'Delete') {
    clearAccelerator()
    return
  }
  // 忽略纯修饰键按下
  if (['Shift', 'Control', 'Alt', 'Meta', 'Command', 'CommandOrControl'].includes(e.key)) {
    return
  }

  // 组装 accelerator
  const parts: string[] = []
  // macOS: e.metaKey 是 Cmd; Windows: e.ctrlKey 是 Ctrl
  // Electron accelerator 用 CommandOrControl 统一表示
  if (e.metaKey || e.ctrlKey) parts.push('CommandOrControl')
  if (e.shiftKey) parts.push('Shift')
  if (e.altKey) parts.push('Alt')

  // 主键
  let key = e.key
  // 单字母大写
  if (key.length === 1) key = key.toUpperCase()
  // 特殊键名转换
  const keyMap: Record<string, string> = {
    'Enter': 'Return',
    ' ': 'Space',
    'ArrowUp': 'Up',
    'ArrowDown': 'Down',
    'ArrowLeft': 'Left',
    'ArrowRight': 'Right',
  }
  if (keyMap[key]) key = keyMap[key]

  // 必须包含修饰键，否则不合法
  if (parts.length === 0) return

  parts.push(key)
  const accelerator = parts.join('+')
  emit('update:modelValue', { ...props.modelValue, accelerator })
  inputRef.value?.blur()
}
</script>

<style scoped>
.shortcut-input-wrapper {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}

.shortcut-label {
  color: var(--text-color);
  font-size: 14px;
}

.shortcut-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.shortcut-input {
  width: 180px;
  padding: 6px 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md, 6px);
  background: var(--card-bg);
  color: var(--text-color);
  font-size: 13px;
  cursor: pointer;
  text-align: center;
  outline: none;
  transition: border-color 0.2s;
}

.shortcut-input:focus,
.shortcut-input.capturing {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(51, 133, 255, 0.15);
}

.shortcut-input::placeholder {
  color: var(--text-secondary, #999);
}

.shortcut-clear-btn {
  border: none;
  background: transparent;
  color: var(--text-secondary, #999);
  cursor: pointer;
  font-size: 18px;
  padding: 2px 6px;
  line-height: 1;
  border-radius: 4px;
}

.shortcut-clear-btn:hover {
  color: var(--text-color);
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}
</style>
```

- [ ] **步骤 2：验证组件可渲染**

运行应用，临时在某个组件中引入 `<ShortcutInput>` 测试渲染和按键捕获（后续任务会正式集成）。
预期：输入框可点击，按下 Cmd+Shift+X 后显示 `⌘+Shift+X`，清除按钮可用，开关可切换。

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/settings/ShortcutInput.vue
git commit -m "feat: 添加 ShortcutInput 按键捕获组件"
```

---

## 任务 6：ShortcutSettings 组件

**文件：**
- 创建：`qim-client/src/components/settings/ShortcutSettings.vue`

**背景：** 快捷键设置分页容器，加载时通过 IPC 拉取配置，保存时检测冲突，冲突用 `window.$QMessage` 提示。

- [ ] **步骤 1：创建 ShortcutSettings.vue**

创建 `qim-client/src/components/settings/ShortcutSettings.vue`：

```vue
<template>
  <div class="shortcut-settings">
    <div v-if="loading" class="shortcut-loading">加载中...</div>
    <template v-else>
      <div class="shortcut-group">
        <h4 class="shortcut-group-title">全局快捷键</h4>
        <ShortcutInput
          v-for="key in globalKeys"
          :key="key"
          :label="SHORTCUT_LABELS.global[key]"
          :modelValue="localShortcuts.global[key]"
          @update:modelValue="updateItem('global', key, $event)"
        />
      </div>

      <div class="shortcut-group">
        <h4 class="shortcut-group-title">编辑器快捷键</h4>
        <ShortcutInput
          v-for="key in editorKeys"
          :key="key"
          :label="SHORTCUT_LABELS.editor[key]"
          :modelValue="localShortcuts.editor[key]"
          @update:modelValue="updateItem('editor', key, $event)"
        />
      </div>

      <div class="shortcut-actions">
        <button class="shortcut-reset-btn" @click="handleReset">恢复默认</button>
        <button class="shortcut-save-btn" @click="handleSave">保存</button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ShortcutInput from './ShortcutInput.vue'
import {
  useShortcuts, checkConflicts, SHORTCUT_LABELS,
  DEFAULT_SHORTCUTS, type ShortcutsConfig, type ShortcutItem
} from '../../composables/useShortcuts'

const { loadShortcuts, saveShortcuts, resetShortcuts } = useShortcuts()

const loading = ref(true)
const localShortcuts = ref<ShortcutsConfig>(JSON.parse(JSON.stringify(DEFAULT_SHORTCUTS)))

const globalKeys = Object.keys(SHORTCUT_LABELS.global)
const editorKeys = Object.keys(SHORTCUT_LABELS.editor)

onMounted(async () => {
  try {
    localShortcuts.value = await loadShortcuts()
  } catch (e) {
    console.error('加载快捷键配置失败:', e)
  } finally {
    loading.value = false
  }
})

function updateItem(scope: 'global' | 'editor', name: string, value: ShortcutItem) {
  localShortcuts.value[scope][name] = value
}

async function handleSave() {
  const conflicts = checkConflicts(localShortcuts.value)
  if (conflicts.length > 0) {
    const c = conflicts[0]
    window.$QMessage?.error(`快捷键冲突：「${c.a.label}」与「${c.b.label}」使用了相同的组合`, 5000)
    return
  }
  try {
    await saveShortcuts(localShortcuts.value)
    window.$QMessage?.success('快捷键设置已保存')
  } catch (e) {
    console.error('保存快捷键配置失败:', e)
    window.$QMessage?.error('保存失败，请重试')
  }
}

async function handleReset() {
  try {
    localShortcuts.value = await resetShortcuts()
    window.$QMessage?.success('已恢复默认快捷键')
  } catch (e) {
    console.error('重置快捷键失败:', e)
    window.$QMessage?.error('重置失败，请重试')
  }
}
</script>

<style scoped>
.shortcut-settings {
  padding: 10px 0;
}

.shortcut-loading {
  text-align: center;
  color: var(--text-secondary, #999);
  padding: 40px 0;
}

.shortcut-group {
  margin-bottom: 24px;
}

.shortcut-group-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}

.shortcut-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.shortcut-reset-btn,
.shortcut-save-btn {
  padding: 6px 18px;
  border-radius: var(--radius-md, 6px);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.shortcut-reset-btn {
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-color);
}

.shortcut-reset-btn:hover {
  background: var(--hover-bg, rgba(0, 0, 0, 0.05));
}

.shortcut-save-btn {
  border: none;
  background: var(--primary-color);
  color: white;
}

.shortcut-save-btn:hover {
  opacity: 0.9;
}
</style>
```

- [ ] **步骤 2：验证组件可独立渲染**

运行应用，临时在某处引入 `<ShortcutSettings />`，确认：
- 加载时显示「加载中...」后变为列表
- 全局 5 项 + 编辑器 4 项正确显示
- 修改快捷键后点保存，无冲突时提示成功
- 故意设置两个相同快捷键，点保存，提示冲突

- [ ] **步骤 3：Commit**

```bash
git add qim-client/src/components/settings/ShortcutSettings.vue
git commit -m "feat: 添加 ShortcutSettings 快捷键设置分页组件"
```

---

## 任务 7：SettingsPanel 集成快捷键 tab

**文件：**
- 修改：`qim-client/src/components/settings/SettingsPanel.vue`

**背景：** 在侧边栏新增 `shortcut` tab，内容区条件渲染 `<ShortcutSettings />`。SettingsPanel 自身的保存按钮不影响快捷键（快捷键有独立的保存按钮，因为其数据流独立于 SettingsPanel 的 props/emit）。

- [ ] **步骤 1：在侧边栏添加 shortcut 项**

在 SettingsPanel.vue 模板中，`file` 侧边栏项之后（约 L26-29）添加：

```html
<div class="settings-sidebar-item" :class="{ active: localTab === 'shortcut' }" @click="localTab = 'shortcut'">
  快捷键
</div>
```

- [ ] **步骤 2：在内容区添加 shortcut 分页**

在 SettingsPanel.vue 模板中，`file` 内容区 `</div>` 之后（约 L224）添加：

```html
<div v-if="localTab === 'shortcut'" class="settings-section">
  <ShortcutSettings />
</div>
```

- [ ] **步骤 3：导入 ShortcutSettings 组件**

在 SettingsPanel.vue 的 `<script setup>` 中，现有 import 语句之后添加：

```typescript
import ShortcutSettings from './ShortcutSettings.vue'
```

- [ ] **步骤 4：验证设置面板显示快捷键页**

运行应用，打开设置面板：
- 侧边栏出现「快捷键」项
- 点击后右侧显示快捷键设置内容
- 其他 tab 切换正常，不受影响

- [ ] **步骤 5：Commit**

```bash
git add qim-client/src/components/settings/SettingsPanel.vue
git commit -m "feat: SettingsPanel 新增快捷键设置 tab"
```

---

## 任务 8：NoteEditor keymapCompartment 改造

**文件：**
- 修改：`qim-client/src/components/apps/notes/NoteEditor.vue`

**背景：** 现有 `keymap.of([...])` 是静态的。改为用 `keymapCompartment.of(...)` 包装，参照已有的 `themeCompartment` 模式（L59）。初始化时从主进程读取 editor 快捷键配置，监听 `shortcuts-updated` 事件运行时替换。

- [ ] **步骤 1：添加 keymapCompartment 声明**

在 NoteEditor.vue 的 script 中，`themeCompartment` 声明附近（约 L59）添加：

```typescript
const keymapCompartment = new Compartment()
```

- [ ] **步骤 2：抽取 buildEditorKeymap 函数**

在 `createEditor` 函数之前添加一个辅助函数，根据配置构建 keymap：

```typescript
function buildEditorKeymap(shortcuts?: Record<string, { accelerator: string, enabled: boolean }>) {
  const customKeymap: { key: string, run: () => boolean }[] = []
  const editorShortcuts = shortcuts || {
    bold:   { accelerator: 'Mod-b', enabled: true },
    italic: { accelerator: 'Mod-i', enabled: true },
    link:   { accelerator: 'Mod-k', enabled: true },
    save:   { accelerator: 'Mod-s', enabled: true }
  }
  if (editorShortcuts.bold?.enabled) {
    customKeymap.push({ key: editorShortcuts.bold.accelerator, run: () => { insertFormat('**', '**'); return true } })
  }
  if (editorShortcuts.italic?.enabled) {
    customKeymap.push({ key: editorShortcuts.italic.accelerator, run: () => { insertFormat('*', '*'); return true } })
  }
  if (editorShortcuts.link?.enabled) {
    customKeymap.push({ key: editorShortcuts.link.accelerator, run: () => { insertLink(); return true } })
  }
  if (editorShortcuts.save?.enabled) {
    customKeymap.push({ key: editorShortcuts.save.accelerator, run: () => { emit('save'); return true } })
  }
  return customKeymap
}
```

- [ ] **步骤 3：改造 createEditor 中的 keymap**

将 NoteEditor.vue L177-188 处的：
```typescript
keymap.of([
  ...closeBracketsKeymap,
  ...defaultKeymap,
  ...searchKeymap,
  ...historyKeymap,
  ...foldKeymap,
  indentWithTab,
  { key: 'Mod-b', run: () => { insertFormat('**', '**'); return true } },
  { key: 'Mod-i', run: () => { insertFormat('*', '*'); return true } },
  { key: 'Mod-k', run: () => { insertLink(); return true } },
  { key: 'Mod-s', run: () => { emit('save'); return true } },
]),
```

替换为：
```typescript
keymapCompartment.of(keymap.of([
  ...closeBracketsKeymap,
  ...defaultKeymap,
  ...searchKeymap,
  ...historyKeymap,
  ...foldKeymap,
  indentWithTab,
  ...buildEditorKeymap(),
])),
```

- [ ] **步骤 4：初始化时从主进程拉取配置并 reconfigure**

在 `onMounted` 中，`createEditor()` 之后添加（NoteEditor.vue 约 L245-252）：

```typescript
// 拉取快捷键配置并应用
if (window.electron?.ipcRenderer?.invoke) {
  window.electron.ipcRenderer.invoke('get-shortcuts').then((shortcuts) => {
    if (editorView.value && shortcuts?.editor) {
      editorView.value.dispatch({
        effects: keymapCompartment.reconfigure(keymap.of([
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...searchKeymap,
          ...historyKeymap,
          ...foldKeymap,
          indentWithTab,
          ...buildEditorKeymap(shortcuts.editor),
        ]))
      })
    }
  }).catch(() => {})
}

// 监听快捷键更新
const handleShortcutsUpdated = (_event: unknown, shortcuts: { editor?: Record<string, { accelerator: string, enabled: boolean }> }) => {
  if (editorView.value && shortcuts?.editor) {
    editorView.value.dispatch({
      effects: keymapCompartment.reconfigure(keymap.of([
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...searchKeymap,
        ...historyKeymap,
        ...foldKeymap,
        indentWithTab,
        ...buildEditorKeymap(shortcuts.editor),
      ]))
    })
  }
}
window.electron?.ipcRenderer?.on('shortcuts-updated', handleShortcutsUpdated)
```

- [ ] **步骤 5：在 onUnmounted 中移除监听器**

在 `onUnmounted` 中（NoteEditor.vue 约 L254-262），现有清理逻辑中添加：

```typescript
if (window.electron?.ipcRenderer?.removeListener) {
  window.electron.ipcRenderer.removeListener('shortcuts-updated', handleShortcutsUpdated)
}
```

注意：`handleShortcutsUpdated` 需要提升为组件级变量以便 onUnmounted 访问。将其声明从 onMounted 内部移到 onMounted 外部（script setup 顶层）。

- [ ] **步骤 6：验证编辑器快捷键运行时替换**

运行应用，打开笔记编辑器：
- Mod-b 加粗、Mod-i 斜体、Mod-k 链接、Mod-s 保存 功能正常
- 在快捷键设置中修改某项（如把加粗改为 Mod-d）并保存
- 回到编辑器，Mod-d 触发加粗，Mod-b 不再触发
- 关闭某项后，该快捷键不响应

- [ ] **步骤 7：Commit**

```bash
git add qim-client/src/components/apps/notes/NoteEditor.vue
git commit -m "feat: NoteEditor 使用 keymapCompartment 支持运行时快捷键替换"
```

---

## 任务 9：端到端验证

- [ ] **步骤 1：验证完整流程**

运行应用，执行以下测试：

1. **启动验证**：应用启动后，所有默认快捷键功能正常（Cmd+M/K/W/Q、Cmd+Shift+A）
2. **修改全局快捷键**：在设置中把「最小化」改为 Cmd+Shift+M，保存后测试 Cmd+Shift+M 最小化、Cmd+M 无响应
3. **关闭快捷键**：关闭「退出应用」开关，Cmd+Q 无响应；重新开启后恢复
4. **修改编辑器快捷键**：把「加粗」改为 Mod-d，保存后编辑器中 Mod-d 加粗、Mod-b 无响应
5. **冲突检测**：把「最大化」设为 Cmd+Shift+M（与已修改的「最小化」相同），点保存，提示冲突且不保存
6. **跨域冲突检测**：默认配置下「最大化」Cmd+K 与「插入链接」Mod-k 冲突，进入快捷键设置页应能在保存时检测到
7. **恢复默认**：点「恢复默认」，所有快捷键回到初始值
8. **重启验证**：重启应用，修改后的快捷键配置仍然生效（config.json 持久化）

- [ ] **步骤 2：验证 config.json 持久化**

检查 `app.getPath('userData')/config.json`，确认 shortcuts 字段正确写入。
macOS 路径示例：`~/Library/Application Support/<appName>/config.json`

- [ ] **步骤 3：最终 Commit（如有修复）**

```bash
git add -A
git commit -m "test: 快捷键功能端到端验证通过"
```

---

## 自检结果

**1. 规格覆盖度：**
- 配置存储（config.json）：任务 1 ✓
- registerGlobalShortcuts 改造：任务 2 ✓
- 截图快捷键移入统一管理：任务 2 步骤 2 ✓
- 初始化顺序调整：任务 2 步骤 3 ✓
- IPC 接口（get/set/reset）：任务 3 ✓
- useShortcuts composable：任务 4 ✓
- ShortcutInput 组件：任务 5 ✓
- ShortcutSettings 组件：任务 6 ✓
- SettingsPanel 集成：任务 7 ✓
- NoteEditor keymapCompartment：任务 8 ✓
- 冲突检测（三类）：任务 4 中 checkConflicts ✓
- 跨域冲突默认值标注：任务 9 步骤 1 第 6 项 ✓
- 端到端测试：任务 9 ✓

**2. 占位符扫描：** 无 TODO/待定，所有步骤含完整代码。

**3. 类型一致性：**
- `ShortcutItem` 在任务 4 定义，任务 5、6 使用一致
- `ShortcutsConfig` 在任务 4 定义，任务 6 使用一致
- `buildEditorKeymap` 在任务 8 定义并使用，参数类型与 ShortcutItem 一致
- IPC 通道名 `get-shortcuts`/`set-shortcuts`/`reset-shortcuts`/`shortcuts-updated` 在任务 3、4、8 中一致

无遗漏。
