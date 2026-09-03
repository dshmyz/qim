/**
 * AI 消息判定单一来源：origin/sender/is_ai_message 任一命中即 AI 消息。
 * MessageItem 的 isAIMessage 计算与 AI 反馈批量筛选共用，避免两处判据漂移。
 */
export function isAIMessage(m: any): boolean {
  const fromOrigin = m?.origin === 'assistant' || m?.origin === 'avatar'
  const fromSenderIsBot = m?.sender?.type === 'bot' || m?.sender?.type === 'system'
  const fromField = !!(m?.is_ai_message || m?.isAIMessage)
  return fromOrigin || fromSenderIsBot || fromField
}
