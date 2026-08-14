<template>
  <div class="message-list-container">
    <div class="list-toolbar">
      <div class="toolbar-left">
        <h3 class="list-title">最新消息</h3>
        <span class="message-count">{{ messages.length }} 条消息</span>
      </div>
      <div class="toolbar-right">
        <button
          class="refresh-btn"
          @click="$emit('refresh')"
          aria-label="刷新消息"
          title="刷新消息"
        >
          <i class="fas fa-sync-alt"></i>
        </button>
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

    <LoadingSpinner v-if="loading" text="加载消息中..." />

    <EmptyState
      v-else-if="!messages || messages.length === 0"
      icon="fa-comment-alt"
      title="暂无消息"
      :description="isCreator ? '发布第一条消息吧！' : '等待创建者发布第一条消息'"
    />

    <div v-else class="list-content">
      <div class="card-grid">
        <MessageCard
          v-for="message in sortedMessages"
          :key="message.id"
          :id="'channel-msg-' + message.id"
          :message="message"
          :channel="channel"
          :is-creator="isCreator"
          :interactive="interactive"
          @like="handleLike"
          @unlike="handleUnlike"
          @comment="handleComment"
          @copy-link="handleCopyLink"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, nextTick } from 'vue'
import LoadingSpinner from '../shared/LoadingSpinner.vue'
import EmptyState from '../shared/EmptyState.vue'
import MessageCard from './MessageCard.vue'
import type { ChannelMessage, Channel } from '../../types'

type SortOrder = 'asc' | 'desc'

interface Props {
  messages: ChannelMessage[]
  channel?: Channel
  isCreator?: boolean
  loading?: boolean
  sortOrder?: SortOrder
  creatorId?: string | number
  interactive?: boolean
  highlightMessageId?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  isCreator: false,
  loading: false,
  sortOrder: 'desc',
  creatorId: '',
  interactive: true,
  highlightMessageId: null
})

const emit = defineEmits<{
  'update:sortOrder': [sortOrder: SortOrder]
  refresh: []
  like: [message: ChannelMessage]
  unlike: [message: ChannelMessage]
  comment: [message: ChannelMessage]
  copyLink: [message: ChannelMessage]
  'highlight-consumed': []
}>()

const toggleSort = () => {
  const newSortOrder = props.sortOrder === 'desc' ? 'asc' : 'desc'
  emit('update:sortOrder', newSortOrder)
}

const sortedMessages = computed(() => {
  const sorted = [...props.messages]
  sorted.sort((a, b) => {
    const timeA = new Date(a.created_at).getTime()
    const timeB = new Date(b.created_at).getTime()
    return props.sortOrder === 'desc' ? timeB - timeA : timeA - timeB
  })
  return sorted
})

// 深链定位：当待定位消息 id 到来且消息已加载渲染时，滚动到目标并把该条高亮闪烁，随后通知上层消费清除。
let highlightTimer: number | undefined
watch(
  () => [props.highlightMessageId, props.loading, props.messages.length] as const,
  async ([messageId, loading]) => {
    if (highlightTimer) {
      clearTimeout(highlightTimer)
      highlightTimer = undefined
    }
    if (!messageId || loading) return
    await nextTick()
    const target = document.getElementById('channel-msg-' + messageId)
    if (target) {
      target.scrollIntoView({ block: 'center', behavior: 'smooth' })
      target.classList.add('msg-highlight')
      highlightTimer = window.setTimeout(() => {
        target.classList.remove('msg-highlight')
        emit('highlight-consumed')
        highlightTimer = undefined
      }, 2600)
    } else {
      // 目标元素未找到（如消息已被清理），直接消费避免卡住
      emit('highlight-consumed')
    }
  },
  { immediate: true }
)

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
  padding: 0 20px;
  height: 56px;
  box-sizing: border-box;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: var(--card-bg);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-3);
}

.list-title {
  margin: 0;
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-semibold);
  color: var(--text-color);
}

.message-count {
  font-size: var(--font-size-xxs);
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
  font-size: var(--font-size-xxs);
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

.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: var(--font-size-xs);
  transition: all 0.2s ease;
}

.refresh-btn:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.refresh-btn:active {
  transform: rotate(180deg);
}

.refresh-btn:focus {
  outline: 2px solid var(--primary-color);
  outline-offset: 2px;
}

.list-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.card-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 760px;
  margin: 0;
}
</style>
