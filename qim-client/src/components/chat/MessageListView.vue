<template>
  <div ref="messageListRef" class="message-list" @scroll="handleScroll">
    <!-- 没有更多消息提示 -->
    <div v-if="!hasMoreMessages" class="no-more-messages">
      <div class="no-more-divider">
        <span class="divider-line"></span>
        <span class="divider-text">已全部加载完毕</span>
        <span class="divider-line"></span>
      </div>
    </div>

    <!-- 加载更多提示 -->
    <div v-if="isLoadingMore" class="loading-more">
      <span>加载中...</span>
    </div>

    <div v-for="(message, index) in messages" :key="message.id">
      <!-- 时间分隔线 -->
      <div v-if="shouldShowTime(index, message, messages)" class="time-divider">
        <span class="time-divider-text">{{ formatTime(message.timestamp) }}</span>
      </div>

      <MessageItem
        :message="message"
        :is-self="message.isSelf"
        :is-recalled="!!message.isRecalled"
        :conversation-type="conversationType"
        :read-users-map="readUsersMap"
        :server-url="serverUrl"
        @contextmenu="(e: MouseEvent) => emit('message-contextmenu', e, message)"
        @show-user-profile="(user: any) => emit('show-user-profile', user)"
        @scroll-to-quoted-message="(id: string) => emit('scroll-to-quoted-message', id)"
        @preview-image="(data: string) => emit('preview-image', data)"
        @download-file="(data: string) => emit('download-file', data)"
        @save-as="(data: string) => emit('save-as', data)"
        @view-shared-content="(content: string) => emit('view-shared-content', content)"
        @open-mini-app="(app: any) => emit('open-mini-app', app)"
        @open-news-link="(url: string) => emit('open-news-link', url)"
        @retry-send-message="(msg: any) => emit('retry-send-message', msg)"
        @show-read-users="(msg: Message) => emit('show-read-users', msg)"
        @image-loaded="handleImageLoaded"
      />
    </div>

    <!-- AI 思考中指示器 -->
    <slot name="thinking-indicator" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import type { Message, User } from '../../types'
import MessageItem from '../message/MessageItem.vue'
import { useChatUtils } from '../../composables/useChatUtils'

const { formatTime, shouldShowTimeDivider } = useChatUtils()

interface Props {
  messages: Message[]
  hasMoreMessages: boolean
  conversationType: string
  readUsersMap: Record<string, { read_users: User[]; total_members: number }>
  serverUrl: string
}

interface Emits {
  'message-contextmenu': [event: MouseEvent, message: Message]
  'show-user-profile': [user: User]
  'scroll-to-quoted-message': [id: string]
  'preview-image': [data: string]
  'download-file': [data: string]
  'save-as': [data: string]
  'view-shared-content': [content: string]
  'open-mini-app': [app: Message['miniAppData']]
  'open-news-link': [url: string]
  'retry-send-message': [msg: Message]
  'show-read-users': [msg: Message]
  'scroll-to-bottom': []
  'load-more': []
  'mark-read': []
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const messageListRef = ref<HTMLDivElement>()
const isLoadingMore = ref(false)
const lastMarkReadTime = ref(0)
// 记录用户是否应该自动滚动到底部（用户在底部附近时）
const shouldAutoScroll = ref(true)

const shouldShowTime = (index: number, message: Message, messages: Message[]) => {
  return shouldShowTimeDivider(index, message, messages)
}

const throttle = (func: Function, delay: number) => {
  let timeoutId: number | null = null
  return function (this: any, ...args: any[]) {
    if (timeoutId === null) {
      timeoutId = window.setTimeout(() => {
        func.apply(this, args)
        timeoutId = null
      }, delay)
    }
  }
}

const handleScroll = throttle(() => {
  if (!messageListRef.value) return

  const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
  // 用户距离底部50px以内，认为是"在底部"
  shouldAutoScroll.value = scrollHeight - scrollTop - clientHeight < 50
  
  if (shouldAutoScroll.value) {
    markMessagesAsRead()
  }

  if (scrollTop < 50 && !isLoadingMore.value) {
    loadMoreMessages()
  }
}, 100)

const markMessagesAsRead = async () => {
  const now = Date.now()
  if (now - lastMarkReadTime.value < 3000) return
  lastMarkReadTime.value = now
  emit('mark-read')
}

const loadMoreMessages = async () => {
  if (!props.hasMoreMessages) return
  isLoadingMore.value = true
  try {
    emit('load-more')
  } finally {
    isLoadingMore.value = false
  }
}

const scrollToBottom = () => {
  if (messageListRef.value) {
    messageListRef.value.scrollTop = messageListRef.value.scrollHeight
    markMessagesAsRead()
  }
}

// 图片加载完成后的处理
const handleImageLoaded = () => {
  // 使用 nextTick 等待 DOM 更新后检查滚动位置
  nextTick(() => {
    if (!messageListRef.value) return
    const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
    const distanceToBottom = scrollHeight - scrollTop - clientHeight
    
    // 如果用户本来就在底部附近（考虑图片加载后scrollHeight可能变化，使用100px阈值）
    // 或者标志位显示应该自动滚动，则滚动到底部
    if (shouldAutoScroll.value || distanceToBottom < 100) {
      scrollToBottom()
    }
  })
}

defineExpose({
  scrollToBottom,
  messageListRef
})

onMounted(() => {
  scrollToBottom()
})
</script>

<style scoped>
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
  opacity: 0.95;
  -webkit-overflow-scrolling: touch;
}

.time-divider {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 15px 0;
  position: relative;
}

.time-divider-text {
  background-color: var(--color-gray-200);
  color: var(--color-gray-500);
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 12px;
  text-align: center;
  font-weight: 400;
}

[data-theme="elegant-dark"] .time-divider-text {
  background-color: var(--sidebar-bg);
  color: var(--color-gray-700);
}

.no-more-messages {
  text-align: center;
  padding: 12px 0;
}

.no-more-divider {
  display: flex;
  align-items: center;
  gap: 12px;
}

.divider-line {
  flex: 1;
  height: 1px;
  background-color: var(--color-gray-200);
}

.divider-text {
  color: var(--color-gray-400);
  font-size: 12px;
  white-space: nowrap;
  font-weight: 400;
}

.loading-more {
  text-align: center;
  padding: 10px 0;
  color: var(--color-gray-500);
  font-size: 12px;
}
</style>
