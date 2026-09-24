import { describe, expect, it, beforeEach, afterEach, beforeAll, vi } from 'vitest'
import vm from 'node:vm'
import { readFileSync, existsSync } from 'node:fs'
import { resolve } from 'node:path'
import { execSync } from 'node:child_process'
import process from 'node:process'
import os from 'node:os'

/**
 * 托盘点击真实行为测试：vm 沙箱加载实际产物 electron/main.bundle.cjs
 * （不是字符串检查），注入 electron mock，驱动真实的 tray click / flash-tray IPC，
 * 断言各窗口状态组合下的调用。核心回归：闪烁态复位不依赖 window 'focus' 事件。
 */
const root = resolve(__dirname, '../..')
const bundlePath = resolve(root, 'electron/main.bundle.cjs')

beforeAll(() => {
  // bundle 是 gitignore 的构建产物，干净环境先构建一次
  if (!existsSync(bundlePath)) execSync('node scripts/build-main.cjs', { cwd: root })
})

type Listener = (...args: any[]) => void

function makeWindow(isMain: boolean) {
  const events: Record<string, Listener> = {}
  return {
    loadFile: vi.fn(),
    loadURL: vi.fn(),
    show: vi.fn(),
    hide: vi.fn(),
    restore: vi.fn(),
    focus: vi.fn(),
    close: vi.fn(),
    minimize: vi.fn(),
    maximize: vi.fn(),
    unmaximize: vi.fn(),
    isVisible: vi.fn(() => true),
    isMinimized: vi.fn(() => false),
    isFocused: vi.fn(() => true),
    isMaximized: vi.fn(() => false),
    isDestroyed: vi.fn(() => false),
    flashFrame: vi.fn(),
    setContentProtection: vi.fn(),
    setMenu: vi.fn(),
    setBounds: vi.fn(),
    webContents: { on: vi.fn(), send: vi.fn(), openDevTools: vi.fn() },
    on(event: string, cb: Listener) { events[event] = cb },
    once(event: string, cb: Listener) { events[event] = cb },
    emit(event: string, ...args: any[]) { events[event]?.(...args) },
    // splash 初始在屏；主窗 close 时会顺手 splash.close()
    _splash: !isMain,
  }
}

function loadHarness(platform: NodeJS.Platform) {
  const splash = makeWindow(false)
  const win = makeWindow(true)
  // BrowserWindow 构造队列：createWindow 依次取 splash、main
  let queue: any[] = [splash, win]
  const BrowserWindow = vi.fn(function (this: unknown) {
    return queue.shift()
  })

  const trayEvents: Record<string, Listener> = {}
  const tray = {
    on(event: string, cb: Listener) { trayEvents[event] = cb },
    emit(event: string) { trayEvents[event]?.() },
    setImage: vi.fn(),
    setToolTip: vi.fn(),
    setContextMenu: vi.fn(),
  }
  const Tray = vi.fn(function (this: unknown) {
    return tray
  })

  const ipcHandlers: Record<string, Listener> = {}
  const ipcMain = { on: vi.fn((ch: string, cb: Listener) => { ipcHandlers[ch] = cb }), handle: vi.fn() }

  let resolveReady: () => void
  const userData = resolve(os.tmpdir(), `qim-tray-test-${Date.now()}-${Math.random().toString(36).slice(2)}`)
  const app = {
    isPackaged: false,
    getAppPath: () => root,
    setAsDefaultProtocolClient: vi.fn(),
    disableHardwareAcceleration: vi.fn(),
    requestSingleInstanceLock: vi.fn(() => true),
    whenReady: () => new Promise<void>((res) => { resolveReady = res }),
    on: vi.fn(),
    quit: vi.fn(),
    exit: vi.fn(),
    getPath: (kind: string) => kind === 'userData' ? userData : os.tmpdir(),
    dock: platform === 'darwin'
      ? { bounce: vi.fn(), setBadge: vi.fn(), setIcon: vi.fn() }
      : undefined,
  }

  const electronMock = {
    app: new Proxy(app, {
      // 未列出的 app 方法兜底 no-op；dock 已显式声明（linux 为 undefined）不受影响
      get(target, prop, receiver) {
        if (Reflect.has(target, prop)) return Reflect.get(target, prop, receiver)
        return () => {}
      },
    }),
    BrowserWindow,
    Tray,
    Menu: { buildFromTemplate: vi.fn(() => ({})) },
    nativeImage: {
      createFromBuffer: vi.fn(() => ({})),
      createFromPath: vi.fn(() => ({})),
      createFromDataURL: vi.fn(() => ({})),
    },
    ipcMain,
    globalShortcut: { register: vi.fn(() => true), unregister: vi.fn(), unregisterAll: vi.fn() },
    desktopCapturer: { getSources: vi.fn() },
    dialog: { showErrorBox: vi.fn(), showSaveDialog: vi.fn(), showOpenDialog: vi.fn(), showMessageBox: vi.fn() },
    screen: { getCursorScreenPoint: vi.fn(), getDisplayNearestPoint: vi.fn(() => ({})) },
    systemPreferences: new Proxy({
      getMediaAccessStatus: vi.fn(() => 'granted'),
      askForMediaAccess: vi.fn(() => Promise.resolve(true)),
      subscribeNotification: vi.fn(),
    }, {
      get(target, prop, receiver) {
        if (Reflect.has(target, prop)) return Reflect.get(target, prop, receiver)
        return () => {}
      },
    }),
    session: { defaultSession: { setPermissionRequestHandler: vi.fn() } },
    shell: { openPath: vi.fn(), showItemInFolder: vi.fn(), openExternal: vi.fn() },
    Notification: vi.fn(function (this: unknown) {
      return { on: vi.fn(), show: vi.fn() }
    }),
    clipboard: { readText: vi.fn() },
    net: { request: vi.fn() },
  }

  const screenshotsMock = {
    default: vi.fn(function (this: unknown) {
      return { on: vi.fn(), startCapture: vi.fn(), getOverlayDiagnostics: vi.fn(() => ({})) }
    }),
  }
  const updaterMock = {
    autoUpdater: new Proxy({
      on: vi.fn(),
      removeAllListeners: vi.fn(),
      checkForUpdates: vi.fn(() => Promise.resolve({})),
      quitAndInstall: vi.fn(),
      downloadUpdate: vi.fn(() => Promise.resolve()),
    }, {
      get(target, prop, receiver) {
        if (Reflect.has(target, prop)) return Reflect.get(target, prop, receiver)
        return () => {}
      },
    }),
  }

  const builtinRequire = require
  const customRequire = (id: string) => {
    if (id === 'electron') return electronMock
    if (id === 'electron-updater') return updaterMock
    if (id.startsWith('.') && id.includes('screenshots')) return screenshotsMock
    if (id === 'node:module') {
      // 劫持 createRequire：bundle 用它在运行时加载 screenshots cjs，
      // 改指 customRequire，避免真实加载原生 node-screenshots
      return { ...builtinRequire('node:module'), createRequire: () => customRequire }
    }
    return builtinRequire(id)
  }

  const proc = new Proxy(process, {
    get(target, prop, receiver) {
      if (prop === 'platform') return platform
      return Reflect.get(target, prop, receiver)
    },
  })

  const sandbox = {
    // banner 用裸 __filename 拼 import.meta.url 锚（CJS 模块包装器在正常运行时提供，
    // VM sandbox 须显式注入）
    __filename: bundlePath,
    __dirname: resolve(bundlePath, '..'),
    require: customRequire,
    module: { exports: {} },
    exports: {},
    process: proc,
    console,
    Buffer,
    URL,
    setTimeout,
    clearTimeout,
    setInterval,
    clearInterval,
    queueMicrotask,
  }
  vm.createContext(sandbox)
  const listenersBefore = process.listeners('uncaughtException')
  vm.runInContext(readFileSync(bundlePath, 'utf8'), sandbox, { filename: bundlePath })
  // bundle 顶层 process.on('uncaughtException') 注册在真实 process 上，
  // 测试结束必须摘掉，否则每加载一次泄漏一个引用整个 sandbox 的监听器
  const addedUncaught = process.listeners('uncaughtException')
    .filter((l) => !listenersBefore.includes(l))
  const cleanup = () => {
    for (const l of addedUncaught) process.removeListener('uncaughtException', l)
  }

  const start = async () => {
    resolveReady()
    // flush ready 回调里的 awaits（ensureMediaPermissions 等）
    for (let i = 0; i < 10; i++) await Promise.resolve()
  }

  return {
    start,
    cleanup,
    win,
    splash,
    tray,
    dock: app.dock,
    click: () => tray.emit('click'),
    flash: () => ipcHandlers['flash-tray'](),
    stopFlash: () => ipcHandlers['stop-tray-flash'](),
    /** 再建窗口（mainWindow 已 null 的场景）：队列补 splash+main */
    queueNextWindow: () => { queue = [makeWindow(false), makeWindow(true)] },
    get queued() { return queue },
  }
}

// 每个用例独立加载，避免状态（hasUnread/mainWindow）串场
let harness: ReturnType<typeof loadHarness>

afterEach(() => {
  // 兜底清掉闪烁 interval，避免 active handle 悬挂
  try { harness?.stopFlash() } catch { /* harness 已废弃 */ }
  harness?.cleanup()
})

describe.each([
  ['darwin'],
  ['linux'],
] as const)('tray click on %s', (platform) => {
  beforeEach(async () => {
    harness = loadHarness(platform)
    await harness.start()
    harness.win.emit('ready-to-show')
    vi.clearAllMocks()
  })

  it('正常可见且无未读：toggle 隐藏', () => {
    harness.win.isVisible.mockReturnValue(true)
    harness.win.isMinimized.mockReturnValue(false)

    harness.click()

    expect(harness.win.hide).toHaveBeenCalledTimes(1)
    expect(harness.win.show).not.toHaveBeenCalled()
    expect(harness.win.focus).not.toHaveBeenCalled()
    expect(harness.win.restore).not.toHaveBeenCalled()
  })

  it('闪烁中窗口可见（焦点已在本窗）：唤起并直接停闪，不依赖 focus 事件', () => {
    harness.win.isVisible.mockReturnValue(true)
    harness.flash()
    expect(harness.win.flashFrame).toHaveBeenLastCalledWith(true)

    // 全程不 emit 'focus'——模拟 WM 拒绝聚焦/窗口已聚焦时无 focus 事件
    harness.click()

    expect(harness.win.hide).not.toHaveBeenCalled()
    expect(harness.win.show).toHaveBeenCalledTimes(1)
    expect(harness.win.focus).toHaveBeenCalledTimes(1)
    expect(harness.win.flashFrame).toHaveBeenLastCalledWith(false)

    // hasUnread 已复位：再点一次恢复 toggle 隐藏（hasUnread 锁死回归断言）
    harness.click()
    expect(harness.win.hide).toHaveBeenCalledTimes(1)
  })

  it('最小化且仍可见（Windows 报告态）：restore + show + focus', () => {
    harness.win.isVisible.mockReturnValue(true)
    harness.win.isMinimized.mockReturnValue(true)

    harness.click()

    expect(harness.win.restore).toHaveBeenCalledTimes(1)
    expect(harness.win.show).toHaveBeenCalledTimes(1)
    expect(harness.win.focus).toHaveBeenCalledTimes(1)
    expect(harness.win.hide).not.toHaveBeenCalled()
  })

  it('隐藏到托盘：show + focus，不调 restore', () => {
    harness.win.isVisible.mockReturnValue(false)
    harness.win.isMinimized.mockReturnValue(false)

    harness.click()

    expect(harness.win.show).toHaveBeenCalledTimes(1)
    expect(harness.win.focus).toHaveBeenCalledTimes(1)
    expect(harness.win.restore).not.toHaveBeenCalled()
    expect(harness.win.hide).not.toHaveBeenCalled()
  })

  it('最小化+不可见（部分 Linux WM）且闪烁：restore + show + focus + 停闪', () => {
    harness.win.isVisible.mockReturnValue(false)
    harness.win.isMinimized.mockReturnValue(true)
    harness.flash()

    harness.click()

    expect(harness.win.restore).toHaveBeenCalledTimes(1)
    expect(harness.win.show).toHaveBeenCalledTimes(1)
    expect(harness.win.focus).toHaveBeenCalledTimes(1)
    expect(harness.win.flashFrame).toHaveBeenLastCalledWith(false)
    expect(harness.win.hide).not.toHaveBeenCalled()
  })

  it('mainWindow 已销毁时点击：重新创建窗口', () => {
    harness.win.emit('close') // close handler 把 mainWindow 置 null
    harness.queueNextWindow()
    const [, nextWin] = harness.queued

    harness.click()

    expect(nextWin.loadURL).toHaveBeenCalledTimes(1)
    expect(harness.win.loadURL).not.toHaveBeenCalled()
  })

  it('闪烁时托盘图标与 tooltip 恢复正常', () => {
    harness.flash()
    expect(harness.tray.setToolTip).toHaveBeenLastCalledWith(expect.stringContaining('有新消息'))

    harness.click()

    expect(harness.tray.setToolTip).toHaveBeenLastCalledWith('QIM 应用')
  })
})

describe('tray flash 平台差异', () => {
  it('darwin：闪烁走 dock bounce/badge，停闪清 badge', async () => {
    harness = loadHarness('darwin')
    await harness.start()
    harness.win.emit('ready-to-show')

    harness.flash()
    expect(harness.dock.bounce).toHaveBeenCalledWith('informational')
    expect(harness.dock.setBadge).toHaveBeenCalledWith('!')

    harness.click()
    expect(harness.dock.setBadge).toHaveBeenLastCalledWith('')
    expect(harness.tray.setToolTip).toHaveBeenLastCalledWith('QIM 应用')
  })

  it('停闪后闪烁 interval 被清除：图标不再交替（darwin）', async () => {
    harness = loadHarness('darwin')
    await harness.start()
    harness.win.emit('ready-to-show')

    harness.flash()
    await new Promise((r) => setTimeout(r, 1100)) // 500ms × 2 次交替
    expect(harness.tray.setImage.mock.calls.length).toBeGreaterThanOrEqual(2)

    harness.click()
    const callsAtStop = harness.tray.setImage.mock.calls.length
    await new Promise((r) => setTimeout(r, 1100))
    expect(harness.tray.setImage.mock.calls.length).toBe(callsAtStop)
  }, 12000)

  it('linux：app.dock 不存在时闪烁/停闪不报错', async () => {
    harness = loadHarness('linux')
    await harness.start()
    harness.win.emit('ready-to-show')

    expect(harness.dock).toBeUndefined()
    expect(() => {
      harness.flash()
      harness.click()
    }).not.toThrow()
    expect(harness.tray.setToolTip).toHaveBeenLastCalledWith('QIM 应用')
  })
})
