const GROUP_RESULT_TYPES = new Set(['group', 'discussion'])

export function isPrivateChatSearchResult(item: { type?: string } | null | undefined): boolean {
  if (!item?.type) return true
  return !GROUP_RESULT_TYPES.has(item.type)
}

export function normalizePrivateChatUserId(id: unknown): number | null {
  if (typeof id === 'number') {
    return Number.isFinite(id) && id > 0 ? id : null
  }

  if (typeof id !== 'string') {
    return null
  }

  const raw = id.startsWith('emp') ? id.slice(3) : id
  if (!/^\d+$/.test(raw)) {
    return null
  }

  const parsed = Number.parseInt(raw, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

export function getPrivateChatUserId(user: unknown): number | null {
  if (typeof user === 'string' || typeof user === 'number') {
    return normalizePrivateChatUserId(user)
  }

  if (!user || typeof user !== 'object') {
    return null
  }

  const record = user as Record<string, unknown>
  return (
    normalizePrivateChatUserId(record.id) ??
    normalizePrivateChatUserId(record.user_id) ??
    normalizePrivateChatUserId(record.userId) ??
    normalizePrivateChatUserId(record.UserID)
  )
}
