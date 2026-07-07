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
