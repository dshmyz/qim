interface ElectronAPI {
  ipcRenderer: {
    send: (channel: string, data?: any) => void
    on: (channel: string, callback: (event: any, ...args: any[]) => void) => void
    once: (channel: string, callback: (event: any, ...args: any[]) => void) => void
    removeListener: (channel: string, callback: (event: any, ...args: any[]) => void) => void
    removeAllListeners: (channel: string) => void
    invoke: (channel: string, data?: any) => Promise<any>
  }
  shell: {
    openExternal: (url: string) => void
  }
  screenshot: {
    take: () => void
    onTaken: (callback: (data: any) => void) => void
    removeOnTaken: (callback: (data: any) => void) => void
    confirmSelection: (imageData: string, bounds: { x: number; y: number; width: number; height: number }) => void
    cancelSelection: () => void
    getScreenInfo: () => Promise<any>
  }
  websocket: {
    send: (message: any) => void
    onMessage: (callback: (message: any) => void) => void
    removeOnMessage: (callback: (message: any) => void) => void
  }
  webrtc: {
    send: (message: any) => void
    onMessage: (callback: (message: any) => void) => void
    removeOnMessage: (callback: (message: any) => void) => void
  }
  tray: {
    flash: () => void
    stopFlash: () => void
  }
}

interface WailsRuntime {
  WindowMinimise: () => void
  WindowMaximise: () => void
  WindowUnmaximise: () => void
  WindowIsMaximised: () => boolean
  WindowHide: () => void
  BrowserOpenURL: (url: string) => void
  EventsOn: (channel: string, callback: (data?: any) => void, options?: { once?: boolean }) => void
  EventsOff: (channel: string) => void
  EventsEmit: (channel: string, data?: any) => void
  OpenDirectoryDialog: (opts: any) => Promise<string>
  OpenFileDialog: (opts: any) => Promise<string>
  SaveFileDialog: (opts: any) => Promise<string>
}

interface Window {
  electron: ElectronAPI
  runtime?: WailsRuntime
  go?: {
    main: {
      App: {
        MinimizeWindow: () => void
        MaximizeWindow: () => void
        CloseWindow: () => void
        IsMaximized: () => Promise<boolean>
        OpenExternal: (url: string) => void
        OpenFileDialog: (opts: string) => Promise<any>
        SaveFileAs: (fileName: string, data: Uint8Array) => Promise<any>
        DownloadFile: (fileName: string, data: Uint8Array, saveDir: string) => Promise<any>
        GetAppInfo: () => Promise<any>
        CacheAvatar: (url: string) => Promise<string>
        CleanupAvatarCache: (maxAgeDays: number) => Promise<void>
        FlashTray: (enabled: boolean) => void
        CheckForUpdates: () => Promise<any>
        DownloadUpdate: () => Promise<any>
        GetScreenSources: () => Promise<any>
        StartScreenshot: () => void
        GetPlatform: () => Promise<string>
      }
    }
  }
  api?: {
    invoke: (channel: string, data?: any) => Promise<any>
    on: (channel: string, callback: (...args: any[]) => void) => void
    removeListener: (channel: string, callback: (...args: any[]) => void) => void
  }
}
