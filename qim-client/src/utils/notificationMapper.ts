export interface Notification {
  id: string
  title: string
  content: string
  timestamp: number
  read: boolean
  type: string
  category: 'message' | 'system' | 'group'
  priority: 'normal' | 'important'
  actionType: string
  actionPayload: Record<string, any>
  pinned: boolean
  important: boolean
  handled: boolean
  data?: Record<string, any>
}

const CATEGORY_MAP: Record<string, 'message' | 'system' | 'group'> = {
  group_invitation: 'group',
  group_member_added: 'group',
  channel_message: 'message',
  system_message: 'system',
  event_reminder: 'system',
  todo_assigned: 'system',
  anomaly_alert: 'system',
  sensitive_content: 'system',
  user_message_flood: 'system',
  inactive_group_activated: 'system',
}

function toCategory(type: string): 'message' | 'system' | 'group' {
  return CATEGORY_MAP[type] || 'system'
}

function toTimestamp(value: string | number | undefined): number {
  if (typeof value === 'number') return value
  if (typeof value === 'string') {
    const parsed = new Date(value)
    if (!isNaN(parsed.getTime())) return parsed.getTime()
  }
  return Date.now()
}

function parsePayload(raw: any): Record<string, any> {
  if (raw.action_payload) {
    try { return JSON.parse(raw.action_payload) } catch { return {} }
  }
  return {}
}

export function mapNotification(raw: any): Notification {
  return {
    id: String(raw.id ?? Date.now()),
    title: raw.title || '',
    content: raw.content || '',
    timestamp: toTimestamp(raw.created_at ?? raw.timestamp),
    read: raw.read ?? false,
    type: raw.type || 'system',
    category: toCategory(raw.type || 'system'),
    priority: raw.priority === 'important' ? 'important' : 'normal',
    actionType: raw.action_type || '',
    actionPayload: parsePayload(raw),
    pinned: raw.pinned ?? false,
    important: raw.important ?? false,
    handled: raw.handled ?? false,
    data: raw.data ?? {},
  }
}

export function mapNotifications(raws: any[]): Notification[] {
  return raws.map(mapNotification)
}