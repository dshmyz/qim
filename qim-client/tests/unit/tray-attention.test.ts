import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
const preloadProcess = readFileSync(resolve(__dirname, '../../electron/preload.cjs'), 'utf8')
const electronTypes = readFileSync(resolve(__dirname, '../../src/types/electron.d.ts'), 'utf8')

describe('tray attention behavior', () => {
  it('keeps tray flashing until it is explicitly stopped', () => {
    const flashTrayBody = mainProcess.match(/function flashTray\(\) \{[\s\S]*?\n\}/)?.[0] || ''

    expect(flashTrayBody).not.toContain('maxFlashCount')
    expect(flashTrayBody).not.toContain('clearInterval(trayFlashInterval)')
  })

  it('stops external flashing when the main window receives focus', () => {
    expect(mainProcess).toContain("mainWindow.on('focus'")
    expect(mainProcess).toMatch(/mainWindow\.on\('focus'[\s\S]*?stopTrayFlash\(\)/)
  })

  it('uses a sized transparent tray icon for the blank flash frame', () => {
    expect(mainProcess).toContain('function getBlankTrayIcon()')
    expect(mainProcess).toContain('nativeImage.createFromDataURL')
    expect(mainProcess).not.toContain('nativeImage.createEmpty()')
  })

  it('reports the main window as active only when it is viewable and focused', () => {
    expect(mainProcess).toContain("ipcMain.handle('is-main-window-active'")
    expect(mainProcess).toMatch(/!mainWindow\.isDestroyed\(\)[\s\S]*?mainWindow\.isVisible\(\)[\s\S]*?!mainWindow\.isMinimized\(\)[\s\S]*?mainWindow\.isFocused\(\)/)
  })

  it('treats main-window state query failures as inactive', () => {
    expect(mainProcess).toMatch(/ipcMain\.handle\('is-main-window-active',[\s\S]*?try\s*\{[\s\S]*?catch\s*\([^)]*\)\s*\{[\s\S]*?return false/)
    expect(mainProcess).toMatch(/ipcMain\.handle\('is-main-window-active',[\s\S]*?catch\s*\([^)]*\)\s*\{[\s\S]*?console\.warn/)
  })

  it('exposes main-window activity through the preload bridge and types', () => {
    expect(preloadProcess).toContain("ipcRenderer.invoke('is-main-window-active')")
    expect(electronTypes).toContain('windowState: {')
    expect(electronTypes).toContain('isActive: () => Promise<boolean>')
  })
})
