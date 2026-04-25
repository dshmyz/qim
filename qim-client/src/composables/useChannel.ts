import { ref } from 'vue'
import { ElMessage } from 'element-plus'

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
      
      if (response.ok) {
        channel.is_subscribed = true
        ElMessage.success('订阅成功')
      } else {
        ElMessage.error('订阅失败')
      }
    } catch (error) {
      console.error('订阅频道失败:', error)
      ElMessage.error('订阅失败')
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
      
      if (response.ok) {
        channel.is_subscribed = false
        ElMessage.success('取消订阅成功')
      } else {
        ElMessage.error('取消订阅失败')
      }
    } catch (error) {
      console.error('取消订阅频道失败:', error)
      ElMessage.error('取消订阅失败')
    }
  }

  const sendChannelMessage = async (channel: any) => {
    if (!channelMessage.value.trim()) return
    
    try {
      const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          content: channelMessage.value
        })
      })
      
      if (response.ok) {
        const newMessage = await response.json()
        if (!channel.messages) {
          channel.messages = []
        }
        channel.messages.push(newMessage)
        channelMessage.value = ''
        ElMessage.success('发送成功')
      } else {
        ElMessage.error('发送失败')
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
