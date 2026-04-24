import { ref, reactive } from 'vue'

interface ContextMenuState {
  visible: boolean
  x: number
  y: number
  target: any
}

export function useContextMenu() {
  const contextMenuState = reactive<ContextMenuState>({
    visible: false,
    x: 0,
    y: 0,
    target: null
  })

  const showContextMenu = (event: MouseEvent, target?: any) => {
    event.preventDefault()
    contextMenuState.x = event.clientX
    contextMenuState.y = event.clientY
    contextMenuState.target = target
    contextMenuState.visible = true
  }

  const hideContextMenu = () => {
    contextMenuState.visible = false
    contextMenuState.target = null
  }

  const handleClickOutside = (event: MouseEvent) => {
    if (contextMenuState.visible) {
      hideContextMenu()
    }
  }

  return {
    contextMenuState,
    showContextMenu,
    hideContextMenu,
    handleClickOutside
  }
}
