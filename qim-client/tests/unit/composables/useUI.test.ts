import { describe, expect, it, vi } from 'vitest'
import { useUI } from '@/composables/useUI'

const setWindowSize = (width: number, height: number) => {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: height })
}

describe('useUI settings menu', () => {
  it('positions the settings menu higher than the clicked settings button', () => {
    vi.spyOn(document, 'addEventListener').mockImplementation(() => {})
    const ui = useUI()
    const button = document.createElement('button')
    button.getBoundingClientRect = () => ({
      x: 0,
      y: 720,
      width: 44,
      height: 44,
      top: 720,
      right: 60,
      bottom: 764,
      left: 16,
      toJSON: () => {},
    })

    ui.showSettingsMenu({
      stopPropagation: vi.fn(),
      currentTarget: button,
      clientY: 742,
    } as unknown as MouseEvent)

    expect(ui.settingsMenuPosition.value.y).toBeLessThan(560)
  })
})

describe('useUI group context menu', () => {
  it('keeps the group context menu inside the bottom of the viewport', () => {
    setWindowSize(1200, 800)
    const ui = useUI()

    ui.showGroupContextMenu({
      preventDefault: vi.fn(),
      clientX: 200,
      clientY: 750,
    } as unknown as MouseEvent, { id: 'group-1' })

    expect(ui.groupContextMenuPosition.value.y).toBeLessThanOrEqual(590)
  })
})
