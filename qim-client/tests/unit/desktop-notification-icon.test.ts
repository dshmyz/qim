import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')

describe('desktop notification icons', () => {
  it('uses an app icon fallback instead of omitting notification icons', () => {
    expect(mainView).toContain("const DEFAULT_NOTIFICATION_ICON = '/app-logo.png'")
    expect(mainView).toContain('const getNotificationIcon =')
    expect(mainView).not.toContain('icon: getAvatarUrl(')
  })

  it('keeps do-not-disturb and desktop notification settings explicit', () => {
    expect(mainView).toContain('const isInDndTimeRange =')
    expect(mainView).toContain('const isGlobalDndEnabled =')
    expect(mainView).toContain("dndMode === 'all_day'")
    expect(mainView).toContain("dndMode === 'custom'")
    expect(mainView).toContain("dndStartTime || '22:00'")
    expect(mainView).toContain("dndEndTime || '08:00'")
    expect(mainView).toContain('const canAlert =')
    expect(mainView).toContain('const canPlaySound =')
    expect(mainView).toContain('const canShowDesktopNotification =')
    expect(mainView).toContain('messageSettings.value.desktopNotificationsEnabled')
  })
})
