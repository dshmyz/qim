import { ref } from 'vue'
import QMessage from '../utils/qmessage'

export function useChannel(serverUrl: any, currentUser: any) {
  const channelMessage = ref('')

  const isChannelCreator = (channel: any) => {
    if (!currentUser.value?.id || !channel.creator_id) return false
    return currentUser.value.id === channel.creator_id || currentUser.value.id.toString() === channel.creator_id.toString()
  }

  const subscribeChannel = async (channel: any) => {
    try {
      const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/subscribe`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      })
      
      const data = await response.json()
      if (data.code === 0) {
        channel.is_subscribed = true
        QMessage.success('订阅成功')
      } else {
        QMessage.error(data.message || '订阅失败')
      }
    } catch (error) {
      console.error('订阅频道失败:', error)
      QMessage.error('订阅失败')
    }
  }

  const unsubscribeChannel = async (channel: any) => {
    try {
      const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/unsubscribe`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      })
      
      const data = await response.json()
      if (data.code === 0) {
        channel.is_subscribed = false
        QMessage.success('取消订阅成功')
      } else {
        QMessage.error(data.message || '取消订阅失败')
      }
    } catch (error) {
      console.error('取消订阅频道失败:', error)
      QMessage.error('取消订阅失败')
    }
  }

  const sendChannelMessage = async (channel: any, message?: string) => {
    const content = message || channelMessage.value
    if (!content?.trim()) return
    
    try {
      const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          content: content
        })
      })
      
      const data = await response.json()
      if (data.code === 0) {
        const newMessage = data.data
        if (!channel.messages) {
          channel.messages = []
        }
        channel.messages.push(newMessage)
        channelMessage.value = ''
        QMessage.success('发送成功')
      } else {
        QMessage.error(data.message || '发送失败')
      }
    } catch (error) {
      console.error('发送频道消息失败:', error)
      QMessage.error('发送失败')
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
