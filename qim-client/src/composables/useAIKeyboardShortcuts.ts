import { onMounted, onUnmounted, ref } from 'vue'

/**
 * 快捷键配置接口
 */
export interface ShortcutConfig {
  key: string
  ctrlKey?: boolean
  shiftKey?: boolean
  metaKey?: boolean
  action: () => void
  description: string
}

/**
 * AI 快捷键管理 composable
 * 提供全局快捷键注册、启用/禁用、事件监听等功能
 * 
 * @param shortcuts 快捷键配置列表
 * @param enabled 是否启用，默认为 true
 * 
 * @example
 * ```ts
 * const shortcuts: ShortcutConfig[] = [
 *   {
 *     key: 'k',
 *     ctrlKey: true,
 *     action: () => openAIQuickPanel(),
 *     description: '打开 AI 快捷面板'
 *   },
 *   {
 *     key: 'S',
 *     ctrlKey: true,
 *     shiftKey: true,
 *     action: () => generateSummary(),
 *     description: '快速生成会话摘要'
 *   }
 * ]
 * 
 * useAIKeyboardShortcuts(shortcuts, true)
 * ```
 */
export function useAIKeyboardShortcuts(
  shortcuts: ShortcutConfig[],
  enabled: boolean = true
) {
  const isEnabled = ref(enabled)

  const handleKeydown = (event: KeyboardEvent) => {
    // 如果快捷键被禁用，不处理
    if (!isEnabled.value) return

    // 如果用户在输入框中，不触发全局快捷键（除非是明确需要在全局触发的）
    const target = event.target as HTMLElement
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable
    ) {
      // 允许 Ctrl+Shift 组合快捷键即使在输入框中也触发
      const hasCtrlShift = event.ctrlKey || event.metaKey
      if (!hasCtrlShift) return
    }

    for (const shortcut of shortcuts) {
      const keyMatch = event.key.toLowerCase() === shortcut.key.toLowerCase()
      const ctrlMatch = shortcut.ctrlKey ? (event.ctrlKey || event.metaKey) : true
      const shiftMatch = shortcut.shiftKey ? event.shiftKey : !event.shiftKey

      if (keyMatch && ctrlMatch && shiftMatch) {
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
