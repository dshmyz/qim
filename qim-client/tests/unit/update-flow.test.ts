import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('update flow safeguards', () => {
  it('resets downloading state when update-error is received', () => {
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')

    expect(useUI).toContain("channel: 'update-error'")
    expect(useUI).toContain('isDownloading.value = false')
    expect(useUI).toContain('downloadProgress.value = 0')
  })

  it('preserves download failure messages instead of replacing them with check failure', () => {
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

    expect(useUI).toContain("friendlyMessage = error")
    expect(mainProcess).toContain("updatePhase = 'downloading'")
    expect(mainProcess).toContain("formatUpdateError(error, 'download')")
  })

  it('waits for user confirmation before installing a downloaded update', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
    const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')
    const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')

    const downloadedHandler = mainProcess.slice(
      mainProcess.indexOf("autoUpdater.on('update-downloaded'"),
      mainProcess.indexOf("async function installLinuxUpdate")
    )

    expect(downloadedHandler).not.toContain('autoUpdater.quitAndInstall()')
    expect(mainProcess).toContain("ipcMain.on('install-update'")
    expect(mainProcess).toContain('autoUpdater.quitAndInstall(false, true)')
    expect(useUI).toContain('isUpdateReadyToInstall.value = true')
    expect(mainDialogs).toContain("$emit('installUpdate')")
    expect(mainDialogs).toContain('立即重启安装')
    expect(mainView).toContain('@installUpdate="installUpdate"')
    expect(mainView).toContain("window.electron.ipcRenderer.send('install-update')")
  })

  it('shows installing status and forces install when install-update is clicked', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')

    const installHandler = mainProcess.slice(
      mainProcess.indexOf("ipcMain.on('install-update'"),
      mainProcess.indexOf("ipcMain.on('start-screen-share'")
    )

    expect(installHandler).toContain("sendToWindow('update-installing')")
    expect(installHandler).toContain('autoUpdater.quitAndInstall(false, true)')
    expect(useUI).toContain("channel: 'update-installing'")
    expect(useUI).toContain("updateResult.value = '正在重启并安装更新...'")
  })

  it('blocks reload shortcuts while a force update dialog is active', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')

    expect(mainProcess).toContain('let forceUpdateActive = false')
    expect(mainProcess).toContain('forceUpdateActive = !!info.forceUpdate')
    expect(mainProcess).toContain("mainWindow.webContents.on('before-input-event'")
    expect(mainProcess).toContain("input.key.toLowerCase() === 'r'")
    expect(mainProcess).toContain('input.meta || input.control')
    expect(mainProcess).toContain('event.preventDefault()')
    expect(mainProcess).toContain("mainWindow.webContents.on('will-navigate'")
    expect(mainProcess).toContain("sendToWindow('update-available'")
  })

  it('shows new version metadata in the update dialog', () => {
    const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')

    expect(mainDialogs).toContain('updateInfo.version')
    expect(mainDialogs).toContain('updateInfo.releaseDate')
    expect(mainDialogs).toContain('updateInfo.releaseNotes')
  })

  it('keeps admin version publishing upload-only for update packages', () => {
    const versionManagement = readFileSync(
      resolve(__dirname, '../../../qim-admin/src/views/VersionManagement.vue'),
      'utf8'
    )

    expect(versionManagement).toContain(':disabled="true"')
    expect(versionManagement).toContain('上传安装包后自动生成下载链接')
    expect(versionManagement).not.toContain('请输入安装包下载链接或上传文件')
  })
})
