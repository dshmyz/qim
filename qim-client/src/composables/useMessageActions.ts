import { ref } from 'vue'
import type { Message } from '../types'
import { request } from './useRequest'

/**
 * 消息操作相关逻辑
 * 包含：消息已读、撤回、删除、转发、复制、已读回执等
 */
export function useMessageActions(
  serverUrl: { value: string },
  currentUser: { value: any }
) {
  const readUsersMap = ref<Record<string, { read_users: any[], total_members: number }>>({})
  const showReadUsersModal = ref(false)
  const currentReadUsers = ref<{ read_users: any[], total_members: number }>({ read_users: [], total_members: 0 })
  const isMounted = ref(true)
  const lastMarkReadTime = ref(0)

  /**
   * 获取消息已读用户列表
   */
  const fetchReadUsers = async (messageId: string) => {
    if (!isMounted.value) return { read_users: [], total_members: 0 }

    try {
      const response = await request(`/api/v1/messages/${messageId}/read-users`, {
        method: 'GET'
      })

      if (response.code === 0) {
        const data = response.data
        // 对已读用户列表进行去重处理
        const uniqueReadUsers: any[] = []
        const seenUserIds = new Set<string>()

        if (data.read_users) {
          for (const user of data.read_users) {
            if (user.id && !seenUserIds.has(user.id)) {
              seenUserIds.add(user.id)
              uniqueReadUsers.push(user)
            }
          }
        }

        readUsersMap.value[messageId] = {
          ...data,
          read_users: uniqueReadUsers
        }
        return data
      }
    } catch (error) {
      console.error('获取已读用户列表失败:', error)
    }
    return { read_users: [], total_members: 0 }
  }

  /**
   * 显示已读用户列表弹窗
   */
  const showReadUsers = async (message: Message) => {
    if (!message.isSelf || !isMounted.value) return
    const data = await fetchReadUsers(message.id)
    if (isMounted.value) {
      currentReadUsers.value = data
      showReadUsersModal.value = true
    }
  }

  /**
   * 标记消息为已读
   */
  const markMessagesAsRead = async (conversationId: string) => {
    if (!conversationId) return

    // 限制调用频率，避免短时间内重复调用
    const now = Date.now()
    if (now - lastMarkReadTime.value < 3000) return
    lastMarkReadTime.value = now

    try {
      const response = await request(`/api/v1/conversations/${conversationId}/read`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })

      if (response.code === 0) {
        // 标记成功，可以触发后续操作
      }
    } catch (error) {
      console.error('标记消息已读失败:', error)
    }
  }

  /**
   * 撤回消息
   */
  const recallMessage = async (conversationId: string, messageId: string) => {
    try {
      const response = await request(`/api/v1/messages/${messageId}/recall`, {
        method: 'POST'
      })
      if (response.code === 0) {
        return { success: true }
      }
      return { success: false, message: response.message || '撤回失败' }
    } catch (error) {
      console.error('撤回消息失败:', error)
      return { success: false, message: '撤回失败' }
    }
  }

  /**
   * 删除消息
   */
  const deleteMessage = async (conversationId: string, messageId: string) => {
    try {
      const response = await request(`/api/v1/messages/${messageId}`, {
        method: 'DELETE'
      })
      if (response.code === 0) {
        return { success: true }
      }
      return { success: false, message: response.message || '删除失败' }
    } catch (error) {
      console.error('删除消息失败:', error)
      return { success: false, message: '删除失败' }
    }
  }

  /**
   * 发送消息
   */
  const sendMessage = async (conversationId: string, content: string, type = 'text', extraData?: Record<string, any>) => {
    try {
      const response = await request('/api/v1/messages', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          conversationId,
          content,
          type,
          ...extraData
        })
      })
      if (response.code === 0) {
        return { success: true, data: response.data }
      }
      return { success: false, message: response.message || '发送失败' }
    } catch (error) {
      console.error('发送消息失败:', error)
      return { success: false, message: '发送失败' }
    }
  }

  /**
   * 重新发送消息
   */
  const retrySendMessage = async (message: Message) => {
    if (!message) return { success: false, message: '消息为空' }
    return sendMessage(
      message.conversationId,
      message.content,
      message.type
    )
  }

  /**
   * 复制消息内容
   */
  const copyMessage = (message: Message) => {
    if (message && message.content) {
      navigator.clipboard.writeText(message.content).catch((err) => {
        console.error('复制失败:', err)
      })
    }
  }

  /**
   * 加载消息后获取已读用户列表
   */
  const loadReadUsersForMessages = async (messages: Message[], conversationType: string) => {
    if (!isMounted.value || conversationType !== 'group') return

    const promises = messages
      .filter(message => message.isSelf)
      .map(message => fetchReadUsers(message.id))

    await Promise.all(promises)
  }

  /**
   * 清理资源
   */
  const cleanup = () => {
    isMounted.value = false
  }

  return {
    // 状态
    readUsersMap,
    showReadUsersModal,
    currentReadUsers,
    isMounted,
    lastMarkReadTime,
    // 方法
    fetchReadUsers,
    showReadUsers,
    markMessagesAsRead,
    recallMessage,
    deleteMessage,
    sendMessage,
    retrySendMessage,
    copyMessage,
    loadReadUsersForMessages,
    cleanup
  }
}
