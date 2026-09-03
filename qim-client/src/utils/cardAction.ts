/**
 * card_action 消息的 content JSON 载荷解析。
 * 主窗口 MessageItem 的「✓ 已选择」气泡与 BotChatView 的 bot 卡片动作共用，
 * 避免同一 wire 协议两个解析器漂移。
 */
export function parseCardActionContent(content: string): { ok: boolean; data?: Record<string, any> } {
  try {
    const p = JSON.parse(content)
    return p && typeof p === 'object' ? { ok: true, data: p } : { ok: false }
  } catch {
    return { ok: false }
  }
}
