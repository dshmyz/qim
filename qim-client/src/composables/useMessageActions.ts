import { ref } from 'vue'
import { storeToRefs } from 'pinia'
import QMessage from '../utils/qmessage'
import type { Message } from '../types'
import { request } from './useRequest'
import { useUIStore } from '../stores/ui'
import { decodeToPlainText } from '../utils/mentions'

function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  return (...args: Parameters<T>) => {
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    timeoutId = setTimeout(() => {
      func(...args)
    }, wait)
  }
}

// 全局节流变量，确保所有实例共享同一个节流状态
const lastMarkReadTime = ref(0)

export function useMessageActions(
  serverUrl: { value: string },
  currentUser: { value: any }
) {
  const uiStore = useUIStore()
  const { showReadUsersModal } = storeToRefs(uiStore)

  const readUsersMap = ref<Record<string, { read_users: any[], total_members: number, read_count?: number }>>({})
  const currentReadUsers = ref<{ read_users: any[], total_members: number }>({ read_users: [], total_members: 0 })
  const isMounted = ref(true)

  const fetchReadUsers = async (messageId: string, forceRefresh: boolean = false) => {
    if (!isMounted.value) return { read_users: [], total_members: 0 }
    
    if (!forceRefresh && readUsersMap.value[messageId]) {
      return readUsersMap.value[messageId]
    }

    try {
      const response = await request(`/api/v1/messages/${messageId}/read-users`, {
        method: 'GET'
      })

      if (!isMounted.value) return { read_users: [], total_members: 0 }

      if (response.code === 0) {
        const data = response.data
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

        if (isMounted.value) {
          readUsersMap.value[messageId] = {
            ...data,
            read_users: uniqueReadUsers
          }
        }
        return data
      }
    } catch (error) {
      if (isMounted.value) {
        console.error('获取已读用户列表失败:', error)
      }
    }
    return { read_users: [], total_members: 0 }
  }

  const batchFetchReadUsers = async (messageIds: string[], forceRefresh: boolean = false) => {
    if (!isMounted.value || messageIds.length === 0) return

    const idsToFetch = forceRefresh 
      ? messageIds 
      : messageIds.filter(id => !readUsersMap.value[id])

    if (idsToFetch.length === 0) return

    try {
      const response = await request('/api/v1/messages/batch/read-users', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          message_ids: idsToFetch.map(id => parseInt(id))
        })
      })

      if (!isMounted.value) return

      if (response.code === 0) {
        const data = response.data
        
        for (const [messageId, readData] of Object.entries(data)) {
          if (!isMounted.value) return
          
          const typedData = readData as { read_users: any[], total_members: number, read_count: number }
          const uniqueReadUsers: any[] = []
          const seenUserIds = new Set<string>()

          if (typedData.read_users) {
            for (const user of typedData.read_users) {
              if (user.id && !seenUserIds.has(user.id)) {
                seenUserIds.add(user.id)
                uniqueReadUsers.push(user)
              }
            }
          }

          readUsersMap.value[messageId] = {
            ...typedData,
            read_users: uniqueReadUsers
          }
        }
      }
    } catch (error) {
      if (isMounted.value) {
        console.error('批量获取已读用户列表失败:', error)
      }
    }
  }

  /**
   * 显示已读用户列表弹窗
   */
  const showReadUsers = async (message: Message) => {
    if (!message.isSelf || !isMounted.value) return
    try {
      // 强制刷新已读用户列表，确保显示最新数据
      const data = await fetchReadUsers(message.id, true)
      if (isMounted.value) {
        currentReadUsers.value = data
        showReadUsersModal.value = true
      }
    } catch (error) {
      console.error('显示已读用户列表失败:', error)
    }
  }

  /**
   * 标记消息为已读
   */
  const markMessagesAsRead = async (conversationId: string) => {
    console.log('[useMessageActions] markMessagesAsRead 被调用', {
      conversationId,
      isMounted: isMounted.value,
      timeSinceLastCall: Date.now() - lastMarkReadTime.value
    })
    
    if (!conversationId || !isMounted.value) {
      console.log('[useMessageActions] markMessagesAsRead 提前返回', {
        reason: !conversationId ? 'conversationId 为空' : '组件未挂载'
      })
      return
    }

    const now = Date.now()
    if (now - lastMarkReadTime.value < 1000) {
      console.log('[useMessageActions] markMessagesAsRead 被节流', {
        timeSinceLastCall: now - lastMarkReadTime.value,
        threshold: 1000
      })
      return
    }
    lastMarkReadTime.value = now

    console.log('[useMessageActions] 发送标记已读请求', {
      conversationId,
      url: `/api/v1/conversations/${conversationId}/read`
    })

    try {
      const response = await request(`/api/v1/conversations/${conversationId}/read`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })

      if (!isMounted.value) {
        console.log('[useMessageActions] 请求返回时组件已卸载')
        return
      }

      if (response.code === 0) {
        console.log('[useMessageActions] 标记已读成功', { conversationId })
      }
    } catch (error) {
      if (isMounted.value) {
        console.error('[useMessageActions] 标记消息已读失败:', error)
      }
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
      const message = error instanceof Error ? error.message : '撤回失败'
      return { success: false, message }
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

        QMessage.success('图片已复制')
      } else {
        // 其他消息：复制文本内容
        await navigator.clipboard.writeText(decodeToPlainText(message.content))
        QMessage.success('已复制')
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

  const debouncedLoadReadUsers = debounce(
    async (messages: Message[], conversationType: string, forceRefresh: boolean = false) => {
      if (!isMounted.value || conversationType !== 'group') return

      const messagesToLoad = messages
        .filter(message => message.isSelf)
        .filter(message => forceRefresh || !readUsersMap.value[message.id])

      if (messagesToLoad.length === 0) return

      const messageIds = messagesToLoad.map(message => message.id)
      await batchFetchReadUsers(messageIds, forceRefresh)
    },
    500
  )

  const loadReadUsersForMessages = async (messages: Message[], conversationType: string, forceRefresh: boolean = false) => {
    if (!isMounted.value || conversationType !== 'group') return

    const messagesToLoad = messages
      .filter(message => message.isSelf)
      .filter(message => forceRefresh || !readUsersMap.value[message.id])

    if (messagesToLoad.length === 0) return

    const messageIds = messagesToLoad.map(message => message.id)
    await batchFetchReadUsers(messageIds, forceRefresh)
  }

  /**
   * 清理资源
   */
  const cleanup = () => {
    isMounted.value = false
  }

  return {
    readUsersMap,
    showReadUsersModal,
    currentReadUsers,
    isMounted,
    lastMarkReadTime,
    fetchReadUsers,
    batchFetchReadUsers,
    showReadUsers,
    markMessagesAsRead,
    recallMessage,
    deleteMessage,
    sendMessage,
    retrySendMessage,
    copyMessage,
    loadReadUsersForMessages,
    debouncedLoadReadUsers,
    cleanup
  }
}
