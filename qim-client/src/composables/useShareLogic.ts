import { Ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { useServerUrl } from './useServerUrl'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'
import { generateAvatar, isAbsoluteUrl, getAvatarUrl } from '../utils/avatar'
import { createMergedForwardPayload } from '../utils/mergedForward'

export function useShareLogic(
  shareData: Ref<any>,
  shareType: Ref<string>,
  shareUsers: Ref<any[]>,
  shareGroups: Ref<any[]>,
  conversations: Ref<any[]>,
  currentConversationId: Ref<string | null>,
  loadConversations: () => Promise<void>,
  handleSwitchConversation: (id: string) => Promise<void>,
  closeShareModal: () => void
) {
  const chatStore = useChatStore()
  const { currentUser } = useCurrentUser()
  const { serverUrl } = useServerUrl()

  const buildFileContent = (file: any): string => {
    const downloadUrl = file.id ? `/api/v1/files/${file.id}/download` : (file.url ?? file.content ?? file.storage_path)
    return JSON.stringify({
      url: downloadUrl,
      id: file.id,
      name: file.name ?? file.original_name,
      size: file.size,
      mimeType: file.mime_type ?? file.mimeType,
    })
  }

  const loadShareUsersAndGroups = async () => {
    try {
      const orgResponse = await request('/api/v1/organization/tree')
      if (orgResponse.code === 0) {
        const users: any[] = []
        const extractUsers = (departments: any[]) => {
          departments.forEach(dept => {
            if (dept.employees) {
              dept.employees.forEach((emp: any) => {
                users.push({
                  id: emp.id.toString(),
                  name: emp.nickname || emp.username || emp.real_name,
                  username: emp.username,
                  avatar: (emp.avatar && isAbsoluteUrl(emp.avatar)) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar(emp.nickname || emp.username || emp.real_name || '员工')),
                  department: dept.name
                })
              })
            }
            if (dept.subDepartments) {
              extractUsers(dept.subDepartments)
            }
          })
        }
        extractUsers(orgResponse.data.departments)
        shareUsers.value = users
      }

      const convResponse = await request('/api/v1/conversations')
      if (convResponse.code === 0) {
        const conversationList = Array.isArray(convResponse.data) ? convResponse.data : (convResponse.data?.list || [])
        const groups = conversationList.filter((conv: any) => conv.type === 'group')
        shareGroups.value = groups.map((group: any) => ({
          id: group.id.toString(),
          name: group.name,
          avatar: getAvatarUrl(group.avatar, group.name || 'group', serverUrl.value),
          members: group.members || []
        }))
      }
    } catch (error) {
      logger.error('加载分享数据失败:', error)
      QMessage.error('加载分享数据失败')
    }
  }

  const handleShareConfirm = async (selection: any) => {
    try {
      const { users, groups } = selection
      logger.log('分享数据:', shareData.value)

      const forwardedMessages = Array.isArray(shareData.value) ? shareData.value : [shareData.value]
      const primaryShareData = forwardedMessages[0]
      let shareContent = ''
      let shareName = ''
      switch (shareType.value) {
        case 'file':
          shareContent = `分享了文件: ${shareData.value.name}`
          shareName = shareData.value.name
          break
        case 'note':
          shareContent = `分享了笔记: ${shareData.value.title}`
          shareName = shareData.value.title
          break
        case 'sticky':
          shareContent = `分享了便签: ${shareData.value.title}`
          shareName = shareData.value.title
          break
        case 'message':
          if (primaryShareData.type === 'text' || primaryShareData.type === 'markdown') {
            shareContent = `转发了消息: ${primaryShareData.content.substring(0, 20)}${primaryShareData.content.length > 20 ? '...' : ''}`
            shareName = primaryShareData.type === 'markdown' ? 'AI 消息' : '文本消息'
          } else if (primaryShareData.type === 'image') {
            shareContent = '转发了图片'
            shareName = '图片消息'
          } else {
            shareContent = '转发了消息'
            shareName = '消息'
          }
          break
        default:
          shareContent = '分享了内容'
          shareName = '内容'
      }

      const shareDataObj = {
        type: shareType.value,
        id: primaryShareData.id || primaryShareData.messageId,
        name: shareName,
        content: shareContent,
        originalContent: primaryShareData.content,
        originalMessage: shareType.value === 'message' ? primaryShareData : undefined
      }

      const buildForwardedMessage = (message: any, destination: 'user' | 'group') => {
        if (shareType.value === 'file' && message) {
          return { type: 'file', content: buildFileContent(message) }
        }

        if (shareType.value === 'message' && message) {
          if (message.type === 'text') {
            return { type: 'text', content: `[转发] ${message.content}` }
          }
          if (message.type === 'markdown' && destination === 'user') {
            return { type: 'markdown', content: `[转发] ${message.content}` }
          }
          if (message.type === 'image' || message.type === 'file' || message.type === 'miniApp' || message.type === 'share') {
            return { type: message.type, content: message.content }
          }
        }

        return { type: 'share', content: JSON.stringify(shareDataObj) }
      }

      const buildMessageData = (destination: 'user' | 'group') => shareType.value === 'message' && forwardedMessages.length > 1
        ? { type: 'merged_forward', content: JSON.stringify(createMergedForwardPayload(forwardedMessages)) }
        : buildForwardedMessage(primaryShareData, destination)

      for (const userId of users) {
        const messageData = buildMessageData('user')
        const convResponse = await request('/api/v1/conversations', {
          method: 'POST',
          body: JSON.stringify({ type: 'single', user_id: parseInt(userId) })
        })

        if (convResponse.code === 0) {
          const messageResponse = await request(`/api/v1/conversations/${convResponse.data.id}/messages`, {
            method: 'POST',
            body: JSON.stringify(messageData)
          })

          const newMessage = {
            id: messageResponse.data.id.toString(),
            content: messageData.content,
            sender: currentUser.value,
            timestamp: Date.now(),
            type: messageData.type,
            isSelf: true,
            isRead: false,
            conversationId: convResponse.data.id.toString()
          }
          chatStore.receiveMessage(convResponse.data.id.toString(), newMessage as any,
            currentConversationId.value === convResponse.data.id.toString())
        }
      }

      for (const groupId of groups) {
        const messageData = buildMessageData('group')
        const messageResponse = await request(`/api/v1/conversations/${parseInt(groupId)}/messages`, {
          method: 'POST',
          body: JSON.stringify(messageData)
        })

        const newMessage = {
          id: messageResponse.data.id.toString(),
          content: messageData.content,
          sender: currentUser.value,
          timestamp: Date.now(),
          type: messageData.type,
          isSelf: true,
          isRead: false,
          conversationId: groupId
        }
        chatStore.receiveMessage(groupId, newMessage as any, currentConversationId.value === groupId)
      }

      QMessage.success('分享成功')

      if (users.length > 0) {
        const firstUserId = users[0]
        await loadConversations()
        const conversation = conversations.value.find((conv: any) =>
          conv.type === 'single' &&
          conv.members &&
          conv.members.some((member: any) => member.id === firstUserId)
        )
        if (conversation) {
          await handleSwitchConversation(conversation.id)
        }
      } else if (groups.length > 0) {
        const firstGroupId = groups[0]?.id
        if (firstGroupId) {
          await handleSwitchConversation(firstGroupId)
        }
      }
    } catch (error) {
      logger.error('分享失败:', error)
      QMessage.error('分享失败')
    } finally {
      closeShareModal()
    }
  }

  return {
    loadShareUsersAndGroups,
    handleShareConfirm,
    buildFileContent
  }
}
