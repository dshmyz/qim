export function sameConversationId(a: unknown, b: unknown): boolean {
  if (a === null || a === undefined || b === null || b === undefined) {
    return false
  }
  return String(a) === String(b)
}
