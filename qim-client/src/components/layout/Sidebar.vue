<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import GroupList from '../shared/GroupList.vue'
import OrgTree from '../shared/OrgTree.vue'
import AppPanel from '../shared/AppPanel.vue'
import SearchResult from '../conversation/SearchResult.vue'
import ConversationList from '../conversation/ConversationList.vue'
import ChannelSidebar from '../channel/ChannelSidebar.vue'
import MessageHitsPanel from '../chat/MessageHitsPanel.vue'
import type { Conversation, User } from '../../types'
import Avatar from '../shared/Avatar.vue'
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
  type: 'user' | 'group' | 'discussion' | 'bot'
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
  unassignedUsers?: any[]
  selectedGroup: any
  selectedChannel: any
  appCategories: AppCategory[]
  searchResults: SearchResultItem[]
  collapsed?: boolean
  hasMoreConversations?: boolean
  isLoadingConversations?: boolean
  groups?: any[]
  /** 全局消息搜索命中（按会话分组渲染在 SearchResult 之后） */
  messageHits?: import('../chat/MessageHitsPanel.vue').MessageHit[]
  messageHitsTotal?: number
  messageHitsLoading?: boolean
  messageHitsExhausted?: string[]
  /** 会话元信息解析：由 Main 提供（优先后端 conversation_meta，本地列表兜底） */
  resolveMessageHitConversation?: (conversationId: number | string) => { name: string, avatar: string } | null
  /** 人员/群组搜索进行中（含防抖等待窗口）：空结果区据此显示"搜索中…"而非"无搜索结果" */
  searchLoading?: boolean
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
  (e: 'openChat', user: any): void
  (e: 'userContextMenu', event: MouseEvent, user: any): void
  (e: 'selectGroup', group: any): void
  (e: 'enterGroup', conversation: Conversation): void
  (e: 'inviteMembers', conversation: Conversation): void
  (e: 'groupContextMenu', event: MouseEvent, conversation: Conversation): void
  (e: 'selectChannel', channel: any): void
  (e: 'openApp', appId: string): void
  (e: 'openExternalApp', url: string): void
  (e: 'resetApp'): void
  (e: 'toggleCategory', categoryId: string): void
  (e: 'searchResultSelect', item: SearchResultItem): void
  (e: 'searchResultPrivateChat', item: SearchResultItem): void
  (e: 'searchResultApplyJoin', item: SearchResultItem): void
  (e: 'createChannel'): void
  (e: 'loadMoreConversations'): void
  (e: 'messageHitSelect', hit: import('../chat/MessageHitsPanel.vue').MessageHit): void
  (e: 'messageHitsLoadMore', group: import('../chat/MessageHitsPanel.vue').MessageHitGroup): void
  (e: 'messageHitsOpenConversation', conversationId: number | string): void
}>()

const userName = computed(() => {
  return props.currentUser?.nickname || props.currentUser?.username || '我的账号'
})

// 搜索结果面的滚动条显隐：静止隐藏，指针进入/滚动中浮现（与会话列表同款）。
// 显隐由 JS 切换 class 驱动——不能用 ::-webkit-scrollbar-thumb:hover，
// Chromium 伪元素不响应容器 :hover 且拖动后会粘滞。
const searchHovered = ref(false)
const searchScrolling = ref(false)
let searchScrollTimer: ReturnType<typeof setTimeout> | undefined
const handleSearchScroll = () => {
  searchScrolling.value = true
  if (searchScrollTimer) clearTimeout(searchScrollTimer)
  searchScrollTimer = setTimeout(() => {
    searchScrolling.value = false
  }, 600)
}
onUnmounted(() => {
  if (searchScrollTimer) clearTimeout(searchScrollTimer)
})

// 结果容器在查询之间复用（同一个 v-if 节点），浏览器会保留上一次的 scrollTop；
// 换词时归零，避免新结果从中段开始、前几条藏在视口上方。
const searchResultsRef = ref<HTMLElement | null>(null)
watch(() => props.searchQuery, () => {
  if (searchResultsRef.value) searchResultsRef.value.scrollTop = 0
})

// 消息命中的会话名/头像解析：默认查本地会话列表；Main 传入 resolver 时
// 以它为准（优先后端 conversation_meta，覆盖不在本地列表的历史会话）。
const resolveHitConversation = (conversationId: number | string) => {
  if (props.resolveMessageHitConversation) {
    return props.resolveMessageHitConversation(conversationId)
  }
  const conv = props.conversations.find(c => String(c.id) === String(conversationId))
  if (!conv) return null
  const avatar = (conv as any).avatar || (conv as any).other_member_avatar || ''
  return { name: conv.name || (conv as any).other_member_name || '会话', avatar }
}

// Ctrl/Cmd+K 全局聚焦搜索框
const searchInputRef = ref<HTMLInputElement | null>(null)
const focusSearch = () => {
  searchInputRef.value?.focus()
}

defineExpose({ focusSearch })
</script>

<template>
  <div class="sidebar" :class="{ 'sidebar-collapsed': collapsed }">
    <!-- 侧边栏头部 -->
    <div class="sidebar-header" v-show="!collapsed">
      <div class="user-info" @click="$emit('showUserProfile')">
        <Avatar
          :src="currentUser?.avatar"
          :name="userName"
          :server-url="serverUrl"
          :alt="userName"
          size="lg"
          class="user-avatar"
        />
        <span class="user-name">{{ userName }}</span>
      </div>
      <div class="header-actions">
        <button class="icon-btn" @click="$emit('showNotification', $event)" title="通知">
          <i class="fas fa-bell"></i>
          <span v-if="unreadNotificationCount > 0" class="notification-badge">{{ unreadNotificationCount > 99 ? '99+' : unreadNotificationCount }}</span>
        </button>
        <button class="icon-btn" data-menu-trigger @click="$emit('showActionMenu', $event)">
          <i class="fas fa-plus"></i>
        </button>
      </div>
    </div>

    <!-- 搜索框 -->
    <div class="search-box" v-show="!collapsed && activeOption !== 'channels'">
      <input
        ref="searchInputRef"
        :value="searchQuery"
        @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        type="text"
        class="search-input"
        :placeholder="activeOption === 'recent' ? '搜索用户、群组、聊天记录...' : '搜索...'"
      />
      <button
        v-if="searchQuery"
        class="search-clear-btn"
        @click="$emit('update:searchQuery', '')"
        aria-label="清空搜索"
      >
        <i class="fas fa-times-circle"></i>
      </button>
    </div>

    <!-- 侧边栏内容 -->
    <div class="sidebar-content" v-show="!collapsed">
      <KeepAlive>
        <div v-if="activeOption === 'recent'" key="recent" class="content-section recent-content-section">
          <!-- 搜索态与列表态互斥：结果面是内容区唯一的滚动容器，人员/群组与聊天记录
               作为分组排在其中，不再与会话列表在同一 flex 列里竞争高度（此前多个独立
               max-height + flex 收缩会把结果行压成半行）。 -->
          <div
            v-if="searchQuery.trim()"
            ref="searchResultsRef"
            class="search-results"
            :class="{ 'is-hovered': searchHovered, 'is-scrolling': searchScrolling }"
            @scroll.passive="handleSearchScroll"
            @mouseenter="searchHovered = true"
            @mouseleave="searchHovered = false"
          >
            <SearchResult
              v-if="searchResults.length > 0"
              :searchQuery="searchQuery"
              :searchResults="searchResults"
              @select="(item) => $emit('searchResultSelect', item)"
              @privateChat="(item) => $emit('searchResultPrivateChat', item)"
              @applyJoin="(item) => $emit('searchResultApplyJoin', item)"
            />
            <MessageHitsPanel
              v-if="messageHits?.length || messageHitsLoading"
              :hits="messageHits || []"
              :total="messageHitsTotal || 0"
              :keyword="searchQuery"
              :server-url="serverUrl"
              :resolve-conversation="resolveHitConversation"
              :exhausted-ids="messageHitsExhausted || []"
              :is-loading-more="messageHitsLoading"
              @select="(hit) => $emit('messageHitSelect', hit)"
              @loadMore="(group) => $emit('messageHitsLoadMore', group)"
              @openConversation="(cid) => $emit('messageHitsOpenConversation', cid)"
            />
            <div v-if="!searchResults.length && !messageHits?.length && !messageHitsLoading" class="search-empty">
              {{ searchLoading ? '搜索中…' : '无搜索结果' }}
            </div>
          </div>
          <!-- 会话列表只隐藏不卸载：保留实例与滚动位置，退出搜索回到原来的位置 -->
          <div v-show="!searchQuery.trim()" class="conversation-list-slot">
            <ConversationList
              :conversations="conversations"
              :currentConversationId="currentConversationId"
              :serverUrl="serverUrl"
              :hasMore="hasMoreConversations"
              :isLoading="isLoadingConversations"
              @select="(conv) => $emit('selectConversation', conv)"
              @contextMenu="(event, conv) => $emit('conversationContextMenu', event, conv)"
              @loadMore="$emit('loadMoreConversations')"
            />
          </div>
        </div>
        
        <div v-else-if="activeOption === 'org'" key="org" class="content-section">
          <OrgTree
            :orgStructure="orgStructure"
            :unassignedUsers="unassignedUsers"
            :searchQuery="searchQuery"
            @selectUser="$emit('selectUser', $event)"
            @openChat="$emit('openChat', $event)"
            @userContextMenu="(...args) => $emit('userContextMenu', ...args)"
          />
        </div>
        
        <div v-else-if="activeOption === 'groups'" key="groups" class="content-section">
          <GroupList
            :groups="groups || []"
            :selectedGroup="selectedGroup"
            :searchQuery="searchQuery"
            @select="(group) => { logger.log('Sidebar - Selected group:', group); $emit('selectGroup', group) }"
            @enter="(conv) => $emit('enterGroup', conv)"
            @showContextMenu="(event, conv) => $emit('groupContextMenu', event, conv)"
          />
        </div>
        
        <div v-else-if="activeOption === 'channels'" key="channels" class="content-section">
          <ChannelSidebar
            :currentUser="currentUser"
            @createChannel="$emit('createChannel')"
          />
        </div>
        
        <div v-else-if="activeOption === 'apps'" key="apps" class="content-section">
          <AppPanel
            :appCategories="appCategories"
            :searchQuery="searchQuery"
            @openApp="$emit('openApp', $event)"
            @openExternalApp="$emit('openExternalApp', $event)"
            @resetApp="$emit('resetApp')"
            @toggleCategory="$emit('toggleCategory', $event)"
          />
        </div>
      </KeepAlive>
    </div>
  </div>
</template>

<style scoped>
.sidebar {
  min-width: 300px;
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
  padding: 0 20px;
  height: 56px;
  background: var(--sidebar-bg);
  box-sizing: border-box;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  flex-shrink: 0;
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
  font-size: var(--font-size-sm);
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
  background: #f56c6c;
  color: white;
  font-size: var(--font-size-tiny);
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
  font-size: var(--font-size-sm);
}

.user-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.user-bio {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.search-box {
  padding: 8px 8px;
  background: transparent;
  box-shadow: var(--shadow-xs);
  position: relative;
}

.search-input {
  width: 100%;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: var(--font-size-xs);
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

.search-clear-btn {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
  font-size: var(--font-size-xxs);
  display: flex;
  align-items: center;
}

.search-clear-btn:hover {
  color: var(--text-color);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  background: var(--sidebar-bg);
  position: relative;
  user-select: none;
}

.content-section {
  height: 100%;
  overflow-y: auto;
  position: relative;
}

/* 搜索结果面：搜索态下内容区唯一的滚动容器。
   人员/群组与聊天记录作为分组排列其中，各分组不再自带 max-height 与滚动，
   高度策略只存在于这一处——这是多块并存时不再互相挤压的关键。 */
.search-results {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 8px;
}

.search-results::-webkit-scrollbar-track {
  background: transparent;
}

.search-results::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 3px;
}

.search-results.is-hovered::-webkit-scrollbar-thumb,
.search-results.is-scrolling::-webkit-scrollbar-thumb {
  background: var(--color-gray-300);
}

.search-empty {
  padding: 24px 16px;
  text-align: center;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

/* 会话列表槽位：搜索期间 v-show 隐藏而非卸载，退出搜索保留滚动位置。
   ConversationList 是 fragment 根，v-show 只能落在这一层包装上（flex 列容器，
   让列表自身的 flex:1 / 空态的 min-height 行为保持不变）。 */
.conversation-list-slot {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.recent-content-section {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* 移动设备适配 */
@media (max-width: 768px) {
  .sidebar-header {
    padding: 0 15px;
  }
  
  .icon-btn {
    width: 28px;
    height: 28px;
    font-size: var(--font-size-xs);
  }
}

@media (max-width: 1200px) {
  .sidebar-header {
    padding: 0 18px;
  }
}
</style>
