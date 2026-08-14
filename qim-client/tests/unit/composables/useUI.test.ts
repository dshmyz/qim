import { afterEach, describe, expect, it, vi } from 'vitest'
import { useUI, activeMenuPosition } from '@/composables/useUI'

const setWindowSize = (width: number, height: number) => {
  Object.defineProperty(window, 'innerWidth', { configurable: true, value: width })
  Object.defineProperty(window, 'innerHeight', { configurable: true, value: height })
}

afterEach(() => {
  vi.restoreAllMocks()
  delete (window as unknown as { electron?: unknown }).electron
})

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

    // 菜单位置写入模块级 activeMenuPosition（useUI 内部 openMenu 的产物）
    expect(activeMenuPosition.value.y).toBeLessThan(560)
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

    expect(activeMenuPosition.value.y).toBeLessThanOrEqual(590)
  })
})

describe('useUI update progress', () => {
  it('rounds update download progress to one decimal place', () => {
    const listeners = new Map<string, (...args: unknown[]) => void>()
    ;(window as unknown as { electron: unknown }).electron = {
      ipcRenderer: {
        on: vi.fn((channel: string, handler: (...args: unknown[]) => void) => {
          listeners.set(channel, handler)
        }),
        removeListener: vi.fn(),
      },
    }

    const ui = useUI()
    ui.registerUpdateEventListeners()
    listeners.get('update-progress')?.({}, { percent: 12.3456789 })

    expect(ui.downloadProgress.value).toBe(12.3)
  })

  it('formats update download byte counts', () => {
    const listeners = new Map<string, (...args: unknown[]) => void>()
    ;(window as unknown as { electron: unknown }).electron = {
      ipcRenderer: {
        on: vi.fn((channel: string, handler: (...args: unknown[]) => void) => {
          listeners.set(channel, handler)
        }),
        removeListener: vi.fn(),
      },
    }

    const ui = useUI()
    ui.registerUpdateEventListeners()
    listeners.get('update-progress')?.({}, {
      percent: 25,
      transferred: 5 * 1024 * 1024,
      total: 20 * 1024 * 1024,
    })

    expect(ui.downloadSizeText.value).toBe('5.0 MB / 20.0 MB')
  })
})
