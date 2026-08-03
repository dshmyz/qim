/// <reference types="vite/client" />
/// <reference path="./types/electron.d.ts" />

declare const __APP_NAME__: string
declare const __APP_VERSION__: string
declare const __APP_PRODUCT_NAME__: string
declare const __APP_PRODUCT_NAME_CN__: string
declare const __APP_COPYRIGHT_YEAR__: string

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

interface QMessageAPI {
  success: (content: string, duration?: number) => void
  error: (content: string, duration?: number) => void
  warning: (content: string, duration?: number) => void
  info: (content: string, duration?: number) => void
}

interface Window {
  $QMessageBox: QMessageBoxAPI
  $QMessage: QMessageAPI
  // WebSocket 单例（useSignaling 等模块直接复用全局 ws 连接）
  ws?: WebSocket
  // 全局搜索防抖计时器（Main.vue 联系人搜索）
  searchTimeout?: ReturnType<typeof setTimeout>
}
