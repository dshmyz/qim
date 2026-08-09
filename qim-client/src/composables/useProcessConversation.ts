import { Ref } from 'vue'
import { generateAvatar, isAbsoluteUrl, getAvatarUrl } from '../utils/avatar'
import { resolveMessageDisplay } from '../utils/messageDisplay'

export interface Conversation {
  id: string
  name: string
  avatar: string
  type: 'group' | 'discussion' | 'bot' | 'single'
  lastMessage?: {
    id: string
    content: string
    sender: {
      id: string
      name: string
      username: string
      avatar: string
      user?: any
    }
    timestamp: number
    type: string
    isSelf: boolean
    file_name?: string
    file_size?: number
    miniAppData?: any
    shareData?: any
  }
  unread_count: number
  timestamp: number
  members: any[]
  is_pinned: boolean
  muted: boolean
  announcement?: string
  ip?: string
  status?: string
  signature?: string
  other_member_id?: string
  other_member_name?: string
  other_member_type?: string
  [key: string]: any
}

export function useProcessConversation(serverUrl: Ref<string>, currentUser: Ref<any>) {
  const processConversation = (conv: any): Conversation => {
    const members = conv.members ? conv.members.map((member: any) => {
      const memberName = member.user ? (member.user.nickname || member.user.username || '') : (member.User ? (member.User.Nickname || member.User.Username || '') : '')
      const memberAvatar = member.user?.avatar || member.User?.Avatar || ''
      return {
        id: member.user && member.user.id ? member.user.id.toString() : (member.UserID ? member.UserID.toString() : (member.user_id ? member.user_id.toString() : '')),
        name: memberName,
        username: member.user ? member.user.username || '' : (member.User ? member.User.Username || '' : ''),
        avatar: getAvatarUrl(memberAvatar, memberName || '用户', serverUrl.value),
        role: member.role || member.Role || 'member',
        type: member.user?.type || member.User?.Type || '',
        disabled: member.disabled || member.user?.disabled || member.User?.Disabled,
        is_disabled: member.is_disabled || member.user?.is_disabled || member.User?.IsDisabled,
        is_deleted: member.is_deleted || member.user?.is_deleted || member.User?.IsDeleted,
        deletedAt: member.deletedAt || member.user?.deletedAt || member.User?.DeletedAt,
        deleted_at: member.deleted_at || member.user?.deleted_at || member.User?.deleted_at,
        status: member.status || member.user?.status || member.User?.Status
      }
    }) : []
    
    let avatar = conv.avatar || ''
    let name = conv.name || ''
    const currentUserId = currentUser.value?.id?.toString() || ''

    // bot 会话的身份由后端统一取自 bots 表（顶层 name/avatar 稳定提供），
    // 不再走下方 self-chat / 成员对端的覆盖逻辑——否则无虚拟用户成员的 bot
    // 会因 members.length===1 被误判为「自聊」而显示成用户自己。
    const isBotConv = conv.type === 'bot'
    const isSelfChat = (conv.type !== 'group' && conv.type !== 'discussion') && members.length === 1 && members[0].id === currentUserId

    if (isBotConv) {
      const botMember = members.find((m: any) => m.type === 'bot') || (members.length === 1 ? members[0] : undefined)
      if (!name) name = botMember?.name || '机器人'
      if (!avatar) avatar = botMember?.avatar || ''
    } else if (isSelfChat) {
      avatar = members[0].avatar || avatar || generateAvatar('self')
      name = members[0].name || currentUser.value?.nickname || currentUser.value?.username || '自己'
    } else if ((conv.type !== 'group' && conv.type !== 'discussion') && members.length > 1) {
      const otherMember = members.find((m: any) => m.id !== currentUserId)
      if (otherMember) {
        avatar = otherMember.avatar || ''
        name = otherMember.name || ''
      }
    }
    
    if ((conv.type === 'group' || conv.type === 'discussion') && conv.avatar) {
      avatar = (isAbsoluteUrl(conv.avatar)) ? conv.avatar : serverUrl.value + conv.avatar
    }
    
    const unreadCount = conv.unread_count || 0
    const lastMessage = conv.lastMessage || conv.last_message
    const lastMessageContent = lastMessage?.content || ''
    const lastMessageType = lastMessage?.type || 'text'
    const lastMessageDisplay = lastMessage
      ? resolveMessageDisplay({ type: lastMessageType, content: lastMessageContent })
      : undefined
    
    const getSenderInfo = (senderId: string, members: any[]) => {
      const member = members.find(m => m.id === senderId)
      if (member) {
        return {
          id: member.id,
          name: member.name || '',
          avatar: member.avatar || ''
        }
      }
      return {
        id: '',
        name: '',
        avatar: ''
      }
    }
    
    const conversationObj: Conversation = {
      id: conv.id ? conv.id.toString() : (conv.ID ? conv.ID.toString() : ''),
      name: name || '',
      avatar: avatar || generateAvatar(name || 'user'),
      ip: conv.ip || '',
      status: conv.status || 'offline',
      signature: conv.signature || '',
      other_member_id: conv.other_member_id || conv.OtherMemberID || '',
      other_member_name: conv.other_member_name || conv.OtherMemberName || '',
      other_member_type: conv.other_member_type || conv.OtherMemberType || '',
      lastMessage: lastMessage ? {
        id: (conv.lastMessage?.id || conv.last_message?.id) ? (conv.lastMessage?.id || conv.last_message?.id).toString() : '',
        content: lastMessageContent,
        file_name: conv.lastMessage?.file_name || conv.last_message?.file_name,
        file_size: conv.lastMessage?.file_size || conv.last_message?.file_size,
        sender: (conv.lastMessage?.sender || conv.last_message?.sender) ? {
          id: (conv.lastMessage?.sender?.id || conv.last_message?.sender?.id) ? (conv.lastMessage?.sender?.id || conv.last_message?.sender?.id).toString() : '',
          name: conv.lastMessage?.sender?.nickname || conv.lastMessage?.sender?.username || conv.lastMessage?.sender?.name || conv.lastMessage?.sender?.user?.nickname || conv.lastMessage?.sender?.user?.username || conv.last_message?.sender?.nickname || conv.last_message?.sender?.username || conv.last_message?.sender?.name || conv.last_message?.sender?.user?.nickname || conv.last_message?.sender?.user?.username || '',
          username: conv.lastMessage?.sender?.username || conv.lastMessage?.sender?.user?.username || conv.last_message?.sender?.username || conv.last_message?.sender?.user?.username || '',
          avatar: getAvatarUrl(conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar, conv.lastMessage?.sender?.nickname || conv.lastMessage?.sender?.username || conv.lastMessage?.sender?.name || conv.last_message?.sender?.nickname || conv.last_message?.sender?.username || conv.last_message?.sender?.name || '用户', serverUrl.value),
          user: conv.lastMessage?.sender || conv.last_message?.sender
        } : {
          id: '',
          name: '',
          username: '',
          avatar: ''
        },
        timestamp: conv.lastMessage?.created_at || conv.last_message?.created_at ? new Date(conv.lastMessage?.created_at || conv.last_message?.created_at).getTime() : Date.now(),
        type: lastMessageType,
        isSelf: false,
        miniAppData: lastMessageDisplay?.kind === 'miniApp' ? lastMessageDisplay.data : undefined,
        shareData: lastMessageDisplay?.kind === 'share' ? lastMessageDisplay.data : undefined,
      } : undefined,
      unread_count: unreadCount,
      timestamp: conv.last_message_at ? new Date(conv.last_message_at).getTime() : (conv.created_at ? new Date(conv.created_at).getTime() : Date.now()),
      type: (conv.type === 'group' || conv.type === 'Group' || conv.type === 'GROUP') ? 'group' : (conv.type === 'discussion' || conv.type === 'Discussion' || conv.type === 'DISCUSSION') ? 'discussion' : (conv.type === 'bot' ? 'bot' : 'single'),
      members: members,
      is_pinned: conv.is_pinned || false,
      muted: conv.muted || false,
      creator_id: conv.creator_id || conv.CreatorID || '',
      owner_id: conv.owner_id || conv.OwnerID || '',
      ownerId: conv.ownerId || conv.owner_id || conv.creator_id || conv.OwnerID || conv.CreatorID || '',
      announcement: conv.announcement || '',
      invite_permission: conv.invite_permission || 'owner_admin'
    }
    
    if ((conversationObj.type === 'group' || conversationObj.type === 'discussion') && conversationObj.lastMessage) {
      const senderId = conversationObj.lastMessage.sender?.id || (conv.lastMessage?.sender_id || conv.last_message?.sender_id)?.toString() || ''
      if (senderId && (!conversationObj.lastMessage.sender?.name || conversationObj.lastMessage.sender.name === '')) {
        const senderInfo = getSenderInfo(senderId, members)
        if (senderInfo.name) {
          conversationObj.lastMessage.sender.name = senderInfo.name
          if (senderInfo.avatar) {
            conversationObj.lastMessage.sender.avatar = senderInfo.avatar
          }
        }
      }
    }
    
    return conversationObj
  }

  return {
    processConversation
  }
}
