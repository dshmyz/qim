<template>
  <div class="message-list-wrapper">
    <div ref="messageListRef" class="message-list" data-viewer-gallery @scroll="throttledHandleScroll">
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
        <div v-if="shouldShowTimeDivider(index, message, messages)" class="time-divider">
          <span class="time-divider-text">{{ formatTime(message.timestamp) }}</span>
        </div>

        <MessageItem
          :class="{ 'message-selection-active': selectionMode && selectedMessageIds.has(String(message.id)) }"
          :message="message"
          :is-self="isMessageSelf(message)"
          :is-recalled="!!message.isRecalled"
          :conversation-type="conversationType"
          :read-users-map="readUsersMap"
          :show-read-receipt="showReadReceipt"
          :server-url="serverUrl"
          @contextmenu="(e: MouseEvent) => emit('message-contextmenu', e, message)"
          @show-user-profile="(user: any) => emit('show-user-profile', user)"
          @scroll-to-quoted-message="(id: string) => emit('scroll-to-quoted-message', id)"
          @download-file="(data: string, id?: string) => emit('download-file', data, id)"
          @save-as="(data: string, id?: string) => emit('save-as', data, id)"
          @open-mini-app="(app: any) => emit('open-mini-app', app)"
          @open-news-link="(url: string) => emit('open-news-link', url)"
          @retry-send-message="(msg: any) => emit('retry-send-message', msg)"
          @show-read-users="(msg: Message) => emit('show-read-users', msg)"
          @image-loaded="handleImageLoaded"
        >
          <template #selection-control>
            <label v-if="selectionMode && isMessageSelectionEligible(message)" class="message-selection-control">
              <input
                type="checkbox"
                :data-testid="`message-select-${message.id}`"
                :checked="selectedMessageIds.has(String(message.id))"
                @change="emit('toggle-message-selection', String(message.id))"
              >
            </label>
          </template>
        </MessageItem>
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
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import Viewer from 'viewerjs'
import 'viewerjs/dist/viewer.css'
import type { Message, User } from '../../types'
import MessageItem from '../message/MessageItem.vue'
import { useChatUtils } from '../../composables/useChatUtils'
import { isMessageSelectionEligible } from '../../utils/messageSelection'

const { formatTime, shouldShowTimeDivider } = useChatUtils()

interface Props {
  messages: Message[]
  hasMoreMessages: boolean
  conversationType: string
  readUsersMap: Record<string, { read_users: User[]; total_members: number }>
  showReadReceipt: boolean
  serverUrl: string
  currentUserId?: string | number
  selectionMode: boolean
  selectedMessageIds: Set<string>
}

interface Emits {
  'message-contextmenu': [event: MouseEvent, message: Message]
  'show-user-profile': [user: User]
  'scroll-to-quoted-message': [id: string]
  'download-file': [data: string, id?: string]
  'save-as': [data: string, id?: string]
  'open-mini-app': [app: Message['miniAppData']]
  'open-news-link': [url: string]
  'retry-send-message': [msg: Message]
  'show-read-users': [msg: Message]
  'scroll-to-bottom': []
  'load-more': []
  'mark-read': []
  'toggle-message-selection': [messageId: string]
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
let imageViewer: Viewer | null = null

const normalizeId = (id: unknown): string => {
  return id === undefined || id === null ? '' : String(id)
}

const getMessageSenderId = (message: any): string => {
  return normalizeId(
    message.sender_id ??
    message.senderId ??
    message.from_user_id ??
    message.fromUserId ??
    message.user_id ??
    message.userId ??
    message.sender?.id ??
    message.sender?.user_id ??
    message.sender?.userId ??
    message.sender?.UserID ??
    message.sender?.user?.id ??
    message.sender?.User?.ID
  )
}

const isMessageSelf = (message: any): boolean => {
  if (message.isSelf === true) return true

  const currentUserId = normalizeId(props.currentUserId)
  if (!currentUserId) return false

  return getMessageSenderId(message) === currentUserId
}

const destroyImageViewer = () => {
  imageViewer?.destroy()
  imageViewer = null
}

const initImageViewer = () => {
  if (!messageListRef.value || imageViewer) return
  imageViewer = new Viewer(messageListRef.value, {
    inline: false,
    transition: false,
    filter(image: HTMLImageElement) {
      return image.matches('img[data-viewer-image]')
    },
    navbar: true,
    title: false,
    toolbar: {
      zoomIn: 1,
      zoomOut: 1,
      oneToOne: 1,
      reset: 1,
      prev: 1,
      next: 1,
      rotateLeft: 1,
      rotateRight: 1,
    },
  })
}

const updateImageViewer = async () => {
  await nextTick()
  if (!isMounted.value) return
  initImageViewer()
  imageViewer?.update()
}

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

// 信创系统（UOS/Kylin 等）将系统滚轮速度设为“最慢”时，Chromium 收到的 wheel
// 事件 deltaY 过小甚至为 0，原生 overflow 滚动几乎无法移动，导致聊天窗口滚轮
// 看似失效。这里仅在判定为“离散滚轮”且 delta 过小时补一个最小步长，保证可
// 滚动；触控板/连续滚动以及正常 delta 仍交给原生处理，不影响其它平台体验。
const WHEEL_DELTA_THRESHOLD = 16 // 像素：低于此值视为 OS 报告的过小 delta
const WHEEL_MIN_STEP = 40        // 每次离散滚轮的最小滚动距离（像素）
const WHEEL_DISCRETE_GAP = 80   // 两次 wheel 间隔大于此值（ms）视为离散滚轮
let lastWheelTime = 0

const handleWheel = (event: WheelEvent) => {
  const el = messageListRef.value
  if (!el) return

  const now = event.timeStamp
  const isDiscrete = now - lastWheelTime > WHEEL_DISCRETE_GAP
  lastWheelTime = now

  // 连续/触控板滚动交给原生处理
  if (!isDiscrete) return

  let delta = event.deltaY
  if (event.deltaMode === 1) delta *= 16 // 行 -> 像素
  else if (event.deltaMode === 2) delta *= el.clientHeight // 页 -> 像素

  // delta 足够大时让原生滚动处理，避免影响正常平台体验
  if (Math.abs(delta) >= WHEEL_DELTA_THRESHOLD) return

  event.preventDefault()
  el.scrollTop += delta < 0 ? -WHEEL_MIN_STEP : WHEEL_MIN_STEP
}

const throttledHandleScroll = () => {
  if (throttleTimeoutId !== null) return
  throttleTimeoutId = window.setTimeout(() => {
    throttleTimeoutId = null
    handleScroll()
    emit('scroll')
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
  updateImageViewer()
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
  updateImageViewer()
  messageListRef.value?.addEventListener('wheel', handleWheel, { passive: false })
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
  messageListRef.value?.removeEventListener('wheel', handleWheel)
  destroyImageViewer()
})

watch(() => props.messages, () => {
  updateImageViewer()
}, { deep: true })
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
  opacity: 0.95;
  -webkit-overflow-scrolling: touch;
}

.message-selection-control {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  min-width: 32px;
  height: 40px;
  margin: 0;
  border-radius: 50%;
  cursor: pointer;
}

.message-selection-active {
  background: color-mix(in srgb, var(--primary-color), transparent 92%);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--primary-color), transparent 80%);
  border-radius: 12px;
}

.message-selection-control input {
  width: 18px;
  height: 18px;
  margin: 0;
  appearance: none;
  border: 2px solid var(--border-color);
  border-radius: 50%;
  background: var(--card-bg);
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

.message-selection-control input:checked {
  border-color: var(--primary-color);
  background-color: var(--primary-color);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16'%3E%3Cpath d='m3.25 8.25 2.75 2.75 6.75-6.75' fill='none' stroke='white' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/%3E%3C/svg%3E");
  box-shadow: 0 3px 10px color-mix(in srgb, var(--primary-color), transparent 55%);
}

.message-selection-control input:focus-visible {
  outline: 2px solid var(--primary-color);
  outline-offset: 3px;
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
