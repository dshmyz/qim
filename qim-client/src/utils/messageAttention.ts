export interface MessageAttentionInput {
  isCurrentConversation: boolean
  isStreaming: boolean
  isWindowActive: boolean
}

export function shouldRequestMessageAttention({
  isCurrentConversation,
  isStreaming,
  isWindowActive,
}: MessageAttentionInput): boolean {
  if (isStreaming) return false
  return !isCurrentConversation || !isWindowActive
}
