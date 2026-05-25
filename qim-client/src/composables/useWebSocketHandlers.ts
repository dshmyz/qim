import { ref } from 'vue'
import type { Conversation } from '../types'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { useServerUrl } from './useServerUrl'
import { getAvatarUrl } from '../utils/avatar'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'

export function useWebSocketHandlers() {
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

  const handleReadReceipt = (data: any) => {
    logger.log('收到已读回执:', data)
    const { conversation_id, user_id, last_read_message_id } = data
    
    chatStore.updateReadReceipt(conversation_id, user_id, last_read_message_id)
  }

  const handleMessageRecalled = (data: any) => {
    logger.log('消息撤回:', data)
    const { conversation_id, message_id } = data
    
    chatStore.recallMessage(conversation_id, message_id)
    
    showMessage({
      message: '消息已被撤回',
      type: 'info',
      duration: 3000
    })
  }

  const handleMessageDeleted = (data: any) => {
    logger.log('消息删除:', data)
    const { conversation_id, message_id } = data
    
    chatStore.deleteMessage(conversation_id, message_id)
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
    
    const existingIndex = chatStore.conversations.findIndex(c => c.id === groupConversation.id)
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
    logger.log('群成员离开:', data)
    const { group_id, user_id, user_name } = data
    
    showMessage({
      message: `${user_name} 已离开群聊`,
      type: 'info',
      duration: 3000
    })
  }

  const handleGroupMemberJoined = (data: any) => {
    logger.log('群成员加入:', data)
    const { group_id, user_id, user_name } = data
    
    showMessage({
      message: `${user_name} 加入了群聊`,
      type: 'info',
      duration: 3000
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

  const handleConversationUpdated = (data: any) => {
    logger.log('会话更新:', data)
    const { conversation_id, ...updates } = data
    
    chatStore.patchConversation(conversation_id, updates)
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

  const handleNotification = (data: any) => {
    logger.log('收到通知:', data)
  }

  const handleNewNotification = (notification: any) => {
    logger.log('收到新通知:', notification)
  }

  const handleSystemMessage = (data: any) => {
    logger.log('系统消息:', data)
    const { content } = data
    
    showMessage({
      message: content,
      type: 'info',
      duration: 5000
    })
  }

  const handleUserStatusChange = (data: any) => {
    logger.log('用户状态变化:', data)
  }

  return {
    handleReadReceipt,
    handleMessageRecalled,
    handleMessageDeleted,
    handleGroupInvitation,
    handleAddedToGroup,
    handleGroupMemberLeft,
    handleGroupMemberJoined,
    handleGroupMemberRoleUpdated,
    handleGroupOwnerTransferred,
    handleConversationUpdated,
    handleGroupAnnouncementUpdated,
    handleNotification,
    handleNewNotification,
    handleSystemMessage,
    handleUserStatusChange
  }
}
