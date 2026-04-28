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

    <!-- 虚拟滚动容器 -->
    <div class="virtualizer-container" :style="{ height: `${virtualizer.getTotalSize()}px`, width: '100%', position: 'relative' }">
      <div
        v-for="virtualItem in virtualizer.getVirtualItems()"
        :key="String(virtualItem.key)"
        :data-index="virtualItem.index"
        :style="{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          transform: `translateY(${virtualItem.start}px)`
        }"
      >
          <!-- 时间分隔线 -->
          <div
            v-if="virtualItem.index < messages.length && shouldShowTime(virtualItem.index, messages[virtualItem.index], messages)"
            class="time-divider"
          >
            <span class="time-divider-text">{{ formatTime(messages[virtualItem.index].timestamp) }}</span>
          </div>

          <MessageItem
            v-if="virtualItem.index < messages.length"
            :message="messages[virtualItem.index]"
            :is-self="messages[virtualItem.index].isSelf"
            :is-recalled="!!messages[virtualItem.index].isRecalled"
            :conversation-type="conversationType"
            :read-users-map="readUsersMap"
            :server-url="serverUrl"
            @contextmenu="(e: MouseEvent) => emit('message-contextmenu', e, messages[virtualItem.index])"
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
          />
        </div>
      </div>
    </div>

    <!-- AI 思考中指示器 -->
    <slot name="thinking-indicator" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { useVirtualizer } from '@tanstack/vue-virtual'
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

const messageListRef = ref<HTMLDivElement | null>(null)
const isLoadingMore = ref(false)
const lastMarkReadTime = ref(0)

// 虚拟滚动配置
const virtualizer = useVirtualizer({
  count: props.messages.length,
  getItemKey: (index) => props.messages[index]?.id || index,
  estimateSize: () => 80,
  overscan: 5,
  getScrollElement: () => messageListRef.value
})

const shouldShowTime = (index: number, message: Message, messages: Message[]) => {
  return shouldShowTimeDivider(index, message, messages)
}

const handleScroll = () => {
  if (!messageListRef.value) return

  const { scrollTop } = messageListRef.value
  if (scrollTop < 50 && !isLoadingMore.value) {
    loadMoreMessages()
  }
}

const loadMoreMessages = () => {
  if (!props.hasMoreMessages || isLoadingMore.value) return
  isLoadingMore.value = true
  emit('load-more')
  // isLoadingMore 由父组件通过 hasMoreMessages prop 控制
  // 此处设置一个安全的超时重置，防止加载失败时永远显示加载中
  setTimeout(() => {
    isLoadingMore.value = false
  }, 5000)
}

const markMessagesAsRead = () => {
  const now = Date.now()
  if (now - lastMarkReadTime.value < 3000) return
  lastMarkReadTime.value = now
  emit('mark-read')
}

const scrollToBottom = () => {
  const el = messageListRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
  markMessagesAsRead()
}

defineExpose({
  scrollToBottom,
  messageListRef
})

// 消息变化时滚动到底部并标记已读
// 仅当消息追加时（长度增加）才滚动到底部，避免加载历史消息时强制跳转
watch(() => props.messages.length, (newLen, oldLen) => {
  if (oldLen === undefined || newLen > oldLen) {
    nextTick(() => {
      scrollToBottom()
      markMessagesAsRead()
    })
  }
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
}

.virtualizer-container {
  width: 100%;
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
