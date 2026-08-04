import { describe, expect, it } from 'vitest'
import { resolveNotificationNav } from '../../src/utils/notificationNavigation'
import { mapNotification } from '../../src/utils/notificationMapper'

describe('resolveNotificationNav - 通知点击深链路由决策', () => {
  it('channel_message 命中 channel_id → 打开频道', () => {
    const n = mapNotification({ type: 'channel_message', action_payload: '{"channel_id":5,"channel_name":"公告"}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'channel', channelId: '5' })
  })

  it('group_invitation 命中 conversation_id → 进入会话', () => {
    const n = mapNotification({ type: 'group_invitation', action_payload: '{"conversation_id":8}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'conversation', conversationId: '8' })
  })

  it('group_join_request 命中 conversation_id → 进入会话', () => {
    const n = mapNotification({ type: 'group_join_request', action_payload: '{"conversation_id":3,"user_id":9}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'conversation', conversationId: '3' })
  })

  it('group_member_added 无 payload → 切到群组面板', () => {
    const n = mapNotification({ type: 'group_member_added' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'groups' })
  })

  it('todo_assigned 命中 task_id → 打开任务', () => {
    const n = mapNotification({ type: 'todo_assigned', action_payload: '{"task_id":42}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'task', taskId: '42' })
  })

  it('event_reminder 命中 event_id → 打开日历', () => {
    const n = mapNotification({ type: 'event_reminder', action_payload: '{"event_id":77}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'calendar', eventId: 77 })
  })

  it('event_reminder 无 actionPayload 时回退读 data.event_id', () => {
    const n = mapNotification({ type: 'event_reminder', data: { event_id: 123 } })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'calendar', eventId: 123 })
  })

  it('system_message 无目标 → none', () => {
    const n = mapNotification({ type: 'system_message', action_payload: '{}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'none' })
  })

  it('缺 type / 空对象 → none', () => {
    expect(resolveNotificationNav({})).toEqual({ kind: 'none' })
    expect(resolveNotificationNav({ type: 'unknown_type', actionPayload: {} })).toEqual({ kind: 'none' })
  })

  it('channel_message 缺 channel_id → none', () => {
    const n = mapNotification({ type: 'channel_message', action_payload: '{}' })
    expect(resolveNotificationNav(n)).toEqual({ kind: 'none' })
  })
})
