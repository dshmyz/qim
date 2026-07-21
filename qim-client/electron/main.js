// ==================== Imports & Setup ====================

import { app, BrowserWindow, Tray, Menu, nativeImage, ipcMain, globalShortcut, desktopCapturer, dialog, screen, systemPreferences, session, shell, Notification } from 'electron'
import path from 'path'
import { fileURLToPath } from 'url'
import fs from 'fs'
import crypto from 'crypto'
import { createRequire } from 'node:module'
import { createUpdateService } from './auto-update.js'
import { DownloadRegistry } from './download-registry.js'

const require = createRequire(import.meta.url)
const screenshots = require('./screenshots/lib/index.cjs').default

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
  // app.whenReady 前不能用异步 API，用 Atomics.wait 同步阻塞（跨平台）
  const sleepSync = (ms) => {
    const arr = new Int32Array(new SharedArrayBuffer(4))
    Atomics.wait(arr, 0, 0, ms)
  }

  let gotTheLock = app.requestSingleInstanceLock()

  // 拿不到锁时重试，等待前一个实例的 quitAndInstall 释放锁（最多 2.5 秒）
  // 陈旧锁（前实例崩溃残留）由 Electron 的 requestSingleInstanceLock 自身处理，无需手动删锁
  let retry = 0
  while (!gotTheLock && retry < 5) {
    retry++
    sleepSync(500)
    gotTheLock = app.requestSingleInstanceLock()
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

function loadConfig() {
  try {
    const configPath = getConfigPath()
    if (fs.existsSync(configPath)) {
      const config = JSON.parse(fs.readFileSync(configPath, 'utf-8'))
      if (!config.shortcuts) {
        config.shortcuts = getDefaultShortcuts()
      }
      return config
    }
  } catch (error) {
    console.error('读取配置失败:', error)
  }
  return { serverUrl: null, shortcuts: getDefaultShortcuts() }
}

function saveConfig(config) {
  try {
    const configPath = getConfigPath()
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2))
  } catch (error) {
    console.error('保存配置失败:', error)
  }
}

function getDefaultShortcuts() {
  return {
    global: {
      minimize:   { accelerator: 'CommandOrControl+M', enabled: false },
      maximize:   { accelerator: 'CommandOrControl+K', enabled: false },
      hide:       { accelerator: 'CommandOrControl+Shift+H', enabled: false },
      quit:       { accelerator: 'CommandOrControl+Q', enabled: false },
      screenshot: { accelerator: 'CommandOrControl+Shift+A', enabled: true }
    },
    editor: {
      bold:   { accelerator: 'Mod-b', enabled: true },
      italic: { accelerator: 'Mod-i', enabled: true },
      link:   { accelerator: 'Mod-k', enabled: true },
      save:   { accelerator: 'Mod-s', enabled: true }
    }
  }
}

// 兼容旧接口：只读写 serverUrl
function loadServerConfig() {
  return loadConfig().serverUrl
}

function saveServerConfig(serverUrl) {
  const config = loadConfig()
  config.serverUrl = serverUrl
  saveConfig(config)
}

function getWindowsVersion() {
  if (process.platform !== 'win32') return null
  if (typeof process.getSystemVersion === 'function') {
    const systemVersion = process.getSystemVersion()
    const major = Number(systemVersion.split('.')[0])
    if (Number.isFinite(major)) return major
  }
  const userAgent = app.getUserAgent()
  const match = userAgent.match(/Windows NT (\d+)/)
  if (match) {
    return parseInt(match[1], 10)
  }
  return 10
}

const savedUrl = loadServerConfig()
let currentUpdateBaseUrl = savedUrl || getUpdateServerUrl()

// ==================== Global State ====================

let mainWindow
let tray
const updateService = createUpdateService({
  app,
  ipcMain,
  sendToWindow,
  getUpdateBaseUrl: () => currentUpdateBaseUrl,
  setUpdateBaseUrl: serverUrl => { currentUpdateBaseUrl = serverUrl },
  saveServerConfig
})

// Screenshot state
let screenshotInstance = null
let screenshotInitError = null
let screenshotContentProtectionEnabled = false

// Tray flash state
let trayFlashInterval = null
let isTrayFlashing = false
let normalTrayIcon = null
let blankTrayIcon = null
let hasUnread = false

// ==================== File Download (built-in download manager) ====================
// 用 Electron 内置下载管理器流式写盘，内存恒定，不受大文件影响。
// 每次下载生成唯一 requestUrl，避免同一文件并发下载互相覆盖元信息。
const downloadRegistry = new DownloadRegistry()
let downloadHandlerRegistered = false

function registerDownloadHandlers(sess) {
  if (downloadHandlerRegistered) return
  downloadHandlerRegistered = true

  sess.webRequest.onBeforeSendHeaders({ urls: ['<all_urls>'] }, (details, cb) => {
    const authorization = downloadRegistry.getAuthHeader(details.url)
    if (!authorization) {
      cb({ requestHeaders: details.requestHeaders })
      return
    }
    cb({ requestHeaders: { ...details.requestHeaders, Authorization: authorization } })
  })

  sess.on('will-download', (event, item) => {
    const actualUrl = item.getURL()
    const meta = downloadRegistry.consume(actualUrl)
    if (!meta) return

    const name = meta.fileName || item.getFilename()
    if (meta.savePath) {
      // 前端已选好路径（另存为模式），直接用
      item.setSavePath(meta.savePath)
    } else {
      const targetDir = meta.saveDir && meta.saveDir !== '~/Downloads' ? meta.saveDir : app.getPath('downloads')
      if (!fs.existsSync(targetDir)) fs.mkdirSync(targetDir, { recursive: true })
      item.setSavePath(path.join(targetDir, name))
    }

    const sendProgress = (percent, state) => {
      if (meta.downloadId == null || !mainWindow || mainWindow.isDestroyed()) return
      mainWindow.webContents.send('download-progress', {
        downloadId: meta.downloadId,
        percent,
        received: item.getReceivedBytes(),
        total: item.getTotalBytes(),
        state
      })
    }

    item.on('updated', (_e, state) => {
      if (state !== 'progressing') return
      const total = item.getTotalBytes()
      const percent = total > 0 ? Math.floor((item.getReceivedBytes() / total) * 100) : 0
      sendProgress(percent, 'progressing')
    })

    item.once('done', (_e, state) => {
      console.log('[download] done state:', state, 'downloadId:', meta.downloadId, 'path:', item.getSavePath())
      if (mainWindow && !mainWindow.isDestroyed()) {
        sendProgress(state === 'completed' ? 100 : 0, state)
        if (state === 'completed') {
          const payload = {
            success: true,
            filePath: item.getSavePath(),
            downloadId: meta.downloadId
          }
          mainWindow.webContents.send(meta.completeChannel, payload)
        } else if (state === 'cancelled') {
          mainWindow.webContents.send(meta.completeChannel, {
            success: false,
            cancelled: true,
            downloadId: meta.downloadId
          })
        } else {
          mainWindow.webContents.send(meta.completeChannel, {
            success: false,
            error: state === 'interrupted' ? '下载中断' : state,
            downloadId: meta.downloadId
          })
        }
      }
    })
  })
}

function triggerDownload({ url, token, fileName, saveDir, savePath, downloadId, completeChannel }) {
  try {
    if (!mainWindow || mainWindow.isDestroyed()) return
    const contents = mainWindow.webContents
    const sess = contents.session
    registerDownloadHandlers(sess)
    const meta = downloadRegistry.create({ url, token, fileName, saveDir, savePath, downloadId, completeChannel })
    contents.downloadURL(meta.requestUrl)
  } catch (error) {
    console.error('文件下载失败:', error)
    mainWindow?.webContents.send(completeChannel, { success: false, error: error.message })
  }
}

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
  if (isLinux) {
    windowOptions.transparent = true
    windowOptions.icon = loadIcon(64)
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
    const lastForceUpdateInfo = updateService.getLastForceUpdateInfo()
    if (updateService.isForceUpdateActive() && lastForceUpdateInfo) {
      sendToWindow('update-available', lastForceUpdateInfo)
    }
  })

  mainWindow.webContents.on('before-input-event', (event, input) => {
    if (!updateService.isForceUpdateActive()) return
    if ((input.meta || input.control) && input.key.toLowerCase() === 'r') {
      event.preventDefault()
    }
  })

  mainWindow.webContents.on('will-navigate', (event, url) => {
    if (!updateService.isForceUpdateActive()) return
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

  mainWindow.on('focus', () => {
    stopTrayFlash()
  })

  initScreenshot()
  registerGlobalShortcuts()
}

function registerGlobalShortcuts() {
  const { shortcuts } = loadConfig()
  const handlers = {
    minimize:   () => { if (mainWindow && !mainWindow.isDestroyed()) mainWindow.minimize() },
    maximize:   () => {
      if (!mainWindow || mainWindow.isDestroyed()) return
      if (mainWindow.isMaximized()) mainWindow.unmaximize()
      else mainWindow.maximize()
    },
    hide:       () => { if (mainWindow && !mainWindow.isDestroyed()) mainWindow.hide() },
    quit:       () => app.quit(),
    screenshot: () => screenshotInstance?.startCapture?.()
  }
  for (const [name, conf] of Object.entries(shortcuts.global)) {
    if (!conf.enabled || !conf.accelerator) continue
    const registered = globalShortcut.register(conf.accelerator, handlers[name])
    if (!registered) {
      console.warn(`[shortcut] 注册失败: ${conf.accelerator} (${name})，可能被其他应用占用`)
    }
  }
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

function shouldHideMainWindowForScreenshot() {
  return process.platform === 'linux' || (process.platform === 'win32' && getWindowsVersion() <= 6)
}

function waitForWindowHiddenBeforeScreenshot(win) {
  if (!win || win.isDestroyed()) return Promise.resolve()
  const settleDelayMs = process.platform === 'win32' ? 180 : 80

  return new Promise((resolve) => {
    if (!win.isVisible()) {
      setTimeout(resolve, settleDelayMs)
      return
    }

    let settled = false
    const done = () => {
      if (settled) return
      settled = true
      setTimeout(resolve, settleDelayMs)
    }

    win.once('hide', done)
    win.hide()
    setTimeout(done, settleDelayMs)
  })
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
    if (shouldHideMainWindowForScreenshot()) {
      // Win7 的内容保护会被截图成灰色面板；Linux 上 setContentProtection 也是 no-op。
      await waitForWindowHiddenBeforeScreenshot(mainWindow)
    } else {
      try {
        mainWindow.setContentProtection(true)
        screenshotContentProtectionEnabled = true
      } catch (err) {
        console.error('[screenshot] Failed to enable content protection:', err)
      }
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
      // 通知渲染进程截图已取消，避免 screenshotStatus 卡在非 idle 状态
      if (mainWindow && !mainWindow.isDestroyed()) {
        mainWindow.webContents.send('screenshot-taken', null)
      }
    })

    screenshotInstance.on('error', (err) => {
      console.error('[screenshot] Error:', err)
      screenshotInitError = err
      sendScreenshotError('截图失败，请检查屏幕录制权限或稍后重试', err)
    })

    console.log('[screenshot] Instance created successfully')
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

function getBlankTrayIcon() {
  if (!blankTrayIcon) {
    blankTrayIcon = nativeImage.createFromDataURL(
      'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABYAAAAWCAYAAADEtGw7AAAAF0lEQVR4nGNgGAWjYBSMglEwCkbB4AQAB6YAAW3k4IMAAAAASUVORK5CYII='
    )
  }
  return blankTrayIcon
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

  trayFlashInterval = setInterval(() => {
    flashCount++
    if (flashCount % 2 === 0) {
      if (normalTrayIcon) {
        tray.setImage(normalTrayIcon)
      }
    } else {
      tray.setImage(getBlankTrayIcon())
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

// ==================== IPC Handlers ====================

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

  // 主进程通知：渲染进程通过 IPC 触发。用打包内 app 图标的绝对路径，
  // Linux libnotify 能稳定渲染（Web Notification 的 data URL/相对路径 icon 在 Linux 下不工作）。
  ipcMain.handle('notification:show', (_e, { title, body }) => {
    try {
      const iconPath = path.join(app.getAppPath(), 'electron/icons/icon_512x512.png')
      const icon = nativeImage.createFromPath(iconPath)
      const n = new Notification({ title: title || 'QIM', body: body || '', icon, silent: false })
      n.on('click', () => {
        if (mainWindow && !mainWindow.isDestroyed()) {
          mainWindow.show()
          mainWindow.focus()
        }
      })
      n.show()
      return true
    } catch (err) {
      console.error('[notification] show failed:', err)
      return false
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

  ipcMain.handle('is-main-window-active', () => {
    try {
      return Boolean(
        mainWindow &&
        !mainWindow.isDestroyed() &&
        mainWindow.isVisible() &&
        !mainWindow.isMinimized() &&
        mainWindow.isFocused()
      )
    } catch (error) {
      console.warn('Failed to read main window activity; treating it as inactive:', error)
      return false
    }
  })

  updateService.registerUpdateIpc()

  ipcMain.handle('get-default-download-path', () => {
    return app.getPath('downloads')
  })

  ipcMain.handle('get-shortcuts', () => {
    return loadConfig().shortcuts
  })

  ipcMain.handle('set-shortcuts', (event, shortcuts) => {
    const config = loadConfig()
    config.shortcuts = shortcuts
    saveConfig(config)
    globalShortcut.unregisterAll()
    registerGlobalShortcuts()
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('shortcuts-updated', shortcuts)
    }
    return shortcuts
  })

  ipcMain.handle('reset-shortcuts', () => {
    const defaults = getDefaultShortcuts()
    const config = loadConfig()
    config.shortcuts = defaults
    saveConfig(config)
    globalShortcut.unregisterAll()
    registerGlobalShortcuts()
    if (mainWindow && !mainWindow.isDestroyed()) {
      mainWindow.webContents.send('shortcuts-updated', defaults)
    }
    return defaults
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

  ipcMain.on('download-file', (event, { url, token, fileName, saveDir, downloadId }) => {
    triggerDownload({ url, token, fileName, saveDir, downloadId, completeChannel: 'download-complete' })
  })

  ipcMain.handle('show-save-dialog', async (event, { defaultPath }) => {
    const result = await dialog.showSaveDialog(mainWindow, {
      title: '保存文件',
      defaultPath,
      filters: [{ name: 'All Files', extensions: ['*'] }]
    })
    return result
  })

  ipcMain.on('save-file-as', (event, { url, token, fileName, savePath, downloadId }) => {
    triggerDownload({ url, token, fileName, savePath, downloadId, completeChannel: 'save-file-complete' })
  })

  // 用系统默认应用打开文件
  ipcMain.handle('open-file', async (event, filePath) => {
    try {
      await shell.openPath(filePath)
      return { success: true }
    } catch (error) {
      console.error('打开文件失败:', error)
      return { success: false, error: String(error) }
    }
  })

  // 在资源管理器/访达中显示文件所在位置
  ipcMain.handle('show-file-in-folder', async (event, filePath) => {
    try {
      shell.showItemInFolder(filePath)
      return { success: true }
    } catch (error) {
      console.error('显示文件位置失败:', error)
      return { success: false, error: String(error) }
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

// 请求麦克风/摄像头系统权限（macOS），确保 getUserMedia 能正常触发授权
async function ensureMediaPermissions() {
  // macOS: 主动触发系统授权弹窗
  if (process.platform === 'darwin') {
    try {
      await systemPreferences.askForMediaAccess('microphone')
      await systemPreferences.askForMediaAccess('camera')
    } catch (err) {
      console.error('[MediaPermission] 请求媒体权限失败:', err)
    }
  }

  // 所有平台: 配置权限请求处理器，允许 media 和安全的剪贴板写入请求
  session.defaultSession.setPermissionRequestHandler((_webContents, permission, callback) => {
    // 白名单内的权限放行；其余拒绝。
    // 注意：未列权限调 callback(false) 会让对应 Web API 的 Promise 挂起（不 reject、不报错），
    // 新增用到需要权限的 Web API 时务必同步加白名单。
    // - media: 通话/屏幕共享 getUserMedia
    // - clipboard-sanitized-write: writeText 写剪贴板
    // - clipboard-read: readText 读剪贴板（粘贴、小程序粘贴）
    // - notifications: 便签/日历提醒 Web Notification
    // - fullscreen: Element.requestFullscreen（屏幕共享全屏，缺则挂起）
    if (['media', 'clipboard-sanitized-write', 'clipboard-read', 'notifications', 'fullscreen'].includes(permission)) {
      callback(true)
    } else {
      callback(false)
    }
  })
}

app.whenReady().then(async () => {
  console.log('App ready')
  await ensureMediaPermissions()
  createWindow()
  createTray()
  registerIPC()
  updateService.startUpdateService()

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

app.on('before-quit', () => {
  updateService.stopUpdateService()
})
