import pkg from 'electron-updater'
import fs from 'fs'
import path from 'path'
import crypto from 'crypto'

const { autoUpdater } = pkg

// Linux 下 electron-updater 默认按 gksudo>kdesudo>pkexec>beesu>sudo 的顺序选择提权命令，
// 优先使用 sudo 以便配合免密配置
if (process.platform === 'linux') {
  autoUpdater.determineSudoCommand = () => 'sudo'
}

const CHECK_UPDATE_TIMEOUT_MS = 12000
const AUTO_CHECK_INTERVAL_MS = 4 * 60 * 60 * 1000
const STARTUP_CHECK_DELAY_MS = 30000

// 强制更新下载失败自动重试上限与指数退避基数（首次 delay=base，之后翻倍）
const FORCE_DOWNLOAD_MAX_RETRY = 3
const FORCE_DOWNLOAD_RETRY_BASE_MS = 15 * 1000

// deb/桌面启动时主进程 console 不可见，把更新检查的关键结果/报错落盘，
// 便于在 Linux 等看不到终端日志的环境排查（文件：userData/update-check.log）。
function appendUpdateLog(app, line) {
  try {
    const ts = new Date().toISOString()
    fs.appendFileSync(path.join(app.getPath('userData'), 'update-check.log'), `[${ts}] ${line}\n`)
  } catch (error) {
    // 落盘失败不影响更新主流程
    console.error('[update-log] 写入更新日志失败:', error)
  }
}

export function createUpdateService({
  app,
  ipcMain,
  sendToWindow,
  getUpdateBaseUrl,
  setUpdateBaseUrl,
  saveServerConfig
}) {
  let updatePhase = 'idle'
  let forceUpdateActive = false
  let lastForceUpdateInfo = null
  let downloadedUpdateInfo = null
  let updaterEventsReady = false
  let updateClientId = null
  // 一次操作（检查/下载）周期内是否已发送错误，防止 timeout/.catch/on('error') 重复报错
  let errorReported = false
  // 自动检查与启动延迟检查的定时器，应用退出时需清理
  let autoCheckTimer = null
  let startupCheckTimer = null
  // 强制更新下载失败重试计数与退避定时器
  let forceDownloadRetry = 0
  let forceDownloadRetryTimer = null
  // 最近一次检查更新的触发来源：'manual'（用户手动检查）| 'auto'（自动静默检查）
  let currentCheckSource = null
  // 静默强制自动升级是否激活（仅「自动检查 + 强制版本」走静默路径；手动检查保留原交互不强制）
  let silentForceActive = false
  // 本次静默强制流程内是否已经开始自动下载（保证一段静默流程只下载一次）
  let silentDownloadStarted = false

  function getOrCreateUpdateClientId() {
    if (updateClientId) return updateClientId

    const clientIdPath = path.join(app.getPath('userData'), 'update-client-id')
    try {
      const existing = fs.readFileSync(clientIdPath, 'utf8').trim()
      if (existing) {
        updateClientId = existing
        return updateClientId
      }
    } catch (error) {
      if (error.code !== 'ENOENT') {
        console.warn('读取更新客户端 ID 失败，将重新生成:', error)
      }
    }

    updateClientId = crypto.randomUUID()
    try {
      fs.mkdirSync(path.dirname(clientIdPath), { recursive: true })
      fs.writeFileSync(clientIdPath, updateClientId, 'utf8')
    } catch (error) {
      console.warn('保存更新客户端 ID 失败，本次运行仍会使用内存 ID:', error)
    }
    return updateClientId
  }

  function withRolloutClientQuery(url) {
    const separator = url.includes('?') ? '&' : '?'
    return `${url}${separator}client=${encodeURIComponent(getOrCreateUpdateClientId())}`
  }

  function updateFeedOptions(feedUrl) {
    const clientId = getOrCreateUpdateClientId()
    return {
      provider: 'generic',
      url: feedUrl,
      requestHeaders: {
        'X-QIM-Update-Client': clientId
      }
    }
  }

  function resolveUpdateFeedUrl() {
    const baseUrl = (getUpdateBaseUrl() || '').replace(/\/+$/, '')
    if (!baseUrl) return null // 未配置服务器地址，交给调用方明确报错

    if (process.platform === 'win32') {
      const electronMajor = parseInt(process.versions.electron.split('.')[0], 10)
      return withRolloutClientQuery(`${baseUrl}/api/v1/updates/${electronMajor <= 22 ? 'win7' : 'win10'}/`)
    }
    if (process.platform === 'linux') return withRolloutClientQuery(`${baseUrl}/api/v1/updates/linux/`)
    if (process.platform === 'darwin') return withRolloutClientQuery(`${baseUrl}/api/v1/updates/mac/`)
    return null
  }

  function applyUpdateFeedUrl() {
    const feedUrl = resolveUpdateFeedUrl()
    if (feedUrl) {
      console.log(`设置更新服务器地址: ${feedUrl}`)
      autoUpdater.setFeedURL(updateFeedOptions(feedUrl))
    } else {
      console.warn('无法设置更新服务器地址: feedUrl 为空, currentUpdateBaseUrl:', getUpdateBaseUrl(), 'platform:', process.platform)
    }
  }

  function formatUpdateError(error, phase = updatePhase) {
    const isDownload = phase === 'download' || phase === 'downloading'
    const fallback = isDownload ? '下载更新失败' : '检查更新失败'
    let errorMessage = fallback

    if (error?.message) {
      const msg = error.message.toLowerCase()

      if (msg.includes('404') || msg.includes('cannot find channel')) {
        errorMessage = isDownload ? '下载更新失败：暂无可用安装包' : '暂无可用更新'
      } else if (msg.includes('timeout') || msg.includes('etimedout')) {
        errorMessage = '网络连接超时，请稍后重试'
      } else if (msg.includes('enotfound') || msg.includes('econnrefused')) {
        errorMessage = '无法连接到更新服务器'
      } else if (msg.includes('net::err')) {
        errorMessage = '网络错误，请检查网络连接'
      } else {
        errorMessage = error.message.split('\n')[0]
      }
    }

    if (isDownload && !errorMessage.includes('下载')) {
      errorMessage = `下载更新失败：${errorMessage}`
    }

    return errorMessage
  }

  function resetDownloadedUpdate() {
    downloadedUpdateInfo = null
  }

  // 强制更新下载重试计数/定时器复位（新版本或下载成功后调用）
  function resetForceDownloadRetry() {
    if (forceDownloadRetryTimer) {
      clearTimeout(forceDownloadRetryTimer)
      forceDownloadRetryTimer = null
    }
    forceDownloadRetry = 0
  }

  function clearForceUpdate() {
    forceUpdateActive = false
    lastForceUpdateInfo = null
    silentForceActive = false
    silentDownloadStarted = false
    resetForceDownloadRetry()
  }

  // 强制更新下载失败：指数退避自动重试，次数耗尽后放弃（保持强制弹窗不可关闭）。
  // 由 downloadUpdate().catch 与 autoUpdater.on('error') 共同调用，用 timer 是否已存在防重复调度。
  function handleForceDownloadFailure(error) {
    if (forceDownloadRetryTimer) return // 已有退避重试在等待，避免重复调度

    if (forceDownloadRetry >= FORCE_DOWNLOAD_MAX_RETRY) {
      appendUpdateLog(app, `强制更新下载反复失败（${FORCE_DOWNLOAD_MAX_RETRY} 次），放弃本次自动重试`)
      // 撤销静默锁并复位「本次流程已启动下载」标记，使下一次自动检查能重新进入静默流程、
      // 从而在网络恢复后自动续传，而不是把用户锁死在无法恢复的弹窗里。
      clearForceUpdate()
      sendToWindow(
        'update-error',
        `强制更新下载失败，已自动重试 ${FORCE_DOWNLOAD_MAX_RETRY} 次仍未成功。请检查网络后等待应用自动重试，升级完成前本提示无法关闭。`
      )
      return
    }

    forceDownloadRetry++
    const delay = FORCE_DOWNLOAD_RETRY_BASE_MS * Math.pow(2, forceDownloadRetry - 1)
    console.log(
      `强制更新下载失败，第 ${forceDownloadRetry}/${FORCE_DOWNLOAD_MAX_RETRY} 次重试，${Math.round(delay / 1000)} 秒后重试`
    )
    appendUpdateLog(app, `强制更新下载失败，第 ${forceDownloadRetry}/${FORCE_DOWNLOAD_MAX_RETRY} 次重试，${Math.round(delay / 1000)} 秒后重试`)
    forceDownloadRetryTimer = setTimeout(() => {
      forceDownloadRetryTimer = null
      downloadUpdate('force-retry')
    }, delay)
  }

  function checkForUpdates() {
    if (updatePhase === 'checking' || updatePhase === 'downloading') {
      console.log('更新检查已在进行中，忽略重复请求, currentUpdatePhase:', updatePhase)
      return
    }
    console.log('收到检查更新请求, currentUpdateBaseUrl:', getUpdateBaseUrl(), 'platform:', process.platform)
    const feedUrl = resolveUpdateFeedUrl()
    if (!feedUrl) {
      const error = !getUpdateBaseUrl()
        ? '无法检查更新: 未配置更新服务器地址，请先在登录页「服务器设置」中填写服务器地址'
        : `无法检查更新: 当前平台 ${process.platform} 不支持或服务器地址未配置 (currentUpdateBaseUrl: ${getUpdateBaseUrl()})`
      console.error(error)
      appendUpdateLog(app, error)
      sendToWindow('update-error', error)
      return
    }

    console.log('设置更新服务器地址:', feedUrl)
    updatePhase = 'checking'
    errorReported = false
    currentCheckSource = 'manual'
    autoUpdater.setFeedURL(updateFeedOptions(feedUrl))

    const timeout = setTimeout(() => {
      if (errorReported) return
      errorReported = true
      updatePhase = 'idle'
      console.error(`检查更新超时（${CHECK_UPDATE_TIMEOUT_MS / 1000}秒）`)
      appendUpdateLog(app, `检查更新超时（${CHECK_UPDATE_TIMEOUT_MS / 1000}秒），feed=${feedUrl}`)
      sendToWindow('update-error', '检查更新超时，请检查网络连接或服务器地址')
    }, CHECK_UPDATE_TIMEOUT_MS)

    const clearTimeoutHandler = () => clearTimeout(timeout)
    autoUpdater.once('update-not-available', clearTimeoutHandler)
    autoUpdater.once('update-available', clearTimeoutHandler)
    autoUpdater.once('error', clearTimeoutHandler)

    autoUpdater.checkForUpdates()
      .then(result => {
        clearTimeout(timeout)
        console.log('检查更新结果:', result)
        appendUpdateLog(app, `检查更新完成 feed=${feedUrl} result=${JSON.stringify(result)}`)
      })
      .catch(error => {
        clearTimeout(timeout)
        if (errorReported) return // 超时已处理，避免重复报错
        errorReported = true
        updatePhase = 'idle'
        console.error('检查更新失败:', error)
        appendUpdateLog(app, `检查更新失败 feed=${feedUrl} error=${error?.message || error}`)
        sendToWindow('update-error', formatUpdateError(error, 'check'))
      })
  }

  function downloadUpdate(source = 'manual') {
    updatePhase = 'downloading'
    errorReported = false
    autoUpdater.downloadUpdate()
      .catch(error => {
        console.error('下载更新失败:', error)
        updatePhase = 'idle'
        resetDownloadedUpdate()
        if (errorReported) return // on('error') 已报告，避免重复报错
        errorReported = true
        if (silentForceActive) {
          // 静默强制更新：自动退避重试（次数耗尽后放弃，弹窗保持强制状态）
          handleForceDownloadFailure(error)
        } else {
          sendToWindow('update-error', formatUpdateError(error, 'download'))
        }
      })
  }

  function installDownloadedUpdate() {
    if (!downloadedUpdateInfo) {
      sendToWindow('update-error', '更新文件尚未下载完成')
      return
    }

    clearForceUpdate() // 同时清掉 forceUpdateActive / silentForceActive / current 相关状态
    sendToWindow('update-installing')
    autoUpdater.quitAndInstall(false, true)
  }

  function listenToUpdaterEvents() {
    if (updaterEventsReady) return
    updaterEventsReady = true

    autoUpdater.on('checking-for-update', () => {
      updatePhase = 'checking'
      resetDownloadedUpdate()
      console.log('正在检查更新...')
      sendToWindow('update-checking')
    })

    autoUpdater.on('update-available', (info) => {
      updatePhase = 'available'
      // 仅当新发现的版本不同于已下载版本时才作废旧的下载状态，
      // 避免同一版本被再次广播 update-available 时误清已下载的安装包。
      if (!downloadedUpdateInfo || downloadedUpdateInfo.version !== info.version) {
        resetDownloadedUpdate()
      }
      resetForceDownloadRetry()
      forceUpdateActive = !!info.forceUpdate
      // 已有「手动下载完成、等待安装」的包时，不再被自动检查抢成静默强制：
      // 保持手动路径，由用户确认安装（保证「两路并存；手动不强制」）。
      const hasManualPendingDownload = !silentForceActive && !!downloadedUpdateInfo
      // 静默强制流程一旦启动即锁定，直到安装/放弃/无新版本才结束；
      // 期间收到手动检查触发的 update-available 也不降级，保证强制必达。
      if (!silentForceActive && !hasManualPendingDownload) {
        silentForceActive = forceUpdateActive && currentCheckSource === 'auto'
        silentDownloadStarted = false
      }
      lastForceUpdateInfo = {
        version: info.version,
        forceUpdate: info.forceUpdate || false,
        silent: silentForceActive,
        releaseDate: info.releaseDate,
        releaseName: info.releaseName,
        releaseNotes: info.releaseNotes
      }
      console.log('发现新版本:', info.version, '强制更新:', info.forceUpdate, '静默强制:', silentForceActive)
      appendUpdateLog(app, `发现新版本 version=${info.version} force=${!!info.forceUpdate} silent=${silentForceActive}`)
      sendToWindow('update-available', lastForceUpdateInfo)
      // 静默强制更新：立即自动下载（每段静默流程只启动一次），无需用户点击「立即升级」
      if (silentForceActive && !silentDownloadStarted) {
        silentDownloadStarted = true
        downloadUpdate('force-auto')
      }
    })

    autoUpdater.on('update-not-available', () => {
      updatePhase = 'idle'
      resetDownloadedUpdate()
      clearForceUpdate()
      currentCheckSource = null
      console.log('当前已是最新版本')
      appendUpdateLog(app, '当前已是最新版本（update-not-available）')
      sendToWindow('update-not-available')
    })

    autoUpdater.on('error', (error) => {
      console.error('更新错误:', error)
      appendUpdateLog(app, `更新错误 error=${error?.message || error}`)
      const errorMessage = formatUpdateError(error) // formatUpdateError 依赖当前 phase，需在重置前计算
      updatePhase = 'idle'
      resetDownloadedUpdate()
      if (errorReported) return // 错误已由 timeout 或 .catch 报告，这里只做状态清理
      errorReported = true
      if (silentForceActive) {
        // 静默强制更新：保持强制状态不关闭弹窗，进入自动退避重试
        handleForceDownloadFailure(error)
      } else {
        clearForceUpdate()
        currentCheckSource = null
        sendToWindow('update-error', errorMessage)
      }
    })

    autoUpdater.on('download-progress', (progressObj) => {
      updatePhase = 'downloading'
      console.log('下载进度:', progressObj.percent)
      sendToWindow('update-progress', progressObj)
    })

    autoUpdater.on('update-downloaded', (info) => {
      updatePhase = 'downloaded'
      downloadedUpdateInfo = info
      if (silentForceActive) {
        // 静默强制更新：下载完成立即自动重启安装，不等待用户确认
        resetForceDownloadRetry() // 清除未决的退避重试定时器与计数，避免安装后残留触发
        sendToWindow('update-downloaded', { ...info, forceUpdate: true, silent: true, autoInstall: true })
        installDownloadedUpdate()
      } else {
        console.log('更新下载完成，等待用户确认安装')
        sendToWindow('update-downloaded', info)
      }
    })
  }

  function checkForUpdatesQuietly(source) {
    if (!app.isPackaged) return
    if (updatePhase === 'checking' || updatePhase === 'downloading') {
      console.log(`[自动更新] ${source}跳过：更新检查已在进行中, currentUpdatePhase: ${updatePhase}`)
      return
    }

    console.log(`[自动更新] ${source}...`)
    updatePhase = 'checking'
    errorReported = false
    currentCheckSource = 'auto'
    applyUpdateFeedUrl()
    autoUpdater.checkForUpdates().catch(error => {
      console.error(`[自动更新] ${source}失败:`, error)
      // 兜底：若 Promise reject 且未触发 error 事件，重置状态避免卡在 checking
      if (updatePhase === 'checking') {
        updatePhase = 'idle'
      }
    })
  }

  function startUpdateService() {
    applyUpdateFeedUrl()
    autoUpdater.autoDownload = false
    listenToUpdaterEvents()

    const autoCheckForUpdates = () => {
      checkForUpdatesQuietly('定期检查更新')
    }

    autoCheckTimer = setInterval(autoCheckForUpdates, AUTO_CHECK_INTERVAL_MS)
    startupCheckTimer = setTimeout(() => {
      checkForUpdatesQuietly('启动后首次检查更新')
    }, STARTUP_CHECK_DELAY_MS)
  }

  function stopUpdateService() {
    if (autoCheckTimer) {
      clearInterval(autoCheckTimer)
      autoCheckTimer = null
    }
    if (startupCheckTimer) {
      clearTimeout(startupCheckTimer)
      startupCheckTimer = null
    }
    if (forceDownloadRetryTimer) {
      clearTimeout(forceDownloadRetryTimer)
      forceDownloadRetryTimer = null
    }
  }

  function registerUpdateIpc() {
    ipcMain.on('set-server-url', (event, serverUrl) => {
      console.log('收到服务器地址更新:', serverUrl)
      if (serverUrl && typeof serverUrl === 'string') {
        const nextUrl = serverUrl.replace(/\/+$/, '')
        setUpdateBaseUrl(nextUrl)
        saveServerConfig(nextUrl)
        applyUpdateFeedUrl()
        console.log('更新服务器地址已保存:', nextUrl)
      }
    })

    ipcMain.on('get-server-url', (event) => {
      event.sender.send('server-url', getUpdateBaseUrl())
    })

    ipcMain.on('check-for-updates', () => {
      checkForUpdates()
    })

    ipcMain.on('download-update', () => {
      downloadUpdate()
    })

    ipcMain.on('install-update', () => {
      installDownloadedUpdate()
    })
  }

  return {
    startUpdateService,
    stopUpdateService,
    registerUpdateIpc,
    isForceUpdateActive: () => forceUpdateActive,
    getLastForceUpdateInfo: () => lastForceUpdateInfo
  }
}
