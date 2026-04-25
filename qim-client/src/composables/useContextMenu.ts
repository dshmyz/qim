import { reactive, readonly, onMounted, onUnmounted } from 'vue'

/**
 * 上下文菜单项接口
 */
export interface ContextMenuItem {
  label: string
  icon?: string
  handler: () => void
  disabled?: boolean
  divider?: boolean
  danger?: boolean
}

/**
 * 上下文菜单状态
 */
export interface ContextMenuState {
  visible: boolean
  x: number
  y: number
  target: any
  items: ContextMenuItem[]
}

/**
 * 右键菜单 composable
 * 提供右键菜单的显示、隐藏、位置管理等功能
 */
export function useContextMenu() {
  const contextMenuState = reactive<ContextMenuState>({
    visible: false,
    x: 0,
    y: 0,
    target: null,
    items: []
  })

  // 菜单位置偏移量
  const OFFSET = 2

  /**
   * 显示上下文菜单
   * @param event 鼠标事件
   * @param target 目标对象
   * @param items 菜单项
   */
  const showContextMenu = (
    event: MouseEvent,
    target?: any,
    items: ContextMenuItem[] = []
  ) => {
    event.preventDefault()
    event.stopPropagation()

    // 计算菜单位置，确保不超出屏幕
    let x = event.clientX
    let y = event.clientY

    // 假设菜单最大宽度 180px，高度 300px
    const menuWidth = 180
    const menuHeight = Math.min(items.length * 40, 300)
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight

    // 调整 x 坐标
    if (x + menuWidth + OFFSET > windowWidth) {
      x = windowWidth - menuWidth - OFFSET
    }

    // 调整 y 坐标
    if (y + menuHeight + OFFSET > windowHeight) {
      y = windowHeight - menuHeight - OFFSET
    }

    // 确保坐标不为负
    x = Math.max(OFFSET, x)
    y = Math.max(OFFSET, y)

    contextMenuState.x = x
    contextMenuState.y = y
    contextMenuState.target = target
    contextMenuState.items = items
    contextMenuState.visible = true
  }

  /**
   * 隐藏上下文菜单
   */
  const hideContextMenu = () => {
    contextMenuState.visible = false
    contextMenuState.target = null
    contextMenuState.items = []
  }

  /**
   * 点击菜单项
   */
  const handleMenuItemClick = (item: ContextMenuItem) => {
    if (item.disabled) return
    item.handler()
    hideContextMenu()
  }

  /**
   * 处理点击外部关闭菜单
   */
  const handleClickOutside = (_event: MouseEvent) => {
    if (contextMenuState.visible) {
      hideContextMenu()
    }
  }

  /**
   * 处理 ESC 键关闭菜单
   */
  const handleEscape = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && contextMenuState.visible) {
      hideContextMenu()
    }
  }

  /**
   * 注册全局事件监听
   */
  const attachGlobalListeners = () => {
    document.addEventListener('click', handleClickOutside)
    document.addEventListener('contextmenu', handleClickOutside)
    document.addEventListener('keydown', handleEscape)
  }

  /**
   * 移除全局事件监听
   */
  const detachGlobalListeners = () => {
    document.removeEventListener('click', handleClickOutside)
    document.removeEventListener('contextmenu', handleClickOutside)
    document.removeEventListener('keydown', handleEscape)
  }

  onMounted(() => {
    attachGlobalListeners()
  })

  onUnmounted(() => {
    detachGlobalListeners()
  })

  return {
    // 状态
    contextMenuState: readonly(contextMenuState),

    // 方法
    showContextMenu,
    hideContextMenu,
    handleMenuItemClick,
    handleClickOutside,
    handleEscape,
    attachGlobalListeners,
    detachGlobalListeners
  }
}
