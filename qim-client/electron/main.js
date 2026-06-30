// ==================== Imports & Setup ====================

import { app, BrowserWindow, Tray, Menu, nativeImage, ipcMain, globalShortcut, desktopCapturer, dialog, screen, systemPreferences } from 'electron'
import path from 'path'
import { fileURLToPath } from 'url'
import { execSync } from 'child_process'
import fs from 'fs'
import crypto from 'crypto'
import pkg from 'electron-updater'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const screenshots = require('./screenshots/lib/index.cjs').default
const { autoUpdater } = pkg

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const UPDATE_SERVER_URL = process.env.QIM_UPDATE_URL || 'http://localhost:8080'
const SCREENSHOT_CAPTURE_TIMEOUT_MS = 12000

// ==================== Single Instance & Protocol ====================

// 获取更新服务器地址（优先级：环境变量 > 配置文件 > 根据 isPackaged 判断）
function getUpdateServerUrl() {
  // 优先使用环境变量
  if (process.env.QIM_UPDATE_URL) {
    return process.env.QIM_UPDATE_URL
  }
  
  // 尝试从配置文件加载
  const savedUrl = loadServerConfig()
  if (savedUrl) {
    return savedUrl
  }
  
  // 根据是否打包判断环境
  return app.isPackaged 
    ? 'https://api.qim.work' 
    : 'http://localhost:8080'
}

if (app.isPackaged) {
  let gotTheLock = app.requestSingleInstanceLock()
  if (!gotTheLock) {
    // 锁获取失败，可能是进程被强杀后残留的锁文件，尝试清理后重试
    const lockPath = path.join(app.getPath('userData'), 'SingletonLock')
    try {
      if (fs.existsSync(lockPath)) {
        console.log('检测到残留锁文件，尝试清理:', lockPath)
        fs.unlinkSync(lockPath)
        gotTheLock = app.requestSingleInstanceLock()
      }
    } catch (e) {
      console.error('清理锁文件失败:', e)
    }
  }
  if (!gotTheLock) {
    console.log('应用已在运行，退出当前实例')
    app.quit()
    process.exit(0)
  }

  app.on('second-instance', (event, commandLine) => {
    const protocolUrl = commandLine.find(arg => arg.startsWith('qim://'))
    if (protocolUrl) {
      const httpUrl = protocolUrl.replace('qim://', 'http://localhost:3001/')
      handleAuthCallback(httpUrl)
    }

      showAndFocusWindow()
  })
}

app.setAsDefaultProtocolClient('qim')

// ==================== Icons ====================

function getIconPath(size = 512) {
  const iconDir = path.join(__dirname, 'icons')
  return path.join(iconDir, `icon_${size}x${size}.png`)
}

function loadIcon(size = 512) {
  const iconPath = getIconPath(size)
  try {
    const iconImage = fs.readFileSync(iconPath)
    return nativeImage.createFromBuffer(iconImage)
  } catch (error) {
    console.error('Error loading icon:', error)
    return null
  }
}

function getIconDataURL(size = 512) {
  const iconPath = getIconPath(size)
  try {
    const iconImage = fs.readFileSync(iconPath)
    const base64 = iconImage.toString('base64')
    return `data:image/png;base64,${base64}`
  } catch (error) {
    console.error('Error creating icon data URL:', error)
    return null
  }
}

// ==================== Helpers ====================

function sendToWindow(channel, ...args) {
  if (mainWindow) mainWindow.webContents.send(channel, ...args)
}

function showAndFocusWindow() {
  if (mainWindow) {
    if (mainWindow.isMinimized()) mainWindow.restore()
    mainWindow.show()
    mainWindow.focus()
  }
}

// ==================== Auth ====================

let authWindow = null
let isHandlingCallback = false
const AUTH_CALLBACK_BASE = 'http://localhost:23578'

function handleAuthCallback(callbackUrl) {
  if (isHandlingCallback) return
  isHandlingCallback = true

  try {
    const url = new URL(callbackUrl)
    const isOAuth = url.pathname.startsWith('/oauth')
    const code = url.searchParams.get('code') || ''
    const ticket = url.searchParams.get('ticket') || ''
    const state = url.searchParams.get('state') || ''

    console.log(`收到${isOAuth ? 'OAuth' : 'CAS'}回调:`, callbackUrl)

    if (authWindow && !authWindow.isDestroyed()) {
      authWindow.close()
      authWindow = null
    }

    if (mainWindow && !mainWindow.isDestroyed() && (code || ticket)) {
      showAndFocusWindow()

      const callbackData = isOAuth
        ? { code, state, type: 'oauth' }
        : { ticket, state, type: 'cas' }

      mainWindow.webContents.send(`${isOAuth ? 'oauth' : 'cas'}-callback`, callbackData)
    }
  } catch (err) {
    console.error('解析回调URL失败:', err)
  } finally {
    isHandlingCallback = false
  }
}

// ==================== Server Config ====================

function getConfigPath() {
  return path.join(app.getPath('userData'), 'config.json')
}

function loadServerConfig() {
  try {
    const configPath = getConfigPath()
    if (fs.existsSync(configPath)) {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'))
      if (config.serverUrl) {
        return config.serverUrl
      }
    }
  } catch (error) {
    console.error('读取配置失败:', error)
  }
  return null
}

function saveServerConfig(serverUrl) {
  try {
    const configPath = getConfigPath()
    const config = { serverUrl }
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2))
  } catch (error) {
    console.error('保存配置失败:', error)
  }
}

function getWindowsVersion() {
  if (process.platform !== 'win32') return null
  const userAgent = app.getUserAgent()
  const match = userAgent.match(/Windows NT (\d+)/)
  if (match) {
    return parseInt(match[1], 10)
  }
  return 10
}

function getAutoUpdateFeedUrl() {
  const baseUrl = currentUpdateBaseUrl
  if (process.platform === 'win32') {
    // Electron 22.x 用于 Windows 7 构建，对应 win7/ 更新通道
    const electronMajor = parseInt(process.versions.electron.split('.')[0], 10)
    if (electronMajor <= 22) {
      return `${baseUrl}/api/v1/updates/win7/`
    }
    return `${baseUrl}/api/v1/updates/win10/`
  }
  if (process.platform === 'linux') {
    return `${baseUrl}/api/v1/updates/linux/`
  }
  if (process.platform === 'darwin') {
    return `${baseUrl}/api/v1/updates/mac/`
  }
  return null
}

const savedUrl = loadServerConfig()
let currentUpdateBaseUrl = savedUrl || getUpdateServerUrl()

// ==================== Global State ====================

let mainWindow
let tray
let forceUpdateActive = false
let lastForceUpdateInfo = null

// Screenshot state
let screenshotInstance = null
let screenshotInitError = null
let screenshotContentProtectionEnabled = false

// Tray flash state
let trayFlashInterval = null
let isTrayFlashing = false
let normalTrayIcon = null
let hasUnread = false

// ==================== Window: Core ====================

function createWindow() {
  if (mainWindow) {
    console.log('Window already exists, showing it')
    mainWindow.show()
    return
  }
  console.log('Creating new window')

  const icon = loadIcon(256)
  const isMac = process.platform === 'darwin'
  const isLinux = process.platform === 'linux'

  // Splash
  const splashWindow = new BrowserWindow({
    width: 360,
    height: 320,
    frame: false,
    backgroundColor: isLinux ? '#e8ecf1' : '#00000000',
    transparent: !isLinux,
    alwaysOnTop: true,
    resizable: false,
    skipTaskbar: true,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true
    }
  })

  const splashPath = path.join(__dirname, 'splash.html')
  splashWindow.loadFile(splashPath)
  console.log(`Loading splash for version: v${app.getVersion()}`)

  // Main window
  const windowOptions = {
    width: 1200,
    height: 800,
    icon: icon,
    show: false,
    backgroundColor: '#e8ecf1',
    transparent: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false,
      webSecurity: false
    },
    frame: false,
    // 信创/Linux：关闭窗口阴影，避免合成器在窗口外圈再抠一次角导致黑边；
    // roundedCorners 仅 macOS 生效，保持默认（圆角），Linux/Windows 上传 false 会误关 macOS 系统圆角
    hasShadow: !isLinux
  }

  if (isMac) {
    windowOptions.titleBarStyle = 'customButtonsOnHover'
    windowOptions.titleBarOverlay = { visible: false, height: 0 }
    windowOptions.trafficLightPosition = { x: -100, y: -100 }
  }

  mainWindow = new BrowserWindow(windowOptions)

  const isDev = !app.isPackaged
  const url = isDev
    ? 'http://localhost:3000'
    : `file://${path.join(__dirname, '../dist/index.html')}`

  mainWindow.loadURL(url)
  console.log(`Loading URL: ${url}`)

  mainWindow.webContents.on('did-finish-load', () => {
    console.log('Render process loaded')
    if (isDev) {
      console.log('Opening DevTools in development mode')
      mainWindow.webContents.openDevTools()
    }
    if (forceUpdateActive && lastForceUpdateInfo) {
      sendToWindow('update-available', lastForceUpdateInfo)
    }
  })

  mainWindow.webContents.on('before-input-event', (event, input) => {
    if (!forceUpdateActive) return
    if ((input.meta || input.control) && input.key.toLowerCase() === 'r') {
      event.preventDefault()
    }
  })

  mainWindow.webContents.on('will-navigate', (event, url) => {
    if (!forceUpdateActive) return
    event.preventDefault()
  })

  mainWindow.webContents.on('did-fail-load', (event, errorCode, errorDescription) => {
    console.error(`Failed to load main window: ${errorDescription} (${errorCode})`)
    try {
      if (splashWindow && !splashWindow.isDestroyed()) {
        splashWindow.close()
      }
    } catch (e) {
      console.error('关闭splash窗口失败:', e)
    }
    mainWindow.loadURL(`data:text/html;charset=utf-8,${encodeURIComponent(`
      <html><body style="font-family:system-ui;background:#f0f2f5;display:flex;align-items:center;justify-content:center;">
        <div style="background:#fff;padding:32px;border-radius:12px;text-align:center;max-width:400px;">
          <h2 style="color:#f5222d;margin:0 0 12px;">加载失败</h2>
          <p style="color:#666;margin:0 0 8px;">${errorDescription}</p>
          ${isDev ? '<p style="color:#999;font-size:12px;margin:0;">请先运行 <code>npm run dev</code> 启动 Vite 开发服务器</p>' : ''}
        </div>
      </body></html>
    `)}`)
    mainWindow.show()
  })

  const startupTimeout = setTimeout(() => {
    if (splashWindow && !splashWindow.isDestroyed() && !mainWindow.isVisible()) {
      console.warn('Startup timeout: main window not ready after 10s')
      try {
        splashWindow.close()
      } catch (e) {
        console.error('关闭splash窗口失败:', e)
      }
      mainWindow.show()
    }
  }, 10000)

  mainWindow.once('ready-to-show', () => {
    clearTimeout(startupTimeout)
    console.log('Main window ready to show, closing splash')
    mainWindow.show()
    try {
      if (splashWindow && !splashWindow.isDestroyed()) {
        splashWindow.close()
      }
    } catch (e) {
      console.error('关闭splash窗口失败:', e)
    }
  })

  mainWindow.on('close', function () {
    try {
      if (splashWindow && !splashWindow.isDestroyed()) {
        splashWindow.close()
      }
    } catch (e) {
      console.error('关闭splash窗口失败:', e)
    }
    globalShortcut.unregisterAll()
    mainWindow = null
  })

  mainWindow.on('destroyed', function () {
    console.log('Window destroyed event triggered')
    mainWindow = null
  })

  registerGlobalShortcuts()
  initScreenshot()
}

function registerGlobalShortcuts() {
  globalShortcut.register('CommandOrControl+M', () => {
    if (mainWindow && !mainWindow.isDestroyed()) mainWindow.minimize()
  })
  globalShortcut.register('CommandOrControl+K', () => {
    if (!mainWindow || mainWindow.isDestroyed()) return
    if (mainWindow.isMaximized()) mainWindow.unmaximize()
    else mainWindow.maximize()
  })
  globalShortcut.register('CommandOrControl+W', () => {
    if (mainWindow && !mainWindow.isDestroyed()) mainWindow.hide()
  })
  globalShortcut.register('CommandOrControl+Q', () => app.quit())
}

// ==================== Screenshot ====================

function restoreMainWindowAfterScreenshot() {
  if (screenshotContentProtectionEnabled && mainWindow && !mainWindow.isDestroyed()) {
    try {
      mainWindow.setContentProtection(false)
    } catch (err) {
      console.error('[screenshot] Failed to disable content protection:', err)
    } finally {
      screenshotContentProtectionEnabled = false
    }
  }
  if (mainWindow && !mainWindow.isDestroyed()) {
    showAndFocusWindow()
  }
}

function getScreenshotDiagnostics() {
  const cursorPoint = screen.getCursorScreenPoint()
  const display = screen.getDisplayNearestPoint(cursorPoint)
  return {
    platform: process.platform,
    sessionType: process.env.XDG_SESSION_TYPE || 'unknown',
    desktopSession: process.env.DESKTOP_SESSION || 'unknown',
    waylandDisplay: process.env.WAYLAND_DISPLAY || '',
    x11Display: process.env.DISPLAY || '',
    displayId: display.id,
    scaleFactor: display.scaleFactor,
    bounds: display.bounds,
    screenshotOverlay: screenshotInstance?.getOverlayDiagnostics?.() || null
  }
}

function withScreenshotTimeout(capturePromise) {
  let timer
  const timeoutPromise = new Promise((_, reject) => {
    timer = setTimeout(() => {
      const err = Object.assign(
        new Error(`Screenshot capture timed out after ${SCREENSHOT_CAPTURE_TIMEOUT_MS}ms`),
        { code: 'capture_timeout' }
      )
      reject(err)
    }, SCREENSHOT_CAPTURE_TIMEOUT_MS)
  })

  return Promise.race([capturePromise, timeoutPromise]).finally(() => {
    clearTimeout(timer)
  })
}

function sendScreenshotError(message, err, code = 'capture_failed') {
  const diagnostics = getScreenshotDiagnostics()
  const errorCode = err?.code || code
  console.error('[screenshot]', message, { code: errorCode, diagnostics, err })
  restoreMainWindowAfterScreenshot()
  if (mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.webContents.send('screenshot-error', { message, code: errorCode, diagnostics })
  }
}

function ensureMacScreenRecordingPermission() {
  if (process.platform !== 'darwin') return true

  const status = systemPreferences.getMediaAccessStatus('screen')
  if (status === 'granted') return true

  try {
    systemPreferences.openSystemPreferences('security', 'Privacy_ScreenCapture')
  } catch (err) {
    console.error('[screenshot] Failed to open Screen Recording preferences:', err)
  }

  sendScreenshotError(
    '请在系统设置中允许 QIM 进行屏幕录制，然后重启应用后再截图',
    { code: 'screen_permission_denied', status },
    'screen_permission_denied'
  )
  return false
}

async function startScreenshotCapture({ hideMainWindow = false } = {}) {
  console.log('[screenshot] start capture', { hideMainWindow, diagnostics: getScreenshotDiagnostics() })

  if (!ensureMacScreenRecordingPermission()) {
    return
  }

  if (!screenshotInstance) {
    sendScreenshotError('截图组件尚未初始化，请稍后重试', null, 'not_initialized')
    return
  }

  if (!screenshotInstance._initialized) {
    mainWindow?.webContents?.send('screenshot-loading')
  }

  if (hideMainWindow && mainWindow && !mainWindow.isDestroyed()) {
    try {
      mainWindow.setContentProtection(true)
      screenshotContentProtectionEnabled = true
    } catch (err) {
      console.error('[screenshot] Failed to enable content protection:', err)
    }
  }

  try {
    await withScreenshotTimeout(screenshotInstance.startCapture())
  } catch (err) {
    sendScreenshotError('截图失败，请检查屏幕录制权限或稍后重试', err)
  }
}

function initScreenshot() {
  try {
    console.log('Initializing screenshots...')
    screenshotInstance = new screenshots({ singleWindow: true })

    screenshotInstance.on('ok', (e, buffer) => {
      console.log('[screenshot] Captured, buffer length:', buffer.length)
      restoreMainWindowAfterScreenshot()
      if (!mainWindow || mainWindow.isDestroyed()) return
      mainWindow.webContents.send('screenshot-taken', buffer)
    })

    screenshotInstance.on('cancel', () => {
      console.log('[screenshot] Cancelled')
      restoreMainWindowAfterScreenshot()
    })

    screenshotInstance.on('save', (e, buffer) => {
      console.log('[screenshot] Save triggered, buffer length:', buffer.length)
    })

    screenshotInstance.on('ready', () => {
      console.log('[screenshot] Tool ready')
    })

    screenshotInstance.on('error', (err) => {
      console.error('[screenshot] Error:', err)
      screenshotInitError = err
      sendScreenshotError('截图失败，请检查屏幕录制权限或稍后重试', err)
    })

    console.log('[screenshot] Instance created successfully')

    globalShortcut.register('CommandOrControl+Shift+A', () => {
      screenshotInstance?.startCapture?.()
    })
  } catch (error) {
    console.error('[screenshot] Failed to initialize:', error)
    screenshotInitError = error
  }
}

// ==================== Tray ====================

function createTray() {
  if (tray) {
    console.log('Tray already exists, returning')
    return
  }

  try {
    console.log('开始创建托盘图标')

    const image = loadIcon(22)
    tray = new Tray(image)
    console.log('托盘实例创建成功')

    const contextMenu = Menu.buildFromTemplate([
      {
        label: '显示应用',
        click: () => {
          if (mainWindow) {
            mainWindow.show()
          } else {
            createWindow()
          }
        }
      },
      {
        label: '退出',
        click: () => app.quit()
      }
    ])

    tray.setToolTip('QIM 应用')
    tray.setContextMenu(contextMenu)

    tray.on('click', () => {
      if (mainWindow) {
        if (mainWindow.isVisible()) {
          mainWindow.hide()
        } else {
          mainWindow.show()
        }
      } else {
        createWindow()
      }
    })

    console.log('托盘图标创建成功')
  } catch (error) {
    console.error('创建托盘图标时出错:', error)
  }
}

function flashTray() {
  if (!tray) return

  hasUnread = true

  if (mainWindow) {
    mainWindow.flashFrame(true)
  }

  if (process.platform === 'darwin') {
    app.dock?.bounce('informational')
    app.dock?.setBadge('!')
  }

  tray.setToolTip('QIM 应用 - 有新消息!')

  if (isTrayFlashing) return

  isTrayFlashing = true
  if (!normalTrayIcon) {
    normalTrayIcon = loadIcon(22)
  }
  let flashCount = 0
  const maxFlashCount = 20

  trayFlashInterval = setInterval(() => {
    flashCount++
    if (flashCount > maxFlashCount) {
      clearInterval(trayFlashInterval)
      trayFlashInterval = null
      isTrayFlashing = false
      if (normalTrayIcon) {
        tray.setImage(normalTrayIcon)
      }
      return
    }

    if (flashCount % 2 === 0) {
      if (normalTrayIcon) {
        tray.setImage(normalTrayIcon)
      }
    } else {
      tray.setImage(nativeImage.createEmpty())
    }
  }, 500)
}

function stopTrayFlash() {
  hasUnread = false

  if (trayFlashInterval) {
    clearInterval(trayFlashInterval)
    trayFlashInterval = null
  }
  isTrayFlashing = false

  if (normalTrayIcon && tray) {
    tray.setImage(normalTrayIcon)
  }

  if (process.platform === 'darwin') {
    app.dock?.setBadge('')
  }

  if (mainWindow) {
    mainWindow.flashFrame(false)
  }

  if (tray) {
    tray.setToolTip('QIM 应用')
  }
}

// ==================== Auto Updater ====================

function setupAutoUpdateUrl() {
  const feedUrl = getAutoUpdateFeedUrl()
  if (feedUrl) {
    console.log(`设置更新服务器地址: ${feedUrl}`)
    autoUpdater.setFeedURL({ provider: 'generic', url: feedUrl })
  } else {
    console.warn('无法设置更新服务器地址: feedUrl 为空, currentUpdateBaseUrl:', currentUpdateBaseUrl, 'platform:', process.platform)
  }
}

let updatePhase = 'idle'
let downloadedUpdateInfo = null

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

// Linux 平台安装更新（通过 sudo 执行安装脚本，因为 quitAndInstall 对 deb/AppImage 不生效）
async function installLinuxUpdate(info) {
  const downloadPath = info.path || info.downloadedFile
  if (!downloadPath || !fs.existsSync(downloadPath)) {
    console.error('Linux update: 下载文件未找到:', downloadPath)
    sendToWindow('update-error', '更新文件未找到')
    return
  }

  const isDeb = downloadPath.endsWith('.deb')
  const isRpm = downloadPath.endsWith('.rpm')

  if (!isDeb && !isRpm) {
    console.error('Linux update: 不支持的包格式:', downloadPath)
    sendToWindow('update-error', '不支持的 Linux 包格式')
    return
  }

  const helperScript = path.join(process.resourcesPath, 'install-update-linux.sh')
  if (!fs.existsSync(helperScript)) {
    console.error('Linux update: 安装脚本未找到:', helperScript)
    sendToWindow('update-error', '更新安装脚本未找到')
    return
  }

  try {
    execSync('which sudo', { stdio: 'ignore' })
  } catch {
    console.error('Linux update: sudo 未安装')
    sendToWindow('update-error', '未找到 sudo 命令')
    return
  }

  const escapedPath = downloadPath.replace(/'/g, "'\\''")
  const installCmd = `sudo -n "${helperScript}" "${escapedPath}"`

  try {
    console.log('Linux update: 执行安装命令:', installCmd)
    const result = execSync(installCmd, { timeout: 180000 })
    console.log('Linux update: 安装成功:', result.toString())

    sendToWindow('update-installed')

    setTimeout(() => {
      app.relaunch()
      app.quit()
    }, 2000)
  } catch (error) {
    console.error('Linux update: 安装失败:', error)
    let errorMsg = '安装更新失败'
    if (error.status === 1) {
      errorMsg = '更新安装失败，请检查系统包管理器状态'
    } else if (error.stderr) {
      const stderr = error.stderr.toString().trim()
      if (stderr.includes('a password is required')) {
        errorMsg = 'sudo 免密配置未生效，请运行: sudo visudo -f /etc/sudoers.d/qim-update'
      } else {
        errorMsg = stderr.split('\n').pop()
      }
    }
    sendToWindow('update-error', errorMsg)
  }
}

function setupAutoUpdater() {
  setupAutoUpdateUrl()

  // 不自动下载，等用户确认后再下载
  autoUpdater.autoDownload = false

  autoUpdater.on('checking-for-update', () => {
    updatePhase = 'checking'
    downloadedUpdateInfo = null
    console.log('正在检查更新...')
    sendToWindow('update-checking')
  })

  autoUpdater.on('update-available', (info) => {
    updatePhase = 'available'
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
    forceUpdateActive = false
    lastForceUpdateInfo = null
    console.log('当前已是最新版本')
    sendToWindow('update-not-available')
  })

  autoUpdater.on('error', (error) => {
    console.error('更新错误:', error)
    const errorMessage = formatUpdateError(error)
    updatePhase = 'idle'
    forceUpdateActive = false
    lastForceUpdateInfo = null
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

  // 定期自动检查更新（每 4 小时）
  const AUTO_UPDATE_INTERVAL = 4 * 60 * 60 * 1000
  let autoUpdateTimer = null

  // 自动检查更新（静默，不干扰用户）
  const autoCheckForUpdates = () => {
    if (!app.isPackaged) return
    console.log('[自动更新] 定期检查更新...')
    setupAutoUpdateUrl()
    autoUpdater.checkForUpdates().catch(error => {
      console.error('[自动更新] 定期检查失败:', error)
    })
  }

  // 启动定期检查
  autoUpdateTimer = setInterval(autoCheckForUpdates, AUTO_UPDATE_INTERVAL)

  // 应用启动后延迟 30 秒自动检查一次（避免阻塞启动）
  setTimeout(() => {
    if (app.isPackaged) {
      console.log('[自动更新] 启动后首次检查更新...')
      setupAutoUpdateUrl()
      autoUpdater.checkForUpdates().catch(error => {
        console.error('[自动更新] 启动检查失败:', error)
      })
    }
  }, 30000)

  if (app.isPackaged) {
    autoUpdater.checkForUpdates().catch(error => {
      console.error('检查更新失败:', error)
    })
  }
}

// ==================== IPC Handlers ====================

function checkForUpdates() {
  console.log('收到检查更新请求, currentUpdateBaseUrl:', currentUpdateBaseUrl, 'platform:', process.platform)
  const feedUrl = getAutoUpdateFeedUrl()
  if (!feedUrl) {
    const error = `无法检查更新: 当前平台 ${process.platform} 不支持或服务器地址未配置 (currentUpdateBaseUrl: ${currentUpdateBaseUrl})`
    console.error(error)
    sendToWindow('update-error', error)
    return
  }
  
  console.log('设置更新服务器地址:', feedUrl)
  updatePhase = 'checking'
  autoUpdater.setFeedURL({ provider: 'generic', url: feedUrl })
  
  // 设置超时，防止长时间无响应（增加到 10 秒）
  const timeout = setTimeout(() => {
    console.error('检查更新超时（10秒）')
    sendToWindow('update-error', '检查更新超时，请检查网络连接或服务器地址')
  }, 10000)
  
  // 监听一次 update-not-available 和 update-available 来清除超时
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
      updatePhase = 'idle'
      console.error('检查更新失败:', error)
      sendToWindow('update-error', formatUpdateError(error, 'check'))
    })
}

function registerIPC() {
  ipcMain.on('minimize-window', () => {
    if (mainWindow && !mainWindow.isDestroyed()) mainWindow.minimize()
  })

  ipcMain.on('maximize-window', () => {
    if (!mainWindow || mainWindow.isDestroyed()) return
    if (mainWindow.isMaximized()) {
      mainWindow.unmaximize()
    } else {
      mainWindow.maximize()
    }
  })

  ipcMain.on('close-window', () => {
    if (mainWindow) {
      mainWindow.hide()
    }
  })

  ipcMain.on('take-screenshot', () => {
    console.log('[screenshot] Received take-screenshot event')
    startScreenshotCapture({ hideMainWindow: false })
  })

  ipcMain.on('take-screenshot-hidden', () => {
    console.log('[screenshot] Received take-screenshot-hidden event')
    startScreenshotCapture({ hideMainWindow: true })
  })

  ipcMain.on('open-auth-login', (event, data) => {
    const { type, config, state } = data
    console.log('打开授权登录:', type, config)

    let authURL
    if (type === 'oauth') {
      const callbackUrl = `${AUTH_CALLBACK_BASE}/oauth/callback`
      authURL = `${config.auth_url}?client_id=${config.client_id}&redirect_uri=${encodeURIComponent(callbackUrl)}&response_type=code&scope=${config.scope}&state=${state}`
    } else if (type === 'cas') {
      const callbackUrl = `${AUTH_CALLBACK_BASE}/cas/callback?state=${encodeURIComponent(state)}`
      authURL = `${config.server_url}/login?service=${encodeURIComponent(callbackUrl)}`
    } else {
      console.error('未知的认证类型:', type)
      return
    }

    console.log('授权URL:', authURL)

    try {
      const parsed = new URL(authURL)
      if (!['https:', 'http:'].includes(parsed.protocol)) {
        console.error('不允许的协议:', parsed.protocol)
        event.sender.send('auth-error', '不允许的协议类型')
        return
      }
    } catch (e) {
      console.error('无效的授权URL:', authURL)
      event.sender.send('auth-error', '无效的授权URL，请检查认证配置')
      return
    }

    if (authWindow && !authWindow.isDestroyed()) {
      authWindow.close()
    }

    authWindow = new BrowserWindow({
      width: 1000,
      height: 800,
      title: '授权登录',
      autoHideMenuBar: true,
      webPreferences: {
        nodeIntegration: false,
        contextIsolation: true
      }
    })

    authWindow.setMenu(null)

    authWindow.webContents.on('did-fail-load', (event, errorCode, errorDescription, validatedURL) => {
      console.error('页面加载失败:', errorCode, errorDescription, validatedURL)
      event.sender.send('auth-error', `页面加载失败: ${errorDescription}`)
    })

    authWindow.webContents.on('did-finish-load', () => {
      console.log('页面加载完成')
    })

    authWindow.webContents.on('will-redirect', (event, url) => {
      if (url.startsWith(AUTH_CALLBACK_BASE)) {
        event.preventDefault()
        handleAuthCallback(url)
      }
    })

    authWindow.webContents.on('will-navigate', (event, url) => {
      if (url.startsWith(AUTH_CALLBACK_BASE)) {
        event.preventDefault()
        handleAuthCallback(url)
      }
    })

    authWindow.loadURL(authURL)
    authWindow.on('closed', () => { authWindow = null })
  })

  ipcMain.on('flash-tray', () => {
    flashTray()
  })

  ipcMain.on('stop-tray-flash', () => {
    stopTrayFlash()
  })

  ipcMain.on('set-server-url', (event, serverUrl) => {
    console.log('收到服务器地址更新:', serverUrl)
    if (serverUrl && typeof serverUrl === 'string') {
      currentUpdateBaseUrl = serverUrl.replace(/\/+$/, '')
      saveServerConfig(currentUpdateBaseUrl)
      setupAutoUpdateUrl()
      console.log('更新服务器地址已保存:', currentUpdateBaseUrl)
    }
  })

  ipcMain.on('get-server-url', (event) => {
    event.sender.send('server-url', currentUpdateBaseUrl)
  })

  ipcMain.handle('get-default-download-path', () => {
    return app.getPath('downloads')
  })

  ipcMain.on('check-for-updates', () => {
    checkForUpdates()
  })

  ipcMain.on('download-update', () => {
    updatePhase = 'downloading'
    autoUpdater.downloadUpdate().catch(error => {
      console.error('下载更新失败:', error)
      updatePhase = 'idle'
      sendToWindow('update-error', formatUpdateError(error, 'download'))
    })
  })

  ipcMain.on('install-update', () => {
    if (!downloadedUpdateInfo) {
      sendToWindow('update-error', '更新文件尚未下载完成')
      return
    }
    forceUpdateActive = false
    lastForceUpdateInfo = null
    sendToWindow('update-installing')
    // Linux 平台 quitAndInstall 不生效，需通过脚本安装 deb/AppImage
    if (process.platform === 'linux') {
      installLinuxUpdate(downloadedUpdateInfo)
    } else {
      autoUpdater.quitAndInstall(false, true)
    }
  })

  ipcMain.on('start-screen-share', async () => {
    try {
      console.log('启动屏幕共享')
      const sources = await desktopCapturer.getSources({
        types: ['screen', 'window'],
        thumbnailSize: { width: 640, height: 360 }
      })

      const sourcesWithThumbnails = sources.map(source => ({
        id: source.id,
        name: source.name,
        thumbnail: source.thumbnail.toDataURL()
      }))

      if (mainWindow) {
        mainWindow.webContents.send('screen-sources', sourcesWithThumbnails)
      }
    } catch (error) {
      console.error('获取屏幕源失败:', error)
    }
  })

  ipcMain.on('send-websocket-message', (event, message) => {
    console.log('发送WebSocket消息:', message.type)
    setTimeout(() => {
      if (mainWindow) {
        mainWindow.webContents.send('websocket-message', {
          type: message.type,
          data: {
            ...message.data,
            from_user_id: 1
          }
        })
      }
    }, 100)
  })

  ipcMain.on('webrtc-message', (event, message) => {
    console.log('处理WebRTC消息:', message.type)
    if (mainWindow) {
      mainWindow.webContents.send('webrtc-message', message)
    }
  })

  ipcMain.on('cache-avatar', async (event, avatarUrl) => {
    console.log('Received cache-avatar event for:', avatarUrl)
    try {
      const cachedUrl = await cacheAvatar(avatarUrl)
      event.sender.send('avatar-cached', cachedUrl || avatarUrl)
    } catch (error) {
      console.error('Error caching avatar:', error)
      event.sender.send('avatar-cached', avatarUrl)
    }
  })

  ipcMain.on('download-file', async (event, { buffer, fileName, mime, saveDir }) => {
    try {
      const targetDir = saveDir && saveDir !== '~/Downloads' ? saveDir : app.getPath('downloads')
      if (!fs.existsSync(targetDir)) {
        fs.mkdirSync(targetDir, { recursive: true })
      }
      const filePath = path.join(targetDir, fileName)
      fs.writeFileSync(filePath, Buffer.from(buffer))
      mainWindow?.webContents.send('download-complete', { success: true, filePath })
    } catch (error) {
      console.error('文件下载失败:', error)
      mainWindow?.webContents.send('download-complete', { success: false, error: error.message })
    }
  })

  ipcMain.on('save-file-as', async (event, { buffer, fileName, mime }) => {
    try {
      const result = await dialog.showSaveDialog(mainWindow, {
        title: '保存文件',
        defaultPath: fileName,
        filters: [{ name: 'All Files', extensions: ['*'] }]
      })

      if (!result.canceled && result.filePath) {
        fs.writeFileSync(result.filePath, Buffer.from(buffer))
        mainWindow?.webContents.send('save-file-complete', { success: true, filePath: result.filePath })
      }
    } catch (error) {
      console.error('文件保存失败:', error)
      mainWindow?.webContents.send('save-file-complete', { success: false, error: error.message })
    }
  })

  ipcMain.on('open-file-dialog', async (event, { properties }) => {
    try {
      const result = await dialog.showOpenDialog(mainWindow, {
        properties: properties || ['openDirectory']
      })
      event.sender.send('file-dialog-result', result)
    } catch (error) {
      console.error('打开文件对话框失败:', error)
      event.sender.send('file-dialog-result', { canceled: true })
    }
  })
}

// ==================== Avatar Cache ====================

function getCacheDir() {
  const appDataPath = app.getPath('userData')
  const cacheDir = path.join(appDataPath, 'avatar-cache')

  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true })
  }

  return cacheDir
}

function generateCacheFileName(avatarUrl) {
  const hash = crypto.createHash('md5').update(avatarUrl).digest('hex')
  let ext = 'png'

  const extMatch = avatarUrl.match(/\.([^.]+)(?:\?|$)/)
  if (extMatch && extMatch[1]) {
    ext = extMatch[1].split('?')[0].split('/')[0]
    if (ext.length > 10) {
      ext = 'png'
    }
  }

  return `${hash}.${ext}`
}

async function cacheAvatar(avatarUrl) {
  try {
    const cacheDir = getCacheDir()
    const cacheFileName = generateCacheFileName(avatarUrl)
    const cacheFilePath = path.join(cacheDir, cacheFileName)

    if (fs.existsSync(cacheFilePath)) {
      return `file://${cacheFilePath}`
    }

    const response = await fetch(avatarUrl)
    if (!response.ok) {
      throw new Error(`Failed to fetch avatar: ${response.status}`)
    }

    const buffer = await response.arrayBuffer()
    fs.writeFileSync(cacheFilePath, Buffer.from(buffer))

    return `file://${cacheFilePath}`
  } catch (error) {
    console.error('Error caching avatar:', error)
    return null
  }
}

function cleanupAvatarCache(maxAge = 7 * 24 * 60 * 60 * 1000) {
  try {
    const cacheDir = getCacheDir()
    const now = Date.now()

    fs.readdirSync(cacheDir).forEach(file => {
      const filePath = path.join(cacheDir, file)
      const stats = fs.statSync(filePath)

      if (now - stats.mtime.getTime() > maxAge) {
        fs.unlinkSync(filePath)
      }
    })
  } catch (error) {
    console.error('Error cleaning up avatar cache:', error)
  }
}

// ==================== App Lifecycle ====================

app.whenReady().then(() => {
  console.log('App ready')
  createWindow()
  createTray()
  registerIPC()
  setupAutoUpdater()

  if (app.dock) {
    const image = loadIcon(512)
    if (image) {
      app.dock.setIcon(image)
    }
  }

  cleanupAvatarCache()
  setInterval(() => {
    cleanupAvatarCache()
  }, 24 * 60 * 60 * 1000)
})

app.on('open-url', (event, url) => {
  console.log('收到 open-url:', url)
  event.preventDefault()
  if (url.startsWith('qim://')) {
    const httpUrl = url.replace('qim://', 'http://localhost:3001/')
    handleAuthCallback(httpUrl)
  }
})

app.on('activate', function () {
  if (!mainWindow) {
    console.log('activate event triggered')
    createWindow()
  }
})

app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') app.quit()
  globalShortcut.unregisterAll()
})
