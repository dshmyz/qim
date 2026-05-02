<script setup lang="ts">
import { computed, ref } from 'vue'
import GroupList from '../shared/GroupList.vue'
import OrgTree from '../shared/OrgTree.vue'
import AppPanel from '../shared/AppPanel.vue'
import SearchResult from '../conversation/SearchResult.vue'
import ConversationList from '../conversation/ConversationList.vue'
import type { Conversation, User } from '../../types'
import { generateAvatar, isAbsoluteUrl } from '../../utils/avatar'
import { logger } from '../../utils/logger';

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

interface SearchResultItem {
  id: string
  name: string
  type: 'user' | 'group' | 'discussion'
  username?: string
  avatar?: string
  status?: 'online' | 'offline'
  isMember?: boolean
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
  searchResults: SearchResultItem[]
  collapsed?: boolean
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
  (e: 'searchResultSelect', item: SearchResultItem): void
  (e: 'searchResultPrivateChat', item: SearchResultItem): void
  (e: 'searchResultApplyJoin', item: SearchResultItem): void
}>()

const userAvatar = computed(() => {
  if (props.currentUser?.avatar && isAbsoluteUrl(props.currentUser.avatar)) {
    return props.currentUser.avatar
  }
  if (props.currentUser?.avatar) {
    return props.serverUrl + props.currentUser.avatar
  }
  return generateAvatar(props.currentUser?.username || '用户')
})

const userName = computed(() => {
  return props.currentUser?.nickname || props.currentUser?.username || '我的账号'
})

const filteredConversations = computed(() => {
  if (!props.searchQuery) return props.conversations
  const query = props.searchQuery.toLowerCase()
  return props.conversations.filter(conv =>
    conv.name?.toLowerCase().includes(query)
  )
})

defineExpose({})
</script>

<template>
  <div class="sidebar" :class="{ 'sidebar-collapsed': collapsed }">
    <!-- 侧边栏头部 -->
    <div class="sidebar-header" v-show="!collapsed">
      <div class="user-info" @click="$emit('showUserProfile')">
        <img
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

    <!-- 搜索框 -->
    <div class="search-box" v-show="!collapsed">
      <input
        :value="searchQuery"
        @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        type="text"
        class="search-input"
        :placeholder="activeOption === 'recent' ? '搜索用户或群组...' : '搜索...'"
      />
    </div>

    <!-- 侧边栏内容 -->
    <div class="sidebar-content" v-show="!collapsed">
      <div v-if="activeOption === 'recent'" class="content-section">
        <SearchResult
          v-if="searchQuery && searchResults.length > 0"
          :searchQuery="searchQuery"
          :searchResults="searchResults"
          @select="(item) => $emit('searchResultSelect', item)"
          @privateChat="(item) => $emit('searchResultPrivateChat', item)"
          @applyJoin="(item) => $emit('searchResultApplyJoin', item)"
        />
        <ConversationList
          :conversations="filteredConversations"
          :currentConversationId="currentConversationId"
          :serverUrl="serverUrl"
          @select="(conv) => $emit('selectConversation', conv)"
          @contextMenu="(event, conv) => $emit('conversationContextMenu', event, conv)"
        />
      </div>
      
      <div v-else-if="activeOption === 'org'" class="content-section">
        <OrgTree
          :orgStructure="orgStructure"
          @selectUser="$emit('selectUser', $event)"
          @startPrivateChat="$emit('startPrivateChat', $event)"
          @userContextMenu="(...args) => $emit('userContextMenu', ...args)"
        />
      </div>
      
      <div v-else-if="activeOption === 'groups'" class="content-section">
        <GroupList
          :conversations="conversations"
          :selectedGroup="selectedGroup"
          @select="(group) => { logger.log('Sidebar - Selected group:', group); $emit('selectGroup', group) }"
          @enter="(conv) => $emit('enterGroup', conv)"
          @invite="(conv) => $emit('inviteMembers', conv)"
          @showContextMenu="(event, conv) => $emit('groupContextMenu', event, conv)"
        />
      </div>
      
      <div v-else-if="activeOption === 'channels'" class="content-section">
        <!-- 频道功能已迁移到 Main.vue 中的新布局，这里不再渲染旧的 ChannelList -->
      </div>
      
      <div v-else-if="activeOption === 'apps'" class="content-section">
        <AppPanel
          :appCategories="appCategories"
          @openApp="$emit('openApp', $event)"
          @openExternalApp="$emit('openExternalApp', $event)"
          @resetApp="$emit('resetApp')"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.sidebar {
  min-width: 320px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: var(--shadow-sm);
  z-index: 5;
  transition: width 0.3s ease;
}

.sidebar.sidebar-collapsed {
  width: 0;
  min-width: 0;
  overflow: hidden;
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

.icon-btn:hover {
  background: var(--hover-color);
  opacity: 1;
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
  cursor: pointer;
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

.search-box {
  padding: 12px 8px;
  background: transparent;
  box-shadow: var(--shadow-xs);
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
  box-shadow: 0 0 0 2px var(--primary-light);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  background: var(--sidebar-bg);
  position: relative;
}

.content-section {
  height: 100%;
  overflow-y: auto;
  position: relative;
}

/* 移动设备适配 */
@media (max-width: 768px) {
  .sidebar-header {
    padding: 12px 15px;
  }
  
  .icon-btn {
    width: 28px;
    height: 28px;
    font-size: 13px;
  }
}

@media (max-width: 1200px) {
  .sidebar-header {
    padding: 14px 18px;
  }
}
</style>