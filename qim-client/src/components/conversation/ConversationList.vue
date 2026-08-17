<template>
  <div v-if="conversations.length === 0 && !isLoading" class="empty-conversations">
    <div class="placeholder-content">
      <i class="fas fa-comments fa-4x"></i>
      <h3>暂无会话</h3>
      <p>从通讯录或群聊中发起对话吧</p>
    </div>
  </div>
  <div
    v-else
    ref="listRef"
    class="conversation-list conversation-list--scrollable"
    :class="{ 'is-hovered': isHovered, 'is-scrolling': isScrolling }"
    @scroll="handleScroll"
    @mouseenter="isHovered = true"
    @mouseleave="isHovered = false"
  >
    <div
      v-for="conversation in conversations"
      :key="conversation.id"
      class="conversation-item"
      :class="{ active: sameConversationId(conversation.id, currentConversationId) }"
      @click="$emit('select', conversation)"
      @contextmenu.prevent="$emit('contextMenu', $event, conversation)"
    >
      <div class="conversation-avatar">
        <Avatar
          :src="conversation.avatar"
          :name="conversation.name || '用户'"
          :server-url="serverUrl"
          :badge="conversationBadge(conversation)"
          size="md"
        />
      </div>
      <div class="conversation-info">
        <div class="conversation-name">
          <span class="conversation-name-text">{{ conversation.name }}</span>
          <span v-if="conversation.type === 'group' && conversation.members" class="member-count">
            ({{ conversation.members.length }}人)
          </span>
        </div>
        <div class="conversation-preview" :class="{ 'has-draft': hasDraft(conversation) }">
          <template v-if="hasDraft(conversation)">
            <i class="fas fa-edit draft-icon"></i>
            <span class="conversation-preview-text" v-html="`[草稿] ${previewTextToHtml(getDraftPreview(conversation))}`"></span>
          </template>
          <template v-else>
            <span class="conversation-preview-text" v-html="formatMessagePreviewHtml(conversation.lastMessage, conversation)"></span>
          </template>
        </div>
      </div>
      <div class="conversation-meta">
        <span v-if="conversation.muted" class="muted-icon" title="免打扰"><i class="fas fa-bell-slash"></i></span>
        <div class="conversation-time">{{ formatTime(conversation.timestamp) }}</div>
        <div v-if="getUnreadCount(conversation) > 0" class="unread-badge">
          {{ getUnreadCount(conversation) > 99 ? '99+' : getUnreadCount(conversation) }}
        </div>
      </div>
    </div>
    <!-- 加载更多指示器 -->
    <div v-if="isLoading" class="loading-more">
      <i class="fas fa-spinner fa-spin"></i>
      <span>加载中...</span>
    </div>
    <div v-else-if="hasTriggeredLoadMore && !hasMore && conversations.length > 0" class="no-more">
      <span>没有更多会话了</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { DRAFT_CHANGED_EVENT, type DraftChangedDetail } from '../../utils/drafts'
import { sameConversationId } from '../../utils/conversationId'
import { resolveMessageDisplay } from '../../utils/messageDisplay'
import { previewTextToHtml } from '../../utils/emoji'
import { buildConversationBadge } from '../../utils/user'

interface User {
  id: string
  name: string
  username?: string
  type?: 'user' | 'bot' | 'system' | 'api'
}

interface LastMessage {
  content?: string
  senderId?: string
  sender?: {
    id?: string
    name?: string
    nickname?: string
    username?: string
    user?: any
  }
  type?: string
  title?: string
  file_name?: string
  file_size?: number
  miniAppData?: any
  shareData?: any
}

interface Conversation {
  id: string
  name: string
  type: string
  avatar?: string
  lastMessage?: LastMessage
  timestamp?: string | number
  unread_count?: number
  muted?: boolean
  members?: User[]
  otherMemberType?: string
  status?: 'online' | 'offline' | 'away' | 'busy'
}

const props = defineProps<{
  conversations: Conversation[]
  currentConversationId: string | null
  serverUrl: string
  hasMore?: boolean
  isLoading?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', conversation: Conversation): void
  (e: 'contextMenu', event: MouseEvent, conversation: Conversation): void
  (e: 'loadMore'): void
}>()

const listRef = ref<HTMLElement | null>(null)
const hasTriggeredLoadMore = ref(false)

// 滚动条显隐状态：Chromium 的滚动条伪元素不响应容器 :hover
// （命中原生滚动条会取消元素 hover），且伪元素 :hover 在拖动滑块后会
// 粘滞，故显隐完全由 class 驱动，CSS 侧不引用任何伪元素悬停态。
const isHovered = ref(false)
const isScrolling = ref(false)
let scrollTimer: number | undefined

const handleScroll = () => {
  // 滚动条：滚动期间保持可见，停止 600ms 后淡出
  isScrolling.value = true
  if (scrollTimer) clearTimeout(scrollTimer)
  scrollTimer = window.setTimeout(() => {
    isScrolling.value = false
  }, 600)

  if (!listRef.value || !props.hasMore || props.isLoading) return

  const { scrollTop, scrollHeight, clientHeight } = listRef.value
  const distanceToBottom = scrollHeight - scrollTop - clientHeight

  if (distanceToBottom < 200) {
    hasTriggeredLoadMore.value = true
    emit('loadMore')
  }
}

interface DraftCache {
  hasDraft: boolean
  preview: string
}

const draftsCache = ref<Map<string, DraftCache>>(new Map())

function loadDraftForConversation(id: string): DraftCache {
  try {
    const draft = localStorage.getItem(`qim_draft_${id}`)
    if (!draft) return { hasDraft: false, preview: '' }
    const { text } = JSON.parse(draft)
    if (!text) return { hasDraft: false, preview: '' }
    return {
      hasDraft: true,
      preview: text.length > 50 ? text.substring(0, 50) + '...' : text
    }
  } catch {
    return { hasDraft: false, preview: '' }
  }
}

function updateDraftsCache() {
  const newCache = new Map<string, DraftCache>()
  for (const conversation of props.conversations) {
    newCache.set(conversation.id, loadDraftForConversation(conversation.id))
  }
  draftsCache.value = newCache
}

function refreshDraft(conversationId: string) {
  draftsCache.value.set(conversationId, loadDraftForConversation(conversationId))
}

function handleStorageChange(event: StorageEvent) {
  if (event.key?.startsWith('qim_draft_')) {
    const id = event.key.replace('qim_draft_', '')
    refreshDraft(id)
  }
}

function handleDraftChanged(event: Event) {
  const conversationId = (event as CustomEvent<DraftChangedDetail>).detail?.conversationId
  if (conversationId) refreshDraft(conversationId)
}

onMounted(() => {
  updateDraftsCache()
  window.addEventListener('storage', handleStorageChange)
  window.addEventListener(DRAFT_CHANGED_EVENT, handleDraftChanged)
})

onUnmounted(() => {
  window.removeEventListener('storage', handleStorageChange)
  window.removeEventListener(DRAFT_CHANGED_EVENT, handleDraftChanged)
  if (scrollTimer) clearTimeout(scrollTimer)
})

watch(() => props.conversations, () => {
  updateDraftsCache()
}, { deep: true })

const hasDraft = (conversation: Conversation): boolean => {
  // 当前正在打开的会话不显示草稿标记，只有离开后才显示
  if (sameConversationId(conversation.id, props.currentConversationId)) return false
  return draftsCache.value.get(conversation.id)?.hasDraft ?? false
}

// 会话头像角标：统一由 buildConversationBadge 构造。
// 这里仅做一层薄封装以保持模板可读性；currentUserId 不传时工具函数会从 localStorage 读取。
const conversationBadge = (conversation: Conversation) => buildConversationBadge(conversation)

const getDraftPreview = (conversation: Conversation): string => {
  return draftsCache.value.get(conversation.id)?.preview ?? ''
}

const formatTime = (timestamp?: string | number): string => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / (1000 * 60))
  
  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins}分钟前`
  
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}小时前`
  
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays === 1) return '昨天'
  if (diffDays < 7) return `${diffDays}天前`
  
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${month}-${day}`
}

const formatMessagePreview = (lastMessage?: LastMessage, conversation?: Conversation): string => {
  if (!lastMessage) return '暂无消息'
  const previewText = resolveMessageDisplay(lastMessage).summary
  
  const isGroupChat = conversation?.type === 'group' || conversation?.type === 'discussion'
  
  if (isGroupChat && lastMessage.sender) {
    const senderName = lastMessage.sender.name || 
                       lastMessage.sender.nickname || 
                       lastMessage.sender.username || 
                       lastMessage.sender.user?.nickname || 
                       lastMessage.sender.user?.username ||
                       ''
    if (senderName) {
      return `${senderName}: ${previewText}`
    }
  }
  
  return previewText
}

const formatMessagePreviewHtml = (lastMessage?: LastMessage, conversation?: Conversation): string => {
  return previewTextToHtml(formatMessagePreview(lastMessage, conversation))
}

const getUnreadCount = (conversation: Conversation): number => {
  return conversation.unread_count ?? 0
}
</script>

<style scoped>
.conversation-list {
  width: 100%;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* 滚动条：静止时隐藏，滚动中/指针进入列表时浮现（微信/飞书同款交互）。
   滑块高度由浏览器按内容/视口比例计算，CSS 无法直接限制，此处仅控制显隐。
   显隐只由 JS 切换的 .is-hovered / .is-scrolling class 驱动——不得使用
   ::-webkit-scrollbar-thumb:hover：Chromium 的滚动条伪元素不响应容器
   :hover（命中原生滚动条会取消元素 hover），且拖动滑块后伪元素的 :hover
   会被浏览器粘滞保留（指针移开后仍视为悬停），会让滚动条拖完不消失。 */
.conversation-list::-webkit-scrollbar-track {
  background: transparent;
}

.conversation-list::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 3px;
  transition: background var(--transition-fast);
}

.conversation-list.is-hovered::-webkit-scrollbar-thumb,
.conversation-list.is-scrolling::-webkit-scrollbar-thumb {
  background: var(--color-gray-300);
}

.conversation-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: var(--panel-bg);
  cursor: pointer;
  transition: background 0.2s;
  gap: 12px;
}

.conversation-item:hover {
  background: var(--hover-color);
}

.conversation-item.active {
  background: var(--hover-color);
}

.empty-conversations {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 300px;
}

.empty-conversations .placeholder-content {
  text-align: center;
  color: var(--text-secondary, #666);
}

.empty-conversations .placeholder-content i {
  color: var(--text-tertiary, #999);
  margin-bottom: 16px;
}

.empty-conversations .placeholder-content h3 {
  margin: 0 0 8px 0;
  color: var(--text-primary, #333);
}

.empty-conversations .placeholder-content p {
  margin: 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary, #666);
}

.conversation-avatar {
  position: relative;
  flex-shrink: 0;
}

.conversation-avatar img {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.conversation-info {
  flex: 1;
  min-width: 0;
}

/* 名字 + 徽标（AI机器人/成员数）同行：名字可截断（flex ellipsis），badge flex-shrink:0 常显 */
.conversation-name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.conversation-name-text {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex: 0 1 auto;
}

.member-count {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #999);
  font-weight: normal;
  flex-shrink: 0;
}

.conversation-preview {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #666);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.conversation-preview.has-draft {
  color: var(--color-warning-500, #f59e0b);
}

.conversation-preview-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.conversation-preview-text :deep(.emoji-img) {
  width: 16px;
  height: 16px;
  vertical-align: middle;
  margin: 0 1px;
}

.draft-icon {
  font-size: var(--font-size-xxxs);
  flex-shrink: 0;
}

.conversation-meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}

.conversation-time {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary, #999);
}

.unread-badge {
  background: var(--primary-color, #1976d2);
  color: white;
  font-size: var(--font-size-tiny);
  min-width: 18px;
  height: 18px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 5px;
  font-weight: 600;
}

.muted-icon {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary, #999);
}

.loading-more,
.no-more {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  color: var(--text-secondary, #999);
  font-size: var(--font-size-xs);
  gap: 8px;
}

.loading-more i {
  font-size: var(--font-size-sm);
}
</style>
