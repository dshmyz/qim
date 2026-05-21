<template>
  <div class="message-list-wrapper">
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
        <div v-if="timeDividerIndices.has(index)" class="time-divider">
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

    <!-- 跳转到最新消息按钮 -->
    <div v-if="showScrollToBottomBtn" class="scroll-to-bottom-btn" @click="scrollToBottom">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M7 13l5 5 5-5M7 6l5 5 5-5" />
      </svg>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
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
const shouldAutoScroll = ref(true)
const showScrollToBottomBtn = ref(false)
const isMounted = ref(false)

let scrollTimeoutId: number | null = null
let throttleTimeoutId: number | null = null

// 优化：computed 预处理时间分隔位置，避免模板中每次 render 全量计算
const timeDividerIndices = computed<Set<number>>(() => {
  const indices = new Set<number>()
  for (let i = 0; i < props.messages.length; i++) {
    if (shouldShowTimeDivider(i, props.messages[i], props.messages)) {
      indices.add(i)
    }
  }
  return indices
})

const handleScroll = () => {
  if (!messageListRef.value) return

  const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
  const distanceToBottom = scrollHeight - scrollTop - clientHeight
  shouldAutoScroll.value = distanceToBottom < 50
  showScrollToBottomBtn.value = distanceToBottom > 200

  if (shouldAutoScroll.value) {
    emit('mark-read')
  }

  if (scrollTop < 50 && !isLoadingMore.value) {
    loadMoreMessages()
  }
}

const throttledHandleScroll = () => {
  if (throttleTimeoutId !== null) return
  throttleTimeoutId = window.setTimeout(() => {
    throttleTimeoutId = null
    handleScroll()
  }, 100)
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

const scrollToBottom = (instant: boolean = false) => {
  if (!isMounted.value || !messageListRef.value) return

  messageListRef.value.scrollTo({
    top: messageListRef.value.scrollHeight,
    behavior: instant ? 'auto' : 'smooth'
  })
  showScrollToBottomBtn.value = false
}

const scrollToBottomWithDelay = (delay: number = 100) => {
  if (scrollTimeoutId) {
    clearTimeout(scrollTimeoutId)
  }

  scrollTimeoutId = window.setTimeout(() => {
    if (isMounted.value) {
      scrollToBottom(true)
    }
  }, delay)
}

const handleImageLoaded = () => {
  nextTick(() => {
    if (!isMounted.value || !messageListRef.value) return

    const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
    const distanceToBottom = scrollHeight - scrollTop - clientHeight

    if (shouldAutoScroll.value && distanceToBottom < 50) {
      messageListRef.value.scrollTop = scrollHeight - clientHeight
    }
  })
}

defineExpose({
  scrollToBottom,
  scrollToBottomWithDelay,
  messageListRef
})

onMounted(() => {
  isMounted.value = true
  scrollToBottomWithDelay(100)
})

onUnmounted(() => {
  isMounted.value = false
  if (scrollTimeoutId) {
    clearTimeout(scrollTimeoutId)
    scrollTimeoutId = null
  }
  if (throttleTimeoutId) {
    clearTimeout(throttleTimeoutId)
    throttleTimeoutId = null
  }
})
</script>

<style scoped>
.message-list-wrapper {
  flex: 1;
  display: flex;
  position: relative;
  overflow: hidden;
}

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

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 20px;
  right: 20px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s ease;
  animation: slideIn 0.3s ease;
  z-index: 10;
}

.scroll-to-bottom-btn:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

.scroll-to-bottom-btn:active {
  transform: scale(0.95);
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
