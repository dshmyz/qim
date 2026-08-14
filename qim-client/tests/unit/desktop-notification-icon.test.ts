import { readFileSync } from 'fs'
import { resolve } from 'path'
import { describe, expect, it } from 'vitest'

const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')
const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

describe('desktop notification icons', () => {
  it('uses a main-process app icon fallback instead of omitting notification icons', () => {
    // 主进程负责桌面通知图标：把打包内 app 图标（icon_512x512.png）拷到临时目录，
    // Linux libnotify 只认磁盘真实路径（ASAR 内路径读不到），经 notification:show 使用。
    expect(mainProcess).toContain('const getNotifIconPath =')
    expect(mainProcess).toContain('electron/icons/icon_512x512.png')
    // 渲染进程不再自带图标常量：Main.vue 只把发送者头像传给浏览器回退路径（notify.ts 的 icon 参数），
    // Electron 路径始终使用主进程的 app 图标。
    expect(mainView).not.toContain("const DEFAULT_NOTIFICATION_ICON =")
    expect(mainView).toContain('getAvatarUrl(newMessage.sender?.avatar, senderName, serverUrl.value)')
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
