export interface ElectronAPI {
  ipcRenderer: {
    send: (channel: string, data?: any) => void
    on: (channel: string, callback: (event: any, ...args: any[]) => void) => void
    once: (channel: string, callback: (event: any, ...args: any[]) => void) => void
    removeAllListeners: (channel: string) => void
    invoke: (channel: string, data?: any) => Promise<any>
  }
  shell: {
    openExternal: (url: string) => void
  }
  screenshot: {
    take: () => void
    confirmSelection: (imageData: string, bounds: { x: number; y: number; width: number; height: number }) => void
    cancelSelection: () => void
    getScreenInfo: () => Promise<any>
  }
}

declare global {
  interface Window {
    electron: ElectronAPI
  }
}
