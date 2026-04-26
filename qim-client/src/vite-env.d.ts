/// <reference types="vite/client" />
/// <reference path="./types/electron.d.ts" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface MessageBoxOptions {
  title?: string
  message: string
  type?: 'warning' | 'error' | 'success' | 'info'
  confirmButtonText?: string
  cancelButtonText?: string
  showCancelButton?: boolean
  showClose?: boolean
  inputType?: 'text' | 'password' | ''
  inputPlaceholder?: string
}

interface MessageBoxResult {
  action: 'confirm' | 'cancel' | 'close'
  value?: string
}

interface QMessageBoxAPI {
  show: (options: MessageBoxOptions) => Promise<MessageBoxResult>
  confirm: (message: string, title?: string, options?: Partial<MessageBoxOptions>) => Promise<MessageBoxResult>
  alert: (message: string, title?: string) => Promise<MessageBoxResult>
  prompt: (message: string, title?: string, placeholder?: string) => Promise<MessageBoxResult>
}

interface Window {
  $QMessageBox: QMessageBoxAPI
}
