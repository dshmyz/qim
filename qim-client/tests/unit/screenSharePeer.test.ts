import { describe, expect, it } from 'vitest'
import { resolveScreenShareTarget } from '@/utils/screenSharePeer'

// 回归：startScreenShareView 原实现里「members.length===2 但 find 不到对端」的分支
// 既不赋值也不清理，残留上一会话的 receiverId 会把共享/查看申请发给错误用户。
// 统一收敛为纯函数：解析不出对端一律返回 null，由调用方清理并提示。
const conv = (overrides: Record<string, unknown> = {}) => ({
  id: 10,
  type: 'single',
  name: '会话名',
  ...overrides,
})

describe('resolveScreenShareTarget', () => {
  it('优先使用 other_member_id / other_member_name', () => {
    const target = resolveScreenShareTarget(conv({ other_member_id: 7, other_member_name: '小张' }), 3)
    expect(target).toEqual({ receiverId: 7, conversationId: 10, peerName: '小张' })
  })

  it('other_member_name 缺失时回退会话名', () => {
    const target = resolveScreenShareTarget(conv({ other_member_id: 7 }), 3)
    expect(target?.peerName).toBe('会话名')
  })

  it('无 other_member_id 时从双成员 members 中解析对端', () => {
    const target = resolveScreenShareTarget(
      conv({ members: [{ id: 3, name: '我' }, { id: 7, name: '对端' }] }),
      3
    )
    expect(target).toEqual({ receiverId: 7, conversationId: 10, peerName: '对端' })
  })

  it('members 数量不为 2 时返回 null', () => {
    const target = resolveScreenShareTarget(conv({ members: [{ id: 3, name: '我' }] }), 3)
    expect(target).toBeNull()
  })

  it('members 为 2 但找不到对端（脏数据）时返回 null，不得沿用残留值', () => {
    const target = resolveScreenShareTarget(
      conv({ members: [{ id: 3, name: '我' }, { id: 3, name: '重复' }] }),
      3
    )
    expect(target).toBeNull()
  })

  it('对端 name 为空时兜底「未知用户」', () => {
    const target = resolveScreenShareTarget(
      conv({ other_member_id: 7, other_member_name: '' , name: '' }),
      3
    )
    expect(target?.peerName).toBe('未知用户')
  })
})
