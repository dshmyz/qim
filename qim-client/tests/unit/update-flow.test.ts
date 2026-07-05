import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('update flow safeguards', () => {
  it('uses the preferred update service naming', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    expect(updateModule).toContain('export function createUpdateService')
    expect(updateModule).toContain('function startUpdateService')
    expect(updateModule).toContain('function registerUpdateIpc')
    expect(updateModule).toContain('function installDownloadedUpdate')
    expect(updateModule).not.toContain('createAutoUpdateManager')
    expect(updateModule).not.toContain('function setup()')
    expect(updateModule).not.toContain('function registerIpc()')
    expect(mainProcess).toContain('const updateService = createUpdateService')
    expect(mainProcess).toContain('updateService.registerUpdateIpc()')
    expect(mainProcess).toContain('updateService.startUpdateService()')
  })

  it('resets downloading state when update-error is received', () => {
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')

    expect(useUI).toContain("channel: 'update-error'")
    expect(useUI).toContain('isDownloading.value = false')
    expect(useUI).toContain('downloadProgress.value = 0')
  })

  it('preserves download failure messages instead of replacing them with check failure', () => {
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    expect(useUI).toContain("friendlyMessage = error")
    expect(updateModule).toContain("updatePhase = 'downloading'")
    expect(updateModule).toContain("formatUpdateError(error, 'download')")
  })

  it('waits for user confirmation before installing a downloaded update', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
    const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')
    const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')

    const downloadedHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-downloaded'"),
      updateModule.indexOf('const autoCheckForUpdates')
    )

    expect(downloadedHandler).not.toContain('autoUpdater.quitAndInstall()')
    expect(updateModule).toContain("ipcMain.on('install-update'")
    expect(updateModule).toContain('autoUpdater.quitAndInstall(false, true)')
    expect(useUI).toContain('isUpdateReadyToInstall.value = true')
    expect(mainDialogs).toContain("$emit('installUpdate')")
    expect(mainDialogs).toContain('立即重启安装')
    expect(mainView).toContain('@installUpdate="installUpdate"')
    expect(mainView).toContain("window.electron.ipcRenderer.send('install-update')")
  })

  it('shows installing status and forces install when install-update is clicked', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')

    const installHandler = updateModule.slice(
      updateModule.indexOf('function installDownloadedUpdate'),
      updateModule.indexOf('function listenToUpdaterEvents')
    )

    expect(installHandler).toContain("sendToWindow('update-installing')")
    expect(installHandler).toContain('autoUpdater.quitAndInstall(false, true)')
    expect(useUI).toContain("channel: 'update-installing'")
    expect(useUI).toContain("updateResult.value = '正在重启并安装更新...'")
  })

  it('blocks reload shortcuts while a force update dialog is active', () => {
    const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    expect(updateModule).toContain('let forceUpdateActive = false')
    expect(updateModule).toContain('forceUpdateActive = !!info.forceUpdate')
    expect(mainProcess).toContain("mainWindow.webContents.on('before-input-event'")
    expect(mainProcess).toContain("input.key.toLowerCase() === 'r'")
    expect(mainProcess).toContain('input.meta || input.control')
    expect(mainProcess).toContain('event.preventDefault()')
    expect(mainProcess).toContain("mainWindow.webContents.on('will-navigate'")
    expect(mainProcess).toContain('updateService.isForceUpdateActive()')
    expect(mainProcess).toContain("sendToWindow('update-available'")
  })

  it('shows new version metadata in the update dialog', () => {
    const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')

    expect(mainDialogs).toContain('updateInfo.version')
    expect(mainDialogs).toContain('updateInfo.releaseDate')
    expect(mainDialogs).toContain('updateInfo.releaseNotes')
  })

  it('shows downloaded size in the update progress dialog', () => {
    const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')
    const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')
    const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')

    expect(useUI).toContain('downloadSizeText')
    expect(mainView).toContain(':downloadSizeText="downloadSizeText"')
    expect(mainDialogs).toContain('downloadSizeText')
  })

  it('keeps admin version publishing upload-only for update packages', () => {
    const versionManagement = readFileSync(
      resolve(__dirname, '../../../qim-admin/src/views/ClientManagement/components/VersionFormDialog.vue'),
      'utf8'
    )

    expect(versionManagement).toContain(':disabled="true"')
    expect(versionManagement).toContain('上传安装包后自动生成下载链接')
    expect(versionManagement).not.toContain('请输入安装包下载链接或上传文件')
  })
})

describe('auto update feed URL per platform', () => {
  it('uses win7/ path for Windows 7 builds (Electron 22.x)', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    // getFeedUrl 应区分 win7 和 win10
    expect(updateModule).toContain("'win7' : 'win10'")
    // 不应再有 win/（无后缀）的路径
    expect(updateModule).not.toMatch(/updates\/win["']/)
  })

  it('adds a stable client id to update feed URLs for rollout bucketing', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    expect(updateModule).toContain('getOrCreateUpdateClientId')
    expect(updateModule).toContain('client=${encodeURIComponent(getOrCreateUpdateClientId())}')
    expect(updateModule).toContain("'X-QIM-Update-Client': clientId")
  })
})

describe('websocket version reporting', () => {
  it('uses the module app config version instead of an optional window global', () => {
    const useWebSocket = readFileSync(resolve(__dirname, '../../src/composables/useWebSocket.ts'), 'utf8')
    const websocketManager = readFileSync(resolve(__dirname, '../../src/utils/websocketManager.ts'), 'utf8')

    expect(useWebSocket).toContain("import { APP_CONFIG } from '../config/appConfig'")
    expect(useWebSocket).toContain('const version = APP_CONFIG.version')
    expect(useWebSocket).not.toContain('(window as any).APP_CONFIG?.version')

    expect(websocketManager).toContain("import { APP_CONFIG } from '../config/appConfig'")
    expect(websocketManager).toContain('const version = APP_CONFIG.version')
    expect(websocketManager).not.toContain('(window as any).APP_CONFIG?.version')
  })
})

describe('install-update routing', () => {
  it('uses electron-updater quitAndInstall for every platform', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    const installHandler = updateModule.slice(
      updateModule.indexOf('function installDownloadedUpdate'),
      updateModule.indexOf('function listenToUpdaterEvents')
    )

    expect(installHandler).toContain('autoUpdater.quitAndInstall(false, true)')
    expect(installHandler).not.toContain("process.platform === 'linux'")
    expect(installHandler).not.toContain('installLinuxUpdate')
  })

  it('does not keep manual Linux package installation helpers', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
    const packageJson = readFileSync(resolve(__dirname, '../../package.json'), 'utf8')

    expect(updateModule).not.toContain('install-update-linux.sh')
    expect(updateModule).not.toContain('resolveDownloadedUpdatePath')
    expect(updateModule).not.toContain('downloadedUpdateFiles')
    expect(packageJson).not.toContain('install-update-linux.sh')
    expect(packageJson).not.toContain('qim-update.sudoers')
    // after-install-linux.sh / before-remove-linux.sh 现用于桌面快捷方式（deb 钩子），不再是 sudo 安装辅助
  })
})
