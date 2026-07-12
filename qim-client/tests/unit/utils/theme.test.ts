import { describe, expect, it, beforeEach, vi } from 'vitest'
import { isAppDarkTheme, onAppThemeChange, DARK_THEMES } from '@/utils/theme'

describe('theme utils', () => {
  beforeEach(() => {
    document.documentElement.removeAttribute('data-theme')
  })

  describe('DARK_THEMES', () => {
    it('contains elegant-dark', () => {
      expect(DARK_THEMES).toContain('elegant-dark')
    })
  })

  describe('isAppDarkTheme', () => {
    it('returns true when data-theme is elegant-dark', () => {
      document.documentElement.dataset.theme = 'elegant-dark'
      expect(isAppDarkTheme()).toBe(true)
    })

    it('returns false when data-theme is modern-light', () => {
      document.documentElement.dataset.theme = 'modern-light'
      expect(isAppDarkTheme()).toBe(false)
    })

    it('returns false when data-theme is not set', () => {
      expect(isAppDarkTheme()).toBe(false)
    })

    it('returns false for other light themes like ocean-blue', () => {
      document.documentElement.dataset.theme = 'ocean-blue'
      expect(isAppDarkTheme()).toBe(false)
    })
  })

  describe('onAppThemeChange', () => {
    it('calls callback when data-theme attribute changes', async () => {
      const cb = vi.fn()
      const off = onAppThemeChange(cb)
      document.documentElement.dataset.theme = 'elegant-dark'
      // MutationObserver 是异步的，需要等待微任务
      await Promise.resolve()
      expect(cb).toHaveBeenCalled()
      off()
    })

    it('does not call callback after unsubscribe', async () => {
      const cb = vi.fn()
      const off = onAppThemeChange(cb)
      off()
      document.documentElement.dataset.theme = 'elegant-dark'
      await Promise.resolve()
      expect(cb).not.toHaveBeenCalled()
    })
  })
})
