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
  // 待定位的消息 id：通知中心/深链指定要滚动到并高亮的某条频道消息，渲染层消费后清除
  const pendingMessageId = ref<string | null>(null)

  const getStoredViewMode = (): 'list' | 'card' => {
    const stored = localStorage.getItem('channel-viewMode')
    return (stored === 'list' || stored === 'card') ? stored : 'card'
  }

  const viewMode = ref<'list' | 'card'>(getStoredViewMode())
  const loading = ref(false)
  const messagesLoading = ref(false)

  // 频道 id 类型可能不一致：后端返回数字（如 2），而深链/通知导航把 channel_id 转成了字符串（"2"）。
  // 统一按字符串比较，避免 `2 === "2"` 恒 false 导致 selectedChannel 找不到、频道详情一直空白。
  // （与 isChannelCreator 的 String(...)===String(...) 约定一致。）
  const matchId = (a: any, b: any) => String(a) === String(b)

  const selectedChannel = computed(() => {
    return channels.value.find(c => matchId(c.id, selectedChannelId.value)) || null
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
        const channel = channels.value.find(c => matchId(c.id, channelId))
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
        const channel = channels.value.find(c => matchId(c.id, channelId))
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
        const channel = channels.value.find(c => matchId(c.id, channelId))
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

  async function selectChannel(channelId: string, messageId?: string) {
    selectedChannelId.value = channelId
    // 深链定位：记录待定位消息，等消息加载完成由渲染层滚动+高亮后消费清除
    pendingMessageId.value = messageId || null
    let channel = channels.value.find(c => matchId(c.id, channelId))
    // 通知中心/深链可能在任何时刻触发：若频道列表尚未加载（例如用户从未打开频道 Tab，
    // 或首次启动即点通知），selectChannel 会因找不到频道而静默失败、无法跳转。
    // 因此在继续前先确保频道列表已加载，再做定位。
    if (!channel) {
      await fetchChannels()
      channel = channels.value.find(c => matchId(c.id, channelId))
    }
    if (channel) {
      addTab(channel)
      await fetchChannelMessages(channelId)
      markChannelRead(channelId)
    }
  }

  function clearPendingMessageId() {
    pendingMessageId.value = null
  }

  function markChannelRead(channelId: string) {
    const channel = channels.value.find(c => matchId(c.id, channelId))
    if (channel) {
      channel.unread_count = 0
    }
  }

  function incrementUnread(channelId: string) {
    const channel = channels.value.find(c => matchId(c.id, channelId))
    if (channel) {
      channel.unread_count = (channel.unread_count || 0) + 1
    }
  }

  function addTab(channel: Channel) {
    const exists = openTabs.value.find(t => matchId(t.id, channel.id))
    if (!exists) {
      openTabs.value.push({ id: channel.id, name: channel.name })
    }
  }

  function removeTab(channelId: string) {
    const index = openTabs.value.findIndex(t => matchId(t.id, channelId))
    if (index > -1) {
      openTabs.value.splice(index, 1)
      if (matchId(selectedChannelId.value, channelId)) {
        selectedChannelId.value = openTabs.value.length > 0 ? openTabs.value[0].id : null
      }
    }
  }

  function isChannelCreator(channel: Channel): boolean {
    const currentUser = getCurrentUser()
    if (!currentUser || !currentUser.id || !channel.creator_id) {
      return false
    }
    return String(currentUser.id) === String(channel.creator_id)
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
      // 透传服务器真实原因（如「频道正在审批中」），避免仅给出泛化的「发送失败」
      const reason = error instanceof Error ? error.message : ''
      QMessage.error(reason || '发送失败')
    }
  }

  function setViewMode(mode: 'list' | 'card') {
    viewMode.value = mode
    localStorage.setItem('channel-viewMode', mode)
  }

  return {
    channels,
    selectedChannelId,
    openTabs,
    pendingMessageId,
    viewMode,
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
    clearPendingMessageId,
    addTab,
    removeTab,
    setViewMode,
    markChannelRead,
    incrementUnread,
    isChannelCreator,
    sendChannelMessage,
  }
})
