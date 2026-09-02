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

describe('force auto-update (强制自动升级 - 静默路径)', () => {
  const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
  const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
  const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')

  it('starts downloading automatically only for silent (auto-check + force) updates', () => {
    const availableHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-available'"),
      updateModule.indexOf("autoUpdater.on('update-not-available'")
    )

    // 静默路径判定：自动检查 + 强制版本；一旦启动即锁定，不被后续手动检查降级
    expect(availableHandler).toContain("silentForceActive = forceUpdateActive && currentCheckSource === 'auto'")
    expect(availableHandler).toContain('if (!silentForceActive && !hasManualPendingDownload) {')
    // 静默路径自动触发下载，且每段静默流程只下载一次
    expect(availableHandler).toContain('if (silentForceActive && !silentDownloadStarted) {')
    expect(availableHandler).toContain("silentDownloadStarted = true")
    expect(availableHandler).toContain("downloadUpdate('force-auto')")
  })

  it('locks silent force once started so a manual check cannot downgrade it', () => {
    // 静默流程开始后(锁定)不再因手动检查重算而降级
    expect(updateModule).toContain('if (!silentForceActive && !hasManualPendingDownload) {')
    expect(updateModule).toContain('silentForceActive = forceUpdateActive && currentCheckSource === \'auto\'')
  })

  it('does not hijack a manual pending-install download into silent force on auto re-announce', () => {
    const availableHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-available'"),
      updateModule.indexOf("autoUpdater.on('update-not-available'")
    )

    // 已有「手动下载完成、等待安装」的包时，自动检查重播同一 update-available 不启动静默强制
    expect(availableHandler).toContain('const hasManualPendingDownload = !silentForceActive && !!downloadedUpdateInfo')
    expect(availableHandler).toContain('if (!silentForceActive && !hasManualPendingDownload) {')
  })

  it('does not discard the downloaded package when the same version is re-announced', () => {
    const availableHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-available'"),
      updateModule.indexOf("autoUpdater.on('update-not-available'")
    )

    // 仅当新版本 ≠ 已下载版本时才清空下载状态，避免周期检查清掉已下载安装包
    expect(availableHandler).toContain("downloadedUpdateInfo.version !== info.version")
    expect(availableHandler).toContain('!downloadedUpdateInfo || downloadedUpdateInfo.version !== info.version')
  })

  it('marks manual check source so manual path is not forced', () => {
    const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')

    expect(updateModule).toContain("currentCheckSource = 'manual'")
    expect(updateModule).toContain("currentCheckSource = 'auto'")
    expect(updateModule).toContain('let silentForceActive = false')
    // 下载失败自动重试仅限静默路径
    expect(updateModule).toContain('if (silentForceActive) {')
    expect(updateModule).toContain('handleForceDownloadFailure(error)')
  })

  it('installs a downloaded silently-forced update immediately (no waiting, no night window)', () => {
    const downloadedHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-downloaded'"),
      updateModule.indexOf('const autoCheckForUpdates')
    )

    expect(downloadedHandler).toContain('if (silentForceActive) {')
    expect(downloadedHandler).toContain('installDownloadedUpdate()')
    // 非静默路径仍等待用户确认
    expect(downloadedHandler).toContain('等待用户确认安装')
    // 不再有夜间安装窗口逻辑
    expect(updateModule).not.toContain('FORCE_INSTALL_HOUR_START')
    expect(updateModule).not.toContain('scheduleForceInstallAtNight')
    expect(updateModule).not.toContain('isInForceInstallWindow')
  })

  it('retries silent force download failures with backoff and gives up after N attempts', () => {
    expect(updateModule).toContain('FORCE_DOWNLOAD_MAX_RETRY = 3')
    expect(updateModule).toContain('FORCE_DOWNLOAD_RETRY_BASE_MS = 15 * 1000')
    expect(updateModule).toContain('function handleForceDownloadFailure')
    expect(updateModule).toContain('forceDownloadRetry >= FORCE_DOWNLOAD_MAX_RETRY')
    expect(updateModule).toContain('已自动重试')
  })

  it('releases the silent-force lock after max retries so a later auto-check can recover', () => {
    // 放弃重试时复位静默锁与「本次已启动下载」标记，避免把用户锁死在无法恢复的弹窗里；
    // 下一次自动检查可重新进入静默流程，网络恢复后自动续传，无需杀进程。
    const failureHandler = updateModule.slice(
      updateModule.indexOf('function handleForceDownloadFailure'),
      updateModule.indexOf('function checkForUpdates')
    )

    expect(failureHandler).toContain('forceDownloadRetry >= FORCE_DOWNLOAD_MAX_RETRY')
    expect(failureHandler).toContain('clearForceUpdate()')
    expect(failureHandler).toContain('等待应用自动重试')
  })

  it('uses resetForceDownloadRetry on silent-force download completion (clears pending backoff timer)', () => {
    const downloadedHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('update-downloaded'"),
      updateModule.indexOf('const autoCheckForUpdates')
    )

    expect(downloadedHandler).toContain('if (silentForceActive) {')
    expect(downloadedHandler).toContain('resetForceDownloadRetry() // 清除未决的退避重试定时器与计数')
  })

  it('keeps the silent-force dialog non-dismissible during download failure', () => {
    const errorHandler = updateModule.slice(
      updateModule.indexOf("autoUpdater.on('error'"),
      updateModule.indexOf("autoUpdater.on('download-progress'")
    )

    expect(errorHandler).toContain('if (silentForceActive) {')
    expect(errorHandler).toContain('handleForceDownloadFailure(error)')
  })

  it('gates the auto-install notice and hides manual buttons on silent force in the UI', () => {
    // 静默路径：下载完成立即安装，不进入等待安装（不显示「立即重启安装」）
    expect(useUI).toContain('if (info?.silent) {')
    expect(useUI).toContain('正在重新启动应用完成升级')
    // 弹窗：静默强制时隐藏「立即升级 / 立即重启安装」，普通/手动强制保留按钮
    expect(mainDialogs).toContain('isUpdateReadyToInstall && !silentForce')
    expect(mainDialogs).toContain('hasNewVersion && !isDownloading && !isInstalling && !silentForce')
    // 手动强制（silent=false）仍不可关闭（保留原 forceUpdate 语义）
    expect(mainDialogs).toContain('v-if="!forceUpdate"')
  })
})

describe('server-pushed update URL (服务端下发更新地址)', () => {
  const mainProcess = readFileSync(resolve(__dirname, '../../electron/main.js'), 'utf8')
  const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
  const systemConfigStore = readFileSync(resolve(__dirname, '../../src/stores/systemConfig.ts'), 'utf8')

  it('wires a separate set-update-server-url IPC that does not touch persisted chat server config', () => {
    expect(updateModule).toContain("ipcMain.on('set-update-server-url'")
    const handler = updateModule.slice(
      updateModule.indexOf("ipcMain.on('set-update-server-url'"),
      updateModule.indexOf("ipcMain.on('get-server-url'")
    )
    expect(handler).toContain('setServerPushedUpdateUrl')
    // 关键：不能写 saveServerConfig —— config.json.serverUrl 是聊天服务器地址，不能被更新地址覆盖
    expect(handler).not.toContain('saveServerConfig')
  })

  it('gives server-pushed URL priority over chat server and baked env in main process', () => {
    expect(mainProcess).toContain('let serverPushedUpdateUrl')
    expect(mainProcess).toContain('getUpdateBaseUrl: () => serverPushedUpdateUrl || currentUpdateBaseUrl')
    expect(mainProcess).toContain('setServerPushedUpdateUrl')
  })

  it('reads client:update_base_url from public config and pushes it to main', () => {
    expect(systemConfigStore).toContain("client:update_base_url")
    expect(systemConfigStore).toContain("'set-update-server-url'")
    expect(systemConfigStore).toContain('applyUpdateBaseUrl')
  })

  it('lets an empty pushed URL clear the override (not just set it)', () => {
    // store 端空值也透传（不清空会让管理员清字段后运行中客户端粘住旧地址）
    expect(systemConfigStore).toMatch(/function applyUpdateBaseUrl[\s\S]*?\(typeof url === 'string' \? url : ''\)/)
    // IPC 端接受空串并清除
    const handler = updateModule.slice(
      updateModule.indexOf("ipcMain.on('set-update-server-url'"),
      updateModule.indexOf("ipcMain.on('get-server-url'")
    )
    expect(handler).toContain("if (typeof updateUrl === 'string')")
    // 切换聊天服务器（set-server-url）时清掉旧服务器的推送地址
    const setServerHandler = updateModule.slice(
      updateModule.indexOf("ipcMain.on('set-server-url'"),
      updateModule.indexOf("ipcMain.on('set-update-server-url'")
    )
    expect(setServerHandler).toContain("setServerPushedUpdateUrl('')")
  })

  it('keeps get-server-url returning the chat server URL, not the effective update URL', () => {
    expect(mainProcess).toContain('getChatServerUrl: () => currentUpdateBaseUrl')
    const getServerHandler = updateModule.slice(
      updateModule.indexOf("ipcMain.on('get-server-url'"),
      updateModule.indexOf("ipcMain.on('check-for-updates'")
    )
    expect(getServerHandler).toContain('getChatServerUrl()')
    expect(getServerHandler).not.toContain('getUpdateBaseUrl()')
  })
})

describe('update check watchdog (检查失败看门狗)', () => {
  const updateModule = readFileSync(resolve(__dirname, '../../electron/auto-update.js'), 'utf8')
  const useUI = readFileSync(resolve(__dirname, '../../src/composables/useUI.ts'), 'utf8')
  const mainDialogs = readFileSync(resolve(__dirname, '../../src/components/modals/MainDialogs.vue'), 'utf8')
  const mainView = readFileSync(resolve(__dirname, '../../src/views/Main.vue'), 'utf8')

  it('counts consecutive check failures and emits update-unreliable at threshold', () => {
    expect(updateModule).toContain('let consecutiveCheckFailures = 0')
    expect(updateModule).toContain('UNRELIABLE_FAILURE_THRESHOLD = 3')
    expect(updateModule).toContain('function recordCheckFailure')
    expect(updateModule).toContain("sendToWindow(\n        'update-unreliable'")
    // 成功时清零，保证故障窗口从最近一次成功重新计算
    expect(updateModule).toContain('function resetCheckFailures')
    expect(updateModule).toContain("resetCheckFailures() // 检查成功")
  })

  it('surfaces the unreliable warning in the renderer update dialog', () => {
    expect(useUI).toContain("channel: 'update-unreliable'")
    expect(useUI).toContain('const updateUnreliable = ref(false)')
    expect(mainView).toContain(':updateUnreliable="updateUnreliable"')
    expect(mainDialogs).toContain('update-unreliable-warning')
    expect(mainDialogs).toContain('请前往下载页手动升级客户端')
  })

  it('keeps the unreliable warning across failed re-checks (persistent, not dismissed at check start)', () => {
    // update-checking 处理器不应清除 updateUnreliable，否则主进程一次性提示被下一次失败的复检吞掉
    const checkingHandler = useUI.slice(
      useUI.indexOf("channel: 'update-checking'"),
      useUI.indexOf("channel: 'update-available'")
    )
    expect(checkingHandler).not.toContain('updateUnreliable.value = false')
    // 触发时自动打开弹窗，让后台自动检查的失败也可见
    const unreliableHandler = useUI.slice(
      useUI.indexOf("channel: 'update-unreliable'"),
      useUI.indexOf("channel: 'update-progress'")
    )
    expect(unreliableHandler).toContain('showUpdateDialog.value = true')
  })

  it('refuses concurrent checks while one is still in flight (no cross-check contamination)', () => {
    expect(updateModule).toContain('let checkInFlight = false')
    expect(updateModule).toContain('checkInFlight) {')
    expect(updateModule).toContain('checkInFlight = true')
  })
})
