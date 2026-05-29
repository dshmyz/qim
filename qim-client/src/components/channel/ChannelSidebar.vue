<template>
  <div class="channel-sidebar">
    <div class="channel-search-box">
      <div class="search-input-wrapper">
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          placeholder="搜索频道..."
          aria-label="搜索频道"
        />
        <button
          v-if="searchQuery"
          class="search-clear-btn"
          @click="searchQuery = ''"
          aria-label="清空搜索"
        >
          <i class="fas fa-times-circle"></i>
        </button>
      </div>
      <div class="view-toggle" role="group" aria-label="视图模式">
        <button
          class="view-btn"
          :class="{ active: viewMode === 'list' }"
          @click="setViewMode('list')"
          aria-label="列表视图"
          title="列表视图"
          :aria-pressed="viewMode === 'list'"
        >
          <i class="fas fa-list"></i>
        </button>
        <button
          class="view-btn"
          :class="{ active: viewMode === 'card' }"
          @click="setViewMode('card')"
          aria-label="卡片视图"
          title="卡片视图"
          :aria-pressed="viewMode === 'card'"
        >
          <i class="fas fa-th-large"></i>
        </button>
        <button
          v-if="isAdmin"
          class="create-btn"
          @click="handleCreateChannel"
          aria-label="创建频道"
          title="创建频道"
        >
          <i class="fas fa-plus"></i>
        </button>
      </div>
    </div>

    <div class="channel-tabs-toggle" role="tablist" aria-label="频道标签">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'subscribed' }"
        @click="activeTab = 'subscribed'"
        role="tab"
        :aria-selected="activeTab === 'subscribed'"
        aria-label="订阅的频道"
      >
        订阅
        <span v-if="subscribedUnreadCount > 0" class="tab-unread">{{ subscribedUnreadCount }}</span>
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'discover' }"
        @click="activeTab = 'discover'"
        role="tab"
        :aria-selected="activeTab === 'discover'"
        aria-label="频道广场"
      >
        广场
      </button>
    </div>

    <div class="channel-sidebar-content">
      <LoadingSpinner v-if="loading" text="加载中..." />

      <EmptyState
        v-else-if="displayChannels.length === 0"
        icon="fa-bullhorn"
        :title="emptyTitle"
        :description="emptyDescription"
        :action-text="activeTab === 'subscribed' ? '浏览频道广场' : undefined"
        @action="activeTab = 'discover'"
      />

      <div v-else :class="['channels-container', viewMode]">
        <div v-if="viewMode === 'list'" class="channel-list-view">
          <ChannelListItem
            v-for="channel in displayChannels"
            :key="channel.id"
            :channel="channel"
            :is-selected="selectedChannelId === channel.id"
            @select="handleSelectChannel"
            @subscribe="handleSubscribe"
            @unsubscribe="handleUnsubscribe"
          />
        </div>

        <div v-else class="channel-card-view">
          <ChannelCard
            v-for="channel in displayChannels"
            :key="channel.id"
            :channel="channel"
            :is-selected="selectedChannelId === channel.id"
            @select="handleSelectChannel"
            @subscribe="handleSubscribe"
            @unsubscribe="handleUnsubscribe"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useChannelStore } from '../../stores/channel'
import type { Channel, User } from '../../types'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import ChannelListItem from './ChannelListItem.vue'
import ChannelCard from './ChannelCard.vue'

interface Props {
  currentUser: User | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  createChannel: []
}>()

const channelStore = useChannelStore()

const activeTab = ref<'subscribed' | 'discover'>('subscribed')
const searchQuery = ref('')

const channels = computed(() => channelStore.channels)
const loading = computed(() => channelStore.loading)
const viewMode = computed(() => channelStore.viewMode)
const selectedChannelId = computed(() => channelStore.selectedChannelId)

const isAdmin = computed(() => {
  const user = props.currentUser as any
  return user?.isAdmin || user?.roles?.includes('system_admin')
})

const subscribedUnreadCount = computed(() => {
  return channels.value
    .filter(c => c.is_subscribed)
    .reduce((sum, c) => sum + (c.unread_count || 0), 0)
})

const displayChannels = computed(() => {
  let result = channels.value

  if (activeTab.value === 'subscribed') {
    result = result.filter(c => c.is_subscribed)
  }

  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase().trim()
    result = result.filter(c =>
      c.name.toLowerCase().includes(query) ||
      c.description?.toLowerCase().includes(query)
    )
  }

  if (activeTab.value === 'subscribed') {
    result = [...result].sort((a, b) => {
      const unreadA = a.unread_count || 0
      const unreadB = b.unread_count || 0
      if (unreadB !== unreadA) return unreadB - unreadA
      const timeA = a.last_active_at || a.created_at || 0
      const timeB = b.last_active_at || b.created_at || 0
      return timeB - timeA
    })
  }

  return result
})

const emptyTitle = computed(() => {
  return activeTab.value === 'subscribed' ? '暂无订阅频道' : '暂无频道'
})

const emptyDescription = computed(() => {
  return activeTab.value === 'subscribed'
    ? '去频道广场订阅感兴趣的频道吧！'
    : '成为第一个创建频道的人吧！'
})

const setViewMode = (mode: 'list' | 'card') => {
  channelStore.setViewMode(mode)
}

const handleCreateChannel = () => {
  emit('createChannel')
}

const handleSelectChannel = (channel: Channel) => {
  channelStore.selectChannel(channel.id)
}

const handleSubscribe = async (channel: Channel) => {
  try {
    await channelStore.subscribeChannel(channel.id)
  } catch (error) {
    console.error('订阅频道失败:', error)
  }
}

const handleUnsubscribe = async (channel: Channel) => {
  try {
    await channelStore.unsubscribeChannel(channel.id)
  } catch (error) {
    console.error('取消订阅失败:', error)
  }
}

onMounted(() => {
  if (channels.value.length === 0) {
    channelStore.fetchChannels()
  }
})
</script>

<style scoped>
.channel-sidebar {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: transparent;
}

.channel-search-box {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px 20px 8px;
}

.search-input-wrapper {
  flex: 1;
  min-width: 0;
  position: relative;
}

.search-input {
  width: 100%;
  padding: 8px 28px 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 13px;
  background: var(--panel-bg);
  color: var(--text-color);
  outline: none;
  transition: all 0.2s;
  box-sizing: border-box;
}

.search-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px var(--primary-light);
}

.search-input::placeholder {
  color: var(--text-secondary);
}

.search-clear-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
  font-size: 12px;
  display: flex;
  align-items: center;
}

.search-clear-btn:hover {
  color: var(--text-color);
}

.view-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.view-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 1px solid transparent;
  background: transparent;
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
}

.view-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.view-btn.active {
  background: var(--hover-color);
  border-color: var(--border-color);
  color: var(--primary-color);
}

.view-btn:focus {
  /* outline: 2px solid var(--primary-color); */
  /* outline-offset: 2px; */
}

.channel-tabs-toggle {
  display: flex;
  gap: 4px;
  padding: 0 20px 8px;
}

.tab-btn {
  flex: 1;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.tab-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.tab-btn.active {
  background: var(--primary-color);
  color: white;
}

.tab-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.tab-unread {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 600;
  color: white;
  background: var(--danger-color);
  border-radius: 8px;
  line-height: 1;
}

.tab-btn.active .tab-unread {
  background: rgba(255, 255, 255, 0.3);
}

.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: 6px;
  cursor: pointer;
  flex-shrink: 0;
}

.create-btn:hover {
  background: var(--primary-dark);
}

.create-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.channel-sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  scrollbar-gutter: stable;
}

.channel-list-view {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.channel-card-view {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.channel-sidebar-content::-webkit-scrollbar {
  width: 6px;
}

.channel-sidebar-content::-webkit-scrollbar-track {
  background: transparent;
}

.channel-sidebar-content::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.channel-sidebar-content::-webkit-scrollbar-thumb:hover {
  background: var(--text-secondary);
}
</style>
