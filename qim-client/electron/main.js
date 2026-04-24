import { app, BrowserWindow, Tray, Menu, nativeImage, ipcMain, globalShortcut, desktopCapturer, screen } from 'electron'
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

function getIconPath(size = 512) {
  const iconDir = path.join(__dirname, 'icons')
  const iconPath = path.join(iconDir, `icon-v2_${size}x${size}.png`)
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

const autoUpdaterConfig = {
  win7: {
    version: '22.3.27',
    feedUrl: 'https://update.example.com/win7'
  },
  win10: {
    version: '41.2.0',
    feedUrl: 'https://update.example.com/win10'
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
  const platform = process.platform
  if (platform === 'win32') {
    const winVersion = getWindowsVersion()
    if (winVersion && winVersion < 10) {
      return autoUpdaterConfig.win7
    }
    return autoUpdaterConfig.win10
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

  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    icon: icon,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js')
    },
    sandbox: false,
    frame: false,
    titleBarStyle: 'customButtonsOnHover',
    titleBarOverlay: {
      visible: false,
      height: 0
    },
    trafficLightPosition: { x: -100, y: -100 }
  })

  const isDev = process.env.NODE_ENV !== 'production'
  const url = isDev
    ? 'http://localhost:3000'
    : `file://${path.join(__dirname, '../dist/index.html')}`

  mainWindow.loadURL(url)
  console.log(`Loading URL: ${url}`)

  mainWindow.webContents.on('did-finish-load', () => {
    console.log('Render process loaded')
    if (process.env.NODE_ENV !== 'production') {
      console.log('Opening DevTools in development mode')
      mainWindow.webContents.openDevTools()
    }
  })

  mainWindow.on('close', function () {
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

  ipcMain.on('take-screenshot', async () => {
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
    normalTrayIcon = tray.getImage()
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
        // 可以创建一个带有徽章的图标，但为了简单，这里我们切换到一个视觉上有区别的状态
        tray.setImage(normalTrayIcon)
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

  app.on('activate', function () {
    if (!mainWindow) {
      console.log('activate event triggered')
      createWindow()
    }
  })


function setupAutoUpdater() {
  const config = getAutoUpdaterConfig()
  if (config) {
    console.log(`配置自动更新: Windows 版本对应的更新服务器: ${config.feedUrl}`)
    autoUpdater.setFeedURL({ provider: 'generic', url: config.feedUrl })
  }

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

    autoUpdater.quitAndInstall()
  })

  if (app.isPackaged) {
    autoUpdater.checkForUpdates().catch(error => {
      console.error('检查更新失败:', error)
    })
  }
}

ipcMain.on('check-for-updates', () => {
  console.log('收到检查更新请求')
  autoUpdater.checkForUpdates().catch(error => {
    console.error('检查更新失败:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })
})

ipcMain.on('download-update', () => {
    console.log('收到下载更新请求')
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
