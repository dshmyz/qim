<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <div class="user-info" @click="$emit('showUserProfile')">
        <img style="width: 55px; height: 55px;"
          :src="userAvatar"
          :alt="userName"
          class="user-avatar"
        />
        <span class="user-name">{{ userName }}</span>
      </div>
      <div class="header-actions">
        <button class="icon-btn" @click="$emit('showNotification', $event)" title="通知">
          <i class="fas fa-bell"></i>
          <span v-if="unreadNotificationCount > 0" class="notification-badge">{{ unreadNotificationCount > 99 ? '99+' : unreadNotificationCount }}</span>
        </button>
        <button class="icon-btn" @click="$emit('showActionMenu', $event)">
          <i class="fas fa-plus"></i>
        </button>
      </div>
    </div>

    <div class="search-box">
      <input
        v-model="searchQuery"
        type="text"
        :placeholder="searchPlaceholder"
        class="search-input"
        @input="$emit('update:searchQuery', searchQuery)"
      />
    </div>

    <div class="sidebar-content">
      <div v-if="activeOption === 'recent'" class="content-section">
        <ConversationList
          :conversations="filteredConversations"
          :currentConversationId="currentConversationId"
          @select="(conv) => $emit('selectConversation', conv)"
          @contextMenu="(event, conv) => $emit('conversationContextMenu', event, conv)"
        />
      </div>

      <div v-else-if="activeOption === 'org'" class="content-section">
        <OrgTree
          :orgStructure="orgStructure"
          @selectUser="(user) => $emit('selectUser', user)"
          @startPrivateChat="(user) => $emit('startPrivateChat', user)"
          @userContextMenu="(event, user) => $emit('userContextMenu', event, user)"
        />
      </div>

      <div v-else-if="activeOption === 'groups'" class="content-section">
        <GroupList
          :conversations="conversations"
          :selectedGroup="selectedGroup"
          @select="(group) => $emit('selectGroup', group)"
          @enter="$emit('enterGroup', $event)"
          @invite="$emit('inviteMembers', $event)"
          @showContextMenu="(event, conv) => $emit('groupContextMenu', event, conv)"
        />
      </div>

      <div v-else-if="activeOption === 'channels'" class="content-section">
        <ChannelList :currentUser="currentUser!" @select-channel="$emit('selectChannel', $event)" />
      </div>

      <div v-else-if="activeOption === 'apps'" class="content-section">
        <AppPanel
          :appCategories="appCategories"
          @openApp="(appId) => $emit('openApp', appId)"
          @openExternalApp="(url) => $emit('openExternalApp', url)"
          @resetApp="() => $emit('resetApp')"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import ConversationList from './ConversationList.vue'
import GroupList from './GroupList.vue'
import ChannelList from './ChannelList.vue'
import OrgTree from './OrgTree.vue'
import AppPanel from './AppPanel.vue'
import type { Conversation } from '../types'

interface User {
  id?: string
  username?: string
  nickname?: string
  avatar?: string
}

interface OrgDepartment {
  id: string
  name: string
  subDepartments: OrgDepartment[]
  employees?: any[]
}

interface AppCategory {
  id: string
  name: string
  icon?: string
  expanded: boolean
  apps: any[]
}

interface Props {
  currentUser: User
  activeOption: string
  searchQuery: string
  conversations: Conversation[]
  currentConversationId: string | null
  unreadNotificationCount: number
  serverUrl: string
  orgStructure: OrgDepartment[]
  selectedGroup: any
  selectedChannel: any
  appCategories: AppCategory[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update:searchQuery', query: string): void
  (e: 'showUserProfile'): void
  (e: 'showNotification', event: MouseEvent): void
  (e: 'showActionMenu', event: MouseEvent): void
  (e: 'selectConversation', conversation: Conversation): void
  (e: 'conversationContextMenu', event: MouseEvent, conversation: Conversation): void
  (e: 'selectUser', user: any): void
  (e: 'startPrivateChat', user: any): void
  (e: 'userContextMenu', event: MouseEvent, user: any): void
  (e: 'selectGroup', group: any): void
  (e: 'enterGroup', conversation: Conversation): void
  (e: 'inviteMembers', conversation: Conversation): void
  (e: 'groupContextMenu', event: MouseEvent, conversation: Conversation): void
  (e: 'selectChannel', channel: any): void
  (e: 'openApp', appId: string): void
  (e: 'openExternalApp', url: string): void
  (e: 'resetApp'): void
}>()

const searchQuery = computed({
  get: () => props.searchQuery,
  set: (val) => emit('update:searchQuery', val)
})

const userAvatar = computed(() => {
  if (!props.currentUser) return 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'
  if (props.currentUser.avatar?.startsWith('http')) return props.currentUser.avatar
  if (props.currentUser.avatar) return props.serverUrl + props.currentUser.avatar
  return `https://api.dicebear.com/7.x/avataaars/svg?seed=${props.currentUser.username || 'user'}`
})

const userName = computed(() => {
  return props.currentUser?.nickname || props.currentUser?.username || '我的账号'
})

const searchPlaceholder = computed(() => {
  switch (props.activeOption) {
    case 'recent': return '搜索聊天记录...'
    case 'org': return '搜索联系人...'
    case 'groups': return '搜索群聊...'
    case 'channels': return '搜索频道...'
    case 'apps': return '搜索应用...'
    default: return '搜索...'
  }
})

const filteredConversations = computed(() => {
  if (!props.searchQuery) return props.conversations
  const query = props.searchQuery.toLowerCase()
  return props.conversations.filter(conv =>
    conv.name.toLowerCase().includes(query) ||
    conv.lastMessage?.content?.toLowerCase().includes(query)
  )
})
</script>

<style scoped>
.sidebar {
  width: 320px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 1px 0 4px rgba(0, 0, 0, 0.05);
  z-index: 5;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--sidebar-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.search-box {
  padding: 12px 8px;
  background: transparent;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-name {
  font-weight: 500;
  color: var(--text-color);
  font-size: 14px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: background 0.2s;
  color: var(--text-color);
  position: relative;
  opacity: 0.7;
}

.icon-btn .notification-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 16px;
  height: 16px;
  border-radius: 8px;
  background: var(--primary-color);
  color: white;
  font-size: 10px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 4px;
  line-height: 1;
}

.icon-btn:hover {
  background: var(--hover-color);
  opacity: 1;
}

.search-box {
  padding: 12px 8px;
  background: transparent;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  outline: none;
  transition: all 0.2s;
  background: var(--panel-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.search-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(25, 118, 210, 0.1);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  background: var(--sidebar-bg);
}

.content-section {
  height: 100%;
  overflow-y: auto;
}
</style>
