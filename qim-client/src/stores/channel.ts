import { defineStore } from 'pinia'
import type { Channel } from '../types'

interface ChannelState {
  channels: Channel[]
  selectedChannelId: string | number | null
  openTabs: Array<{ id: string | number; name: string }>
  viewMode: 'list' | 'card'
  messageMode: 'card' | 'timeline'
  loading: boolean
}

export const useChannelStore = defineStore('channel', {
  state: (): ChannelState => ({
    channels: [],
    selectedChannelId: null,
    openTabs: [],
    viewMode: 'card',
    messageMode: 'card',
    loading: false,
  }),

  getters: {
    selectedChannel: (state) => {
      return state.channels.find(c => c.id === state.selectedChannelId)
    },
    subscribedChannels: (state) => {
      return state.channels.filter(c => c.is_subscribed)
    },
  },

  actions: {
    async fetchChannels() {
      this.loading = true
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels`, {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          this.channels = data.data || []
        }
      } catch (error) {
        console.error('加载频道失败:', error)
      } finally {
        this.loading = false
      }
    },

    async subscribeChannel(channelId: string | number) {
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels/${channelId}/subscribe`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          const channel = this.channels.find(c => c.id === channelId)
          if (channel) {
            channel.is_subscribed = true
          }
        }
      } catch (error) {
        console.error('订阅频道失败:', error)
      }
    },

    async unsubscribeChannel(channelId: string | number) {
      try {
        const serverUrl = localStorage.getItem('serverUrl') || ''
        const response = await fetch(`${serverUrl}/api/v1/channels/${channelId}/unsubscribe`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
            'Content-Type': 'application/json'
          }
        })
        const data = await response.json()
        if (data.code === 0) {
          const channel = this.channels.find(c => c.id === channelId)
          if (channel) {
            channel.is_subscribed = false
          }
        }
      } catch (error) {
        console.error('取消订阅失败:', error)
      }
    },

    selectChannel(channelId: string | number) {
      this.selectedChannelId = channelId
      const channel = this.channels.find(c => c.id === channelId)
      if (channel) {
        this.addTab(channel)
      }
    },

    addTab(channel: Channel) {
      const exists = this.openTabs.find(t => t.id === channel.id)
      if (!exists) {
        this.openTabs.push({ id: channel.id, name: channel.name })
      }
    },

    removeTab(channelId: string | number) {
      const index = this.openTabs.findIndex(t => t.id === channelId)
      if (index > -1) {
        this.openTabs.splice(index, 1)
        if (this.selectedChannelId === channelId) {
          this.selectedChannelId = this.openTabs.length > 0 ? this.openTabs[0].id : null
        }
      }
    },

    setViewMode(mode: 'list' | 'card') {
      this.viewMode = mode
    },

    setMessageMode(mode: 'card' | 'timeline') {
      this.messageMode = mode
    },
  },
})
