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
          title="创建频道"
        >
          <i class="fas fa-plus"></i>
        </button>
      </div>
    </div>

    <!-- 标签切换 -->
    <div class="channel-tabs-toggle">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'subscribed' }"
        @click="activeTab = 'subscribed'"
      >
        订阅
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'discover' }"
        @click="activeTab = 'discover'"
      >
        广场
      </button>
    </div>

    <!-- 视图切换 -->
    <div class="view-toggle">
      <button
        class="view-btn"
        :class="{ active: viewMode === 'list' }"
        @click="setViewMode('list')"
        title="列表视图"
      >
        <i class="fas fa-list"></i>
      </button>
      <button
        class="view-btn"
        :class="{ active: viewMode === 'card' }"
        @click="setViewMode('card')"
        title="卡片视图"
      >
        <i class="fas fa-th-large"></i>
      </button>
    </div>

    <!-- 侧边栏内容 -->
    <div class="channel-sidebar-content">
      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <div class="loading-spinner"></div>
        <span>加载中...</span>
      </div>

      <!-- 空状态 -->
      <div v-else-if="displayChannels.length === 0" class="empty-state">
        <i class="fas fa-bullhorn empty-icon"></i>
        <h4>{{ emptyTitle }}</h4>
        <p>{{ emptyDescription }}</p>
        <button
          v-if="activeTab === 'subscribed'"
          class="switch-btn"
          @click="activeTab = 'discover'"
        >
          浏览频道广场
        </button>
      </div>

      <!-- 频道列表 -->
      <div v-else :class="['channels-container', viewMode]">
        <!-- 列表视图 -->
        <div v-if="viewMode === 'list'" class="channel-list-view">
          <div
            v-for="channel in displayChannels"
            :key="channel.id"
            class="channel-list-item"
            :class="{ active: selectedChannelId === channel.id }"
            @click="handleSelectChannel(channel)"
          >
            <img
              :src="channel.avatar || generateAvatar(channel.name)"
              :alt="channel.name"
              class="channel-avatar"
            />
            <div class="channel-info">
              <div class="channel-name">{{ channel.name }}</div>
              <div class="channel-desc">{{ channel.description }}</div>
            </div>
            <div class="channel-actions">
              <button
                v-if="channel.is_subscribed"
                class="subscribe-btn subscribed"
                @click.stop="handleUnsubscribe(channel)"
              >
                <i class="fas fa-check"></i>
              </button>
              <button
                v-else
                class="subscribe-btn"
                @click.stop="handleSubscribe(channel)"
              >
                <i class="fas fa-plus"></i>
              </button>
            </div>
          </div>
        </div>

        <!-- 卡片视图 -->
        <div v-else class="channel-card-view">
          <div
            v-for="channel in displayChannels"
            :key="channel.id"
            class="channel-card"
            :class="{ active: selectedChannelId === channel.id }"
            @click="handleSelectChannel(channel)"
          >
            <div class="card-header">
              <img
                :src="channel.avatar || generateAvatar(channel.name)"
                :alt="channel.name"
                class="card-avatar"
              />
              <button
                v-if="channel.is_subscribed"
                class="card-subscribe-btn subscribed"
                @click.stop="handleUnsubscribe(channel)"
              >
                <i class="fas fa-check"></i> 已订阅
              </button>
              <button
                v-else
                class="card-subscribe-btn"
                @click.stop="handleSubscribe(channel)"
              >
                <i class="fas fa-plus"></i> 订阅
              </button>
            </div>
            <div class="card-body">
              <h4 class="card-title">{{ channel.name }}</h4>
              <p class="card-description">{{ channel.description }}</p>
            </div>
            <div class="card-footer">
              <span class="card-creator">
                <i class="fas fa-user"></i>
                {{ channel.creator?.name || '未知' }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useChannelStore } from '../../stores/channel'
import { generateAvatar } from '../../utils/avatar'
import type { Channel, User } from '../../types'

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

// 从 store 获取状态
const channels = computed(() => channelStore.channels)
const loading = computed(() => channelStore.loading)
const viewMode = computed(() => channelStore.viewMode)
const selectedChannelId = computed(() => channelStore.selectedChannelId)

// 计算属性
const isAdmin = computed(() => {
  return props.currentUser?.role === 'admin'
})

const displayChannels = computed(() => {
  if (activeTab.value === 'subscribed') {
    return channels.value.filter(c => c.is_subscribed)
  }
  return channels.value
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
  await channelStore.subscribeChannel(channel.id)
}

const handleUnsubscribe = async (channel: Channel) => {
  await channelStore.unsubscribeChannel(channel.id)
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
  width: 280px;
  height: 100%;
  background: var(--card-bg);
  border-right: 1px solid var(--border-color);
  transition: width var(--transition-base);
}

/* 头部样式 */
.channel-sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  min-height: 60px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.sidebar-title {
  margin: 0;
  font-size: var(--font-size-xl);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: var(--primary-color);
  color: white;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.create-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

/* 标签切换 */
.channel-tabs-toggle {
  display: flex;
  gap: var(--spacing-1);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
}

.tab-btn {
  flex: 1;
  padding: var(--spacing-2) var(--spacing-3);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: var(--font-weight-medium);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.tab-btn.active {
  background: var(--primary-color);
  color: white;
}

/* 视图切换 */
.view-toggle {
  display: flex;
  gap: var(--spacing-1);
  padding: var(--spacing-2) var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
}

.view-btn {
  flex: 1;
  padding: var(--spacing-2);
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 14px;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.view-btn:hover {
  background: var(--color-gray-100);
  color: var(--text-color);
}

.view-btn.active {
  background: var(--color-gray-200);
  color: var(--text-color);
}

/* 内容区域 */
.channel-sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-3);
}

/* 加载状态 */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-12) 0;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: var(--spacing-4);
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-12) var(--spacing-4);
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  color: var(--primary-color);
  margin-bottom: var(--spacing-4);
  opacity: 0.5;
}

.empty-state h4 {
  margin: 0 0 var(--spacing-2) 0;
  font-size: var(--font-size-lg);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.empty-state p {
  margin: 0 0 var(--spacing-4) 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.switch-btn {
  padding: var(--spacing-2) var(--spacing-4);
  border: 1px solid var(--primary-color);
  border-radius: var(--radius-md);
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
}

.switch-btn:hover {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
  transform: translateY(-1px);
}

/* 列表视图 */
.channel-list-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

.channel-list-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
  padding: var(--spacing-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.channel-list-item:hover {
  background: var(--color-gray-100);
}

.channel-list-item.active {
  background: var(--color-gray-200);
}

.channel-avatar {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.channel-info {
  flex: 1;
  min-width: 0;
}

.channel-name {
  font-size: 14px;
  font-weight: var(--font-weight-medium);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.channel-desc {
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-top: 2px;
}

.channel-actions {
  flex-shrink: 0;
}

.subscribe-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid var(--primary-color);
  background: transparent;
  color: var(--primary-color);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

/* 卡片视图 */
.channel-card-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-3);
}

.channel-card {
  background: var(--card-bg);
  border-radius: var(--radius-lg);
  padding: var(--spacing-4);
  box-shadow: var(--shadow-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
  border: 1px solid var(--border-color);
}

.channel-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
  border-color: var(--primary-color);
}

.channel-card.active {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(51, 133, 255, 0.1);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-3);
}

.card-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  object-fit: cover;
}

.card-subscribe-btn {
  padding: var(--spacing-1) var(--spacing-3);
  border: 1px solid var(--primary-color);
  background: transparent;
  color: var(--primary-color);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
}

.card-subscribe-btn:hover {
  background: var(--primary-color);
  color: white;
}

.card-subscribe-btn.subscribed {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.card-body {
  margin-bottom: var(--spacing-3);
}

.card-title {
  margin: 0 0 var(--spacing-2) 0;
  font-size: 15px;
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-description {
  margin: 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.5;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-footer {
  font-size: 12px;
  color: var(--text-secondary);
  border-top: 1px solid var(--border-color);
  padding-top: var(--spacing-3);
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
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
  .channel-sidebar {
    width: 100%;
    border-right: none;
  }

  .channel-sidebar-header {
    padding: var(--spacing-3);
  }

  .sidebar-title {
    font-size: var(--font-size-lg);
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

  .channel-list-item {
    padding: var(--spacing-2);
  }

  .channel-avatar {
    width: 36px;
    height: 36px;
  }

  .channel-card {
    padding: var(--spacing-3);
  }

  .card-avatar {
    width: 40px;
    height: 40px;
  }
}
</style>
