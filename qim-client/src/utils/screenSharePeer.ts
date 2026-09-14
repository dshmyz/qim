/**
 * 屏幕共享/查看的对端解析：从单聊会话对象提取目标用户。
 * 解析不出对端一律返回 null（调用方据此清理残留值并提示），
 * 杜绝「members 找不到对端时沿用上一会话残留 id」把申请发给错误用户。
 */

export interface ScreenShareTarget {
  receiverId: number
  conversationId: number
  peerName: string
}

interface ConversationLike {
  id: number | string
  name?: string
  other_member_id?: number | string
  other_member_name?: string
  members?: Array<{ id: number | string; name?: string }>
}

export function resolveScreenShareTarget(
  conv: ConversationLike,
  currentUserId: number | string
): ScreenShareTarget | null {
  // 优先使用后端返回的 other_member_id 字段
  if (conv.other_member_id) {
    return {
      receiverId: Number(conv.other_member_id),
      conversationId: Number(conv.id),
      peerName: conv.other_member_name || conv.name || '未知用户'
    }
  }

  // 降级方案：双成员会话里取与当前用户不同的那个
  if (conv.members && conv.members.length === 2) {
    const otherMember = conv.members.find(m => String(m.id) !== String(currentUserId))
    if (!otherMember) {
      // 双成员却找不到对端（脏数据）：必须返回 null，调用方清理残留
      return null
    }
    return {
      receiverId: Number(otherMember.id),
      conversationId: Number(conv.id),
      peerName: otherMember.name || conv.name || '未知用户'
    }
  }

  return null
}
