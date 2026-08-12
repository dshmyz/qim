import { ref, Ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { useCurrentUser } from './useCurrentUser'
import { useServerUrl } from './useServerUrl'
import { request } from './useRequest'
import { logger } from '../utils/logger'
import QMessage from '../utils/qmessage'
import { generateAvatar, isAbsoluteUrl, getAvatarUrl } from '../utils/avatar'
import { createMergedForwardPayload } from '../utils/mergedForward'

// ── 模块级缓存（5 分钟 TTL） ──
const CACHE_TTL_MS = 5 * 60 * 1000
let cachedDepartments: any[] | null = null
let cachedUsers: any[] | null = null
let cachedGroups: any[] | null = null
let cacheTimestamp = 0

export interface ShareSendResult {
  total: number
  success: number
  failed: number
  failedTargets: string[]
}

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

  // ── 加载 & 发送状态 ──
  const shareLoading = ref(false)
  const shareDepartments = ref<any[]>([])
  const sending = ref(false)
  const sendingProgress = ref({ sent: 0, total: 0 })

  const buildFileContent = (file: any): string => {
    let contentUrl: string | undefined
    if (typeof file.content === 'string') {
      try {
        const content = JSON.parse(file.content)
        contentUrl = typeof content?.url === 'string' ? content.url : undefined
      } catch {
        contentUrl = file.content
      }
    }
    const originalUrl = file.url ?? contentUrl ?? file.storage_path
    const downloadUrl = typeof originalUrl === 'string' && originalUrl
      ? (originalUrl.startsWith('http') || originalUrl.startsWith('/') ? originalUrl : `/${originalUrl}`)
      : (file.id ? `/api/v1/files/${file.id}/download` : '')
    return JSON.stringify({
      url: downloadUrl,
      id: file.id,
      name: file.name ?? file.original_name,
      size: file.size,
      mimeType: file.mime_type ?? file.mimeType,
    })
  }

  // ── 构建部门树 ──
  const buildDepartmentsTree = (subDepartments: any[]): any[] => {
    if (!subDepartments || !Array.isArray(subDepartments)) return []
    return subDepartments.map(dept => ({
      id: dept.id || dept.name,
      name: dept.name,
      employees: (dept.employees || []).map((emp: any) => ({
        id: emp.id.toString(),
        name: emp.nickname || emp.username || emp.real_name,
        username: emp.username,
        avatar: (emp.avatar && isAbsoluteUrl(emp.avatar)) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar(emp.nickname || emp.username || emp.real_name || '员工')),
        department: dept.name
      })),
      subDepartments: buildDepartmentsTree(dept.subDepartments)
    }))
  }

  // ── 加载联系人（带缓存） ──
  const loadShareUsersAndGroups = async () => {
    const now = Date.now()
    if (cachedDepartments && cachedUsers && cachedGroups && now - cacheTimestamp < CACHE_TTL_MS) {
      shareUsers.value = cachedUsers
      shareGroups.value = cachedGroups
      return
    }

    shareLoading.value = true
    try {
      const orgResponse = await request('/api/v1/organization/tree')
      if (orgResponse.code === 0) {
        // 保留部门层级结构
        const departments = (orgResponse.data.departments || []).map((dept: any) => ({
          id: dept.id || dept.name,
          name: dept.name,
          employees: (dept.employees || []).map((emp: any) => ({
            id: emp.id.toString(),
            name: emp.nickname || emp.username || emp.real_name,
            username: emp.username,
            avatar: (emp.avatar && isAbsoluteUrl(emp.avatar)) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : generateAvatar(emp.nickname || emp.username || emp.real_name || '员工')),
            department: dept.name
          })),
          subDepartments: buildDepartmentsTree(dept.subDepartments)
        }))
        shareDepartments.value = departments
        cachedDepartments = departments

        // 同时保留扁平列表（供 handleShareConfirm 失败回溯用）
        const users: any[] = []
        const flattenUsers = (depts: any[]) => {
          depts.forEach(d => {
            (d.employees || []).forEach((emp: any) => users.push(emp))
            if (d.subDepartments?.length > 0) flattenUsers(d.subDepartments)
          })
        }
        flattenUsers(departments)
        shareUsers.value = users
        cachedUsers = users
      }

      const convResponse = await request('/api/v1/conversations?type=group&page_size=100')
      if (convResponse.code === 0) {
        const conversationList = Array.isArray(convResponse.data) ? convResponse.data : (convResponse.data?.list || [])
        const groups = conversationList.filter((conv: any) => conv.type === 'group')
        shareGroups.value = groups.map((group: any) => ({
          id: group.id.toString(),
          name: group.name,
          avatar: getAvatarUrl(group.avatar, group.name || 'group', serverUrl.value),
          members: group.members || []
        }))
        cachedGroups = shareGroups.value
      }

      cacheTimestamp = Date.now()
    } catch (error) {
      logger.error('加载分享数据失败:', error)
      QMessage.error('加载分享数据失败')
    } finally {
      shareLoading.value = false
    }
  }

  // ── 清除缓存（如需强制刷新） ──
  const invalidateShareCache = () => {
    cachedDepartments = null
    cachedUsers = null
    cachedGroups = null
    cacheTimestamp = 0
  }

  // ── 单个目标发送 ──
  const sendToUser = async (userId: string, messageData: any) => {
    const convResponse = await request('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({ type: 'single', user_id: parseInt(userId) })
    })

    if (convResponse.code !== 0) throw new Error('创建会话失败')

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

    return { targetId: userId, targetName: '' }
  }

  const sendToGroup = async (groupId: string, messageData: any) => {
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

    const group = shareGroups.value.find(g => g.id === groupId)
    return { targetId: groupId, targetName: group?.name || '' }
  }

  // ── 确认分享/转发 ──
  const handleShareConfirm = async (selection: any): Promise<ShareSendResult> => {
    try {
      const { users, groups } = selection
      logger.log('分享数据:', shareData.value)

      const forwardedMessages = Array.isArray(shareData.value)
        ? shareData.value
        : Array.isArray(shareData.value?.messages)
          ? shareData.value.messages
          : [shareData.value]
      const sourceTitle = typeof shareData.value?.sourceTitle === 'string' ? shareData.value.sourceTitle : undefined
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
          if (message.type === 'image' || message.type === 'file' || message.type === 'miniApp' || message.type === 'share' || message.type === 'merged_forward') {
            return { type: message.type, content: message.content }
          }
        }

        return { type: 'share', content: JSON.stringify(shareDataObj) }
      }

      const buildMessageData = (destination: 'user' | 'group') => shareType.value === 'message' && forwardedMessages.length > 1
        ? { type: 'merged_forward', content: JSON.stringify(createMergedForwardPayload(forwardedMessages, sourceTitle)) }
        : buildForwardedMessage(primaryShareData, destination)

      // ── 并行发送（批次控制，每批5个） ──
      const totalTargets = users.length + groups.length
      sending.value = true
      sendingProgress.value = { sent: 0, total: totalTargets }

      const BATCH_SIZE = 5
      const allTargets: { type: 'user' | 'group'; id: string }[] = [
        ...users.map((id: string) => ({ type: 'user' as const, id })),
        ...groups.map((id: string) => ({ type: 'group' as const, id })),
      ]

      const results: PromiseSettledResult<any>[] = []
      for (let i = 0; i < allTargets.length; i += BATCH_SIZE) {
        const batch = allTargets.slice(i, i + BATCH_SIZE)
        const batchResults = await Promise.allSettled(
          batch.map(t =>
            (t.type === 'user' ? sendToUser(t.id, buildMessageData('user')) : sendToGroup(t.id, buildMessageData('group')))
              .then(r => {
                sendingProgress.value = { ...sendingProgress.value, sent: sendingProgress.value.sent + 1 }
                return r
              })
          )
        )
        results.push(...batchResults)
      }

      const succeeded = results.filter(r => r.status === 'fulfilled')
      const failed = results.filter(r => r.status === 'rejected')
      const failedTargets: string[] = []

      results.forEach((r, i) => {
        if (r.status === 'rejected') {
          const targetName = i < users.length
            ? (shareUsers.value.find(u => u.id === users[i])?.name || users[i])
            : (shareGroups.value.find(g => g.id === groups[i - users.length])?.name || groups[i - users.length])
          failedTargets.push(targetName)
        }
      })

      const result: ShareSendResult = {
        total: totalTargets,
        success: succeeded.length,
        failed: failed.length,
        failedTargets
      }

      // ── 结果反馈 ──
      if (result.failed === 0) {
        QMessage.success('发送成功')
      } else if (result.success === 0) {
        QMessage.error('发送失败')
      } else {
        QMessage.warning(`已发送 ${result.success} 条，${result.failed} 条失败`)
      }

      // ── 发送后跳转到第一个成功的目标 ──
      if (users.length > 0 && succeeded.length > 0) {
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
      } else if (groups.length > 0 && succeeded.length > 0) {
        const firstGroupId = groups[0]?.id
        if (firstGroupId) {
          await handleSwitchConversation(firstGroupId)
        }
      }

      return result
    } catch (error) {
      logger.error('分享失败:', error)
      QMessage.error('分享失败')
      return { total: 0, success: 0, failed: 0, failedTargets: [] }
    } finally {
      sending.value = false
      sendingProgress.value = { sent: 0, total: 0 }
      closeShareModal()
    }
  }

  return {
    shareLoading,
    shareDepartments,
    sending,
    sendingProgress,
    loadShareUsersAndGroups,
    handleShareConfirm,
    buildFileContent,
    invalidateShareCache
  }
}
