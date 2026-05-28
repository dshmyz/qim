import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Channel, ChannelMessage } from '../types'
import { request, type ApiResponse } from '../composables/useRequest'
import QMessage from '../utils/qmessage'
import { getCurrentUser } from '../utils/user'

export const useChannelStore = defineStore('channel', () => {
  const channels = ref<Channel[]>([])
  const selectedChannelId = ref<string | null>(null)
  const openTabs = ref<Array<{ id: string; name: string }>>([])

  const getStoredViewMode = (): 'list' | 'card' => {
    const stored = localStorage.getItem('channel-viewMode')
    return (stored === 'list' || stored === 'card') ? stored : 'card'
  }
  const getStoredMessageMode = (): 'card' | 'timeline' => {
    const stored = localStorage.getItem('channel-messageMode')
    return (stored === 'card' || stored === 'timeline') ? stored : 'card'
  }

  const viewMode = ref<'list' | 'card'>(getStoredViewMode())
  const messageMode = ref<'card' | 'timeline'>(getStoredMessageMode())
  const loading = ref(false)
  const messagesLoading = ref(false)

  const selectedChannel = computed(() => {
    return channels.value.find(c => c.id === selectedChannelId.value) || null
  })

  const subscribedChannels = computed(() => {
    return channels.value.filter(c => c.is_subscribed)
  })

  const totalUnreadCount = computed(() => {
    return channels.value.reduce((sum, c) => sum + (c.unread_count || 0), 0)
  })

  async function fetchChannels() {
    loading.value = true
    try {
      const response = await request<ApiResponse<Channel[]>>('/api/v1/channels')
      if (response.code === 0) {
        channels.value = response.data || []
      } else {
        QMessage.error(response.message || '加载频道失败')
      }
    } catch (error) {
      console.error('加载频道失败:', error)
      QMessage.error('加载频道失败')
    } finally {
      loading.value = false
    }
  }

  async function subscribeChannel(channelId: string) {
    try {
      const response = await request<ApiResponse<void>>(`/api/v1/channels/${channelId}/subscribe`, {
        method: 'POST'
      })
      if (response.code === 0) {
        const channel = channels.value.find(c => c.id === channelId)
        if (channel) {
          channel.is_subscribed = true
        }
        QMessage.success('订阅成功')
      } else {
        QMessage.error(response.message || '订阅失败')
      }
    } catch (error) {
      console.error('订阅频道失败:', error)
      QMessage.error('订阅失败')
    }
  }

  async function unsubscribeChannel(channelId: string) {
    try {
      const response = await request<ApiResponse<void>>(`/api/v1/channels/${channelId}/subscribe`, {
        method: 'DELETE'
      })
      if (response.code === 0) {
        const channel = channels.value.find(c => c.id === channelId)
        if (channel) {
          channel.is_subscribed = false
        }
        QMessage.success('取消订阅成功')
      } else {
        QMessage.error(response.message || '取消订阅失败')
      }
    } catch (error) {
      console.error('取消订阅频道失败:', error)
      QMessage.error('取消订阅失败')
    }
  }

  async function fetchChannelMessages(channelId: string) {
    messagesLoading.value = true
    try {
      const response = await request<ApiResponse<ChannelMessage[]>>(`/api/v1/channels/${channelId}/messages`)
      if (response.code === 0) {
        const channel = channels.value.find(c => c.id === channelId)
        if (channel) {
          channel.messages = response.data || []
        }
      } else {
        QMessage.error(response.message || '加载频道消息失败')
      }
    } catch (error) {
      console.error('加载频道消息失败:', error)
      QMessage.error('加载频道消息失败')
    } finally {
      messagesLoading.value = false
    }
  }

  async function selectChannel(channelId: string) {
    selectedChannelId.value = channelId
    const channel = channels.value.find(c => c.id === channelId)
    if (channel) {
      addTab(channel)
      if (!channel.messages || channel.messages.length === 0) {
        await fetchChannelMessages(channelId)
      }
      markChannelRead(channelId)
    }
  }

  function markChannelRead(channelId: string) {
    const channel = channels.value.find(c => c.id === channelId)
    if (channel) {
      channel.unread_count = 0
    }
  }

  function incrementUnread(channelId: string) {
    const channel = channels.value.find(c => c.id === channelId)
    if (channel) {
      channel.unread_count = (channel.unread_count || 0) + 1
    }
  }

  function addTab(channel: Channel) {
    const exists = openTabs.value.find(t => t.id === channel.id)
    if (!exists) {
      openTabs.value.push({ id: channel.id, name: channel.name })
    }
  }

  function removeTab(channelId: string) {
    const index = openTabs.value.findIndex(t => t.id === channelId)
    if (index > -1) {
      openTabs.value.splice(index, 1)
      if (selectedChannelId.value === channelId) {
        selectedChannelId.value = openTabs.value.length > 0 ? openTabs.value[0].id : null
      }
    }
  }

  function isChannelCreator(channel: Channel): boolean {
    const currentUser = getCurrentUser()
    console.log('=== isChannelCreator 调试信息 ===')
    console.log('currentUser:', currentUser ? '存在' : '不存在')
    console.log('channel.creator_id:', channel.creator_id, typeof channel.creator_id)
    
    if (!currentUser || !currentUser.id || !channel.creator_id) {
      console.log('返回 false: currentUser、currentUser.id 或 creator_id 不存在')
      return false
    }
    
    const result = String(currentUser.id) === String(channel.creator_id)
    console.log('比较结果:', String(currentUser.id), '===', String(channel.creator_id), '=', result)
    console.log('=== 调试信息结束 ===')
    return result
  }

  async function sendChannelMessage(channel: Channel, message: string) {
    if (!message?.trim()) return
    try {
      const response = await request<ApiResponse<ChannelMessage>>(`/api/v1/channels/${channel.id}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: message.trim() })
      })
      if (response.code === 0) {
        const newMessage = response.data
        if (newMessage) {
          if (!channel.messages) {
            channel.messages = []
          }
          channel.messages.push(newMessage)
        }
        QMessage.success('发送成功')
      } else {
        QMessage.error(response.message || '发送失败')
      }
    } catch (error) {
      console.error('发送频道消息失败:', error)
      QMessage.error('发送失败')
    }
  }

  function setViewMode(mode: 'list' | 'card') {
    viewMode.value = mode
    localStorage.setItem('channel-viewMode', mode)
  }

  function setMessageMode(mode: 'card' | 'timeline') {
    messageMode.value = mode
    localStorage.setItem('channel-messageMode', mode)
  }

  return {
    channels,
    selectedChannelId,
    openTabs,
    viewMode,
    messageMode,
    loading,
    messagesLoading,
    selectedChannel,
    subscribedChannels,
    totalUnreadCount,
    fetchChannels,
    fetchChannelMessages,
    subscribeChannel,
    unsubscribeChannel,
    selectChannel,
    addTab,
    removeTab,
    setViewMode,
    setMessageMode,
    markChannelRead,
    incrementUnread,
    isChannelCreator,
    sendChannelMessage,
  }
})
