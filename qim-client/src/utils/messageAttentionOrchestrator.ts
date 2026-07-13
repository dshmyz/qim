import { shouldRequestMessageAttention } from './messageAttention'

export interface MessageAttentionRequest {
  isCurrentConversation: boolean
  isStreaming: boolean
  getIsWindowActive: () => Promise<boolean>
  onAttention: () => void
  onWindowStateError?: (error: unknown) => void
}

export function requestMessageAttention({
  isCurrentConversation,
  isStreaming,
  getIsWindowActive,
  onAttention,
  onWindowStateError,
}: MessageAttentionRequest): void {
  if (isStreaming) return

  if (!isCurrentConversation) {
    onAttention()
    return
  }

  void getIsWindowActive()
    .then((isWindowActive) => {
      if (shouldRequestMessageAttention({
        isCurrentConversation,
        isStreaming,
        isWindowActive,
      })) {
        onAttention()
      }
    }, (error: unknown) => {
      onWindowStateError?.(error)
      onAttention()
    })
}
