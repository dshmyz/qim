/**
 * 通知中心"点击通知"的深链路由决策（纯函数，便于单测）。
 *
 * 统一读取 mapper 解析后的 actionPayload（来自后端 action_payload），
 * 各类型字段约定：
 *   - channel_message  → channel_id
 *   - group_invitation / group_join_request → conversation_id
 *   - todo_assigned    → task_id
 *   - event_reminder   → event_id
 * 返回"导航意图"，由调用方（Main.vue）执行实际的 UI 跳转。
 */

export type NotificationNavIntent =
  | { kind: 'channel'; channelId: string }
  | { kind: 'conversation'; conversationId: string }
  | { kind: 'groups' }
  | { kind: 'task'; taskId: string }
  | { kind: 'calendar'; eventId: number | string }
  | { kind: 'none' }

export interface NotificationLike {
  type?: string
  actionPayload?: Record<string, any>
  data?: Record<string, any>
}

export function resolveNotificationNav(notification: NotificationLike): NotificationNavIntent {
  const payload = notification.actionPayload || {}

  // 频道消息：打开对应频道
  if (notification.type === 'channel_message' && payload.channel_id !== undefined) {
    return { kind: 'channel', channelId: String(payload.channel_id) }
  }

  // 群聊邀请 / 入群申请：直接进入对应会话（后端 payload 是 conversation_id）
  if (
    (notification.type === 'group_invitation' || notification.type === 'group_join_request') &&
    payload.conversation_id !== undefined
  ) {
    return { kind: 'conversation', conversationId: String(payload.conversation_id) }
  }

  // 新成员加入：payload 无会话 id，退而切到群组面板
  if (notification.type === 'group_member_added') {
    return { kind: 'groups' }
  }

  // 待办指派：打开任务应用并聚焦该任务
  if (notification.type === 'todo_assigned' && payload.task_id !== undefined) {
    return { kind: 'task', taskId: String(payload.task_id) }
  }

  // 日历提醒：打开日历应用并定位到该事件
  if (notification.type === 'event_reminder') {
    const eventId = payload.event_id ?? notification.data?.event_id
    if (eventId !== undefined) {
      return { kind: 'calendar', eventId }
    }
  }

  // 其余类型（system_message / 异常告警 等）无明确跳转目标
  return { kind: 'none' }
}
