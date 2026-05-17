function initWailsShim() {
  const isWails = typeof window.runtime !== 'undefined'

  if (!isWails) {
    return
  }

  const runtime = window.runtime!
  const goApp = window.go?.main?.App

  const electronShim = {
    ipcRenderer: {
      send: (channel: string, data?: any) => {
        switch (channel) {
          case 'minimize-window':
            runtime.WindowMinimise()
            break
          case 'maximize-window':
            if (runtime.WindowIsMaximised()) {
              runtime.WindowUnmaximise()
            } else {
              runtime.WindowMaximise()
            }
            break
          case 'close-window':
            runtime.WindowHide()
            break
          case 'flash-tray':
            runtime.EventsEmit('tray-flash', true)
            break
          case 'stop-tray-flash':
            runtime.EventsEmit('tray-flash', false)
            break
          case 'take-screenshot':
            runtime.EventsEmit('screenshot-requested', true)
            break
          case 'start-screen-share':
            runtime.EventsEmit('screen-share-requested', true)
            break
          case 'download-file':
            if (data && goApp) {
              const { buffer, fileName, saveDir } = data
              goApp.DownloadFile(fileName, new Uint8Array(buffer), saveDir || '')
                .then((result: any) => {
                  runtime.EventsEmit('download-complete', { success: true, filePath: result.filePath })
                })
                .catch((err: any) => {
                  runtime.EventsEmit('download-complete', { success: false, error: err.message })
                })
            }
            break
          case 'save-file-as':
            if (data && goApp) {
              const { buffer, fileName } = data
              goApp.SaveFileAs(fileName, new Uint8Array(buffer))
                .then((result: any) => {
                  runtime.EventsEmit('save-file-complete', { success: true, filePath: result.filePath })
                })
                .catch((err: any) => {
                  runtime.EventsEmit('save-file-complete', { success: false, error: err.message })
                })
            }
            break
          case 'open-file-dialog':
            if (data && goApp) {
              goApp.OpenFileDialog(JSON.stringify(data))
                .then((result: any) => {
                  runtime.EventsEmit('file-dialog-result', result)
                })
            }
            break
          case 'cache-avatar':
            if (data && goApp) {
              goApp.CacheAvatar(data)
                .then((cachedUrl: string) => {
                  runtime.EventsEmit('avatar-cached', cachedUrl)
                })
                .catch(() => {
                  runtime.EventsEmit('avatar-cached', data)
                })
            }
            break
          case 'check-for-updates':
            if (goApp) {
              goApp.CheckForUpdates()
                .then((info: any) => {
                  if (info.available) {
                    runtime.EventsEmit('update-available', info)
                  } else {
                    runtime.EventsEmit('update-not-available')
                  }
                })
            }
            break
          case 'download-update':
            goApp?.DownloadUpdate()
            break
        }
      },
      on: (channel: string, callback: (...args: any[]) => void) => {
        runtime.EventsOn(channel, (data: any) => {
          callback(null, data)
        })
      },
      once: (channel: string, callback: (...args: any[]) => void) => {
        runtime.EventsOn(channel, (data: any) => {
          callback(null, data)
        })
      },
      removeListener: (_channel: string, _callback: (...args: any[]) => void) => {
        runtime.EventsOff(_channel)
      },
      removeAllListeners: (channel: string) => {
        runtime.EventsOff(channel)
      },
      invoke: async (_channel: string, _data?: any) => {
        return null
      },
    },
    shell: {
      openExternal: (url: string) => {
        runtime.BrowserOpenURL(url)
      },
    },
    screenshot: {
      take: () => {
        runtime.EventsEmit('screenshot-requested', true)
      },
      onTaken: (callback: (data: any) => void) => {
        runtime.EventsOn('screenshot-taken', (data: any) => callback(data))
      },
      removeOnTaken: (_callback: (data: any) => void) => {
        runtime.EventsOff('screenshot-taken')
      },
      confirmSelection: (imageData: string, bounds: any) => {
        runtime.EventsEmit('screenshot-confirm', { imageData, bounds })
      },
      cancelSelection: () => {
        runtime.EventsEmit('screenshot-cancel')
      },
      getScreenInfo: async () => {
        return null
      },
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
      flash: () => {
        runtime.EventsEmit('tray-flash', true)
      },
      stopFlash: () => {
        runtime.EventsEmit('tray-flash', false)
      },
    },
  }

  ;(window as any).electron = electronShim
}

initWailsShim()