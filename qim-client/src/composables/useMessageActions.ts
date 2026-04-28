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
  const readUsersMap = ref<Record<string, { read_users: any[], total_members: number, read_count?: number }>>({})
  const showReadUsersModal = ref(false)
  const currentReadUsers = ref<{ read_users: any[], total_members: number }>({ read_users: [], total_members: 0 })
  const isMounted = ref(true)
  const lastMarkReadTime = ref(0)

  /**
   * 获取消息已读用户列表
   */
  const fetchReadUsers = async (messageId: string, forceRefresh: boolean = false) => {
    if (!isMounted.value) return { read_users: [], total_members: 0 }
    
    // 如果不是强制刷新且已有缓存，直接返回缓存数据
    if (!forceRefresh && readUsersMap.value[messageId]) {
      return readUsersMap.value[messageId]
    }

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
    // 强制刷新已读用户列表，确保显示最新数据
    const data = await fetchReadUsers(message.id, true)
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
    if (now - lastMarkReadTime.value < 2000) return
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
      const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
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
  const copyMessage = async (message: Message) => {
    if (!message || !message.content) {
      console.warn('消息内容为空')
      return
    }

    try {
      // 图片消息：复制图片本身
      if (message.type === 'image') {
        let imageUrl = message.content

        // 解析图片 URL（可能是 JSON 格式或纯 URL）
        try {
          const imageData = JSON.parse(imageUrl)
          if (imageData.url) {
            imageUrl = imageData.url
          }
        } catch {
          // 不是 JSON 格式，直接使用原始 content
        }

        // 如果是相对路径，需要拼接完整 URL
        if (!imageUrl.startsWith('http://') && !imageUrl.startsWith('https://')) {
          // 从 serverUrl 获取基础 URL
          const baseUrl = serverUrl.value.replace(/\/$/, '')
          imageUrl = `${baseUrl}/${imageUrl.replace(/^\//, '')}`
        }

        console.log('复制图片，URL:', imageUrl)

        // 使用 Image 对象加载图片
        const img = new Image()
        img.crossOrigin = 'anonymous'
        
        await new Promise((resolve, reject) => {
          img.onload = resolve
          img.onerror = reject
          img.src = imageUrl
        })

        // 创建 canvas 绘制图片
        const canvas = document.createElement('canvas')
        canvas.width = img.width
        canvas.height = img.height
        const ctx = canvas.getContext('2d')
        
        if (!ctx) {
          throw new Error('无法获取 canvas context')
        }

        ctx.drawImage(img, 0, 0)

        // 将 canvas 转换为 blob
        const blob = await new Promise<Blob>((resolve, reject) => {
          canvas.toBlob(
            (blob) => {
              if (blob) {
                resolve(blob)
              } else {
                reject(new Error('canvas 转 blob 失败'))
              }
            },
            'image/png',
            1
          )
        })

        console.log('图片 blob 类型:', blob.type, '大小:', blob.size)

        await navigator.clipboard.write([
          new ClipboardItem({
            [blob.type]: blob
          })
        ])

        console.log('图片复制成功')
      } else {
        // 其他消息：复制文本内容
        await navigator.clipboard.writeText(message.content)
      }
    } catch (err) {
      console.error('复制失败:', err)
      // 如果复制图片失败，尝试复制 URL
      if (message.type === 'image') {
        try {
          await navigator.clipboard.writeText(message.content)
          console.log('已复制图片 URL 作为备选方案')
        } catch (fallbackErr) {
          console.error('备选方案也失败:', fallbackErr)
        }
      }
    }
  }

  /**
   * 加载消息后获取已读用户列表
   */
  const loadReadUsersForMessages = async (messages: Message[], conversationType: string, forceRefresh: boolean = false) => {
    if (!isMounted.value || conversationType !== 'group') return

    const promises = messages
      .filter(message => message.isSelf)
      .filter(message => forceRefresh || !readUsersMap.value[message.id]) // 只请求需要刷新或未缓存的消息
      .map(message => fetchReadUsers(message.id, forceRefresh))

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
