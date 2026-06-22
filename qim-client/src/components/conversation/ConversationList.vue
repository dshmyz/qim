<template>
  <div v-if="conversations.length === 0 && !isLoading" class="empty-conversations">
    <div class="placeholder-content">
      <i class="fas fa-comments fa-4x"></i>
      <h3>暂无会话</h3>
      <p>从通讯录或群聊中发起对话吧</p>
    </div>
  </div>
  <div v-else class="conversation-list" ref="listRef" @scroll="handleScroll">
    <div
      v-for="conversation in conversations"
      :key="conversation.id"
      class="conversation-item"
      :class="{ active: conversation.id === currentConversationId }"
      @click="$emit('select', conversation)"
      @contextmenu.prevent="$emit('contextMenu', $event, conversation)"
    >
      <div class="conversation-avatar">
        <Avatar
          :src="conversation.avatar"
          :name="conversation.name || '用户'"
          :server-url="serverUrl"
          :alt="conversation.name"
          size="md"
        />
        <span v-if="conversation.type === 'group'" class="group-badge">群</span>
        <span v-if="conversation.type === 'discussion'" class="discussion-badge group-badge"><i class="fas fa-comments"></i></span>
        <span v-if="conversation.type === 'bot'" class="bot-badge"><i class="fas fa-robot"></i></span>
        <span v-if="conversation.type === 'single' && conversation.status" class="status-indicator" :class="conversation.status"></span>
      </div>
      <div class="conversation-info">
        <div class="conversation-name">
          {{ conversation.name }}
          <span v-if="conversation.type === 'group' && conversation.members" class="member-count">
            ({{ conversation.members.length }}人)
          </span>
        </div>
        <div class="conversation-preview" :class="{ 'has-draft': hasDraft(conversation) }">
          <template v-if="hasDraft(conversation)">
            <i class="fas fa-edit draft-icon"></i>
            <span class="conversation-preview-text">[草稿] {{ getDraftPreview(conversation) }}</span>
          </template>
          <template v-else>
            <span class="conversation-preview-text">{{ formatMessagePreview(conversation.lastMessage, conversation) }}</span>
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
import { decodeToPlainText } from '../../utils/mentions'

interface User {
  id: string
  name: string
  username?: string
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

const handleScroll = () => {
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
    const cached = draftsCache.value.get(conversation.id)
    if (cached) {
      newCache.set(conversation.id, cached)
    } else {
      newCache.set(conversation.id, loadDraftForConversation(conversation.id))
    }
  }
  draftsCache.value = newCache
}

function handleStorageChange(event: StorageEvent) {
  if (event.key?.startsWith('qim_draft_')) {
    const id = event.key.replace('qim_draft_', '')
    if (event.newValue) {
      draftsCache.value.set(id, loadDraftForConversation(id))
    } else {
      draftsCache.value.set(id, { hasDraft: false, preview: '' })
    }
  }
}

onMounted(() => {
  updateDraftsCache()
  window.addEventListener('storage', handleStorageChange)
})

onUnmounted(() => {
  window.removeEventListener('storage', handleStorageChange)
})

watch(() => props.conversations, () => {
  updateDraftsCache()
}, { deep: true })

const hasDraft = (conversation: Conversation): boolean => {
  return draftsCache.value.get(conversation.id)?.hasDraft ?? false
}

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
  
  let previewText = ''
  
  switch (lastMessage.type) {
    case 'text':
      previewText = lastMessage.content || '无内容'
      break
    case 'image':
      let imageName = '图片'
      try {
        const imageData = JSON.parse(lastMessage.content || '{}')
        imageName = imageData.name || imageData.fileName || lastMessage.file_name || (imageData.url ? imageData.url.split('/').pop() : '图片')
      } catch (e) {
        imageName = lastMessage.file_name || (lastMessage.content ? lastMessage.content.split('/').pop() : '图片') || '图片'
      }
      previewText = `[图片] ${imageName}`
      break
    case 'file':
      let fileName = '文件'
      try {
        const fileData = JSON.parse(lastMessage.content || '{}')
        fileName = fileData.name || fileData.fileName || lastMessage.file_name || (fileData.url ? fileData.url.split('/').pop() : '文件')
      } catch (e) {
        fileName = lastMessage.file_name || (lastMessage.content ? lastMessage.content.split('/').pop() : '文件') || '文件'
      }
      previewText = `[文件] ${fileName}`
      break
    case 'miniApp':
    case 'mini_app':
      if (lastMessage.miniAppData) {
        previewText = `[小程序] ${lastMessage.miniAppData.name || '小程序'}`
      } else {
        try {
          const data = JSON.parse(lastMessage.content || '{}')
          const miniAppName = data.data?.name || data.name || '小程序'
          previewText = `[小程序] ${miniAppName}`
        } catch {
          previewText = '[小程序]'
        }
      }
      break
    case 'share':
      if (lastMessage.shareData) {
        const shareType = lastMessage.shareData.type === 'file' ? '文件' : lastMessage.shareData.type === 'note' ? '笔记' : lastMessage.shareData.type === 'sticky' ? '便签' : '分享'
        const shareName = lastMessage.shareData.name || lastMessage.content || '分享内容'
        previewText = `[${shareType}] ${shareName}`
      } else {
        previewText = '[分享]'
      }
      break
    case 'system':
      previewText = lastMessage.content || '[系统消息]'
      break
    default:
      previewText = lastMessage.content || '无内容'
  }

  // decode mention token（@{mention:3|张三} → @张三），对非文本类型也生效
  if (previewText) {
    previewText = decodeToPlainText(previewText)
  }
  
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

const getUnreadCount = (conversation: Conversation): number => {
  return conversation.unread_count ?? 0
}
</script>

<style scoped>
.conversation-list {
  width: 100%;
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
  font-size: 14px;
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

.group-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  background: var(--primary-color, #1976d2);
  color: white;
  font-size: 10px;
  padding: 0 4px;
  border-radius: 4px;
  line-height: 1.2;
}

.discussion-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  padding: 1px 3px;
}

.bot-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  background: var(--accent-color);
  color: white;
  font-size: 10px;
  padding: 1px 3px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-indicator {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--panel-bg);
}

.status-indicator.online {
  background: #52c41a;
}

.status-indicator.offline {
  background: #d9d9d9;
}

.status-indicator.busy {
  background: #ff4d4f;
}

.conversation-info {
  flex: 1;
  min-width: 0;
}

.conversation-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-count {
  font-size: 12px;
  color: var(--text-secondary, #999);
  font-weight: normal;
}

.conversation-preview {
  font-size: 12px;
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

.draft-icon {
  font-size: 11px;
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
  font-size: 11px;
  color: var(--text-secondary, #999);
}

.unread-badge {
  background: var(--primary-color, #1976d2);
  color: white;
  font-size: 10px;
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
  font-size: 12px;
  color: var(--text-secondary, #999);
}

.loading-more,
.no-more {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  color: var(--text-secondary, #999);
  font-size: 13px;
  gap: 8px;
}

.loading-more i {
  font-size: 14px;
}
</style>
