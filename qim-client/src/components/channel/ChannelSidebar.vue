<!--
  ChannelSidebar.vue - 频道侧边栏容器组件

  功能：
  - 显示频道列表（支持列表视图和卡片视图）
  - 订阅/广场标签切换
  - 视图模式切换（列表/卡片）
  - 创建频道按钮（仅管理员可见）
  - 加载状态和空状态展示

  使用：
  - 通过 useChannelStore 管理频道状态
  - 接收 currentUser prop 用于权限判断
  - 发送 createChannel 事件用于创建频道
-->
<template>
  <div class="channel-sidebar">
    <!-- 侧边栏头部 -->
    <div class="channel-sidebar-header">
      <div class="header-left">
        <h2 class="sidebar-title">频道</h2>
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

    <!-- 标签切换 -->
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

    <!-- 视图切换和搜索 -->
    <div class="view-and-search">
      <input
        v-model="searchQuery"
        type="text"
        class="search-input"
        placeholder="搜索频道..."
        aria-label="搜索频道"
      />
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
      </div>
    </div>

    <!-- 侧边栏内容 -->
    <div class="channel-sidebar-content">
      <!-- 加载状态 -->
      <LoadingSpinner v-if="loading" text="加载中..." />

      <!-- 空状态 -->
      <EmptyState
        v-else-if="displayChannels.length === 0"
        icon="fa-bullhorn"
        :title="emptyTitle"
        :description="emptyDescription"
        :action-text="activeTab === 'subscribed' ? '浏览频道广场' : undefined"
        @action="activeTab = 'discover'"
      />

      <!-- 频道列表 -->
      <div v-else :class="['channels-container', viewMode]">
        <!-- 列表视图 -->
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

        <!-- 卡片视图 -->
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

// 本地状态
const activeTab = ref<'subscribed' | 'discover'>('subscribed')
const searchQuery = ref('')

// 从 store 获取状态
const channels = computed(() => channelStore.channels)
const loading = computed(() => channelStore.loading)
const viewMode = computed(() => channelStore.viewMode)
const selectedChannelId = computed(() => channelStore.selectedChannelId)

// 计算属性
const isAdmin = computed(() => {
  return props.currentUser?.isAdmin || props.currentUser?.roles?.includes('system_admin')
})

const displayChannels = computed(() => {
  let result = channels.value
  
  // 根据标签过滤
  if (activeTab.value === 'subscribed') {
    result = result.filter(c => c.is_subscribed)
  }
  
  // 根据搜索关键词过滤
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase().trim()
    result = result.filter(c => 
      c.name.toLowerCase().includes(query) ||
      c.description?.toLowerCase().includes(query)
    )
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

// 方法
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
    // 这里可以添加错误提示，例如使用消息组件
  }
}

const handleUnsubscribe = async (channel: Channel) => {
  try {
    await channelStore.unsubscribeChannel(channel.id)
  } catch (error) {
    console.error('取消订阅失败:', error)
    // 这里可以添加错误提示，例如使用消息组件
  }
}

// 生命周期
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

/* 头部样式 */
.channel-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-3) var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  min-height: 52px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
}

.sidebar-title {
  margin: 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-md);
  cursor: pointer;
}

.create-btn:hover {
  background: var(--primary-dark);
}

.create-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

/* 标签切换 */
.channel-tabs-toggle {
  display: flex;
  gap: var(--spacing-1);
  padding: var(--spacing-2);
  border-bottom: 1px solid var(--border-color);
}

.tab-btn {
  flex: 1;
  padding: var(--spacing-1) var(--spacing-2);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  cursor: pointer;
}

.tab-btn:hover {
  background: var(--color-gray-100);
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

/* 视图切换和搜索 */
.view-and-search {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2);
  border-bottom: 1px solid var(--border-color);
}

.search-input {
  flex: 1;
  min-width: 0;
  padding: var(--spacing-1) var(--spacing-2);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  font-size: 13px;
  background: var(--input-bg);
  color: var(--text-color);
}

.search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px var(--primary-light);
}

.search-input::placeholder {
  color: var(--text-secondary);
}

/* 视图切换 */
.view-toggle {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
  flex-shrink: 0;
}

.view-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid transparent;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 14px;
  color: var(--text-secondary);
  cursor: pointer;
}

.view-btn:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.view-btn.active {
  background: var(--color-gray-100);
  border-color: var(--border-color);
  color: var(--primary-color);
}

.view-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

/* 内容区域 */
.channel-sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-2);
}

/* 列表视图 */
.channel-list-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

/* 卡片视图 */
.channel-card-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

/* 滚动条样式 */
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

/* 响应式设计 */
@media (max-width: 768px) {
  .channel-sidebar-header {
    padding: var(--spacing-2);
  }

  .sidebar-title {
    font-size: var(--font-size-base);
  }

  .channel-tabs-toggle,
  .view-toggle {
    padding: var(--spacing-1) var(--spacing-2);
  }

  .tab-btn,
  .view-btn {
    padding: var(--spacing-1) var(--spacing-2);
    font-size: 12px;
  }

  .channel-sidebar-content {
    padding: var(--spacing-2);
  }
}
</style>
