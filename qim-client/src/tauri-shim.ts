import { invoke } from '@tauri-apps/api/core'
import { listen, emit } from '@tauri-apps/api/event'
import { getCurrentWindow } from '@tauri-apps/api/window'
import { openUrl } from '@tauri-apps/plugin-opener'

type Unlisten = () => void
const listeners = new Map<string, Unlisten[]>()

function normalizeArgs(payload: any) {
  if (Array.isArray(payload)) return payload
  if (payload === undefined || payload === null) return []
  return [payload]
}

function setupGlobalHotkeys(runtime: any, goApp: any) {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
  const cmdKey = isMac ? 'metaKey' : 'ctrlKey'

  document.addEventListener('keydown', (e: KeyboardEvent) => {
    if (e[cmdKey]) {
      switch (e.key.toLowerCase()) {
        case 'm':
          e.preventDefault()
          runtime.WindowMinimise()
          break
        case 'k':
          e.preventDefault()
          console.log('[hotkey] Cmd+K: toggle devtools')
          break
        case 'w':
          e.preventDefault()
          runtime.WindowHide()
          break
        case 'q':
          e.preventDefault()
          runtime.Quit()
          break
      }
    }
  })
}

async function initTauriShim() {
  const appWindow = getCurrentWindow()

  const runtimeShim = {
    WindowMinimise: () => appWindow.minimize(),
    WindowMaximise: () => appWindow.maximize(),
    WindowUnmaximise: () => appWindow.unmaximize(),
    WindowIsMaximised: () => appWindow.isMaximized(),
    WindowHide: () => appWindow.hide(),
    WindowShow: () => appWindow.show(),
    WindowSetAlwaysOnTop: (enabled: boolean) => appWindow.setAlwaysOnTop(enabled),
    BrowserOpenURL: (url: string) => openUrl(url),
    Quit: () => invoke('quit_app'),
    EventsEmit: async (channel: string, data?: any) => {
      await emit(channel, data)
    },
    EventsOn: (channel: string, callback: (...args: any[]) => void) => {
      listen(channel, (event) => {
        callback(...normalizeArgs(event.payload))
      }).then((unlisten) => {
        const current = listeners.get(channel) ?? []
        current.push(unlisten)
        listeners.set(channel, current)
      })
    },
    EventsOff: (channel: string) => {
      const list = listeners.get(channel) ?? []
      list.forEach((fn) => fn())
      listeners.delete(channel)
    },
  }

  const goApp = {
    MinimizeWindow: () => invoke('minimize_window'),
    MaximizeWindow: () => invoke('maximize_window'),
    CloseWindow: () => invoke('close_window'),
    IsMaximized: () => invoke('is_maximized'),
    OpenExternal: (url: string) => invoke('open_external', { url }),
    OpenFileDialog: (opts: any) => invoke('open_file_dialog', { opts: typeof opts === 'string' ? JSON.parse(opts) : opts }),
    SaveFileAs: (fileName: string, data: Uint8Array) => invoke('save_file_as', { fileName, data: Array.from(data) }),
    DownloadFile: (fileName: string, data: Uint8Array, saveDir: string) => invoke('download_file', { fileName, data: Array.from(data), saveDir }),
    GetAppInfo: () => invoke('get_app_info'),
    CacheAvatar: (url: string) => invoke('cache_avatar', { avatarUrl: url }),
    CleanupAvatarCache: (maxAgeDays: number) => invoke('cleanup_avatar_cache', { maxAgeDays }),
    FlashTray: (enabled: boolean) => invoke('flash_tray', { enabled }),
    CheckForUpdates: () => invoke('check_for_updates'),
    DownloadUpdate: () => invoke('download_update'),
    GetScreenSources: () => invoke('get_screen_sources'),
    StartScreenshot: (hideWindow?: boolean) => invoke('start_screenshot', { hideWindow: hideWindow ?? true }),
    CancelScreenshot: () => invoke('cancel_screenshot'),
    CompleteScreenshot: (dataUrl: string, boundsJson: string) => invoke('complete_screenshot', { dataUrl, boundsJson }),
    SaveScreenshot: (data: Uint8Array, boundsJson: string) => invoke('save_screenshot', { data: Array.from(data), boundsJson }),
    GetPlatform: () => invoke('get_platform'),
  }

  setupGlobalHotkeys(runtimeShim, goApp)

  const electronShim = {
    ipcRenderer: {
      send: (channel: string, data?: any) => {
        switch (channel) {
          case 'minimize-window':
            runtimeShim.WindowMinimise()
            break
          case 'maximize-window':
            runtimeShim.WindowIsMaximised().then((max: boolean) => {
              if (max) runtimeShim.WindowUnmaximise()
              else runtimeShim.WindowMaximise()
            })
            break
          case 'close-window':
            runtimeShim.WindowHide()
            break
          case 'flash-tray':
            runtimeShim.EventsEmit('tray-flash', true)
            break
          case 'stop-tray-flash':
            runtimeShim.EventsEmit('tray-flash', false)
            break
          case 'take-screenshot':
            goApp.StartScreenshot(data?.hideWindow ?? true)
            break
          case 'start-screen-share':
            runtimeShim.EventsEmit('screen-share-requested', true)
            break
          case 'download-file':
            if (data) {
              const { buffer, fileName, saveDir } = data
              goApp.DownloadFile(fileName, new Uint8Array(buffer), saveDir || '')
                .then((result: any) => runtimeShim.EventsEmit('download-complete', { success: true, filePath: result.filePath }))
                .catch((err: any) => runtimeShim.EventsEmit('download-complete', { success: false, error: err.message }))
            }
            break
          case 'save-file-as':
            if (data) {
              const { buffer, fileName } = data
              goApp.SaveFileAs(fileName, new Uint8Array(buffer))
                .then((result: any) => runtimeShim.EventsEmit('save-file-complete', { success: true, filePath: result.filePath }))
                .catch((err: any) => runtimeShim.EventsEmit('save-file-complete', { success: false, error: err.message }))
            }
            break
          case 'open-file-dialog':
            goApp.OpenFileDialog(data || {}).then((result: any) => runtimeShim.EventsEmit('file-dialog-result', result))
            break
          case 'cache-avatar':
            if (data) {
              goApp.CacheAvatar(data)
                .then((cachedUrl: string) => runtimeShim.EventsEmit('avatar-cached', cachedUrl))
                .catch(() => runtimeShim.EventsEmit('avatar-cached', data))
            }
            break
          case 'check-for-updates':
            goApp.CheckForUpdates().then((info: any) => {
              if (info?.available) runtimeShim.EventsEmit('update-available', info)
              else runtimeShim.EventsEmit('update-not-available')
            })
            break
          case 'download-update':
            goApp.DownloadUpdate()
            break
        }
      },
      on: (channel: string, callback: (...args: any[]) => void) => {
        runtimeShim.EventsOn(channel, (data: any) => callback(null, data))
      },
      once: (channel: string, callback: (...args: any[]) => void) => {
        listen(channel, (event) => {
          callback(null, event.payload)
        }).then((unlisten) => {
          unlisten()
        })
      },
      removeListener: (channel: string) => runtimeShim.EventsOff(channel),
      removeAllListeners: (channel: string) => runtimeShim.EventsOff(channel),
      invoke: async () => null,
    },
    shell: {
      openExternal: (url: string) => runtimeShim.BrowserOpenURL(url),
    },
    screenshot: {
      take: () => goApp.StartScreenshot(),
      onTaken: (callback: (data: any) => void) => runtimeShim.EventsOn('screenshot-taken', (data: any) => callback(data)),
      removeOnTaken: () => runtimeShim.EventsOff('screenshot-taken'),
      confirmSelection: (imageData: string, bounds: any) => runtimeShim.EventsEmit('screenshot-confirm', { imageData, bounds }),
      cancelSelection: () => runtimeShim.EventsEmit('screenshot-cancel'),
      getScreenInfo: async () => null,
    },
    websocket: {
      send: (_message: any) => {},
      onMessage: (_callback: (message: any) => void) => {},
      removeOnMessage: (_callback: (message: any) => void) => {},
    },
    webrtc: {
      send: (_message: any) => {},
      onMessage: (_callback: (message: any) => void) => {},
      removeOnMessage: (_callback: (message: any) => void) => {},
    },
    tray: {
      flash: () => runtimeShim.EventsEmit('tray-flash', true),
      stopFlash: () => runtimeShim.EventsEmit('tray-flash', false),
    },
  }

  ;(window as any).runtime = runtimeShim
  ;(window as any).go = { main: { App: goApp } }
  ;(window as any).electron = electronShim
}

initTauriShim()
