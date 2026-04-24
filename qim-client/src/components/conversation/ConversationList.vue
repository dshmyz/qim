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
        <div class="conversation-preview">
          {{ formatMessagePreview(conversation.lastMessage, conversation) }}
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
  type?: string
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
  
  if (lastMessage.type === 'image') return '[图片]'
  if (lastMessage.type === 'file') return '[文件]'
  if (lastMessage.type === 'system') return lastMessage.content || '[系统消息]'
  
  return lastMessage.content || '暂无消息'
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
