import { app, BrowserWindow, Tray, Menu, nativeImage, ipcMain, globalShortcut, desktopCapturer, screen, dialog } from 'electron'
import path from 'path'
import { fileURLToPath } from 'url'
import { execSync } from 'child_process'
import fs from 'fs'
import os from 'os'
import crypto from 'crypto'
import pkg from 'electron-updater'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const screenshots = require('./screenshots/lib/index.cjs').default
const { autoUpdater } = pkg

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

// 单开实例控制 + 自定义协议回调（仅打包模式启用，开发模式允许多开）
if (app.isPackaged) {
  const gotTheLock = app.requestSingleInstanceLock()
  if (!gotTheLock) {
    console.log('应用已在运行，退出当前实例')
    app.quit()
    process.exit(0)
  }

  app.on('second-instance', (event, commandLine, workingDirectory) => {
    // 处理自定义协议回调（Windows/Linux 生产环境）
    const protocolUrl = commandLine.find(arg => arg.startsWith('qim://'))
    if (protocolUrl) {
      const httpUrl = protocolUrl.replace('qim://', 'http://localhost:3001/')
      handleAuthCallback(httpUrl)
    }

    if (mainWindow) {
      if (mainWindow.isMinimized()) {
        mainWindow.restore()
      }
      mainWindow.focus()
      mainWindow.show()
    }
  })
}

function getIconPath(size = 512) {
  const iconDir = path.join(__dirname, 'icons')
  const iconPath = path.join(iconDir, `icon_${size}x${size}.png`)
  if (fs.existsSync(iconPath)) {
    return iconPath
  }
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

const UPDATE_SERVER_URL = process.env.QIM_UPDATE_URL || 'http://localhost:8080'

// 认证回调
let authWindow = null
let isHandlingCallback = false
const AUTH_CALLBACK_BASE = 'http://localhost:3001'

// 处理认证回调URL
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
    
    // 关闭授权窗口（开发模式）
    if (authWindow && !authWindow.isDestroyed()) {
      authWindow.close()
      authWindow = null
    }
    
    if (mainWindow && !mainWindow.isDestroyed() && (code || ticket)) {
      if (mainWindow.isMinimized()) mainWindow.restore()
      mainWindow.show()
      mainWindow.focus()
      
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

// 注册自定义协议（打包后系统浏览器通过此协议唤起应用）
app.setAsDefaultProtocolClient('qim')

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

const savedUrl = loadServerConfig()
let currentUpdateBaseUrl = savedUrl || UPDATE_SERVER_URL

function getUpdateBaseUrl() {
  return currentUpdateBaseUrl
}

const autoUpdaterConfig = {
  win7: {
    version: '22.3.27'
  },
  win10: {
    version: '33.0.0'
  },
  linux: {
    version: '1.0.0'
  }
}

function getWindowsVersion() {
  const platform = process.platform
  if (platform !== 'win32') return null

  const userAgent = app.getUserAgent()
  const match = userAgent.match(/Windows NT (\d+)/)
  if (match) {
    return parseInt(match[1], 10)
  }
  return 10
}

function getAutoUpdaterConfig() {
  const baseUrl = getUpdateBaseUrl()
  const platform = process.platform
  if (platform === 'win32') {
    const winVersion = getWindowsVersion()
    if (winVersion && winVersion < 10) {
      return { ...autoUpdaterConfig.win7, feedUrl: `${baseUrl}/api/v1/updates/win7/` }
    }
    return { ...autoUpdaterConfig.win10, feedUrl: `${baseUrl}/api/v1/updates/win10/` }
  }
  if (platform === 'linux') {
    return { ...autoUpdaterConfig.linux, feedUrl: `${baseUrl}/api/v1/updates/linux/` }
  }
  return null
}

let mainWindow
let tray

function createWindow() {
  if (mainWindow) {
    console.log('Window already exists, showing it')
    mainWindow.show()
    return
  }else {
    console.log('Creating new window')
  }

  const icon = loadIcon(256)

  // 创建启动页面窗口
  const splashWindow = new BrowserWindow({
    width: 360,
    height: 320,
    frame: false,
    transparent: true,
    alwaysOnTop: true,
    resizable: false,
    skipTaskbar: true,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true
    }
  })

  const splashPath = `file://${path.join(__dirname, 'splash.html')}`
  splashWindow.loadURL(splashPath)
  console.log(`Loading splash: ${splashPath}`)

  const isMac = process.platform === 'darwin'
  const windowOptions = {
    width: 1200,
    height: 800,
    icon: icon,
    show: false,
    backgroundColor: '#e8ecf1',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      nodeIntegration: false,
      contextIsolation: true,
      sandbox: false,
      webSecurity: false
    },
    frame: false
  }

  // macOS 专属配置，Windows/Linux 不适用
  if (isMac) {
    windowOptions.titleBarStyle = 'customButtonsOnHover'
    windowOptions.titleBarOverlay = {
      visible: false,
      height: 0
    }
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
  })

  mainWindow.webContents.on('did-fail-load', (event, errorCode, errorDescription) => {
    console.error(`Failed to load main window: ${errorDescription} (${errorCode})`)
    if (splashWindow && !splashWindow.isDestroyed()) {
      splashWindow.close()
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

  // 启动超时保护：10 秒后如果还没 ready，关闭 splash
  const startupTimeout = setTimeout(() => {
    if (splashWindow && !splashWindow.isDestroyed() && !mainWindow.isVisible()) {
      console.warn('Startup timeout: main window not ready after 10s')
      splashWindow.close()
      mainWindow.show()
    }
  }, 10000)

  mainWindow.once('ready-to-show', () => {
    clearTimeout(startupTimeout)
    console.log('Main window ready to show, closing splash')
    mainWindow.show()
    splashWindow.close()
  })

  mainWindow.on('close', function () {
    if (splashWindow && !splashWindow.isDestroyed()) {
      splashWindow.close()
    }
    mainWindow = null
  })

  mainWindow.on('destroyed', function () {
    console.log('Window destroyed event triggered')
    mainWindow = null
  })

  console.log('Setting up window control event listeners')

  globalShortcut.register('CommandOrControl+M', () => {
    console.log('Global shortcut: CommandOrControl+M pressed')
    mainWindow.minimize()
  })

  globalShortcut.register('CommandOrControl+K', () => {
    console.log('Global shortcut: CommandOrControl+K pressed')
    if (mainWindow.isMaximized()) {
      mainWindow.unmaximize()
    } else {
      mainWindow.maximize()
    }
  })

  globalShortcut.register('CommandOrControl+W', () => {
    console.log('Global shortcut: CommandOrControl+W pressed')
    mainWindow.hide()
  })

  globalShortcut.register('CommandOrControl+Q', () => {
    console.log('Global shortcut: CommandOrControl+Q pressed')
    app.quit()
  })

  // 初始化截图功能
  let screenshotInstance = null
  let screenshotInitError = null

  try {
    console.log('Initializing screenshots...')
    screenshotInstance = new screenshots({ singleWindow: true })
    screenshotInstance.on('ok', (e, buffer, data) => {
      console.log('[screenshot] Captured, buffer length:', buffer.length)
      if (mainWindow) {
        mainWindow.show()
        const img = nativeImage.createFromBuffer(buffer)
        const dataUrl = img.toDataURL()
        console.log('[screenshot] DataURL created, length:', dataUrl.length)
        mainWindow.webContents.send('screenshot-taken', dataUrl)
      }
    })

    screenshotInstance.on('cancel', (e) => {
      console.log('[screenshot] Cancelled')
      if (mainWindow) {
        mainWindow.show()
      }
    })

    screenshotInstance.on('save', (e, buffer, data) => {
      console.log('[screenshot] Save triggered, buffer length:', buffer.length)
    })

    screenshotInstance.on('ready', () => {
      console.log('[screenshot] Tool ready')
    })

    screenshotInstance.on('error', (err) => {
      console.error('[screenshot] Error:', err)
      screenshotInitError = err
    })

    console.log('[screenshot] Instance created successfully')
    
    // 注册截图快捷键
    globalShortcut.register('CommandOrControl+Shift+A', () => {
      console.log('Global shortcut: CommandOrControl+Shift+A pressed, starting screenshot')
      if (screenshotInstance) {
        try {
          screenshotInstance.startCapture()
        } catch (error) {
          console.error('[screenshot] Error starting capture:', error)
        }
      } else {
        console.error('[screenshot] Cannot capture: instance not initialized')
      }
    })
  } catch (error) {
    console.error('[screenshot] Failed to initialize:', error)
    screenshotInitError = error
  }

  ipcMain.on('take-screenshot', () => {
    console.log('[screenshot] Received take-screenshot event')
    console.log('[screenshot] Instance exists:', !!screenshotInstance)
    console.log('[screenshot] Init error:', screenshotInitError)

    if (screenshotInstance) {
      try {
        console.log('[screenshot] Starting capture...')
        screenshotInstance.startCapture()
      } catch (error) {
        console.error('[screenshot] Error starting capture:', error)
      }
    } else {
      console.error('[screenshot] Cannot capture: instance not initialized')
    }
  })

  ipcMain.on('minimize-window', () => {
    console.log('Received minimize-window event')
    mainWindow.minimize()
  })

  ipcMain.on('maximize-window', () => {
    console.log('Received maximize-window event')
    if (mainWindow.isMaximized()) {
      console.log('Window is maximized, unmaximizing')
      mainWindow.unmaximize()
    } else {
      console.log('Window is not maximized, maximizing')
      mainWindow.maximize()
    }
  })

  ipcMain.on('close-window', () => {
    console.log('Received close-window event')
    if (mainWindow) {
      mainWindow.hide()
    }
  })

  ipcMain.on('open-auth-login', (event, data) => {
    const { type, config, state } = data
    console.log('打开授权登录:', type, config)

    let authURL
    if (type === 'oauth') {
      const callbackUrl = `${AUTH_CALLBACK_BASE}/oauth/callback`
      authURL = `${config.auth_url}?client_id=${config.client_id}&redirect_uri=${encodeURIComponent(callbackUrl)}&response_type=code&scope=${config.scope}&state=${state}`
    } else if (type === 'cas') {
      const callbackUrl = `${AUTH_CALLBACK_BASE}/cas/callback`
      authURL = `${config.cas_url}/login?service=${encodeURIComponent(callbackUrl)}`
    } else {
      console.error('未知的认证类型:', type)
      return
    }

    console.log('授权URL:', authURL)

    // 校验URL协议
    try {
      const parsed = new URL(authURL)
      if (!['https:', 'http:'].includes(parsed.protocol)) {
        console.error('不允许的协议:', parsed.protocol)
        return
      }
    } catch (e) {
      console.error('无效的授权URL:', authURL)
      return
    }

    // 内嵌窗口：BrowserWindow + 拦截回调重定向
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

  let trayFlashInterval = null
  let isTrayFlashing = false
  let normalTrayIcon = null
  let hasUnread = false

  ipcMain.on('flash-tray', () => {
    console.log('Received flash-tray event')
    if (!tray) return

    hasUnread = true

    // 使用 flashFrame 让任务栏/停靠图标闪烁（跨平台支持）
    if (mainWindow) {
      mainWindow.flashFrame(true)
    }

    if (process.platform === 'darwin') {
      app.dock?.bounce('informational')
      // 设置 Dock 徽章
      app.dock?.setBadge('!')
    }

    // 对于所有平台，更新托盘工具提示
    tray.setToolTip('QIM 应用 - 有新消息!')

    if (isTrayFlashing) return

    isTrayFlashing = true
    // 使用 loadIcon 获取默认图标，而不是 tray.getImage()（Electron Tray 没有此方法）
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

      // 闪烁效果：交替设置空图标和正常图标
      if (flashCount % 2 === 0) {
        if (normalTrayIcon) {
          tray.setImage(normalTrayIcon)
        }
      } else {
        // 创建一个空/透明图标实现闪烁效果
        const emptyImage = nativeImage.createEmpty()
        tray.setImage(emptyImage)
      }
    }, 500)
  })

  ipcMain.on('stop-tray-flash', () => {
    console.log('Received stop-tray-flash event')
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
  })

  console.log('Global shortcuts registered')
}

function createTray() {
  if (tray) {
    console.log('Tray already exists, returning')
    return
  }

  try {
    console.log('开始创建托盘图标')

    const image = loadIcon(22)
    console.log('创建托盘实例')
    tray = new Tray(image)
    console.log('托盘实例创建成功')

    const contextMenu = Menu.buildFromTemplate([
      {
        label: '显示应用',
        click: () => {
          console.log('点击显示应用')
          if (mainWindow) {
            mainWindow.show()
          } else {
            createWindow()
          }
        }
      },
      {
        label: '退出',
        click: () => {
          console.log('点击退出')
          app.quit()
        }
      }
    ])

    console.log('设置托盘工具提示')
    tray.setToolTip('QIM 应用')

    console.log('设置托盘上下文菜单')
    tray.setContextMenu(contextMenu)

    tray.on('click', () => {
      console.log('点击托盘图标')
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

app.whenReady().then(() => {
  console.log('App ready')
  createWindow()
  createTray()

  if (app.dock) {
    const image = loadIcon(512)
    if (image) {
      app.dock.setIcon(image)
    }
  }
})

// macOS: 通过 open-url 事件接收自定义协议回调（生产环境）
app.on('open-url', (event, url) => {
  console.log('收到 open-url:', url)
  event.preventDefault()
  if (url.startsWith('qim://')) {
    // qim://oauth/callback -> http://localhost:3001/oauth/callback
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


function setupAutoUpdater() {
  setupAutoUpdateUrl()

  autoUpdater.on('checking-for-update', () => {
    console.log('正在检查更新...')
    if (mainWindow) {
      mainWindow.webContents.send('update-checking')
    }
  })

  autoUpdater.on('update-available', (info) => {
    console.log('发现新版本:', info.version)
    if (mainWindow) {
      mainWindow.webContents.send('update-available', info)
    }
  })

  autoUpdater.on('update-not-available', () => {
    console.log('当前已是最新版本')
    if (mainWindow) {
      mainWindow.webContents.send('update-not-available')
    }
  })

  autoUpdater.on('error', (error) => {
    console.error('更新错误:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })

  autoUpdater.on('download-progress', (progressObj) => {
    console.log('下载进度:', progressObj.percent)
    if (mainWindow) {
      mainWindow.webContents.send('update-progress', progressObj)
    }
  })

  autoUpdater.on('update-downloaded', (info) => {
    console.log('更新下载完成，准备安装')
    if (mainWindow) {
      mainWindow.webContents.send('update-downloaded', info)
    }

    if (process.platform === 'linux') {
      installLinuxUpdate(info)
    } else {
      autoUpdater.quitAndInstall()
    }
  })

  async function installLinuxUpdate(info) {
    const downloadPath = info.path || info.downloadedFile
    if (!downloadPath || !fs.existsSync(downloadPath)) {
      console.error('Linux update: 下载文件未找到:', downloadPath)
      if (mainWindow) {
        mainWindow.webContents.send('update-error', '更新文件未找到')
      }
      return
    }

    const isDeb = downloadPath.endsWith('.deb')
    const isRpm = downloadPath.endsWith('.rpm')

    if (!isDeb && !isRpm) {
      console.error('Linux update: 不支持的包格式:', downloadPath)
      if (mainWindow) {
        mainWindow.webContents.send('update-error', '不支持的 Linux 包格式')
      }
      return
    }

    const helperScript = path.join(process.resourcesPath, 'install-update-linux.sh')
    if (!fs.existsSync(helperScript)) {
      console.error('Linux update: 安装脚本未找到:', helperScript)
      if (mainWindow) {
        mainWindow.webContents.send('update-error', '更新安装脚本未找到')
      }
      return
    }

    try {
      execSync('which sudo', { stdio: 'ignore' })
    } catch {
      console.error('Linux update: sudo 未安装')
      if (mainWindow) {
        mainWindow.webContents.send('update-error', '未找到 sudo 命令')
      }
      return
    }

    const escapedPath = downloadPath.replace(/'/g, "'\\''")
    const installCmd = `sudo -n "${helperScript}" "${escapedPath}"`

    try {
      console.log('Linux update: 执行安装命令:', installCmd)
      const result = execSync(installCmd, { timeout: 180000 })
      console.log('Linux update: 安装成功:', result.toString())

      if (mainWindow) {
        mainWindow.webContents.send('update-installed')
      }

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
      if (mainWindow) {
        mainWindow.webContents.send('update-error', errorMsg)
      }
    }
  }

  if (app.isPackaged) {
    autoUpdater.checkForUpdates().catch(error => {
      console.error('检查更新失败:', error)
    })
  }
}

function setupAutoUpdateUrl() {
  const config = getAutoUpdaterConfig()
  if (config) {
    console.log(`设置更新服务器地址: ${config.feedUrl}`)
    autoUpdater.setFeedURL({ provider: 'generic', url: config.feedUrl })
  }
}

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
  event.sender.send('server-url', getUpdateBaseUrl())
})

ipcMain.on('check-for-updates', () => {
  console.log('收到检查更新请求')
  setupAutoUpdateUrl()
  autoUpdater.checkForUpdates().catch(error => {
    console.error('检查更新失败:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })
})

ipcMain.on('download-update', () => {
  console.log('收到下载更新请求')
  setupAutoUpdateUrl()
  autoUpdater.checkForUpdates().catch(error => {
    console.error('检查更新失败:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })
})

  // 屏幕共享相关
  ipcMain.on('start-screen-share', async () => {
    try {
      console.log('启动屏幕共享')
      // 获取可用的屏幕和窗口，指定较大的缩略图尺寸
      const sources = await desktopCapturer.getSources({
        types: ['screen', 'window'],
        thumbnailSize: {
          width: 640,
          height: 360
        }
      })
      
      // 转换缩略图为base64
      const sourcesWithThumbnails = sources.map(source => ({
        id: source.id,
        name: source.name,
        thumbnail: source.thumbnail.toDataURL()
      }))
      
      // 发送屏幕源信息到渲染进程
      if (mainWindow) {
        mainWindow.webContents.send('screen-sources', sourcesWithThumbnails)
      }
    } catch (error) {
      console.error('获取屏幕源失败:', error)
    }
  })

  // 处理WebSocket消息发送
  ipcMain.on('send-websocket-message', (event, message) => {
    console.log('发送WebSocket消息:', message.type)
    // 这里需要实现WebSocket消息发送逻辑
    // 实际项目中，应该通过WebSocket连接发送消息
    // 这里只是模拟发送
    setTimeout(() => {
      // 模拟接收方收到消息
      if (mainWindow) {
        mainWindow.webContents.send('websocket-message', {
          type: message.type,
          data: {
            ...message.data,
            from_user_id: 1 // 模拟发送者ID
          }
        })
      }
    }, 100)
  })

  // 处理WebRTC相关消息
  ipcMain.on('webrtc-message', (event, message) => {
    console.log('处理WebRTC消息:', message.type)
    // 这里需要实现WebRTC消息处理逻辑
    if (mainWindow) {
      mainWindow.webContents.send('webrtc-message', message)
    }
  })

  // 处理头像缓存请求
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

  // 处理文件下载（下载到指定目录）
  ipcMain.on('download-file', async (event, { buffer, fileName, mime, saveDir }) => {
    try {
      const targetDir = saveDir || app.getPath('downloads')
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

  // 处理文件另存为（弹出文件选择器）
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

  // 处理选择目录
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

app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') app.quit()

  globalShortcut.unregisterAll()
})

// 获取缓存目录
function getCacheDir() {
  const appDataPath = app.getPath('userData')
  const cacheDir = path.join(appDataPath, 'avatar-cache')
  
  // 确保缓存目录存在
  if (!fs.existsSync(cacheDir)) {
    fs.mkdirSync(cacheDir, { recursive: true })
  }
  
  return cacheDir
}

// 生成头像缓存文件名
function generateCacheFileName(avatarUrl) {
  // 使用URL的哈希作为文件名
  const hash = crypto.createHash('md5').update(avatarUrl).digest('hex')
  // 根据URL确定文件类型，处理特殊情况
  let ext = 'png'
  
  // 检查URL是否包含文件扩展名
  const extMatch = avatarUrl.match(/\.([^.]+)(?:\?|$)/)
  if (extMatch && extMatch[1]) {
    // 获取扩展名并移除可能的查询参数
    ext = extMatch[1].split('?')[0].split('/')[0]
    // 限制扩展名长度，防止恶意URL
    if (ext.length > 10) {
      ext = 'png'
    }
  }
  
  return `${hash}.${ext}`
}

// 缓存头像
async function cacheAvatar(avatarUrl) {
  try {
    const cacheDir = getCacheDir()
    const cacheFileName = generateCacheFileName(avatarUrl)
    const cacheFilePath = path.join(cacheDir, cacheFileName)
    
    // 检查是否已缓存
    if (fs.existsSync(cacheFilePath)) {
      return `file://${cacheFilePath}`
    }
    
    // 下载并缓存头像
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

// 清理过期的头像缓存
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

app.whenReady().then(() => {
  setupAutoUpdater()
  
  // 启动时清理一次
  cleanupAvatarCache()
  
  // 每天清理一次
  setInterval(() => {
    cleanupAvatarCache()
  }, 24 * 60 * 60 * 1000)
})
