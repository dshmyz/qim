type BoxType = 'warning' | 'error' | 'success' | 'info'

interface MessageBoxOptions {
  title?: string
  message: string
  type?: BoxType
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

const showMessageBox = (options: MessageBoxOptions): Promise<MessageBoxResult> => {
  if (typeof window !== 'undefined' && window.$QMessageBox) {
    return window.$QMessageBox.show(options)
  }
  return Promise.reject('MessageBox not initialized')
}

const QMessageBox = {
  show: showMessageBox,
  confirm: (message: string, title?: string, options?: Partial<MessageBoxOptions>): Promise<MessageBoxResult> => {
    if (typeof window !== 'undefined' && window.$QMessageBox) {
      return window.$QMessageBox.confirm(message, title, options)
    }
    return Promise.reject('MessageBox not initialized')
  },
  alert: (message: string, title?: string): Promise<MessageBoxResult> => {
    if (typeof window !== 'undefined' && window.$QMessageBox) {
      return window.$QMessageBox.alert(message, title)
    }
    return Promise.reject('MessageBox not initialized')
  },
  prompt: (message: string, title?: string, placeholder?: string): Promise<MessageBoxResult> => {
    if (typeof window !== 'undefined' && window.$QMessageBox) {
      return window.$QMessageBox.prompt(message, title, placeholder)
    }
    return Promise.reject('MessageBox not initialized')
  }
}

export default QMessageBox
export type { MessageBoxOptions, MessageBoxResult }
