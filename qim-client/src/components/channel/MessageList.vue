<!--
  MessageList.vue - 消息列表容器组件

  功能：
  - 显示消息列表（卡片模式和时间线模式）
  - 支持模式切换
  - 支持排序切换
  - 使用 LoadingSpinner 和 EmptyState 通用组件

  使用示例：
  <MessageList
    :messages="messages"
    :mode="displayMode"
    :is-creator="isCreator"
    :loading="isLoading"
    @update:mode="handleModeChange"
  />
-->
<template>
  <div class="message-list-container">
    <!-- 工具栏 -->
    <div class="list-toolbar">
      <div class="toolbar-left">
        <h3 class="list-title">最新消息</h3>
        <span class="message-count">{{ messages.length }} 条消息</span>
      </div>
      <div class="toolbar-right">
        <!-- 显示模式切换 -->
        <div class="mode-toggle" role="group" aria-label="显示模式">
          <button
            class="mode-btn"
            :class="{ active: mode === 'card' }"
            @click="$emit('update:mode', 'card')"
            :aria-pressed="mode === 'card'"
            aria-label="卡片模式"
          >
            <i class="fas fa-th-large"></i>
          </button>
          <button
            class="mode-btn"
            :class="{ active: mode === 'timeline' }"
            @click="$emit('update:mode', 'timeline')"
            :aria-pressed="mode === 'timeline'"
            aria-label="时间线模式"
          >
            <i class="fas fa-stream"></i>
          </button>
        </div>
        <!-- 排序切换 -->
        <div class="sort-toggle">
          <button
            class="sort-btn"
            @click="toggleSort"
            :aria-label="`排序: ${sortOrder === 'desc' ? '最新优先' : '最早优先'}`"
          >
            <i :class="sortOrder === 'desc' ? 'fas fa-sort-amount-down' : 'fas fa-sort-amount-up'"></i>
            <span>{{ sortOrder === 'desc' ? '最新' : '最早' }}</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <LoadingSpinner v-if="loading" text="加载消息中..." />

    <!-- 空状态 -->
    <EmptyState
      v-else-if="!messages || messages.length === 0"
      icon="fa-comment-alt"
      title="暂无消息"
      description="还没有任何消息，等待创建者发布第一条消息吧！"
    />

    <!-- 消息列表 -->
    <div v-else class="list-content">
      <!-- 卡片模式 -->
      <div v-if="mode === 'card'" class="card-grid">
        <MessageCard
          v-for="message in sortedMessages"
          :key="message.id"
          :message="message"
          :is-creator="isCreator"
          @like="handleLike"
          @unlike="handleUnlike"
          @comment="handleComment"
          @copy-link="handleCopyLink"
        />
      </div>

      <!-- 时间线模式 -->
      <MessageTimeline
        v-else
        :messages="sortedMessages"
        :is-creator="isCreator"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import MessageCard from './MessageCard.vue'
import MessageTimeline from './MessageTimeline.vue'
import type { ChannelMessage } from '../../types'

type DisplayMode = 'card' | 'timeline'
type SortOrder = 'asc' | 'desc'

interface Props {
  messages: ChannelMessage[]
  mode?: DisplayMode
  isCreator?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  mode: 'card',
  isCreator: false,
  loading: false
})

const emit = defineEmits<{
  'update:mode': [mode: DisplayMode]
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
}>()

// 排序顺序
const sortOrder = ref<SortOrder>('desc')

// 切换排序
const toggleSort = () => {
  sortOrder.value = sortOrder.value === 'desc' ? 'asc' : 'desc'
}

// 排序后的消息列表
const sortedMessages = computed(() => {
  const sorted = [...props.messages]
  sorted.sort((a, b) => {
    const timeA = new Date(a.created_at).getTime()
    const timeB = new Date(b.created_at).getTime()
    return sortOrder.value === 'desc' ? timeB - timeA : timeA - timeB
  })
  return sorted
})

// 事件处理
const handleLike = (message: ChannelMessage) => {
  emit('like', message)
}

const handleUnlike = (message: ChannelMessage) => {
  emit('unlike', message)
}

const handleComment = (message: ChannelMessage) => {
  emit('comment', message)
}

const handleCopyLink = (message: ChannelMessage) => {
  emit('copyLink', message)
}
</script>

<style scoped>
.message-list-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.list-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--spacing-4);
  border-bottom: 1px solid var(--border-color);
  background: var(--card-bg);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.list-title {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.message-count {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  padding: 2px var(--spacing-2);
  background: var(--hover-color);
  border-radius: var(--radius-sm);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.mode-toggle {
  display: flex;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.mode-btn {
  padding: var(--spacing-1) var(--spacing-3);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  justify-content: center;
}

.mode-btn:not(:last-child) {
  border-right: 1px solid var(--border-color);
}

.mode-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.mode-btn.active {
  background: var(--primary-color);
  color: white;
}

.mode-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: -2px;
}

.sort-btn {
  display: flex;
  align-items: center;
  gap: var(--spacing-1);
  padding: var(--spacing-1) var(--spacing-3);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-xs);
  transition: all var(--transition-fast);
}

.sort-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
  border-color: var(--primary-color);
}

.sort-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.list-content {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-4);
}

.card-grid {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-4);
}
</style>
