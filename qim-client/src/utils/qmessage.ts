type MessageType = 'success' | 'error' | 'warning' | 'info'

interface MessageInstance {
  (type: MessageType, content: string, duration?: number): void
  success: (content: string, duration?: number) => void
  error: (content: string, duration?: number) => void
  warning: (content: string, duration?: number) => void
  info: (content: string, duration?: number) => void
}

const showMessage = (type: MessageType, content: string, duration = 3000) => {
  if (typeof window !== 'undefined' && window.$QMessage) {
    window.$QMessage[type](content, duration)
  }
}

const QMessage: MessageInstance = Object.assign(showMessage, {
  success: (content: string, duration?: number) => showMessage('success', content, duration),
  error: (content: string, duration?: number) => showMessage('error', content, duration),
  warning: (content: string, duration?: number) => showMessage('warning', content, duration),
  info: (content: string, duration?: number) => showMessage('info', content, duration)
})

export default QMessage
