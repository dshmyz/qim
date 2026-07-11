import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

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
})
