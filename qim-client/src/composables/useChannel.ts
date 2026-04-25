import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { request, getToken } from '@/utils/request'

export function useChannel(serverUrl: any, currentUser: any) {
  const channelMessage = ref('')

  const isChannelCreator = (channel: any) => {
    if (!currentUser.value?.id || !channel.creator_id) return false
    return currentUser.value.id === channel.creator_id || currentUser.value.id.toString() === channel.creator_id.toString()
  }

  const subscribeChannel = async (channel: any) => {
    try {
      const response: any = await request(`/api/v1/channels/${channel.id}/subscribe`, {
        method: 'POST',
        customBaseUrl: serverUrl.value
      })
      
      if (response.code === 0) {
        channel.is_subscribed = true
        ElMessage.success('订阅成功')
      } else {
        ElMessage.error(response.message || '订阅失败')
      }
    } catch (error) {
      console.error('订阅频道失败:', error)
      ElMessage.error('订阅失败')
    }
  }

  const unsubscribeChannel = async (channel: any) => {
    try {
      const response: any = await request('/api/v1/channels/${channel.id}/unsubscribe', {
        method: 'POST',
        customBaseUrl: serverUrl.value
      })
      
      if (response.code === 0) {
        channel.is_subscribed = false
        ElMessage.success('取消订阅成功')
      } else {
        ElMessage.error(response.message || '取消订阅失败')
      }
    } catch (error) {
      console.error('取消订阅频道失败:', error)
      ElMessage.error('取消订阅失败')
    }
  }

  const sendChannelMessage = async (channel: any) => {
    if (!channelMessage.value.trim()) return
    
    try {
      const response: any = await request('/api/v1/channels/${channel.id}/messages', {
        method: 'POST',
        data: {
          content: channelMessage.value
        },
        customBaseUrl: serverUrl.value
      })
      
      if (response.code === 0) {
        const newMessage = response.data
        if (!channel.messages) {
          channel.messages = []
        }
        channel.messages.push(newMessage)
        channelMessage.value = ''
        ElMessage.success('发送成功')
      } else {
        ElMessage.error(response.message || '发送失败')
      }
    } catch (error) {
      console.error('发送频道消息失败:', error)
      ElMessage.error('发送失败')
    }
  }

  return {
    channelMessage,
    isChannelCreator,
    subscribeChannel,
    unsubscribeChannel,
    sendChannelMessage
  }
}
