import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'

/**
 * 快捷键配置接口
 */
export interface ShortcutConfig {
  key: string
  ctrlKey?: boolean
  shiftKey?: boolean
  altKey?: boolean
  metaKey?: boolean
  action: () => void
  description: string
}

/**
 * 将 accelerator 字符串（如 'CommandOrControl+Shift+L'）解析为 ShortcutConfig 片段
 * 返回 { key, ctrlKey, shiftKey, metaKey }（不含 action/description）
 */
export function parseAccelerator(accel: string): { key: string; ctrlKey?: boolean; shiftKey?: boolean; altKey?: boolean; metaKey?: boolean } {
  const parts = accel.split('+').map(p => p.trim())
  const result: { key: string; ctrlKey?: boolean; shiftKey?: boolean; altKey?: boolean; metaKey?: boolean } = { key: '' }
  for (const part of parts) {
    if (part === 'CommandOrControl' || part === 'Mod') {
      result.ctrlKey = true // ctrlKey 同时匹配 Ctrl 和 Meta（见匹配逻辑）
    } else if (part === 'Shift') {
      result.shiftKey = true
    } else if (part === 'Alt' || part === 'Option') {
      result.altKey = true
    } else if (part === 'Control') {
      result.ctrlKey = true
    } else if (part === 'Meta') {
      result.metaKey = true
    } else {
      result.key = part.toLowerCase()
    }
  }
  return result
}

/**
 * AI 快捷键管理 composable
 * 支持静态数组或 Ref<ShortcutConfig[]>，后者可在运行时更新快捷键配置
 */
export function useAIKeyboardShortcuts(
  shortcuts: ShortcutConfig[] | Ref<ShortcutConfig[]>,
  enabled: boolean = true
) {
  const isEnabled = ref(enabled)

  // 始终从 ref 读取，兼容静态数组和 Ref
  const shortcutsRef = ref(
    Array.isArray(shortcuts) ? shortcuts : shortcuts.value
  )
  if (!Array.isArray(shortcuts)) {
    watch(shortcuts, (val: ShortcutConfig[]) => { shortcutsRef.value = val })
  }

  const handleKeydown = (event: KeyboardEvent) => {
    if (!isEnabled.value) return

    const target = event.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable
    ) {
      const hasCtrlShift = event.ctrlKey || event.metaKey
      if (!hasCtrlShift) return
    }

    for (const shortcut of shortcutsRef.value) {
      const keyMatch = event.key.toLowerCase() === shortcut.key.toLowerCase()
      const ctrlMatch = shortcut.ctrlKey ? (event.ctrlKey || event.metaKey) : true
      const shiftMatch = shortcut.shiftKey ? event.shiftKey : !event.shiftKey
      const altMatch = shortcut.altKey ? event.altKey : !event.altKey

      if (keyMatch && ctrlMatch && shiftMatch && altMatch) {
        event.preventDefault()
        shortcut.action()
        break
      }
    }
  }

  /**
   * 启用快捷键
   */
  const enable = () => {
    isEnabled.value = true
  }

  /**
   * 禁用快捷键
   */
  const disable = () => {
    isEnabled.value = false
  }

  /**
   * 切换快捷键启用状态
   */
  const toggle = () => {
    isEnabled.value = !isEnabled.value
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return {
    enabled: isEnabled,
    enable,
    disable,
    toggle
  }
}
