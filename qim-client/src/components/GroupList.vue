<template>
  <div class="groups-list">
    <div v-for="conversation in filteredConversations" :key="conversation.id" class="group-item" :class="{ active: selectedGroup && selectedGroup.id === conversation.id }" @contextmenu.prevent="$emit('showContextMenu', $event, conversation)" @click="$emit('select', conversation)" @dblclick="$emit('enter', conversation)">
      <div class="group-avatar">
        <img :src="getAvatarUrl(conversation)" :alt="conversation.name" />
        <span class="group-badge" :class="conversation.type === 'discussion' ? 'discussion-badge' : ''">{{ conversation.type === 'group' ? '群' : '讨' }}</span>
      </div>
      <div class="group-info">
        <div class="group-name">
          {{ conversation.name }}
          <span v-if="conversation.members" class="member-count">({{ conversation.members.length }}人)</span>
          <span v-if="conversation.type === 'discussion'" class="conversation-type-tag">讨论组</span>
        </div>
      </div>
      <div v-if="conversation.unreadCount && conversation.unreadCount > 0" class="unread-badge">
        {{ conversation.unreadCount > 99 ? '99+' : conversation.unreadCount }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { generateAvatar } from '../utils/avatar'
import { API_BASE_URL } from '../config'
import type { Conversation, User } from '../types'

const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

interface Props {
  conversations: Conversation[]
  selectedGroup: Conversation | null
}

const props = defineProps<Props>()

defineEmits<{
  select: [conversation: Conversation]
  enter: [conversation: Conversation]
  showContextMenu: [event: MouseEvent, conversation: Conversation]
}>()

const filteredConversations = computed(() => {
  const filtered = props.conversations.filter(c => c.type === 'group' || c.type === 'discussion')
  console.log('GroupList - Filtered conversations:', filtered)
  console.log('GroupList - Total conversations:', props.conversations.length)
  return filtered
})

const getAvatarUrl = (conversation: Conversation) => {
  if (conversation.avatar && conversation.avatar.startsWith('http')) {
    return conversation.avatar
  }
  if (conversation.avatar) {
    return serverUrl.value + conversation.avatar
  }
  return generateAvatar(conversation.name || (conversation.type === 'group' ? '群聊' : '讨论组'))
}
</script>

<style scoped>
.groups-list {
  flex-shrink: 0;
  /* border-right: 1px solid #e8e8e8; */
  overflow-y: auto;
  padding: 16px;
  margin: 8px 8px;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  background: #fafafa;
  max-height: calc(100vh - 200px);
}

.group-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s;
  margin-bottom: 8px;
}

.group-item:hover {
  background: var(--hover-color);
}

.group-item.active {
  background: var(--hover-color);
}

.group-avatar {
  position: relative;
  margin-right: 12px;
}

.group-avatar img {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: #1976d2;
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 4px;
}

.discussion-badge {
  background: #ff9800;
}

.conversation-type-tag {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  background: #f5f5f5;
  color: #666;
  margin-left: 6px;
  font-weight: normal;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-count {
  font-size: 12px;
  color: var(--text-secondary);
  margin-left: 6px;
  font-weight: normal;
}

.unread-badge {
  display: inline-block;
  background: var(--error-color);
  color: white;
  font-size: 12px;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
  text-align: center;
  border-radius: 9px;
  padding: 0 6px;
}
</style>
