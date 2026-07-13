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
  tray: {
    flash: () => void
    stopFlash: () => void
  }
  windowState: {
    isActive: () => Promise<boolean>
  }
}

interface Window {
  electron: ElectronAPI
  api?: {
    invoke: (channel: string, data?: any) => Promise<any>
    on: (channel: string, callback: (...args: any[]) => void) => void
    removeListener: (channel: string, callback: (...args: any[]) => void) => void
  }
}
