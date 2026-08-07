export interface User {
  id: string | number
  username: string
  nickname?: string
  avatar?: string
  isAdmin?: boolean
  [key: string]: any
}

const defaultUser: User = {
  id: '1',
  username: 'admin',
  nickname: '管理员',
  avatar: '',
  isAdmin: false
}

export const getCurrentUser = (): User => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr) as User
      if (user && user.id) {
        user.isAdmin = user.roles?.includes('system_admin') || false
        return user
      }
    } catch (error) {
      console.error('解析用户信息失败:', error)
    }
  }
  return { ...defaultUser }
}

export const setCurrentUser = (user: User): void => {
  localStorage.setItem('user', JSON.stringify(user))
}

export const clearCurrentUser = (): void => {
  localStorage.removeItem('user')
}

/**
 * 头像右下角角标描述。与 Avatar.vue 的 badge prop 一一对应，
 * 由业务层（utils/user.ts）构造，Avatar 只负责渲染。
 */
export interface AvatarBadge {
  type: 'status' | 'icon' | 'text'
  /** type='icon' 时的 FontAwesome 类名，如 'fas fa-user-friends' */
  icon?: string
  /** type='text' 时的角标文字 */
  text?: string
  /** type='status' 时的状态 */
  status?: 'online' | 'offline' | 'busy'
  /** 角标背景色（可选，覆盖默认） */
  color?: string
  /** 角标尺寸：'sm' | 'md' | 'lg'（默认 'md'），lg 用于会话列表等需要更醒目角标的场景 */
  size?: 'sm' | 'md' | 'lg'
  /** 角标外圈白环 + 阴影（默认 true；设为 false 可去掉白环边，用于角标较小时更干净） */
  ring?: boolean
  /** 悬浮提示 */
  title?: string
}

export type PartnerType = 'user' | 'bot' | 'system' | 'api'

/**
 * 从会话对象推导「对方/伙伴」的账号类型，用于头像角标。
 * - bot / group / discussion 直接按其自身类型判定；
 * - single 会话默认不预加载 members，优先用后端返回的对方类型字段
 *   `other_member_type`（同时兼容 `otherMemberType`），再兜底从 members 找。
 *
 * 修复点：兜底从 members 找对方时，必须排除当前用户 id，
 * 否则当对方是普通 user 时会取到 members[0]（可能是当前用户）。
 *
 * @param conversation 会话对象
 * @param currentUserId 当前用户 id；不传则从 localStorage 读取
 */
export const conversationPartnerType = (
  conversation: any,
  currentUserId?: string | number
): PartnerType => {
  if (!conversation) return 'user'
  if (conversation.type === 'bot') return 'bot'
  if (conversation.type === 'system') return 'system'
  if (conversation.type === 'api') return 'api'
  if (conversation.type === 'group' || conversation.type === 'discussion') return 'user'

  if (conversation.type === 'single') {
    const otherType = conversation.other_member_type || conversation.otherMemberType
    if (otherType) return otherType as PartnerType

    const members = conversation.members || []
    if (members.length === 0) return 'user'

    // 优先用传入的 currentUserId；未传则回退到 localStorage 中的当前用户
    const meId = currentUserId ?? getCurrentUser().id
    const meIdStr = meId !== undefined && meId !== null ? String(meId) : ''

    // 先按 id 排除当前用户，再退而取第一个；保证返回的是「对方」的类型。
    const other = (meIdStr
      ? members.find((m: any) => m.id !== undefined && String(m.id) !== meIdStr)
      : undefined) || members[0]
    return (other?.type as PartnerType) || 'user'
  }

  return 'user'
}

/**
 * 判断会话的「对方」是否为非人类账号（bot 或 system）。
 * 系统助手与机器人一致：语音通话、屏幕共享、右上角分身设置等约束应同样生效。
 * 兼容 direct 类型（type=bot/system）与 single 会话中对方为 bot/system 两种形态。
 *
 * @param conversation 会话对象
 * @param currentUserId 当前用户 id；不传则从 localStorage 读取（用于 single 会话 partner 推导）
 */
export const isConversationPeerNonHuman = (
  conversation: any,
  currentUserId?: string | number
): boolean => {
  const partner = conversationPartnerType(conversation, currentUserId)
  return partner === 'bot' || partner === 'system'
}

/**
 * 会话头像右下角的角标。统一了 ConversationList / ChatHeader / SearchResult 三处调用：
 * - group → user-friends 图标
 * - discussion → comments 图标
 * - single 系统/bot/api → 对应类型图标
 * - single 真人 → 在线状态点（若有 status）
 *
 * @param conversation 会话对象
 * @param currentUserId 当前用户 id；不传则从 localStorage 读取（用于 single 会话 partner 推导）
 */
export const buildConversationBadge = (
  conversation: any,
  currentUserId?: string | number
): AvatarBadge | null => {
  if (!conversation) return null
  const type = conversation.type
  const partnerType = conversationPartnerType(conversation, currentUserId)

  if (type === 'group') {
    return { type: 'icon', icon: 'fas fa-user-friends', title: '群聊', color: 'var(--primary-color, #1890ff)' }
  }
  if (type === 'discussion') {
    return { type: 'icon', icon: 'fas fa-comments', title: '讨论组', color: 'var(--color-success-500, #52c41a)' }
  }

  if (partnerType === 'system') {
    return { type: 'icon', icon: 'fas fa-cog', title: '系统', color: 'var(--color-warning-500, #faad14)' }
  }
  if (partnerType === 'bot') {
    return { type: 'icon', icon: 'fas fa-robot', title: '机器人', color: 'var(--primary-color, #1890ff)' }
  }
  if (partnerType === 'api') {
    return { type: 'icon', icon: 'fas fa-plug', title: 'API', color: 'var(--color-success-500, #52c41a)' }
  }

  // 普通真人：在线状态点
  if (conversation.status) {
    const s = conversation.status
    return { type: 'status', status: s, title: s === 'online' ? '在线' : s === 'busy' ? '忙碌' : '离线' }
  }
  return null
}

/**
 * 消息发送者头像角标。统一 MessageItem 等处的 sender 类型映射：
 * system → cog / api → plug / bot → robot / user → null（不显示）。
 *
 * @param sender 消息 sender 对象，可能直接含 type，也可能 sender.user.type
 * @param isAI 兜底判定为 bot 的标志（旧逻辑用 origin/senderIsBot 等组合判定）
 */
export const buildSenderBadge = (sender: any, isAI = false): AvatarBadge | null => {
  const senderType: PartnerType =
    sender?.type || sender?.user?.type || (isAI ? 'bot' : 'user')

  if (senderType === 'system') {
    return { type: 'icon', icon: 'fas fa-cog', title: '系统', color: 'var(--color-warning-500, #faad14)' }
  }
  if (senderType === 'api') {
    return { type: 'icon', icon: 'fas fa-plug', title: 'API', color: 'var(--color-success-500, #52c41a)' }
  }
  if (senderType === 'bot') {
    return { type: 'icon', icon: 'fas fa-robot', title: '机器人', color: 'var(--primary-color, #1890ff)' }
  }
  return null
}

/**
 * 搜索结果头像角标。按 item.type 直接映射：
 * group / discussion / bot → 对应图标；其他 → null。
 */
export const buildSearchResultBadge = (item: any): AvatarBadge | null => {
  if (!item) return null
  if (item.type === 'group') {
    return { type: 'icon', icon: 'fas fa-user-friends', title: '群聊', color: 'var(--primary-color, #1890ff)' }
  }
  if (item.type === 'discussion') {
    return { type: 'icon', icon: 'fas fa-comments', title: '讨论组', color: 'var(--color-success-500, #52c41a)' }
  }
  if (item.type === 'bot') {
    return { type: 'icon', icon: 'fas fa-robot', title: '机器人', color: 'var(--primary-color, #1890ff)' }
  }
  return null
}
