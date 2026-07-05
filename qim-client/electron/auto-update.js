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
    const baseUrl = getUpdateBaseUrl()
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
    const fallback = phase === 'download' || phase === 'downloading' ? '下载更新失败' : '检查更新失败'
    let errorMessage = fallback

    if (error?.message) {
      const msg = error.message.toLowerCase()

      if (msg.includes('404') || msg.includes('cannot find channel')) {
        errorMessage = phase === 'download' || phase === 'downloading' ? '下载更新失败：暂无可用安装包' : '暂无可用更新'
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

    if ((phase === 'download' || phase === 'downloading') && !errorMessage.includes('下载')) {
      errorMessage = `下载更新失败：${errorMessage}`
    }

    return errorMessage
  }

  function resetDownloadedUpdate() {
    downloadedUpdateInfo = null
  }

  function clearForceUpdate() {
    forceUpdateActive = false
    lastForceUpdateInfo = null
  }

  function checkForUpdates() {
    if (updatePhase === 'checking' || updatePhase === 'downloading') {
      console.log('更新检查已在进行中，忽略重复请求, currentUpdatePhase:', updatePhase)
      return
    }
    console.log('收到检查更新请求, currentUpdateBaseUrl:', getUpdateBaseUrl(), 'platform:', process.platform)
    const feedUrl = resolveUpdateFeedUrl()
    if (!feedUrl) {
      const error = `无法检查更新: 当前平台 ${process.platform} 不支持或服务器地址未配置 (currentUpdateBaseUrl: ${getUpdateBaseUrl()})`
      console.error(error)
      sendToWindow('update-error', error)
      return
    }

    console.log('设置更新服务器地址:', feedUrl)
    updatePhase = 'checking'
    errorReported = false
    autoUpdater.setFeedURL(updateFeedOptions(feedUrl))

    const timeout = setTimeout(() => {
      if (errorReported) return
      errorReported = true
      updatePhase = 'idle'
      console.error(`检查更新超时（${CHECK_UPDATE_TIMEOUT_MS / 1000}秒）`)
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
      })
      .catch(error => {
        clearTimeout(timeout)
        if (errorReported) return // 超时已处理，避免重复报错
        errorReported = true
        updatePhase = 'idle'
        console.error('检查更新失败:', error)
        sendToWindow('update-error', formatUpdateError(error, 'check'))
      })
  }

  function downloadUpdate() {
    updatePhase = 'downloading'
    errorReported = false
    autoUpdater.downloadUpdate()
      .catch(error => {
        console.error('下载更新失败:', error)
        updatePhase = 'idle'
        resetDownloadedUpdate()
        if (errorReported) return // on('error') 已报告，避免重复报错
        errorReported = true
        sendToWindow('update-error', formatUpdateError(error, 'download'))
      })
  }

  function installDownloadedUpdate() {
    if (!downloadedUpdateInfo) {
      sendToWindow('update-error', '更新文件尚未下载完成')
      return
    }

    clearForceUpdate()
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
      resetDownloadedUpdate()
      forceUpdateActive = !!info.forceUpdate
      lastForceUpdateInfo = {
        version: info.version,
        forceUpdate: info.forceUpdate || false,
        releaseDate: info.releaseDate,
        releaseName: info.releaseName,
        releaseNotes: info.releaseNotes
      }
      console.log('发现新版本:', info.version, '强制更新:', info.forceUpdate)
      sendToWindow('update-available', lastForceUpdateInfo)
    })

    autoUpdater.on('update-not-available', () => {
      updatePhase = 'idle'
      resetDownloadedUpdate()
      clearForceUpdate()
      console.log('当前已是最新版本')
      sendToWindow('update-not-available')
    })

    autoUpdater.on('error', (error) => {
      console.error('更新错误:', error)
      const errorMessage = formatUpdateError(error) // formatUpdateError 依赖当前 phase，需在重置前计算
      updatePhase = 'idle'
      resetDownloadedUpdate()
      clearForceUpdate()
      if (errorReported) return // 错误已由 timeout 或 .catch 报告，这里只做状态清理
      errorReported = true
      sendToWindow('update-error', errorMessage)
    })

    autoUpdater.on('download-progress', (progressObj) => {
      updatePhase = 'downloading'
      console.log('下载进度:', progressObj.percent)
      sendToWindow('update-progress', progressObj)
    })

    autoUpdater.on('update-downloaded', (info) => {
      updatePhase = 'downloaded'
      downloadedUpdateInfo = info
      console.log('更新下载完成，等待用户确认安装')
      sendToWindow('update-downloaded', info)
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
