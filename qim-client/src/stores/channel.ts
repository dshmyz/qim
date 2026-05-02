import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Channel } from '../types'
import { request, type ApiResponse } from '../composables/useRequest'
import QMessage from '../utils/qmessage'

export const useChannelStore = defineStore('channel', () => {
  // 状态
  const channels = ref<Channel[]>([])
  const selectedChannelId = ref<string | null>(null)
  const openTabs = ref<Array<{ id: string; name: string }>>([])

  // 初始化持久化的状态
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

  // 计算属性
  const selectedChannel = computed(() => {
    return channels.value.find(c => c.id === selectedChannelId.value) || null
  })

  const subscribedChannels = computed(() => {
    return channels.value.filter(c => c.is_subscribed)
  })

  // 方法
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
      const response = await request<ApiResponse<void>>(`/api/v1/channels/${channelId}/unsubscribe`, {
        method: 'POST'
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

  function selectChannel(channelId: string) {
    selectedChannelId.value = channelId
    const channel = channels.value.find(c => c.id === channelId)
    if (channel) {
      addTab(channel)
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

  function setViewMode(mode: 'list' | 'card') {
    viewMode.value = mode
    localStorage.setItem('channel-viewMode', mode)
  }

  function setMessageMode(mode: 'card' | 'timeline') {
    messageMode.value = mode
    localStorage.setItem('channel-messageMode', mode)
  }

  return {
    // 状态
    channels,
    selectedChannelId,
    openTabs,
    viewMode,
    messageMode,
    loading,
    // 计算属性
    selectedChannel,
    subscribedChannels,
    // 方法
    fetchChannels,
    subscribeChannel,
    unsubscribeChannel,
    selectChannel,
    addTab,
    removeTab,
    setViewMode,
    setMessageMode,
  }
})
