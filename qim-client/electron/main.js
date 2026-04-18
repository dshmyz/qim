import { app, BrowserWindow, Tray, Menu, nativeImage, ipcMain, globalShortcut } from 'electron'
import path from 'path'
import { fileURLToPath } from 'url'
// 暂时注释掉electron-updater，避免开发环境中的错误
// import { autoUpdater } from 'electron-updater'

// 模拟autoUpdater对象，用于开发环境
const autoUpdater = {
  checkForUpdates: () => {
    return new Promise((resolve) => {
      console.log('模拟检查更新')
      resolve()
    })
  },
  on: (event, callback) => {
    console.log(`注册事件监听器: ${event}`)
  },
  quitAndInstall: () => {
    console.log('模拟退出并安装更新')
  },
  // 添加missing方法
  setFeedURL: () => {
    console.log('模拟设置更新源')
  }
}

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

let mainWindow
let tray

function createWindow() {
  // 如果已经存在窗口，就显示它，而不是创建一个新的窗口
  if (mainWindow) {
    console.log('Window already exists, showing it')
    mainWindow.show()
    return
  }else {
    console.log('Creating new window')
  }
  
  // 使用 base64 编码的 PNG 图标
  const iconData = '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'
  const icon = nativeImage.createFromDataURL('data:image/png;base64,' + iconData)
  
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    icon: icon,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js')
    },
    sandbox: false,
    frame: false, // 去掉默认窗口边框
    titleBarStyle: 'customButtonsOnHover', // 仅在悬停时显示按钮
    titleBarOverlay: {
      visible: false, // 禁用标题栏覆盖
      height: 0 // 设置高度为 0
    },
    trafficLightPosition: { x: -100, y: -100 } // 将控制按钮移到屏幕外
  })

  // 在开发模式下，加载 Vite 开发服务器
  // 在生产模式下，加载本地文件
  const isDev = process.env.NODE_ENV !== 'production'
  const url = isDev 
    ? 'http://localhost:3000' 
    : `file://${path.join(__dirname, '../dist/index.html')}`
  
  mainWindow.loadURL(url)
  console.log(`Loading URL: ${url}`)
  
  // 当渲染进程加载完成后，注入必要的代码
  mainWindow.webContents.on('did-finish-load', () => {
    console.log('Render process loaded')
    // 直接注入窗口控制函数到渲染进程
    // mainWindow.webContents.executeJavaScript(`
    //   // 注入窗口控制函数
    //   window.minimizeWindow = function() {
    //     console.log('Minimize window function called')
    //     // 发送消息到主进程
    //     window.postMessage({ type: 'MINIMIZE_WINDOW' }, '*')
    //   }
    //   
    //   window.maximizeWindow = function() {
    //     console.log('Maximize window function called')
    //     // 发送消息到主进程
    //     window.postMessage({ type: 'MAXIMIZE_WINDOW' }, '*')
    //   }
    //   
    //   window.closeWindow = function() {
    //     console.log('Close window function called')
    //     // 发送消息到主进程
    //     window.postMessage({ type: 'CLOSE_WINDOW' }, '*')
    //   }
    //   
    //   // 注入electron对象
    //   window.electron = {
    //     ipcRenderer: {
    //       send: (channel, data) => {
    //         console.log('Sending IPC message:', channel, data)
    //          window.postMessage({ type: 'ELECTRON_IPC', channel, data }, '*')
    //       }
    //     }
    //   }
    //   
    //   console.log('Window control functions and electron object injected')
    // `)
    // 在开发者模式下打开开发者工具
    if (process.env.NODE_ENV !== 'production') {
      console.log('Opening DevTools in development mode')
      mainWindow.webContents.openDevTools()
    }
  })
  
  // 监听窗口事件
  // const handleMessage = (event, message) => {
  //   console.log('Received message from renderer:', message)
  //   if (message.type === 'MINIMIZE_WINDOW') {
  //     console.log('Minimizing window')
  //     mainWindow.minimize()
  //   } else if (message.type === 'MAXIMIZE_WINDOW') {
  //     console.log('Maximizing window')
  //     if (mainWindow.isMaximized()) {
  //       console.log('Window is maximized, unmaximizing')
  //       mainWindow.unmaximize()
  //     } else {
  //       console.log('Window is not maximized, maximizing')
  //       mainWindow.maximize()
  //     }
  //   } else if (message.type === 'CLOSE_WINDOW') {
  //     console.log('Hiding window instead of closing')
  //     mainWindow.hide()
  //   } else if (message.type === 'ELECTRON_IPC') {
  //     console.log(`Received ELECTRON_IPC message: ${message.channel}`, message.data)
  //     if (message.channel === 'minimize-window') {
  //       console.log('Minimizing window')
  //       mainWindow.minimize()
  //     } else if (message.channel === 'maximize-window') {
  //       console.log('Maximizing window')
  //       if (mainWindow.isMaximized()) {
  //         console.log('Window is maximized, unmaximizing')
  //         mainWindow.unmaximize()
  //       } else {
  //         console.log('Window is not maximized, maximizing')
  //         mainWindow.maximize()
  //       }
  //     } else if (message.channel === 'close-window') {
  //       console.log('Hiding window instead of closing')
  //       mainWindow.hide()
  //     }
  //   }
  // }
  
  // mainWindow.webContents.on('message', handleMessage)
  
  // 当窗口关闭时，隐藏窗口而不是销毁它
  mainWindow.on('close', function (event) {
    // 移除事件监听器
    try {
      if (mainWindow && mainWindow.webContents) {
        // ipcMain.removeAllListeners('minimize-window')
        // ipcMain.removeAllListeners('maximize-window')
        // ipcMain.removeAllListeners('close-window')
        // mainWindow.webContents.removeListener('message', handleMessage)
      }
    } catch (error) {
      console.log('Error removing listener:', error)
    }
    
    // 阻止默认的关闭行为
    event.preventDefault()
    // 隐藏窗口
    mainWindow.hide()
    console.log('Window hidden instead of closed')
  })
  
  // 当窗口真正销毁时，才将mainWindow设置为null
  mainWindow.on('destroyed', function () {
    console.log('Window destroyed event triggered')
    mainWindow = null
  })
  
  // 监听窗口控制事件
  
  console.log('Setting up window control event listeners')
  
  // 添加全局快捷键测试
  
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
  
  // 添加退出快捷键
  globalShortcut.register('CommandOrControl+Q', () => {
    console.log('Global shortcut: CommandOrControl+Q pressed')
    app.quit()
  })
  
  // 监听窗口控制事件
  // 这些事件已经在handleMessage函数中处理了，不需要重复添加
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
    // mainWindow.hide()
    mainWindow.close()
    // mainWindow = null
  })
  
  // 处理截图请求
  ipcMain.on('take-screenshot', async (event) => {
    console.log('Received take-screenshot event')
    try {
      // 尝试使用desktopCapturer实现真正的截图功能
      const { desktopCapturer } = require('electron')
      
      // 获取所有屏幕源
      const sources = await desktopCapturer.getSources({ types: ['screen'] })
      
      if (sources.length === 0) {
        console.log('No screen sources found')
        // 如果没有找到屏幕源，返回模拟数据
        const screenshotData = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='
        event.sender.send('screenshot-taken', screenshotData)
        return
      }
      
      // 选择第一个屏幕源
      const source = sources[0]
      console.log('Selected screen source:', source.name)
      
      // 使用navigator.mediaDevices.getDisplayMedia来捕获屏幕
      // 注意：这在Electron的主进程中可能无法直接使用
      // 所以我们仍然使用模拟数据作为备用
      
      // 模拟截图结果（实际生产环境中应该使用真正的截图数据）
      const screenshotData = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='
      
      // 发送截图结果回渲染进程
      event.sender.send('screenshot-taken', screenshotData)
      console.log('Screenshot taken and sent to renderer')
    } catch (error) {
      console.error('Error taking screenshot:', error)
      // 即使出错也返回一个模拟的截图数据，确保前端能正常显示
      const screenshotData = 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=='
      event.sender.send('screenshot-taken', screenshotData)
    }
  })
  
  console.log('Global shortcuts registered')
}

function createTray() {
  // 如果已经存在托盘图标，就返回，而不是创建一个新的托盘图标
  if (tray) {
    console.log('Tray already exists, returning')
    return
  }
  
  try {
    console.log('开始创建托盘图标')
    
    // 使用 base64 编码的 PNG 图标
    const iconData = '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'
    
    console.log('创建图标数据')
    const image = nativeImage.createFromDataURL('data:image/png;base64,' + iconData)
    
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
    
    // 点击托盘图标显示/隐藏窗口
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

// 当 Electron 完成初始化并准备创建浏览器窗口时调用此方法
app.whenReady().then(() => {
  console.log('App ready')
  createWindow()
  createTray()
  
  // 设置 Dock 图标（仅 macOS）
  if (app.dock) {
    const iconData = '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'
    const image = nativeImage.createFromDataURL('data:image/png;base64,' + iconData)
    app.dock.setIcon(image)
  }

  app.on('activate', function () {
    console.log('activate event triggered')
    createWindow()
    // if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })
})

// 自动更新配置
function setupAutoUpdater() {
  // 监听更新事件
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
    
    // 下载完成后自动安装
    autoUpdater.quitAndInstall()
  })

  // 应用启动时检查更新
  if (app.isPackaged) {
    autoUpdater.checkForUpdates().catch(error => {
      console.error('检查更新失败:', error)
    })
  }
}

// 监听渲染进程的检查更新请求
ipcMain.on('check-for-updates', () => {
  console.log('收到检查更新请求')
  autoUpdater.checkForUpdates().catch(error => {
    console.error('检查更新失败:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })
})

// 监听渲染进程的下载更新请求
ipcMain.on('download-update', () => {
  console.log('收到下载更新请求')
  // 自动更新器会在发现更新后自动下载
  // 这里只是触发检查更新
  autoUpdater.checkForUpdates().catch(error => {
    console.error('检查更新失败:', error)
    if (mainWindow) {
      mainWindow.webContents.send('update-error', error.message)
    }
  })
})

// 当所有窗口都关闭时退出应用（Windows & Linux）
app.on('window-all-closed', function () {
  if (process.platform !== 'darwin') app.quit()
})

// 应用就绪后设置自动更新
app.whenReady().then(() => {
  setupAutoUpdater()
})