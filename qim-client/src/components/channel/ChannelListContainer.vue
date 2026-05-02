<!--
  ChannelListContainer.vue - 频道列表容器组件

  功能：
  - 管理频道列表的展示
  - 支持列表视图和卡片视图切换
  - 处理加载状态和空状态
  - 协调子组件的交互

  使用示例：
  <ChannelListContainer
    :channels="channels"
    :view-mode="viewMode"
    :loading="loading"
    :selected-channel-id="selectedChannelId"
    @select="handleSelect"
    @subscribe="handleSubscribe"
    @unsubscribe="handleUnsubscribe"
  />
-->
<template>
  <div class="channel-list-container">
    <!-- 视图切换工具栏 -->
    <div class="view-toolbar">
      <div class="view-tabs">
        <button
          class="view-tab"
          :class="{ active: viewMode === 'card' }"
          @click="handleViewModeChange('card')"
          :aria-label="'卡片视图'"
          :aria-pressed="viewMode === 'card'"
        >
          <i class="fas fa-th-large"></i>
          <span>卡片</span>
        </button>
        <button
          class="view-tab"
          :class="{ active: viewMode === 'list' }"
          @click="handleViewModeChange('list')"
          :aria-label="'列表视图'"
          :aria-pressed="viewMode === 'list'"
        >
          <i class="fas fa-list"></i>
          <span>列表</span>
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <LoadingSpinner v-if="loading" text="加载频道中..." />

    <!-- 空状态 -->
    <EmptyState
      v-else-if="channels.length === 0"
      icon="fa-bullhorn"
      title="暂无频道"
      description="还没有任何频道数据"
    />

    <!-- 频道列表 -->
    <div v-else :class="['channel-list', viewMode]">
      <!-- 卡片视图 -->
      <template v-if="viewMode === 'card'">
        <ChannelCard
          v-for="channel in channels"
          :key="channel.id"
          :channel="channel"
          :is-selected="channel.id === selectedChannelId"
          @select="handleSelect"
          @subscribe="handleSubscribe"
          @unsubscribe="handleUnsubscribe"
        />
      </template>

      <!-- 列表视图 -->
      <template v-else>
        <ChannelListItem
          v-for="channel in channels"
          :key="channel.id"
          :channel="channel"
          :is-selected="channel.id === selectedChannelId"
          @select="handleSelect"
          @subscribe="handleSubscribe"
          @unsubscribe="handleUnsubscribe"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Channel } from '../../types'
import ChannelListItem from './ChannelListItem.vue'
import ChannelCard from './ChannelCard.vue'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'

interface Props {
  channels: Channel[]
  viewMode: 'list' | 'card'
  loading?: boolean
  selectedChannelId?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  selectedChannelId: null
})

const emit = defineEmits<{
  select: [channel: Channel]
  subscribe: [channel: Channel]
  unsubscribe: [channel: Channel]
  'update:viewMode': [mode: 'list' | 'card']
}>()

// 处理视图模式切换
function handleViewModeChange(mode: 'list' | 'card') {
  emit('update:viewMode', mode)
}

// 处理频道选择
function handleSelect(channel: Channel) {
  emit('select', channel)
}

// 处理订阅
function handleSubscribe(channel: Channel) {
  emit('subscribe', channel)
}

// 处理取消订阅
function handleUnsubscribe(channel: Channel) {
  emit('unsubscribe', channel)
}
</script>

<style scoped>
.channel-list-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* 视图切换工具栏 */
.view-toolbar {
  display: flex;
  justify-content: flex-end;
  padding: var(--spacing-3);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.view-tabs {
  display: flex;
  gap: var(--spacing-1);
  background: var(--color-gray-100);
  border-radius: var(--radius-md);
  padding: var(--spacing-1);
}

.view-tab {
  display: flex;
  align-items: center;
  gap: var(--spacing-2);
  padding: var(--spacing-2) var(--spacing-3);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  font-weight: var(--font-weight-medium);
  transition: all var(--transition-fast);
}

.view-tab:hover {
  color: var(--text-color);
  background: var(--color-gray-200);
}

.view-tab.active {
  background: var(--primary-color);
  color: white;
}

.view-tab:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

/* 频道列表 */
.channel-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-3);
}

.channel-list.card {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--spacing-4);
  align-content: start;
}

.channel-list.list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2);
}

/* 滚动条样式 */
.channel-list::-webkit-scrollbar {
  width: 6px;
}

.channel-list::-webkit-scrollbar-track {
  background: transparent;
}

.channel-list::-webkit-scrollbar-thumb {
  background: var(--color-gray-300);
  border-radius: var(--radius-full);
}

.channel-list::-webkit-scrollbar-thumb:hover {
  background: var(--color-gray-400);
}
</style>
