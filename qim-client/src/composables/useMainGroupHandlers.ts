import { type Ref } from 'vue'
import type { Conversation, Message } from '../types'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { useServerUrl } from './useServerUrl'
import { getAvatarUrl } from '../utils/avatar'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useMainGroupHandlers(
  conversations: Ref<Conversation[]>,
  currentConversationId: Ref<string | null>,
  messages: Ref<Message[]>
) {
  const chatStore = useChatStore()
  const { currentUser } = useCurrentUser()
  const { serverUrl } = useServerUrl()

  const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
    const { message, type = 'info', duration } = options
    if (type === 'success') QMessage.success(message, duration)
    else if (type === 'error') QMessage.error(message, duration)
    else if (type === 'warning') QMessage.warning(message, duration)
    else QMessage.info(message, duration)
  }

  const handleGroupInvitation = (data: any) => {
    logger.log('收到群聊邀请:', data)
    showMessage({
      message: `您收到了加入群聊 "${data.group_name}" 的邀请`,
      type: 'info',
      duration: 5000
    })
  }

  const handleAddedToGroup = (data: any) => {
    logger.log('被添加到群聊:', data)
    
    const groupConversation = {
      id: data.conversation_id.toString(),
      name: data.group_name,
      avatar: getAvatarUrl(data.group_avatar, 'group', serverUrl.value),
      lastMessage: null,
      unread_count: 0,
      timestamp: Date.now(),
      type: 'group' as const,
      members: data.members || []
    }
    
    const existingIndex = conversations.value.findIndex(c => c.id === groupConversation.id)
    if (existingIndex === -1) {
      chatStore.addConversation(groupConversation as any)
    } else {
      chatStore.patchConversation(groupConversation.id, { members: data.members || [] })
    }
    
    showMessage({
      message: `您已被添加到群聊 "${data.group_name}"`,
      type: 'success',
      duration: 5000
    })
  }

  const handleGroupMemberLeft = (data: any) => {
    logger.log('成员退出群聊:', data)
    
    const conversationId = data.conversation_id.toString()
    const userId = data.user_id.toString()
    
    chatStore.removeGroupMember(conversationId, userId)
    
    if (userId === currentUser.value?.id?.toString()) {
      chatStore.patchConversation(conversationId, { isExited: true } as any)
    }
  }

  const handleGroupMemberJoined = (data: any) => {
    logger.log('成员加入群聊:', data)

    const conversationId = data.conversation_id.toString()
    const newMember = data.member
    const memberName = newMember.nickname || newMember.username || (newMember.name !== undefined ? newMember.name : '未知用户')
    const memberData = {
      id: newMember.id?.toString() || '',
      name: memberName,
      avatar: newMember.avatar || '',
      type: newMember.type || ''
    }

    chatStore.addGroupMember(conversationId, memberData)

    if (currentConversationId.value === conversationId) {
      const systemMessage = {
        id: `system_${Date.now()}`,
        type: 'system',
        content: `${memberName} 加入了群聊`,
        timestamp: Date.now(),
        sender: {
          id: 'system',
          name: '系统',
          avatar: ''
        },
        isSelf: false,
        isRead: true,
        conversationId: String(conversationId)
      }
      messages.value.push(systemMessage as any)
    }
  }

  const handleGroupAnnouncementUpdated = (data: any) => {
    logger.log('群公告更新:', data)
    const { group_id, announcement, announcer_name } = data
    
    showMessage({
      message: `${announcer_name} 更新了群公告`,
      type: 'info',
      duration: 5000
    })
  }

  const handleGroupMemberRoleUpdated = (data: any) => {
    logger.log('群成员角色更新:', data)
    const { group_id, user_id, user_name, new_role } = data
    
    const roleNames = {
      'admin': '管理员',
      'member': '普通成员',
      'owner': '群主'
    }
    
    showMessage({
      message: `${user_name} 已成为${roleNames[new_role] || new_role}`,
      type: 'info',
      duration: 3000
    })
  }

  const handleGroupOwnerTransferred = (data: any) => {
    logger.log('群主转让:', data)
    const { group_id, new_owner_id, new_owner_name } = data
    
    showMessage({
      message: `群主已转让给 ${new_owner_name}`,
      type: 'warning',
      duration: 5000
    })
  }

  return {
    handleGroupInvitation,
    handleAddedToGroup,
    handleGroupMemberLeft,
    handleGroupMemberJoined,
    handleGroupAnnouncementUpdated,
    handleGroupMemberRoleUpdated,
    handleGroupOwnerTransferred
  }
}
