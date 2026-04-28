<template>
  <div class="conversation-list">
    <div
      v-for="conversation in conversations"
      :key="conversation.id"
      class="conversation-item"
      :class="{ active: conversation.id === currentConversationId }"
      @click="$emit('select', conversation)"
      @contextmenu.prevent="$emit('contextMenu', $event, conversation)"
    >
      <div class="conversation-avatar">
        <img :src="getAvatarUrl(conversation.avatar, conversation.name || '用户', serverUrl)" :alt="conversation.name" />
        <span v-if="conversation.type === 'group'" class="group-badge">群</span>
        <span v-if="conversation.type === 'discussion'" class="discussion-badge group-badge"><i class="fas fa-comments"></i></span>
        <span v-if="conversation.type === 'bot'" class="bot-badge"><i class="fas fa-robot"></i></span>
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
  </div>
</template>

<script setup lang="ts">
import { getAvatarUrl } from '../../utils/avatar'

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
  unreadCount?: number
  muted?: boolean
  members?: User[]
}

defineProps<{
  conversations: Conversation[]
  currentConversationId: string | null
  serverUrl: string
}>()

defineEmits<{
  (e: 'select', conversation: Conversation): void
  (e: 'contextMenu', event: MouseEvent, conversation: Conversation): void
}>()

const hasDraft = (conversation: Conversation): boolean => {
  const draft = localStorage.getItem(`qim_draft_${conversation.id}`)
  if (!draft) return false
  try {
    const { text } = JSON.parse(draft)
    return !!text
  } catch {
    return false
  }
}

const getDraftPreview = (conversation: Conversation): string => {
  const draft = localStorage.getItem(`qim_draft_${conversation.id}`)
  if (!draft) return ''
  try {
    const { text } = JSON.parse(draft)
    if (!text) return ''
    return text.length > 50 ? text.substring(0, 50) + '...' : text
  } catch {
    return ''
  }
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
  return conversation.unreadCount ?? 0
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
  background: var(--color-info-500);
  color: white;
  font-size: 10px;
  padding: 1px 3px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
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
</style>
