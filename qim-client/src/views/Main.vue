<template>
  <div class="im-container">
    <!-- 加载过渡动画 -->
    <div v-if="isLoading" class="loading-overlay">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">加载中...</div>
      </div>
    </div>
    <!-- 网络连接状态提示 -->
    <div v-if="showNetworkError" class="network-error">
      <div class="network-error-content">
        <i class="fas fa-exclamation-circle error-icon"></i>
        <div class="error-message">
          <p>{{ networkErrorMsg }}</p>
          <div class="error-actions">
            <button class="retry-btn" @click="handleManualReconnect">重新连接</button>
            <button class="login-btn" @click="gotoLogin">重新登录</button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 分享弹窗 -->
    <ShareModal
      :visible="showShareModal"
      :shareType="shareType"
      :users="shareUsers"
      :groups="shareGroups"
      @close="closeShareModal"
      @confirm="handleShareConfirm"
    />
    <!-- 顶部区域：窗口控制栏 -->
    <div class="top-bar">
      <div class="top-bar-left"></div>
      <WindowControls />
    </div>
    
    <!-- 左侧垂直选项栏（固定定位，从顶部延伸到底部） -->
    <SideOptions 
      v-model:activeOption="activeOption"
      @showMoreMenu="showMoreMenu"
      @showThemeMenu="showThemeMenu"
      @showSettingsMenu="showSettingsMenu"
    />
    
    <!-- 主内容区域 -->
    <div class="main-content-area">
      <!-- 主内容区域 -->
      <div class="main-content">
      <!-- 侧边栏 -->
      <Sidebar
        :currentUser="currentUser || { username: '用户', name: '用户' }"
        :activeOption="activeOption"
        :searchQuery="searchQuery"
        :conversations="filteredConversations"
        :currentConversationId="currentConversationId"
        :unreadNotificationCount="unreadNotificationCount"
        :serverUrl="serverUrl"
        :orgStructure="orgStructure"
        :selectedGroup="selectedGroup"
        :selectedChannel="selectedChannel"
        :appCategories="appCategories"
        :searchResults="searchResults"
        :collapsed="sidebarCollapsed"
        @update:searchQuery="searchQuery = $event"
        @showUserProfile="showUserProfile = true"
        @showNotification="handleNotificationCenter"
        @showActionMenu="showActionMenu"
        @selectConversation="handleConversationSelect"
        @conversationContextMenu="showContextMenu"
        @selectUser="handleUserClick"
        @startPrivateChat="startPrivateChat"
        @userContextMenu="showUserContextMenu"
        @selectGroup="(group) => { selectedGroup = group }"
        @enterGroup="handleConversationSelect"
        @inviteMembers="handleInviteMembers"
        @groupContextMenu="showGroupContextMenu"
        @selectChannel="handleChannelSelect"
        @openApp="openApp"
        @openExternalApp="openExternalApp"
        @resetApp="backToAppList"
        @searchResultSelect="handleSearchItemClick"
        @searchResultPrivateChat="startPrivateChat"
      />
      
      <!-- 聊天窗口 -->
      <ChatWindow
        ref="chatWindowRef"
        v-if="currentConversation && activeOption === 'recent'"
        :conversation="currentConversation"
        :messages="messages"
        :getReadUsers="getMessageReadUsers"
        :currentUser="currentUser.value"
        :hasMoreMessages="hasMoreMessages"
        :remoteScreenSharing="remoteScreenSharing"
        :remoteScreenUserId="remoteScreenUserId"
        :remoteScreenData="remoteScreenData"
        @send="handleSendMessage"
        @recall="handleRecallMessage"
        @inviteMembers="handleInviteMembers"
        @switchConversation="handleSwitchConversation"
        @switch-app="handleSwitchApp"
        @loadMore="handleLoadMore"
        @retry-send="handleRetrySendMessage"
        @send-screen-share-start="handleScreenShareStart"
        @send-screen-share-stop="handleScreenShareStop"
        @send-screen-share-data="handleScreenShareData"
      />
      <div v-else-if="activeOption === 'recent'" class="right-content">
        <div class="right-content-header">
          <h2>{{ getPageTitle() }}</h2>
          <button class="toggle-sidebar-btn" @click="toggleSidebar">
            <i class="fas fa-compress"></i>
          </button>
        </div>
        <div class="empty-state">
          <div class="empty-content">
            <div class="empty-icon"><i class="fas fa-comments"></i></div>
            <p>选择一个会话开始聊天</p>
          </div>
        </div>
      </div>
      
      <!-- 频道页面的右侧内容 -->
      <ChannelDetail
        v-else-if="activeOption === 'channels' && selectedChannel"
        :channel="selectedChannel"
        :isCreator="isChannelCreator(selectedChannel)"
        :formatTime="formatTime"
        :initialMessage="channelMessage"
        @toggleSidebar="toggleSidebar"
        @subscribe="subscribeChannel"
        @unsubscribe="unsubscribeChannel"
        @sendMessage="sendChannelMessage"
      />
      <div v-else-if="activeOption === 'channels' && !selectedChannel" class="right-content">
        <div class="right-content-header">
            <h2>频道</h2>
            <button class="toggle-sidebar-btn" @click="toggleSidebar">
              <i class="fas fa-compress"></i>
            </button>
        </div>
        <div class="right-content-body">
            <div class="empty-icon"><i class="fas fa-bullhorn"></i></div>
            <p>选择一个频道查看详情</p>
        </div>
      </div>
      
      <!-- 组织架构用户信息 -->
      <UserDetailPanel
        v-else-if="activeOption === 'org' && selectedUser"
        :user="selectedUser"
        :serverUrl="serverUrl"
        :getAvatarUrl="getAvatarUrl"
        @toggleSidebar="toggleSidebar"
        @privateChat="startPrivateChat"
        @showProfile="showUserProfile = true"
      />
      
      <!-- 应用面板 -->
      <AppsPanel
        v-else-if="activeOption === 'apps' && !selectedAppId"
        :recentApps="recentApps"
        :allApps="allApps"
        :pageTitle="getPageTitle()"
        @toggleSidebar="toggleSidebar"
        @openApp="openApp"
      />
      
      <!-- 文件管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '3'" class="right-content">
        <FileManagementApp @back="backToAppList" />
      </div>
      
      <!-- 笔记应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '7'" class="right-content">
        <NotesApp @back="backToAppList" />
      </div>
      
      <!-- 任务管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '5'" class="right-content">
        <TaskManagementApp @back="backToAppList" @toggleSidebar="toggleSidebar" />
      </div>
      <!-- 统计报表应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '1'" class="right-content">
        <StatisticsApp @back="backToAppList" />
      </div>
      
      <!-- 日历应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '2'" class="right-content">
        <CalendarApp @back="backToAppList" />
      </div>
      

      <!-- 便签应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '6'" class="right-content">
        <StickyNotesApp @back="backToAppList" />
      </div>
      
      <!-- 用户创建的应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'user-app' && currentUserApp" class="right-content">
        <div class="right-content-header">
          <div class="header-left">
            <button class="back-button" @click="backToAppList">
              <i class="fas fa-arrow-left"></i>
            </button>
            <h2>{{ currentUserApp.name }}</h2>
          </div>
          <button class="toggle-sidebar-btn" @click="toggleSidebar">
            <i class="fas fa-compress"></i>
          </button>
        </div>
        <div class="user-app-content">
          <div v-if="currentUserApp.url" class="user-app-iframe-container">
            <iframe 
              :src="currentUserApp.url" 
              class="user-app-iframe"
              frameborder="0"
              allowfullscreen
            ></iframe>
          </div>
          <div v-else class="empty-user-app">
            <div class="empty-icon"><i class="fas fa-link"></i></div>
            <p>该应用没有配置URL</p>
            <p class="empty-hint">请在应用管理中编辑应用，添加URL地址</p>
          </div>
        </div>
      </div>
      
      <!-- 应用管理 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'app-management'" class="right-content">
        <AppManagementApp @back="backToAppList" />
      </div>
      
      <!-- AI 助手 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'ai-assistant'" class="right-content">
        <AIAssistantApp @back="backToAppList" />
      </div>

      <!-- AI 大模型配置 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'ai-config'" class="right-content">
        <AIConfigApp @back="backToAppList" />
      </div>
      
      <!-- 短链接管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'short-link'" class="right-content">
        <ShortLinkManager @back="backToAppList" />
      </div>
      
      <!-- 群聊详情 -->
      <div v-else-if="activeOption === 'groups' && selectedGroup" class="right-content">
        <div class="right-content-header">
          <h2>{{ selectedGroup.name }}</h2>
          <button class="toggle-sidebar-btn" @click="toggleSidebar">
            <i class="fas fa-compress"></i>
          </button>
        </div>
        <GroupDetail
          :group="selectedGroup"
          @enter="handleConversationSelect($event)"
          @invite="handleInviteMembers($event)"
          @editAnnouncement="editAnnouncement"
          @showMemberContextMenu="(event, member) => showMemberContextMenu(event, member)"
          @startPrivateChat="startPrivateChat"
        />
      </div>
      
      <div v-else class="right-content">
        <div class="right-content-header">
          <h2>{{ getPageTitle() }}</h2>
        </div>
        <div class="right-content-body">
          <p>选择左侧的{{ getPageTitle() }}查看详情</p>
        </div>
      </div>
      <!-- 隐藏的便签应用实例，用于处理添加到笔记事件 -->
      <div style="display: none">
        <StickyNotesApp />
      </div>


    </div>
    </div>
    
    <!-- 右键菜单 -->
    <MainContextMenus
      :showMenu="showMenu"
      :selectedConversation="selectedConversation"
      :menuPosition="menuPosition"
      :showActionMenuFlag="showActionMenuFlag"
      :actionMenuPosition="actionMenuPosition"
      :showUserContextMenuFlag="showUserContextMenuFlag"
      :userContextMenuPosition="userContextMenuPosition"
      :selectedEmployee="selectedEmployee"
      :showMemberContextMenuFlag="showMemberContextMenuFlag"
      :memberContextMenuPosition="memberContextMenuPosition"
      :showGroupContextMenuFlag="showGroupContextMenuFlag"
      :groupContextMenuPosition="groupContextMenuPosition"
      :isGroupOwner="isGroupOwner(selectedGroup)"
      :showSettingsMenuFlag="showSettingsMenuFlag"
      :settingsMenuPosition="settingsMenuPosition"
      :showThemeMenuFlag="showThemeMenuFlag"
      :themeMenuPosition="themeMenuPosition"
      :showMoreMenuFlag="showMoreMenuFlag"
      :moreMenuPosition="moreMenuPosition"
      :currentUser="currentUser"
      @pin="handlePin"
      @mute="handleMute"
      @exitGroup="handleExitGroup"
      @remove="handleRemove"
      @createGroup="openCreateGroupModal"
      @createDiscussion="createDiscussionGroup"
      @createChannel="createChannel"
      @systemMessage="openSystemMessageModal"
      @viewUserProfile="viewUserProfile"
      @privateChat="startPrivateChat"
      @removeMember="removeMemberFromGroup"
      @viewMemberInfo="viewMemberInfo"
      @setAdmin="setAsAdmin"
      @viewGroupMembers="viewGroupMembers"
      @viewGroupInfo="viewGroupInfo"
      @addMembers="addMembersToGroup"
      @editAnnouncement="editAnnouncement"
      @dissolveGroup="dissolveGroup"
      @about="aboutApp"
      @checkUpdate="checkForUpdates"
      @settings="openSettings"
      @logout="logout"
      @setTheme="setTheme"
      @showChannels="() => { handleSidebarOptionClick('channels'); closeMoreMenu() }"
      @closeMoreMenu="closeMoreMenu"
      @closeAllMenus="handleClickOutside"
    />
    
    <!-- 用户信息弹窗 -->
    <UserProfile 
      v-if="selectedUser"
      :visible="showUserProfile" 
      :user="selectedUser" 
      @close="closeUserProfile"
      @send-private-message="startPrivateChat"
    />
    
    <!-- 个人资料弹窗 -->
    <SelfProfileModal
      :visible="showUserProfile && !selectedUser"
      :currentUser="currentUser"
      :serverUrl="serverUrl"
      :profile="userProfile"
      @close="closeUserProfile"
      @save="saveUserProfile"
      @avatarClick="triggerAvatarInput"
      @avatarChange="handleAvatarChange"
    />
    
    <!-- 创建群聊/讨论组弹窗 -->
    <CreateGroupModal 
      :visible="showCreateConversationModal"
      :type="createConversationType"
      :title="createConversationTitle"
      :members="allEmployees"
      @close="closeCreateConversationModal"
      @created="handleConversationCreated"
    />
    
    <!-- 群模态框 -->
    <GroupModals
      :showGroupMembersModal="showGroupMembersModal"
      :showGroupInfoModal="showGroupInfoModal"
      :showAddMembersModal="showAddMembersModal"
      :showEditAnnouncementModal="showEditAnnouncementModal"
      :selectedGroup="selectedGroup"
      :groupMembers="groupMembers"
      :allEmployees="allEmployees"
      :addMembersSearchQuery="addMembersSearchQuery"
      :selectedAddMembers="selectedAddMembers"
      :editAnnouncementContent="editAnnouncementContent"
      :currentUserId="currentUser?.id"
      :formatTime="formatTime"
      @closeGroupMembers="closeGroupMembersModal"
      @closeGroupInfo="closeGroupInfoModal"
      @closeAddMembers="closeAddMembersModal"
      @closeEditAnnouncement="closeEditAnnouncementModal"
      @removeMember="removeMember"
      @confirmAddMembers="confirmAddMembers"
      @saveAnnouncement="saveAnnouncement"
    />
    
    <!-- 设置、主题、更多菜单 -->
    
    <!-- 通知中心 -->
    <NotificationCenter 
      ref="notificationCenterRef"
      :show="showNotificationCenter" 
      :position="notificationCenterPosition"
      @close="closeNotificationCenter"
      @notification-click="handleNotificationClick"
    />

    <!-- 对话框和弹窗集合 -->
    <MainDialogs
      :showAboutDialog="showAboutDialog"
      :showLogoutDialog="showLogoutDialog"
      :showUpdateDialog="showUpdateDialog"
      :showSystemMessageModal="showSystemMessageModal"
      :showVoiceCallModal="showVoiceCallModal"
      :isCheckingUpdate="isCheckingUpdate"
      :isDownloading="isDownloading"
      :downloadProgress="downloadProgress"
      :hasNewVersion="hasNewVersion"
      :updateResult="updateResult"
      :callStatus="voiceCallStatus"
      :formattedDuration="formatCallDuration(voiceCallDuration)"
      :groupConversations="conversations.filter(c => c.type === 'group')"
      :allEmployees="allEmployees"
      :systemMessage="systemMessage"
      @closeAbout="closeAboutDialog"
      @cancelLogout="cancelLogout"
      @confirmLogout="handleConfirmLogout"
      @closeUpdate="closeUpdateDialog"
      @downloadUpdate="downloadUpdate"
      @closeSystemMessage="closeSystemMessageModal"
      @sendSystemMessage="sendSystemMessage"
      @endCall="endVoiceCall"
    />
  </div>
  
  <!-- 系统设置页面 -->
  <SettingsPanel
    v-if="showSettingsModal"
    :visible="showSettingsModal"
    :currentUser="currentUser"
    :serverUrl="serverUrl"
    :profile="settingsProfile"
    :messageSettings="messageSettings"
    :appearanceSettings="appearanceSettings"
    :advancedSettings="advancedSettings"
    :fileSettings="fileSettings"
    @close="closeSettingsModal"
    @save="handleSaveSettings"
    @clearCache="clearCache"
    @saveTwoFactor="saveTwoFactorSetting"
    @openSecurity="openSecuritySettings"
    @browseDirectory="browseDefaultSaveDirectory"
  />
</template>

<script setup lang="ts">
import { ref, computed, defineComponent, onMounted, onUnmounted, watch, nextTick } from 'vue'
import type { Conversation, Message, User } from '../types'
import QMessage from '../utils/qmessage'
import QMessageBox from '../utils/qmessagebox'
import axios from 'axios'
import CalendarApp from '../components/apps/CalendarApp.vue'
import StatisticsApp from '../components/apps/StatisticsApp.vue'
import StickyNotesApp from '../components/apps/StickyNotesApp.vue'
import NotesApp from '../components/apps/NotesApp.vue'
import TaskManagementApp from '../components/apps/TaskManagementApp.vue'
import FileManagementApp from '../components/apps/FileManagementApp.vue'
import AppManagementApp from '../components/apps/AppManagementApp.vue'
import AIAssistantApp from '../components/apps/AIAssistantApp.vue'
import ShortLinkManager from '../components/apps/ShortLinkManager.vue'
import AIConfigApp from '../components/apps/AIConfigApp.vue'
import * as storage from '../utils/storage'

// 声明 window.electron 变量
declare global {
  interface Window {
    electron: {
      ipcRenderer: {
        send: (channel: string, data?: any) => void
      }
    } | undefined
  }
}
import Sidebar from '../components/layout/Sidebar.vue'
import SideOptions from '../components/layout/SideOptions.vue'
import WindowControls from '../components/layout/WindowControls.vue'
import ChatWindow from '../components/chat/ChatWindow.vue'
import GroupDetail from '../components/shared/GroupDetail.vue'
import ShareModal from '../components/modals/ShareModal.vue'
import UserProfile from '../components/modals/UserProfile.vue'
import NotificationCenter from '../components/notification/NotificationCenter.vue'
import CreateGroupModal from '../components/modals/CreateGroupModal.vue'
import ChannelDetail from '../components/channel/ChannelDetail.vue'
import UserDetailPanel from '../components/user/UserDetailPanel.vue'
import AppsPanel from '../components/apps/AppsPanel.vue'
import SelfProfileModal from '../components/modals/SelfProfileModal.vue'
import GroupModals from '../components/modals/GroupModals.vue'
import MainContextMenus from '../components/menus/MainContextMenus.vue'
import MainDialogs from '../components/modals/MainDialogs.vue'
import SettingsPanel from '../components/settings/SettingsPanel.vue'
import { API_BASE_URL } from '../config'
import { generateAvatar, getAvatarUrl } from '../utils/avatar'
// @ts-ignore - WebRTC module has no type declarations
import { screenShareSender, screenShareReceiver } from '../utils/webrtc'
import { request } from '../composables/useRequest'
import { useChannel } from '../composables/useChannel'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useProcessConversation } from '../composables/useProcessConversation'
import { useSettings } from '../composables/useSettings'
import { useNetwork } from '../composables/useNetwork'
import { useWebSocketManager } from '../composables/useWebSocketManager'
import { useGroup } from '../composables/useGroup'
import { useMessageActions } from '../composables/useMessageActions'

// 服务器地址
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 显示消息提示（兼容现有 showMessage 调用方式）
const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
  const { message, type = 'info', duration } = options
  if (type === 'success') QMessage.success(message, duration)
  else if (type === 'error') QMessage.error(message, duration)
  else if (type === 'warning') QMessage.warning(message, duration)
  else QMessage.info(message, duration)
}

// 当前用户信息
const { currentUser, userProfile, syncUserProfile, getProfileAvatar } = useCurrentUser()

// 频道相关
const {
  channelMessage,
  isChannelCreator,
  subscribeChannel,
  unsubscribeChannel,
  sendChannelMessage
} = useChannel(serverUrl, currentUser)

// 会话数据处理
const { processConversation } = useProcessConversation(serverUrl, currentUser)

// 网络连接状态 - 已从 useNetwork composable 导入

// 使用 composable
const {
  notifications,
  unreadNotificationCount,
  showNotificationCenter,
  notificationCenterPosition,
  handleNotificationCenter,
  closeNotificationCenter,
  handleNotificationClick: _handleNotificationClick,
  handleNewNotification: _handleNewNotification,
  markAllNotificationsAsRead
} = useNotifications()

const {
  activeOption,
  selectedAppId,
  searchQuery,
  searchResults,
  isLoading,
  showNetworkError,
  networkErrorMsg,
  sidebarCollapsed,
  toggleSidebar,
  setLoading,
  setNetworkError
} = useAppState()

// 设置相关
const {
  currentTheme,
  settingsProfile,
  messageSettings,
  appearanceSettings,
  advancedSettings,
  fileSettings,
  loadSettings,
  saveSettings,
  clearCache,
  saveTwoFactorSetting,
  browseDefaultSaveDirectory,
  applyFontSize,
  setTheme,
  initTheme,
  updateSettingsProfile
} = useSettings(currentUser, serverUrl, request)

// 网络相关
const {
  sessionExpired,
  gotoLogin,
  cleanupNetwork
} = useNetwork()

// 手动重连：重新连接 WebSocket
const handleManualReconnect = () => {
  connectWebSocket()
}

// WebSocket 管理
const {
  ws,
  isConnected,
  connectWebSocket: connectWSManager,
  disconnectWebSocket,
  sendMessage,
  addHandler
} = useWebSocketManager(serverUrl)

// 群组相关（使用别名避免与 useUI 中的同名变量冲突）
const groupState = useGroup()
const {
  isGroupOwnerCheck
} = groupState

// 检查当前用户是否是群聊所有者
const isGroupOwner = (group: any) => {
  return isGroupOwnerCheck(group, currentUser.value?.id?.toString())
}

const ui = useUI()

// 解构 UI 状态
const {
  // 右键菜单
  showMenu,
  menuPosition,
  selectedConversation,
  showContextMenu,
  hideContextMenu,
  // 动作菜单
  showActionMenuFlag,
  actionMenuPosition,
  showActionMenu,
  hideActionMenu,
  // 用户右键菜单
  showUserContextMenuFlag,
  userContextMenuPosition,
  selectedEmployee,
  showUserContextMenu,
  hideUserContextMenu,
  // 群聊右键菜单
  showGroupContextMenuFlag,
  groupContextMenuPosition,
  showGroupContextMenu,
  closeGroupContextMenu,
  // 成员右键菜单
  showMemberContextMenuFlag,
  memberContextMenuPosition,
  selectedMember,
  showMemberContextMenu,
  hideMemberContextMenu,
  // 设置菜单
  showSettingsMenuFlag,
  settingsMenuPosition,
  showSettingsMenu,
  hideSettingsMenu,
  // 主题菜单
  showThemeMenuFlag,
  themeMenuPosition,
  showThemeMenu,
  hideThemeMenu,
  // 更多菜单
  showMoreMenuFlag,
  moreMenuPosition,
  showMoreMenu,
  closeMoreMenu,
  // 分享模态框
  showShareModal,
  shareType,
  shareUsers,
  shareGroups,
  openShareModal,
  closeShareModal,
  // 用户资料
  showUserProfile,
  selectedUser,
  openUserProfile,
  closeUserProfile,
  // 创建会话
  showCreateConversationModal,
  createConversationType,
  createConversationTitle,
  openCreateGroupModal,
  closeCreateConversationModal,
  // 系统消息
  showSystemMessageModal,
  systemMessage,
  openSystemMessageModal,
  closeSystemMessageModal,
  // 群成员
  showGroupMembersModal,
  groupMembers,
  openGroupMembersModal,
  closeGroupMembersModal,
  // 邀请成员
  showInviteMembersModal,
  openInviteMembersModal,
  closeInviteMembersModal,
  // 添加成员
  showAddMembersModal,
  addMembersSearchQuery,
  selectedAddMembers,
  openAddMembersModal,
  closeAddMembersModal,
  // 编辑群公告
  showEditAnnouncementModal,
  editAnnouncementContent,
  openEditAnnouncementModal,
  closeEditAnnouncementModal,
  // 群资料
  showGroupInfoModal,
  openGroupInfoModal,
  closeGroupInfoModal,
  // 关于对话框
  showAboutDialog,
  openAboutDialog,
  closeAboutDialog,
  // 退出登录
  showLogoutDialog,
  openLogoutDialog,
  cancelLogout,
  confirmLogout,
  // 更新对话框
  showUpdateDialog,
  isCheckingUpdate,
  isDownloading,
  downloadProgress,
  updateResult,
  hasNewVersion,
  openUpdateDialog,
  closeUpdateDialog,
  // 语音通话
  showVoiceCallModal,
  voiceCallStatus,
  voiceCallDuration,
  openVoiceCall,
  closeVoiceCall,
  // 设置
  showSettingsModal,
  activeSettingsTab,
  openSettings,
  closeSettingsModal,
  switchSettingsTab,
  handleClickOutside
} = ui

// 使用 useConversation composable
const conversation = useConversation()

// 解构会话状态
const {
  conversations,
  currentConversationId,
  messages,
  hasMoreMessages,
  selectedConversation: _selectedConversation,
  selectedGroup,
  selectedChannel,
  groups,
  currentConversation,
  handleConversationSelect: _handleConversationSelect,
  handleGroupChatSelect,
  handleChannelSelect,
  handlePin,
  handleMute,
  handleRemove,
  handleMarkRead,
  updateConversation: _updateConversation,
  addMessage: _addMessage,
  clearMessages: _clearMessages,
  addConversation,
  handleExitGroup,
  loadGroups,
  loadConversations: loadConversationsFromApi,
  resetState: _resetConversationState,
  updateConversations,
  setCurrentConversationId
} = conversation

// 通知中心组件 ref
const notificationCenterRef = ref<any>(null)

// 重写会话选择处理，包含 Main.vue 的特定逻辑
const handleConversationSelect = (conversation: Conversation) => {
  _handleConversationSelect(conversation)
  activeOption.value = 'recent'
  loadMessages(conversation.id)
  const conversationIndex = conversations.value.findIndex(c => c.id === conversation.id)
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex].unreadCount = 0
  }
  if (window.electron?.tray) {
    window.electron.tray.stopFlash()
  }
}

// 重写通知点击处理，包含 Main.vue 的特定逻辑
const handleNotificationClick = (notification: any) => {
  _handleNotificationClick(notification)
  if (notification.type === 'message' && notification.data?.conversationId) {
    activeOption.value = 'recent'
    setCurrentConversationId(notification.data.conversationId)
    loadMessages(notification.data.conversationId)
  } else if (notification.type === 'group' && notification.data?.groupId) {
    activeOption.value = 'groups'
  }
}

// 重写新通知处理，包含 Main.vue 的特定逻辑
const handleNewNotification = (notification: any) => {
  _handleNewNotification(notification)
  console.log('收到新通知:', notification)

  // 显示通知提示
  showMessage({
    message: notification.content || notification.title || '您有一条新通知',
    type: 'info',
    duration: 5000
  })

  // 将通知添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: notification.id || Date.now().toString(),
      title: notification.title || '新通知',
      content: notification.content || '',
      timestamp: notification.timestamp || Date.now(),
      read: false,
      type: notification.type || 'system',
      data: notification.data || {}
    }

    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()
  }
}

// 远程屏幕共享状态
const remoteScreenSharing = ref(false)
const remoteScreenUserId = ref<number | null>(null)
const remoteScreenData = ref<string | null>(null)

// 处理侧边栏选项按钮点击
const handleSidebarOptionClick = (option: string) => {
  activeOption.value = option
  if (sidebarCollapsed.value) {
    sidebarCollapsed.value = false
  }
}

// 会话数据已从 useConversation composable 导入

// 会话数据处理已从 useProcessConversation composable 导入

// 加载会话列表
const loadConversations = async () => {
  try {
    // 从服务器获取最新会话
    const response = await request('/api/v1/conversations')
    if (response.code === 0 && response.data) {
      const serverConversations = response.data.map((conv: any) => processConversation(conv))
      
      // 使用updateConversations方法更新会话列表
      updateConversations(serverConversations)
    } else {
      // 清空会话列表
      updateConversations([])
    }
  } catch (error) {
    console.error('加载会话失败:', error)
    // 清空会话列表
    updateConversations([])
  }
}

// 加载组织架构
const loadOrganizationTree = async () => {
  try {
    const response = await request('/api/v1/organization/tree')
    if (response.code === 0) {
      // 处理组织架构数据
      console.log('组织架构数据:', response.data)
      // 将后端返回的数据转换为前端期望的格式
      const convertDepartments = (departments) => {
        return departments.map(dept => ({
          id: dept.id ? dept.id.toString() : '',
          name: dept.name || '',
          subDepartments: dept.subDepartments ? convertDepartments(dept.subDepartments) : [],
          employees: dept.employees ? dept.employees.map(emp => ({
            id: emp.id ? emp.id.toString() : '',
            name: emp.nickname || emp.username || '',
            username: emp.username || '',
            avatar: (emp.avatar && emp.avatar.startsWith('http')) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=emp'),
            position: '', // 后端没有提供职位信息
            department: dept.name, // 添加部门信息
            status: emp.status || 'offline' // 在线状态，默认为 offline
          })) : []
        }))
      }
      // 将转换后的数据赋值给 orgStructure
      orgStructure.value = convertDepartments(response.data)
    }
  } catch (error) {
    console.error('加载组织架构失败:', error)
  }
}

// 点击组织架构中的用户
const handleUserClick = (employee: any) => {
  selectedUser.value = employee
}

// 计算部门人数和在线人数
const getDepartmentStats = (department: any) => {
  let totalCount = 0
  let onlineCount = 0

  // 统计当前部门员工
  if (department.employees) {
    totalCount += department.employees.length
    // 根据员工的 status 字段判断是否在线
    onlineCount += department.employees.filter(emp => emp.status === 'online').length
  }

  // 递归统计子部门
  if (department.subDepartments) {
    department.subDepartments.forEach(subDept => {
      const stats = getDepartmentStats(subDept)
      totalCount += stats.total
      onlineCount += stats.online
    })
  }

  return { total: totalCount, online: onlineCount }
}

// 轮询检查新消息
let messagePollingInterval: number | null = null

// 开始轮询


// 初始化数据
onMounted(async () => {
  isLoading.value = true
  try {
    // 并行加载数据
    await Promise.all([
      console.log('开始加载数据.......'),
      loadConversations(),
      loadOrganizationTree(),
      loadUserApps()
    ])
  } catch (error) {
    console.error('加载数据失败:', error)
  } finally {
    isLoading.value = false
  }
  
  // 连接WebSocket（不再使用轮询，完全依赖WebSocket）
  connectWebSocket()
  
  // 监听分享便签事件
  window.addEventListener('shareStickyNote', (event: CustomEvent) => {
    const note = event.detail
    openShareModal('sticky', note)
  })
  
  // 监听消息转发事件
  window.addEventListener('forwardMessage', (event: CustomEvent) => {
    const message = event.detail.message
    if (message) {
      openShareModal('message', message)
    }
  })
  
  // 监听文件分享事件
  window.addEventListener('openShareModal', (event: CustomEvent) => {
    const { type, data } = event.detail
    openShareModal(type, data)
  })
  
  // 监听刷新用户应用事件
  window.addEventListener('refresh-user-apps', async () => {
    await loadUserApps()
  })
})

// 导入 composables
import { useNotifications } from '../composables/useNotifications'
import { useAppState } from '../composables/useAppState'
import { useUI } from '../composables/useUI'
import { useConversation } from '../composables/useConversation'

// WebSocket 实例代理变量（已从 useWebSocketManager composable 导入）

// WebSocket连接

// 连接WebSocket
const connectWebSocket = () => {
  // 为每种消息类型添加专门的处理器
  const messageHandlers = {
    'message_read': handleReadReceipt,
    'new_message': handleNewMessage,
    'message_recalled': handleMessageRecalled,
    'message_deleted': handleMessageDeleted,
    'group_invitation': handleGroupInvitation,
    'added_to_group': handleAddedToGroup,
    'group_member_left': handleGroupMemberLeft,
    'group_member_joined': handleGroupMemberJoined,
    'group_member_role_updated': handleGroupMemberRoleUpdated,
    'group_owner_transferred': handleGroupOwnerTransferred,
    'conversation_updated': handleConversationUpdated,
    'group_announcement_updated': handleGroupAnnouncementUpdated,
    'notification': handleNotification,
    'new_notification': handleNewNotification,
    'system_message': handleSystemMessage,
    'screen-share-start': handleRemoteScreenShareStart,
    'screen-share-stop': handleRemoteScreenShareStop,
    'screen-share-data': handleRemoteScreenShareData,
    'screen-share-request': handleScreenShareRequest,
    'screen-share-accepted': handleScreenShareAccepted,
    'screen-share-rejected': handleScreenShareRejected
  }
  
  // 使用WebSocket管理器连接
  connectWSManager(
    () => handleReconnect(connectWebSocket, showNetworkError, networkErrorMsg),
    showNetworkError,
    networkErrorMsg,
    sessionExpired,
    messageHandlers
  )
}

// 注意：由于我们现在使用了按消息类型分类的处理器，这个函数不再需要
// 所有消息处理都通过addMessageHandler注册的专门处理器来完成

// 处理群聊邀请
const handleGroupInvitation = (data: any) => {
  console.log('收到群聊邀请:', data)
  // 显示群聊邀请通知
  showMessage({
    message: `您收到了加入群聊 "${data.group_name}" 的邀请`,
    type: 'info',
    duration: 5000
  })
}

// 处理已读回执事件


// 处理被添加到群聊
const handleAddedToGroup = (data: any) => {
  console.log('被添加到群聊:', data)
  
  // 构建群聊会话对象
  const groupConversation = {
    id: data.conversation_id.toString(),
    name: data.group_name,
    avatar: data.group_avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group',
    lastMessage: null,
    unreadCount: 0,
    timestamp: Date.now(),
    type: 'group',
    members: data.members || []
  }
  
  // 检查会话是否已存在
  const existingIndex = conversations.value.findIndex(c => c.id === groupConversation.id)
  if (existingIndex === -1) {
    // 添加到会话列表
    conversations.value.unshift(groupConversation)
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  } else {
    // 更新现有会话的成员列表
    const updatedConversation = {
      ...conversations.value[existingIndex],
      members: data.members || []
    }
    conversations.value.splice(existingIndex, 1, updatedConversation)
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  }
  
  // 保存会话到本地存储
  storage.saveConversations(conversations.value)
  
  // 显示通知
  showMessage({
    message: `您已被添加到群聊 "${data.group_name}"`,
    type: 'success',
    duration: 5000
  })
}

// 处理成员退出群聊
const handleGroupMemberLeft = (data: any) => {
  console.log('成员退出群聊:', data)
  
  const conversationId = data.conversation_id.toString()
  const userId = data.user_id.toString()
  
  // 更新会话列表中的群聊成员信息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      // 过滤掉退出的成员
      const updatedMembers = conversation.members.filter(member => member.id !== userId)
      
      // 创建新的会话对象，确保响应式更新
      const updatedConversation = {
        ...conversation,
        members: updatedMembers
      }
      
      // 替换会话对象，触发响应式更新
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      
      // 强制触发响应式更新
      conversations.value = [...conversations.value]
      
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
  
  // 如果是当前用户退出群聊，标记为已退出
  if (userId === currentUser.value?.id?.toString()) {
    const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
    if (conversationIndex !== -1) {
      const updatedConversation = {
        ...conversations.value[conversationIndex],
        isExited: true
      }
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      conversations.value = [...conversations.value]
      storage.saveConversations(conversations.value)
    }
  }
}

// 处理成员加入群聊
const handleGroupMemberJoined = (data: any) => {
  console.log('成员加入群聊:', data)

  const conversationId = data.conversation_id.toString()
  const newMember = data.member
  const memberName = newMember.nickname || newMember.username || (newMember.name !== undefined ? newMember.name : '未知用户')

  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]

    const memberExists = conversation.members && conversation.members.some(member => member.id === newMember.id?.toString())

    if (!memberExists) {
      const updatedMembers = [...(conversation.members || []), {
        id: newMember.id?.toString() || '',
        name: memberName,
        avatar: newMember.avatar || ''
      }]

      const updatedConversation = {
        ...conversation,
        members: updatedMembers
      }

      conversations.value.splice(conversationIndex, 1, updatedConversation)
      conversations.value = [...conversations.value]
      storage.saveConversations(conversations.value)

      if (currentConversationId.value === conversationId) {
        const systemMessage = {
          id: `system_${Date.now()}`,
          type: 'system',
          content: `${memberName} 加入了群聊`,
          timestamp: Date.now(),
          sender: {
            id: 'system',
            name: '系统',
            avatar: ''
          },
          isSelf: false,
          isRead: true
        }
        messages.value.push(systemMessage)
      }
    }
  }
}

// 处理系统消息
const handleSystemMessage = (data: any) => {
  console.log('收到系统消息:', data)
  
  // 显示系统消息通知
  showMessage({
    message: `系统消息: ${data.title}`,
    type: 'info',
    duration: 5000
  })
  
  // 将系统消息添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: Date.now().toString(),
      title: data.title,
      content: data.content,
      timestamp: Date.now(),
      read: false,
      type: 'system' as const
    }
    
    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()
    
    // 更新未读通知计数
    unreadNotificationCount.value++
  }
  
  // 可以在这里添加更新系统消息列表的逻辑
  // 例如，重新加载系统消息列表
  // loadSystemMessages()
}

// 处理远程屏幕共享开始
const handleRemoteScreenShareStart = (data: any) => {
  console.log('收到远程屏幕共享开始:', data)
  
  // 设置远程屏幕共享状态
  remoteScreenSharing.value = true
  remoteScreenUserId.value = data.user_id
  remoteScreenData.value = null
  
  // 调用 ChatWindow 组件的 receiveScreenShareStream 函数，初始化视频元素
  if (chatWindowRef.value) {
    chatWindowRef.value.receiveScreenShareStream(data)
  }
  
  showMessage({
    message: `用户 ${data.user_id} 开始共享屏幕`,
    type: 'info',
    duration: 3000
  })
}

// 处理远程屏幕共享停止
const handleRemoteScreenShareStop = (data: any) => {
  console.log('收到远程屏幕共享停止:', data)
  
  // 重置远程屏幕共享状态
  remoteScreenSharing.value = false
  remoteScreenUserId.value = null
  remoteScreenData.value = null
  
  showMessage({
    message: `用户 ${data.user_id} 停止了屏幕共享`,
    type: 'info',
    duration: 3000
  })
}

// 聊天窗口引用
const chatWindowRef = ref(null)

// 处理远程屏幕共享数据
const handleRemoteScreenShareData = (data: any) => {
  console.log('收到远程屏幕共享数据:', data)
  
  // 更新远程屏幕共享数据
  if (data.data) {
    remoteScreenData.value = data.data
    // 将屏幕共享数据传递给 ChatWindow 组件
    if (chatWindowRef.value) {
      chatWindowRef.value.handleRemoteScreenShareData(data.data)
    }
  }
}


// 处理屏幕共享请求
const handleScreenShareRequest = (data: any) => {
  console.log('收到屏幕共享请求:', data)
  
  // 显示确认对话框
  QMessageBox.confirm(
    `用户 ${data.user_id} 请求共享屏幕，是否接受？`,
    '屏幕共享请求',
    {
      confirmButtonText: '接受',
      cancelButtonText: '拒绝',
      type: 'warning'
    }
  )
  .then(() => {
    // 发送接受响应
    sendScreenShareResponse(data.conversation_id, data.user_id, 'accepted')
  })
  .catch(() => {
    // 发送拒绝响应
    sendScreenShareResponse(data.conversation_id, data.user_id, 'rejected')
  })
}

// 发送屏幕共享响应
const sendScreenShareResponse = (conversationId: number, requesterId: number, status: string) => {
  const wsMsg = {
    type: 'screen-share-response',
    data: {
      conversation_id: conversationId,
      requester_id: requesterId,
      status: status
    }
  }
  // 发送WebSocket消息
  sendMessage(wsMsg)
}

// 处理屏幕共享接受
const handleScreenShareAccepted = (data: any) => {
  console.log('屏幕共享请求被接受:', data)
  showMessage({
    message: '对方接受了屏幕共享请求',
    type: 'success',
    duration: 3000
  })
  // 调用ChatWindow组件的handleScreenShareAccepted方法，开始建立WebRTC连接
  if (chatWindowRef.value) {
    console.log('调用ChatWindow组件的handleScreenShareAccepted方法')
    chatWindowRef.value.handleScreenShareAccepted(data)
  }
  console.log('对方已接受屏幕共享请求，开始建立连接...')
}

// 处理屏幕共享拒绝
const handleScreenShareRejected = (data: any) => {
  console.log('屏幕共享请求被拒绝:', data)
  showMessage({
    message: '对方拒绝了屏幕共享请求',
    type: 'info',
    duration: 3000
  })
}


// 处理消息删除
const handleMessageDeleted = (data: any) => {
  console.log('消息被删除:', data)
  // 从消息列表中移除被删除的消息
  const index = messages.value.findIndex(msg => msg.id === data.message_id)
  if (index !== -1) {
    messages.value.splice(index, 1)
  }
}

// 处理通知
const handleNotification = (data: any) => {
  console.log('收到通知:', data)
  // 显示通知
  showMessage({
    message: data.content,
    type: 'info',
    duration: 5000
  })
  
  // 将通知添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: Date.now().toString(),
      title: data.title,
      content: data.content,
      timestamp: Date.now(),
      read: false,
      type: data.type as 'group_invitation' | 'group_member_added' | 'other'
    }
    
    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()
    
    // 更新未读通知计数
    unreadNotificationCount.value++
  }
}

// 处理会话更新
const handleConversationUpdated = (data: any) => {
  console.log('会话更新:', data)
  // 更新会话列表中的会话信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.id.toString())
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex] = {
      ...conversations.value[conversationIndex],
      ...data
    }
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
    // 保存会话到本地存储
    storage.saveConversations(conversations.value)
  }
}

// 处理群公告更新
const handleGroupAnnouncementUpdated = (data: any) => {
  console.log('群公告更新:', data)

  const conversationId = data.conversation_id.toString()
  const newAnnouncement = data.announcement || ''

  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex] = {
      ...conversations.value[conversationIndex],
      announcement: newAnnouncement
    }
    conversations.value = [...conversations.value]
    storage.saveConversations(conversations.value)

    if (currentConversationId.value === conversationId) {
      const updaterName = data.updater_name || data.operator_name || '未知用户'
      const systemMessage = {
        id: `system_${Date.now()}`,
        type: 'system',
        content: `${updaterName} 更新了群公告: ${newAnnouncement || '(无)'}`,
        timestamp: Date.now(),
        sender: {
          id: 'system',
          name: '系统',
          avatar: ''
        },
        isSelf: false,
        isRead: true
      }
      messages.value.push(systemMessage)
    }
  }
}

// 处理群成员角色更新
const handleGroupMemberRoleUpdated = (data: any) => {
  console.log('群成员角色更新:', data)
  // 更新群成员列表中的角色信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.conversation_id.toString())
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      const memberIndex = conversation.members.findIndex(m => m.id === data.user_id.toString())
      if (memberIndex !== -1) {
        conversation.members[memberIndex].role = data.role
        // 强制触发响应式更新
        conversations.value = [...conversations.value]
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
    }
  }
}

// 处理群主转让
const handleGroupOwnerTransferred = (data: any) => {
  console.log('群主转让:', data)
  // 更新群成员列表中的角色信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.conversation_id.toString())
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      // 创建新的成员数组，确保响应式更新
      const updatedMembers = conversation.members.map(member => {
        if (member.id === data.old_owner_id.toString()) {
          return { ...member, role: 'member' }
        }
        if (member.id === data.new_owner_id.toString()) {
          return { ...member, role: 'owner' }
        }
        return member
      })
      
      // 创建新的会话对象
      const updatedConversation = {
        ...conversation,
        members: updatedMembers
      }
      
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      conversations.value = [...conversations.value]
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
}

// 处理已读回执
const handleReadReceipt = (data: any) => {
  const { conversation_id, user_id } = data
  const convIdStr = conversation_id.toString()
  
  if (currentConversationId.value === convIdStr) {
    messages.value = messages.value.map(msg => {
      if (msg.isSelf) {
        return { ...msg, isRead: true }
      }
      return msg
    })
    messages.value = [...messages.value]
  }
  
  const conversationIndex = conversations.value.findIndex(c => c.id === convIdStr)
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex].unreadCount = 0
    conversations.value = [...conversations.value]
    storage.saveConversations(conversations.value)
  }
  
  console.log('处理已读回执，会话:', convIdStr, '用户:', user_id)
}

// 处理消息撤回
const handleMessageRecalled = (data: any) => {
  const messageId = data.id.toString()
  const conversationId = data.conversation_id.toString()
  
  console.log('收到消息撤回通知:', data)
  
  // 更新消息列表中的消息状态
  messages.value = messages.value.map(msg => {
    if (msg.id === messageId) {
      return { ...msg, content: '[消息已撤回]', isRecalled: true }
    }
    return msg
  })
  
  // 更新会话列表中的最后消息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.lastMessage && conversation.lastMessage.id === messageId) {
      // 创建新的会话对象，确保响应式更新
      const updatedConversation = {
        ...conversation,
        lastMessage: {
          ...conversation.lastMessage,
          content: '[消息已撤回]',
          isRecalled: true
        }
      }
      
      // 替换会话对象，触发响应式更新
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      
      // 强制触发响应式更新
      conversations.value = [...conversations.value]
    }
  }
}

// 处理新消息
const handleNewMessage = (data: any) => {
  const conversationId = data.conversation_id.toString()
  
  console.log('收到新消息:', data)
  console.log('新消息中的引用消息:', data.quoted_message)
  
  // 构建新消息对象
  let quotedMessageData = undefined
  if (data.quoted_message) {
    quotedMessageData = {
      id: data.quoted_message.id?.toString() || '',
      content: data.quoted_message.content || '',
      file_name: data.quoted_message.file_name,
      file_size: data.quoted_message.file_size,
      sender: data.quoted_message.sender ? {
        id: data.quoted_message.sender?.id?.toString() || '',
        name: data.quoted_message.sender?.nickname || data.quoted_message.sender?.username || data.quoted_message.sender?.name || '未知用户',
        avatar: data.quoted_message.sender?.avatar || ''
      } : {
        id: '',
        name: data.quoted_message.name || '未知用户',
        avatar: ''
      },
      timestamp: data.quoted_message.created_at ? new Date(data.quoted_message.created_at).getTime() : Date.now(),
      type: data.quoted_message.type || 'text',
      isSelf: data.quoted_message.sender?.id?.toString() === currentUser.value?.id?.toString()
    }
    console.log('构建的引用消息数据:', quotedMessageData)
  }
  
  // 使用 processMessage 函数处理新消息
  const newMessage = processMessage(data, conversationId)
  
  console.log('构建的新消息对象:', newMessage)
  console.log('新消息是否包含引用消息:', !!newMessage.quotedMessage)
  
  // 更新会话列表中的未读计数和最后消息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    // 创建新的会话对象，确保响应式更新
    const updatedConversation = {
      ...conversations.value[conversationIndex],
      lastMessage: newMessage,
      timestamp: newMessage.timestamp
    }
    
    // 更新未读计数（如果不是当前会话）
    if (currentConversationId.value !== conversationId) {
      updatedConversation.unreadCount = (updatedConversation.unreadCount || 0) + 1
      
      // 发送消息通知
      if (messageSettings.value.notificationsEnabled) {
        showMessage({
          message: `收到来自 ${newMessage.sender.name} 的新消息`,
          type: 'info',
          duration: 3000
        })
        
        // 播放消息提示音
        if (messageSettings.value.soundEnabled) {
          playMessageSound()
        }

        // 托盘图标闪动
        if (window.electron?.tray) {
          window.electron.tray.flash()
        }
        
        // 显示桌面通知
        if (messageSettings.value.desktopNotificationsEnabled && 'Notification' in window) {
          if (Notification.permission === 'granted') {
            new Notification('新消息', {
              body: newMessage.content,
              icon: newMessage.sender.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'
            })
          } else if (Notification.permission !== 'denied') {
            Notification.requestPermission().then(permission => {
              if (permission === 'granted') {
                new Notification('新消息', {
                  body: newMessage.content,
                  icon: newMessage.sender.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'
                })
              }
            })
          }
        }
      }
    }
    
    // 替换会话对象，触发响应式更新
    conversations.value.splice(conversationIndex, 1, updatedConversation)
    
    // 重新排序会话列表（按时间倒序）
    conversations.value.sort((a, b) => b.timestamp - a.timestamp)
    
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  } else {
    // 会话不存在于列表中，可能是被移除后又有新消息
    // 重新加载会话列表以获取最新状态
    console.log('收到已移除会话的消息，重新加载会话列表:', conversationId)
    loadConversations().then(() => {
      // 重新加载后，检查会话是否存在，如果是新消息且不是当前会话，确保未读计数正确
      const newConversationIndex = conversations.value.findIndex(c => c.id === conversationId)
      if (newConversationIndex !== -1 && currentConversationId.value !== conversationId) {
        // 确保未读计数至少为1
        const conversation = conversations.value[newConversationIndex]
        if (!conversation.unreadCount || conversation.unreadCount === 0) {
          const updatedConversation = {
            ...conversation,
            unreadCount: 1
          }
          conversations.value.splice(newConversationIndex, 1, updatedConversation)
          // 重新排序并强制更新
          conversations.value.sort((a, b) => b.timestamp - a.timestamp)
          conversations.value = [...conversations.value]
        }
        
        // 发送消息通知
        if (messageSettings.value.notificationsEnabled) {
          showMessage({
            message: `收到来自 ${newMessage.sender.name} 的新消息`,
            type: 'info',
            duration: 3000
          })
          
          // 播放消息提示音
          if (messageSettings.value.soundEnabled) {
            playMessageSound()
          }
        }
      }
    })
  }
  
  // 如果是当前会话的消息，添加到消息列表
  if (currentConversationId.value === conversationId) {
    // 检查消息是否已经存在，避免重复添加
    const messageExists = messages.value.some(msg => msg.id === newMessage.id)
    if (!messageExists) {
      messages.value.push(newMessage)
      console.log('消息列表长度:', messages.value.length)
      console.log('最后一条消息:', messages.value[messages.value.length - 1])
      
      // 滚动到底部
      nextTick(() => {
        const messageContainer = document.querySelector('.message-list')
        if (messageContainer) {
          messageContainer.scrollTop = messageContainer.scrollHeight
        }
      })
    }
  }
}

// 播放消息提示音
const playMessageSound = () => {
  try {
    // 创建音频上下文
    const audioContext = new (window.AudioContext || (window as any).webkitAudioContext)()
    
    // 创建 oscillator 节点
    const oscillator = audioContext.createOscillator()
    const gainNode = audioContext.createGain()
    
    // 连接节点
    oscillator.connect(gainNode)
    gainNode.connect(audioContext.destination)
    
    // 设置参数
    oscillator.type = 'sine'
    oscillator.frequency.setValueAtTime(800, audioContext.currentTime)
    oscillator.frequency.exponentialRampToValueAtTime(400, audioContext.currentTime + 0.1)
    
    gainNode.gain.setValueAtTime(0.1, audioContext.currentTime)
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.1)
    
    // 播放
    oscillator.start(audioContext.currentTime)
    oscillator.stop(audioContext.currentTime + 0.1)
  } catch (error) {
    console.error('播放消息提示音失败:', error)
  }
}

// 重新连接 - 已从 useNetwork composable 导入

// 跳转到登录页 - 已从 useNetwork composable 导入

// 组件销毁时关闭WebSocket连接
onUnmounted(() => {
  // 关闭WebSocket连接
  disconnectWebSocket()
  // 清除重连定时器
  cleanupNetwork()
})

// 过滤后的会话列表
const filteredConversations = computed(() => {
  let filtered = conversations.value
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(conv => 
      // 搜索会话名称
      conv.name.toLowerCase().includes(query) ||
      // 搜索最后一条消息的内容
      (conv.lastMessage?.content && conv.lastMessage.content.toLowerCase().includes(query)) ||
      // 搜索会话中的用户（针对群聊）
      (conv.members && conv.members.some(member => 
        member.name.toLowerCase().includes(query)
      )) ||
      // 搜索会话类型
      (conv.type === 'group' && '群聊'.includes(query)) ||
      (conv.type === 'single' && '用户'.includes(query))
    )
  }
  
  // 排序：置顶的会话排在前面，然后按时间戳降序
  return filtered.sort((a, b) => {
    if (a.pinned && !b.pinned) return -1
    if (!a.pinned && b.pinned) return 1
    return b.timestamp - a.timestamp
  })
})

const isSearching = ref(false)

// 处理搜索
const handleSearch = async (query) => {
  if (!query.trim()) {
    searchResults.value = []
    return
  }
  
  isSearching.value = true
  try {
    const response = await request(`/api/v1/users/search?q=${encodeURIComponent(query)}`)
    if (response.code === 0) {
      searchResults.value = response.data || []
    }
  } catch (error) {
    console.error('搜索失败:', error)
  } finally {
    isSearching.value = false
  }
}

// 处理搜索项点击
const handleSearchItemClick = (item) => {
  if (item.type === 'user') {
    startPrivateChat(item)
  } else if (item.type === 'group' || item.type === 'discussion') {
    // 如果是群聊或讨论组，选中该会话
    handleGroupChatSelect(item)
    activeOption.value = 'recent'
    loadMessages(item.id.toString())
    // 关闭搜索悬浮框
    searchQuery.value = ''
    searchResults.value = []
  }
}

// 监听搜索输入变化
watch(searchQuery, (newQuery) => {
  // 简单的防抖实现
  clearTimeout(window.searchTimeout)
  window.searchTimeout = setTimeout(() => {
    handleSearch(newQuery)
  }, 300)
})

// 消息数据
// 已处理的已读回执，用于避免重复处理
const readReceiptsProcessed = ref<Set<string> | null>(null)

// 标记消息为已读
const markMessagesAsRead = async (conversationId: string) => {
  try {
    console.log('标记消息已读，conversationId:', conversationId)
    const url = `/api/v1/conversations/${conversationId}/read`
    console.log('请求URL:', url)
    const response = await request(url, {
      method: 'POST'
    })
    console.log('标记消息已读成功:', response)
  } catch (error) {
    console.error('标记消息已读失败:', error)
    console.error('错误详情:', error)
  }
}

// 处理消息数据，确保 sender 字段正确
const processMessage = (msg: any, conversationId?: string) => {
  const messageObj: any = {
    id: msg.id ? msg.id.toString() : '',
    content: msg.content || '',
    file_name: msg.file_name,
    file_size: msg.file_size,
    sender: msg.sender ? {
      id: msg.sender.id ? msg.sender.id.toString() : '',
      name: msg.sender.nickname || msg.sender.username || msg.sender.name || msg.sender.user?.nickname || msg.sender.user?.username || '',
      avatar: msg.sender.avatar || '',
      // 保存原始 sender 对象，以便在需要时访问更多属性
      user: msg.sender
    } : {
      id: '',
      name: '',
      avatar: ''
    },
    timestamp: msg.created_at ? new Date(msg.created_at).getTime() : Date.now(),
    type: msg.type || 'text',
    isSelf: msg.sender && msg.sender.id ? msg.sender.id.toString() === currentUser.value?.id?.toString() : false,
    isRead: msg.is_read || false,
    isRecalled: msg.is_recalled || false,
    isFailed: msg.is_failed || false,
    conversationId: msg.conversation_id?.toString() || msg.conversationId || conversationId || '',
    quotedMessage: msg.quoted_message ? {
      id: msg.quoted_message.id?.toString() || '',
      content: msg.quoted_message.content || '',
      file_name: msg.quoted_message.file_name,
      file_size: msg.quoted_message.file_size,
      sender: msg.quoted_message.sender ? {
        id: msg.quoted_message.sender.id?.toString() || '',
        name: msg.quoted_message.sender?.nickname || msg.quoted_message.sender?.username || msg.quoted_message.sender?.name || msg.quoted_message.sender?.user?.nickname || msg.quoted_message.sender?.user?.username || '未知用户',
        avatar: msg.quoted_message.sender.avatar || ''
      } : {
        id: '',
        name: '未知用户',
        avatar: ''
      },
      timestamp: msg.quoted_message.created_at ? new Date(msg.quoted_message.created_at).getTime() : Date.now(),
      type: msg.quoted_message.type || 'text',
      isSelf: msg.quoted_message.sender?.id?.toString() === currentUser.value?.id?.toString()
    } : undefined,
  }
  
  // 处理分享消息（从content字段解析）
  if (msg.type === 'share' && msg.content) {
    try {
      // 尝试解析JSON
      const shareData = JSON.parse(msg.content)
      // 存储解析后的分享数据
      messageObj.shareData = shareData
    } catch (e) {
      // 如果解析失败，将原始内容作为分享数据
      messageObj.shareData = {
        type: 'text',
        content: msg.content
      }
    }
  }
  
  // 处理小程序消息
  if (msg.type === 'miniApp' && msg.content) {
    try {
      messageObj.miniAppData = JSON.parse(msg.content)
    } catch (e) {
      console.error('解析小程序数据失败:', e)
    }
  }
  
  // 处理资讯消息
  if (msg.type === 'news' && msg.content) {
    try {
      messageObj.newsData = JSON.parse(msg.content)
    } catch (e) {
      console.error('解析资讯数据失败:', e)
    }
  }
  
  return messageObj
}

// 加载会话消息
// 分页参数
// 消息分页相关变量已从 useConversation composable 导入
const messagePage = ref(1)
const messagePageSize = ref(20)
const isLoadingMessages = ref(false)

const loadMessages = async (conversationId: string, reset: boolean = true) => {
  if (isLoadingMessages.value) return
  
  if (reset) {
    messagePage.value = 1
    hasMoreMessages.value = true
  } else if (!hasMoreMessages.value) {
    return
  }
  
  isLoadingMessages.value = true
  try {
    // 从服务器获取消息，添加分页参数
    const response = await request(`/api/v1/conversations/${conversationId}/messages?page=${messagePage.value}&page_size=${messagePageSize.value}`)
    if (response.code === 0) {
      // 后端返回的数据结构是 { messages: [...], pagination: {...} }
      const messagesArray = response.data?.messages || []
      const serverMessages = Array.isArray(messagesArray) ? messagesArray.map((msg: any) => processMessage(msg)) : []
      
      // 保存当前滚动位置
      const messageListElement = document.querySelector('.message-list')
      let scrollTop = 0
      let initialHeight = 0
      if (messageListElement) {
        scrollTop = messageListElement.scrollTop
        initialHeight = messageListElement.scrollHeight
      }
      
      if (reset) {
        messages.value = serverMessages
      } else {
        messages.value = [...serverMessages, ...messages.value]
      }
      
      // 调整滚动位置，保持用户查看的内容不变
      setTimeout(() => {
        if (messageListElement) {
          const newHeight = messageListElement.scrollHeight
          const heightDiff = newHeight - initialHeight
          messageListElement.scrollTop = scrollTop + heightDiff
        }
      }, 0)
      
      // 处理分页信息
      if (response.pagination) {
        const { current_page, total_pages } = response.pagination
        // 检查是否还有更多消息
        hasMoreMessages.value = current_page < total_pages
        messagePage.value = current_page + 1
      } else {
        // 兼容旧版本，没有分页信息时的处理
        hasMoreMessages.value = serverMessages.length === messagePageSize.value
        messagePage.value++
      }
      
      const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].unreadCount = 0
      }
      
      try {
        await markMessagesAsRead(conversationId)
      } catch (error) {
        console.error('标记消息已读失败:', error)
      }
    } else {
      if (reset) {
        messages.value = []
      }
      hasMoreMessages.value = false
    }
  } catch (error) {
    console.error('加载消息失败:', error)
    if (reset) {
      messages.value = []
    }
    hasMoreMessages.value = false
  } finally {
    isLoadingMessages.value = false
  }
}

// 获取消息的已读用户列表
const getMessageReadUsers = async (messageId: string) => {
  try {
    const response = await request(`/api/v1/messages/${messageId}/read-users`)
    if (response.code === 0) {
      return response.data
    }
    return { read_users: [], total_members: 0 }
  } catch (error) {
    console.error('获取已读用户列表失败:', error)
    return { read_users: [], total_members: 0 }
  }
}

// 播放消息提示音


// 发送消息
const handleSendMessage = async (messageData: any) => {
  if (!currentConversationId.value) return
  
  // 确保currentConversationId是字符串
  const conversationId = String(currentConversationId.value)
  
  // 检查是否为模拟会话
  if (conversationId.startsWith('conv_')) {
    showMessage({ message: '会话创建失败，请重试', type: 'error' })
    return
  }
  
  console.log('发送消息时的原始数据:', messageData)
  
  // 检查WebSocket连接状态
  const isWebSocketConnected = isConnected.value
  
  try {
    let requestData: any = {}
    let messageType = 'text'
    let messageContent = ''
    let miniAppData = null
    let newsData = null
    
    // 处理JSON字符串格式的消息数据（来自小程序和资讯消息）
    if (typeof messageData === 'string') {
      try {
        const parsedData = JSON.parse(messageData)
        if (parsedData.type === 'miniApp' && parsedData.data) {
          messageType = 'miniApp'
          messageContent = JSON.stringify(parsedData.data)
          miniAppData = parsedData.data
        } else if (parsedData.type === 'news' && parsedData.data) {
          messageType = 'news'
          messageContent = JSON.stringify(parsedData.data)
          newsData = parsedData.data
        } else {
          messageType = parsedData.type || 'text'
          messageContent = parsedData.content || messageData
        }
      } catch (e) {
        // 如果解析失败，当作普通文本消息处理
        messageType = 'text'
        messageContent = messageData
      }
    } else {
      // 处理对象格式的消息数据
      messageType = messageData.type || 'text'
      messageContent = messageData.content
      miniAppData = messageData.miniAppData
      newsData = messageData.newsData
    }
    
    // 准备请求参数
    requestData = {
      type: messageType,
      content: messageContent
    }
    
    // 只有当有引用消息时才添加quoted_message_id
    if (messageData.quotedMessage && messageData.quotedMessage.id) {
      requestData.quoted_message_id = parseInt(messageData.quotedMessage.id)
      console.log('添加引用消息ID:', requestData.quoted_message_id)
    }
    
    // 添加文件消息和图片消息的额外字段
    if (messageType === 'file' || messageType === 'image') {
      requestData.file_size = messageData.fileSize
      requestData.file_name = messageData.fileName
    }
    
    console.log('发送消息的请求数据:', requestData)
    
    // 如果WebSocket连接断开，直接标记消息为发送失败
    if (!isWebSocketConnected) {
      console.error('WebSocket连接已断开，消息发送失败')
      showMessage({ message: '网络连接已断开，消息发送失败', type: 'error' })
      
      // 创建发送失败的消息对象
      const failedMessage = {
        id: Date.now().toString(),
        content: messageContent,
        file_name: messageData.fileName,
        file_size: messageData.fileSize,
        sender: {
          id: currentUser.value?.id?.toString() || '',
          name: currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: messageType,
        isSelf: true,
        isRead: false,
        isFailed: true,
        conversationId: String(currentConversationId.value),
        quotedMessage: messageData.quotedMessage,
        miniAppData: miniAppData,
        newsData: newsData,
        originalData: messageData // 保存原始消息数据，用于重新发送
      }
      
      console.log('添加发送失败的消息:', failedMessage)
      
      messages.value.push(failedMessage)
      
      // 更新会话列表中的最后消息
      const conversationIndex = conversations.value.findIndex(c => c.id.toString() === conversationId)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].lastMessage = failedMessage
        conversations.value[conversationIndex].timestamp = failedMessage.timestamp
        
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
      
      return
    }
    
    // 检查是否是机器人会话
    const currentConv = currentConversation.value
    const isBotConversation = currentConv && (currentConv.type === 'bot' || currentConv.isBot)
    
    if (isBotConversation) {
      // 机器人会话使用流式API
      await handleStreamMessage(conversationId, requestData, messageData, miniAppData, newsData)
    } else {
      // 普通会话使用普通API
      const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
        method: 'POST',
        body: JSON.stringify(requestData)
      })
      
      console.log('发送消息的响应:', response)
      
      if (response.code === 0) {
        // 直接使用客户端的引用消息数据，确保引用消息能正确显示
        const newMessage = {
          id: response.data.id?.toString() || Date.now().toString(),
          content: response.data.content,
          file_name: response.data.file_name || messageData.fileName,
          file_size: response.data.file_size || messageData.fileSize,
          sender: {
            id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
            name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
            avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
          },
          timestamp: new Date().getTime(),
          type: response.data.type || messageType,
          isSelf: true,
          isRead: false,
          quotedMessage: messageData.quotedMessage,
          miniAppData: miniAppData,
          newsData: newsData
        }
        
        console.log('添加到消息列表的新消息:', newMessage)
        
        messages.value.push(newMessage)
        
        // 更新会话列表中的最后消息
        const conversationIndex = conversations.value.findIndex(c => c.id.toString() === conversationId)
        if (conversationIndex !== -1) {
          conversations.value[conversationIndex].lastMessage = newMessage
          conversations.value[conversationIndex].timestamp = newMessage.timestamp
          
          // 保存会话到本地存储
          storage.saveConversations(conversations.value)
        }
        
        // 播放消息发送成功的提示音
        // playMessageSound() // 暂时注释掉，因为该函数未定义
      } else {
        console.error('发送消息失败:', response.message)
        
        // 根据响应码给出更友好的提示
        let errorMessage = '消息发送失败'
        if (response.code === 401) {
          errorMessage = '登录已过期，请重新登录'
          sessionExpired.value = true
        } else if (response.code === 403) {
          errorMessage = '没有发送消息的权限'
        } else if (response.code === 404) {
          errorMessage = '会话不存在或已被解散'
        } else if (response.message) {
          errorMessage = response.message
        }
        
        showMessage({ message: errorMessage, type: 'error' })
        
        // 创建发送失败的消息对象
      const failedMessage = {
        id: Date.now().toString(),
        content: messageContent,
        file_name: messageData.fileName,
        file_size: messageData.fileSize,
        sender: {
          id: currentUser.value?.id?.toString() || '',
          name: currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: messageType,
        isSelf: true,
        isRead: false,
        isFailed: true,
        conversationId: String(currentConversationId.value),
        quotedMessage: messageData.quotedMessage,
        miniAppData: miniAppData,
        newsData: newsData,
        originalData: messageData // 保存原始消息数据，用于重新发送
      }
        
        console.log('添加发送失败的消息:', failedMessage)
        
        messages.value.push(failedMessage)
        
        // 更新会话列表中的最后消息
        const conversationIndex = conversations.value.findIndex(c => c.id.toString() === String(currentConversationId.value))
        if (conversationIndex !== -1) {
          conversations.value[conversationIndex].lastMessage = failedMessage
          conversations.value[conversationIndex].timestamp = failedMessage.timestamp
          
          // 保存会话到本地存储
          storage.saveConversations(conversations.value)
        }
      }
    }
  } catch (error: any) {
    console.error('发送消息失败:', error)
    
    // 根据错误类型给出更友好的提示
    let errorMessage = '消息发送失败'
    if (error?.response?.status === 401) {
      errorMessage = '登录已过期，请重新登录'
      // 触发重新登录
      sessionExpired.value = true
    } else if (error?.message?.includes('Network') || error?.message?.includes('network')) {
      errorMessage = '网络连接失败，请检查网络'
    } else if (error?.response?.data?.message) {
      errorMessage = error.response.data.message
    } else if (error.message) {
      // 处理常见错误消息
      const msg = error.message.toLowerCase()
      if (msg.includes('unauthorized')) {
        errorMessage = '登录已过期，请重新登录'
        sessionExpired.value = true
      } else if (msg.includes('forbidden')) {
        errorMessage = '没有发送消息的权限'
      } else if (msg.includes('not found')) {
        errorMessage = '会话不存在或已被解散'
      } else {
        errorMessage = error.message
      }
    }
    
    showMessage({ message: errorMessage, type: 'error' })
    
    // 创建发送失败的消息对象
    let messageType = 'text'
    let messageContent = ''
    let miniAppData = null
    let newsData = null
    
    if (typeof messageData === 'string') {
      try {
        const parsedData = JSON.parse(messageData)
        if (parsedData.type === 'miniApp' && parsedData.data) {
          messageType = 'miniApp'
          messageContent = JSON.stringify(parsedData.data)
          miniAppData = parsedData.data
        } else if (parsedData.type === 'news' && parsedData.data) {
          messageType = 'news'
          messageContent = JSON.stringify(parsedData.data)
          newsData = parsedData.data
        } else {
          messageType = parsedData.type || 'text'
          messageContent = parsedData.content || messageData
        }
      } catch (e) {
        messageType = 'text'
        messageContent = messageData
      }
    } else {
      messageType = messageData.type || 'text'
      messageContent = messageData.content
      miniAppData = messageData.miniAppData
      newsData = messageData.newsData
    }
    
    const failedMessage = {
      id: Date.now().toString(),
      content: messageContent,
      file_name: messageData.fileName,
      file_size: messageData.fileSize,
      sender: {
        id: currentUser.value?.id?.toString() || '',
        name: currentUser.value?.nickname || currentUser.value?.username || '',
        avatar: currentUser.value?.avatar || ''
      },
      timestamp: new Date().getTime(),
      type: messageType,
      isSelf: true,
      isRead: false,
      isFailed: true,
      conversationId: String(currentConversationId.value),
      quotedMessage: messageData.quotedMessage,
      miniAppData: miniAppData,
      newsData: newsData,
      originalData: messageData // 保存原始消息数据，用于重新发送
    }
    
    console.log('添加发送失败的消息:', failedMessage)
    
    messages.value.push(failedMessage)
    
    // 更新会话列表中的最后消息
    const conversationIndex = conversations.value.findIndex(c => c.id.toString() === String(currentConversationId.value))
    if (conversationIndex !== -1) {
      conversations.value[conversationIndex].lastMessage = failedMessage
      conversations.value[conversationIndex].timestamp = failedMessage.timestamp
      
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
}

// 处理消息撤回
const handleRecallMessage = async (messageId: number) => {
  try {
    const index = messages.value.findIndex(m => m.id === messageId.toString())
    if (index !== -1) {
      messages.value[index].content = '[消息已撤回]'
      messages.value[index].isRecalled = true
    }
  } catch (error) {
    console.error('撤回消息失败:', error)
  }
}

// 处理流式消息
const handleStreamMessage = async (conversationId: string, requestData: any, messageData: any, miniAppData: any, newsData: any) => {
  try {
    // 创建一个唯一的消息ID用于流式消息
    const streamMessageId = `stream_${Date.now()}`
    
    // 创建初始的流式消息对象
    const streamMessage = {
      id: streamMessageId,
      content: '',
      sender: {
        id: '0', // 机器人ID
        name: 'AI助手',
        avatar: ''
      },
      timestamp: new Date().getTime(),
      type: 'streaming',
      isSelf: false,
      isRead: false,
      isStreaming: true,
      conversationId: conversationId
    }
    
    messages.value.push(streamMessage)
    
    // 发送流式请求
    const token = localStorage.getItem('token')
    const response = await fetch(`${serverUrl.value}/api/v1/conversations/${conversationId}/messages/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      },
      body: JSON.stringify(requestData)
    })
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    
    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }
    
    let accumulatedContent = ''
    let buffer = ''
    
    // 处理流式响应
    while (true) {
      const { done, value } = await reader.read()
      if (done) {
        break
      }
      
      // 解码字节流
      const chunk = new TextDecoder('utf-8').decode(value)
      buffer += chunk
      
      // 处理SSE格式数据
      // SSE格式: data: {json}\n\n
      const lines = buffer.split('\n')
      buffer = lines.pop() || '' // 保留不完整的行
      
      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6).trim()
          if (!data) continue
          
          try {
            // 解析统一 JSON 格式的 StreamChunk
            const chunk = JSON.parse(data)
            
            if (chunk.content) {
              accumulatedContent += chunk.content
            }
            
            if (chunk.finish === 'stop') {
              // 流式结束
              break
            }
          } catch (e) {
            // 兼容旧格式：直接作为纯文本
            accumulatedContent += data
          }
        }
      }
      
      // 更新消息内容
      const messageIndex = messages.value.findIndex(m => m.id === streamMessageId)
      if (messageIndex !== -1) {
        messages.value[messageIndex].content = accumulatedContent
        messages.value[messageIndex].isStreaming = true
      }
    }
    
    const messageIndex = messages.value.findIndex(m => m.id === streamMessageId)
    if (messageIndex !== -1) {
      messages.value[messageIndex].isStreaming = false
      messages.value[messageIndex].type = 'markdown'
    }
    
  } catch (error) {
    console.error('流式消息处理失败:', error)
    showMessage({ message: '消息发送失败: ' + (error as Error).message, type: 'error' })
  }
}

// 处理加载更多消息
const handleLoadMore = (conversationId: string) => {
  // 调用loadMessages函数加载更多消息，使用分页逻辑
  loadMessages(conversationId, false)
}

// 处理重新发送失败的消息
const handleRetrySendMessage = (failedMessage: any) => {
  console.log('重新发送失败消息:', failedMessage)
  
  // 从消息列表中移除失败的消息
  const messageIndex = messages.value.findIndex(msg => msg.id === failedMessage.id)
  if (messageIndex !== -1) {
    messages.value.splice(messageIndex, 1)
  }
  
  // 重新发送消息
  if (failedMessage.originalData) {
    handleSendMessage(failedMessage.originalData)
  } else {
    handleSendMessage(failedMessage.content)
  }
}

// 处理屏幕共享开始
const handleScreenShareStart = (data: { conversationId: number; userId: number }) => {
  console.log('===== 发送屏幕共享请求 =====', data)

  const wsMsg = {
    type: 'screen-share-request',
    data: data
  }
  console.log('发送的WebSocket消息:', wsMsg)
  sendMessage(wsMsg)
}

// 处理屏幕共享停止
const handleScreenShareStop = (data: { conversationId: number }) => {
  console.log('发送屏幕共享停止:', data)
  sendMessage({
    type: 'screen-share-stop',
    data: data
  })
}

// 处理屏幕共享数据
const handleScreenShareData = (data: { conversationId: number; data: string }) => {
  // console.log('发送屏幕共享数据:', data)
  sendMessage({
    type: 'screen-share-data',
    data: data
  })
}

// 处理会话选择
// 格式化时间
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const diffDays = Math.floor((today.getTime() - messageDate.getTime()) / (24 * 60 * 60 * 1000))
  
  if (diffDays === 0) {
    // 今天的消息，显示具体时间
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (diffDays === 1) {
    // 昨天的消息，显示"昨天 时间"
    return `昨天 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else if (diffDays < 7) {
    // 本周的消息，显示星期几和时间
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
    const weekday = weekdays[date.getDay()]
    return `${weekday} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else {
    // 更早的消息，显示具体日期
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
  }
}

// 格式化消息预览
// 获取搜索占位符
const getSearchPlaceholder = (): string => {
  switch (activeOption.value) {
    case 'recent':
      return '搜索会话'
    case 'org':
      return '搜索组织成员'
    case 'groups':
      return '搜索群聊'
    case 'apps':
      return '搜索应用'
    default:
      return '搜索'
  }
}

// 获取页面标题
const getPageTitle = (): string => {
  switch (activeOption.value) {
    case 'recent':
      return '最近会话'
    case 'org':
      return '组织架构'
    case 'groups':
      return '群聊'
    case 'channels':
      return '频道'
    case 'apps':
      return '应用'
    default:
      return 'QIM'
  }
}

// 开始私聊
const startPrivateChat = async (user: any) => {
  // 关闭搜索悬浮框
  searchQuery.value = ''
  searchResults.value = []
  
  try {
    // 检查用户ID格式
    let userId = user.id
    
    // 确保userId是数字类型
    if (typeof userId === 'string') {
      // 如果是字符串格式（如 'emp1'），尝试提取数字部分
      if (userId.startsWith('emp')) {
        userId = userId.replace('emp', '')
      }
      // 转换为数字
      userId = parseInt(userId)
    }
    
    const response = await request('/api/v1/conversations/single', {
      method: 'POST',
      body: JSON.stringify({
        user_id: userId
      })
    })
    
    if (response.code === 0) {
      // 切换到最近联系人选项卡
      activeOption.value = 'recent'
      // 重新加载会话列表
      loadConversations()
      // 选择新创建的会话
      setCurrentConversationId(response.data.id.toString())
      loadMessages(response.data.id.toString())
    }
  } catch (error) {
    console.error('创建私聊失败:', error)
    // 模拟创建会话（当API调用失败时）
    activeOption.value = 'recent'
    // 创建一个模拟的会话
    const mockConversation = {
      id: `conv_${Date.now()}`,
      name: user.name,
      avatar: user.avatar,
      lastMessage: null,
      unreadCount: 0,
      timestamp: Date.now(),
      type: 'single',
      members: [
        { id: currentUser.value?.id || 'me', name: currentUser.value?.nickname || currentUser.value?.username || '我', avatar: currentUser.value?.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=me' },
        { id: user.id, name: user.name, avatar: user.avatar }
      ]
    }
    // 添加到会话列表
    conversations.value.unshift(mockConversation)
    // 选择新创建的会话
    setCurrentConversationId(mockConversation.id)
    // 初始化消息列表
    messages.value = []
  }
  hideUserContextMenu()
}

// 发起语音通话
const startVoiceCall = async (userId: string) => {
  try {
    voiceCallStatus.value = 'calling'
    showVoiceCallModal.value = true
    
    // 模拟语音通话连接
    setTimeout(() => {
      voiceCallStatus.value = 'ringing'
    }, 1000)
    
    // 模拟对方接听
    setTimeout(() => {
      voiceCallStatus.value = 'active'
      startVoiceCallTimer()
    }, 3000)
  } catch (error) {
    console.error('发起语音通话失败:', error)
    voiceCallStatus.value = 'ended'
    showMessage({ message: '发起语音通话失败', type: 'error' })
  }
}

// 开始语音通话计时器
const startVoiceCallTimer = () => {
  voiceCallDuration.value = 0
  voiceCallTimer.value = window.setInterval(() => {
    voiceCallDuration.value++
  }, 1000)
}

// 结束语音通话
const endVoiceCall = () => {
  if (voiceCallTimer.value) {
    clearInterval(voiceCallTimer.value)
    voiceCallTimer.value = null
  }
  voiceCallStatus.value = 'ended'
  setTimeout(() => {
    showVoiceCallModal.value = false
    voiceCallStatus.value = 'idle'
    voiceCallDuration.value = 0
  }, 1000)
}

// 格式化通话时长
const formatCallDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`
}


// 触发头像选择
const triggerAvatarInput = () => {
  const input = document.querySelector('.avatar-input') as HTMLInputElement
  if (input) {
    input.click()
  }
}

// 处理头像变化
const handleAvatarChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    const file = input.files[0]
    
    // 验证文件类型
    if (!file.type.startsWith('image/')) {
      showMessage({ message: '请选择图片文件', type: 'error' })
      return
    }
    
    // 验证文件大小
    if (file.size > 5 * 1024 * 1024) { // 5MB限制
      showMessage({ message: '图片大小不能超过5MB', type: 'error' })
      return
    }
    
    try {
      // 创建FormData
      const formData = new FormData()
      formData.append('file', file)
      
      // 上传文件
      const response = await request('/api/v1/upload', {
        method: 'POST',
        headers: {
          // 注意：FormData不需要设置Content-Type
        },
        body: formData
      })
      
      if (response.code === 0 && response.data && response.data.url) {
        // 更新用户头像
        const updateResponse = await request('/api/v1/users/me', {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            avatar: response.data.url
          })
        })
        
        if (updateResponse.code === 0 && updateResponse.data) {
          // 更新当前用户信息
          if (currentUser.value) {
            currentUser.value.avatar = updateResponse.data.avatar || response.data.url
            // 更新本地存储中的用户信息
            localStorage.setItem('user', JSON.stringify(currentUser.value))
          }
          showMessage({ message: '头像更新成功', type: 'success' })
        } else {
          showMessage({ message: '头像更新失败: ' + updateResponse.message, type: 'error' })
        }
      } else {
        showMessage({ message: '文件上传失败: ' + response.message, type: 'error' })
      }
    } catch (error) {
      console.error('头像上传失败:', error)
      showMessage({ message: '头像上传失败: ' + error.message, type: 'error' })
    }
  }
}

// 保存用户资料
const saveUserProfile = async () => {
  try {
    const response = await request('/api/v1/users/me', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        nickname: userProfile.value.nickname,
        signature: userProfile.value.signature
      })
    })
    
    if (response.code === 0) {
      // 更新当前用户信息
      if (currentUser.value) {
        currentUser.value.nickname = userProfile.value.nickname
      }
      showMessage({ message: '保存成功', type: 'success' })
      closeUserProfile()
    } else {
      showMessage({ message: '保存失败: ' + response.message, type: 'error' })
    }
  } catch (error) {
    console.error('保存用户资料失败:', error)
    showMessage({ message: '保存失败: ' + error.message, type: 'error' })
  }
}


// 切换部门展开/折叠
const toggleDepartment = (departmentId: string) => {
  const index = expandedDepartments.value.indexOf(departmentId)
  if (index > -1) {
    expandedDepartments.value.splice(index, 1)
  } else {
    expandedDepartments.value.push(departmentId)
  }
}

// 切换子部门展开/折叠
const toggleSubDepartment = (departmentId: string, subDepartmentId: string) => {
  if (!expandedSubDepartments.value[departmentId]) {
    expandedSubDepartments.value[departmentId] = []
  }
  const index = expandedSubDepartments.value[departmentId].indexOf(subDepartmentId)
  if (index > -1) {
    expandedSubDepartments.value[departmentId].splice(index, 1)
  } else {
    expandedSubDepartments.value[departmentId].push(subDepartmentId)
  }
}

// 组织架构数据（从后端 API 加载）
const orgStructure = ref([])

// 展开的部门（运行时动态设置）
const expandedDepartments = ref<string[]>([])

// 展开的子部门（运行时动态设置）
const expandedSubDepartments = ref<Record<string, string[]>>({})

// 应用相关数据
// 从本地存储加载最近使用的应用
const loadRecentApps = () => {
  try {
    const storedRecentApps = localStorage.getItem('recentApps')
    if (storedRecentApps) {
      return JSON.parse(storedRecentApps)
    }
  } catch (error) {
    console.error('加载最近使用的应用失败:', error)
  }
  // 默认最近使用的应用为空
  return []
}

const recentApps = ref(loadRecentApps())

// 所有应用列表（包括内置应用、外链应用和自定义应用）
const allApps = computed(() => {
  const apps: any[] = []
  
  // 遍历所有应用分类，收集所有应用
  appCategories.value.forEach(category => {
    category.apps.forEach(app => {
      apps.push(app)
    })
  })
  
  return apps
})

// 加载用户创建的应用
const loadUserApps = async () => {
  try {
    const token = localStorage.getItem('token')
    const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL
    const response = await axios.get(`${serverUrl}/api/v1/apps`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      const userApps = response.data.data
      // 找到自定义应用分类
      const customCategory = appCategories.value.find(cat => cat.id === '3')
      if (customCategory) {
        // 清空现有的自定义应用
        customCategory.apps = []
        // 添加用户创建的应用
        userApps.forEach((app: any) => {
          customCategory.apps.push({
            id: 'user-' + app.id.toString(),
            name: app.name,
            icon: app.icon,
            url: app.url,
            openType: app.open_type || app.openType || 'in-app' // 默认为在应用内打开
          })
        })
      }
    }
  } catch (error) {
    console.error('加载用户应用失败:', error)
  }
}

const appCategories = ref([
  {
    id: '1',
    name: '内置应用',
    expanded: true,
    apps: [
      { id: '1', name: '统计报表', icon: 'fas fa-chart-bar' },
      { id: '2', name: '日历', icon: 'fas fa-calendar' },
      { id: '3', name: '文件管理', icon: 'fas fa-folder' },
      { id: '5', name: '任务管理', icon: 'fas fa-check-square' },
      { id: '6', name: '便签', icon: 'fas fa-sticky-note' },
      { id: '7', name: '笔记', icon: 'fas fa-book' },
      { id: 'ai-assistant', name: 'AI 助手', icon: 'fas fa-robot' },
      { id: 'ai-config', name: '大模型配置', icon: 'fas fa-cogs' },
      { id: 'short-link', name: '短链接管理', icon: 'fas fa-link' }
    ]
  },
  {
    id: '2',
    name: '外链应用',
    expanded: false,
    apps: [
      // 外链应用列表
    ]
  },
  {
    id: '3',
    name: '自定义应用',
    expanded: false,
    apps: [
      // 这里可以添加用户自定义的应用
    ]
  },
  {
    id: '4',
    name: '应用管理',
    expanded: false,
    apps: [
      { id: 'app-management', name: '管理应用', icon: 'fas fa-cog' }
    ]
  }
])


// 应用管理相关状态
const apps = ref<any[]>([])
const showAppModal = ref(false)
const editingApp = ref<any>(null)
const newApp = ref({
  name: '',
  icon: 'fas fa-cube',
  url: '',
  categoryId: ''
})

// 应用管理相关函数
const loadApps = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/apps`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      apps.value = response.data.data
    }
  } catch (error) {
    console.error('加载应用失败:', error)
    QMessage.error('加载应用失败，请稍后重试')
  }
}

const createApp = async () => {
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/apps`, newApp.value, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      apps.value.push(response.data.data)
      closeAppModal()
      showMessage({ message: '应用创建成功', type: 'success' })
    }
  } catch (error) {
    console.error('创建应用失败:', error)
    QMessage.error('创建应用失败，请稍后重试')
  }
}

const updateApp = async () => {
  try {
    const token = getToken()
    const response = await axios.put(`${serverUrl.value}/api/v1/apps/${editingApp.value.id}`, editingApp.value, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      const index = apps.value.findIndex(app => app.id === editingApp.value.id)
      if (index !== -1) {
        apps.value[index] = response.data.data
      }
      closeAppModal()
      showMessage({ message: '应用更新成功', type: 'success' })
    }
  } catch (error) {
    console.error('更新应用失败:', error)
    QMessage.error('更新应用失败，请稍后重试')
  }
}

const deleteApp = async (appId: string) => {
  if (confirm('确定要删除这个应用吗？')) {
    try {
      const token = getToken()
      const response = await axios.delete(`${serverUrl.value}/api/v1/apps/${appId}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      if (response.data.code === 0) {
        apps.value = apps.value.filter(app => app.id !== appId)
        showMessage({ message: '应用删除成功', type: 'success' })
      }
    } catch (error) {
      console.error('删除应用失败:', error)
      QMessage.error('删除应用失败，请稍后重试')
    }
  }
}

const openAppModal = (app?: any) => {
  if (app) {
    editingApp.value = { ...app }
  } else {
    newApp.value = {
      name: '',
      icon: 'fas fa-cube',
      url: '',
      categoryId: '1'
    }
    editingApp.value = null
  }
  showAppModal.value = true
}

const closeAppModal = () => {
  showAppModal.value = false
  editingApp.value = null
  newApp.value = {
    name: '',
    icon: 'fas fa-cube',
    url: '',
    categoryId: '1'
  }
}

// 切换应用分类展开/折叠
const toggleCategory = (categoryId: string) => {
  const category = appCategories.value.find(c => c.id === categoryId)
  if (category) {
    category.expanded = !category.expanded
  }
}

// 当前打开的用户应用
const currentUserApp = ref<any>(null)

// 应用面板的tab切换
const activeAppTab = ref('categories')

// 笔记数据


// 打开应用
// 记录最近使用的应用
const addToRecentApps = (appId: string, appName: string, appIcon: string) => {
  // 从最近使用列表中移除已存在的该应用
  recentApps.value = recentApps.value.filter(app => app.id !== appId)
  
  // 将应用添加到最近使用列表的开头
  recentApps.value.unshift({ id: appId, name: appName, icon: appIcon })
  
  // 限制最近使用的应用数量为5个
  if (recentApps.value.length > 5) {
    recentApps.value = recentApps.value.slice(0, 5)
  }
  
  // 保存到本地存储
  localStorage.setItem('recentApps', JSON.stringify(recentApps.value))
}

const openApp = async (appId: string) => {
  console.log('打开应用:', appId)
  
  // 查找应用信息
  let appName = ''
  let appIcon = ''
  let appUrl = ''
  let openType = 'in-app' // 默认为在应用内打开
  
  // 从应用分类中查找应用
  let foundApp: any = null
  for (const category of appCategories.value) {
    const app = category.apps.find(a => a.id === appId)
    if (app) {
      foundApp = app
      appName = app.name
      appIcon = app.icon
      appUrl = app.url || ''
      openType = app.openType || 'in-app'
      break
    }
  }
  
  // 记录最近使用的应用
  if (appName && appIcon) {
    addToRecentApps(appId, appName, appIcon)
  }
  
  // 特殊处理短链接应用
  if (appId === 'short-link') {
    console.log('打开短链接管理应用')
    selectedAppId.value = 'short-link'
    return
  }
  
  // 检查应用是否有URL
  if (appUrl) {
    console.log('打开带URL的应用:', appName, appUrl, 'openType:', openType)
    
    // 根据openType决定如何打开应用
    if (openType === 'external') {
      // 使用默认浏览器打开
      console.log('使用默认浏览器打开应用:', appUrl)
      if (typeof window !== 'undefined') {
        try {
          // 检查是否在Electron环境中
          if (window.electron && window.electron.shell && typeof window.electron.shell.openExternal === 'function') {
            console.log('使用Electron shell.openExternal打开链接（系统默认浏览器）')
            window.electron.shell.openExternal(appUrl)
          } else {
            // 在普通浏览器环境中，使用window.open
            console.log('使用window.open打开链接')
            window.open(appUrl, '_blank', 'noopener,noreferrer')
          }
        } catch (error) {
          console.error('打开外部应用失败:', error)
          // 作为后备，使用window.open在新窗口打开
          window.open(appUrl, '_blank', 'noopener,noreferrer')
        }
      }
    } else {
      // 在应用内打开
      console.log('在应用内打开:', appName, appUrl)
      selectedAppId.value = 'user-app'
      currentUserApp.value = {
        id: appId,
        name: appName,
        icon: appIcon,
        url: appUrl
      }
      console.log('设置selectedAppId:', selectedAppId.value)
      console.log('设置currentUserApp:', currentUserApp.value)
    }
  } else {
    // 没有URL的应用，按原来的方式处理
    selectedAppId.value = appId
    
    // 数据加载由各独立应用组件内部处理
  }
}

// 打开用户创建的应用
const openUserApp = (app: any) => {
  console.log('打开用户创建的应用:', app)
  selectedAppId.value = 'user-app'
  currentUserApp.value = app
  
  // 记录最近使用的应用
  if (app.name && app.icon) {
    addToRecentApps(app.id, app.name, app.icon)
  }
}

// 监听打开用户应用的事件
window.addEventListener('open-user-app', (event: any) => {
  const app = event.detail
  openUserApp(app)
})

// 打开外部应用
const openExternalApp = (url: string) => {
  console.log('打开外部链接:', url)
  
  // 查找外部应用信息
  let appName = ''
  let appIcon = ''
  
  // 从应用分类中查找外部应用
  for (const category of appCategories.value) {
    const app = category.apps.find(a => a.url === url)
    if (app) {
      appName = app.name
      appIcon = app.icon
      break
    }
  }
  
  // 记录最近使用的应用
  if (appName && appIcon) {
    addToRecentApps(url, appName, appIcon)
  }
  
  // 尝试使用系统默认浏览器打开链接
  if (typeof window !== 'undefined') {
    try {
      // 检查是否在Electron环境中
      if (window.electron && window.electron.shell && typeof window.electron.shell.openExternal === 'function') {
        console.log('使用Electron shell.openExternal打开链接（系统默认浏览器）')
        window.electron.shell.openExternal(url)
      } else {
        // 在非Electron环境中，使用新窗口打开
        console.log('使用window.open打开链接（新窗口）')
        window.open(url, '_blank', 'noopener,noreferrer')
      }
    } catch (error) {
      console.error('打开外部链接失败:', error)
      // 出错时回退到使用新窗口打开
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  }
}

// 创建新笔记


// 返回应用列表
const backToAppList = () => {
  selectedAppId.value = ''
}


// 打开应用管理


// 获取所有员工
const allEmployees = computed(() => {
  const employees = []
  
  // 递归收集员工
  const collectEmployees = (departments) => {
    departments.forEach(dept => {
      // 收集当前部门的员工
      if (dept.employees) {
        employees.push(...dept.employees)
      }
      // 递归处理子部门
      if (dept.subDepartments) {
        collectEmployees(dept.subDepartments)
      }
    })
  }
  
  collectEmployees(orgStructure.value)
  return employees
})

// 过滤可添加的成员列表
const filteredAddMembersEmployees = computed(() => {
  if (!addMembersSearchQuery.value) {
    return allEmployees.value
  }
  const query = addMembersSearchQuery.value.toLowerCase()
  return allEmployees.value.filter(employee => 
    employee.name.toLowerCase().includes(query)
  )
})

// 成员和群聊菜单状态已从 useUI composable 导入
// 设置相关状态已从 useUI composable 导入
// 注意: settingsProfile, messageSettings, appearanceSettings, advancedSettings, fileSettings
// 已从 useSettings composable 导入

// 以下函数已从 useUI composable 导入：showActionMenu, hideActionMenu, openCreateGroupModal, closeCreateConversationModal, openSystemMessageModal, closeSystemMessageModal, showMemberContextMenu, closeMemberContextMenu

// 处理会话创建成功
const handleConversationCreated = (newConversation: any) => {
  // 重新加载会话列表
  loadConversations()
  
  // 如果传递了新创建的会话对象，直接切换到新会话
  if (newConversation && newConversation.id) {
    setCurrentConversationId(newConversation.id)
    messages.value = []
    messagePage.value = 1
    hasMoreMessages.value = true
    
    // 加载新会话的消息
    loadMessages(newConversation.id)
  }
}

// 打开系统消息发布模态框
// 关闭系统消息发布模态框
// 发送系统消息
const sendSystemMessage = async () => {
  if (!systemMessage.value.title || !systemMessage.value.content) return
  
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/system-messages`, systemMessage.value, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (response.data.code === 0) {
      showMessage({ message: '系统消息发布成功', type: 'success' })
      closeSystemMessageModal()
    } else {
      showMessage({ message: '系统消息发布失败: ' + response.data.message, type: 'error' })
    }
  } catch (error) {
    console.error('发布系统消息失败:', error)
    showMessage({ message: '系统消息发布失败', type: 'error' })
  }
}


const createChannel = () => {
  hideActionMenu()
  // 这里可以实现创建频道的逻辑
  QMessage.info('创建频道功能开发中...')
  console.log('创建频道')
}

const createDiscussionGroup = () => {
  hideActionMenu()
  // 打开创建群聊模态框，类型为讨论组
  openCreateGroupModal('discussion')
  console.log('创建讨论组')
}

const viewUserProfile = () => {
  if (selectedEmployee.value) {
    openUserProfile(selectedEmployee.value)
  }
  hideUserContextMenu()
}

const editAnnouncement = () => {
  if (selectedGroup.value) {
    editAnnouncementContent.value = selectedGroup.value.announcement || ''
    openEditAnnouncementModal()
  }
  closeGroupContextMenu()
}

const dissolveGroup = async () => {
  if (selectedGroup.value) {
    // 调用 useGroup 中的实现
    const success = await groupState.dissolveGroup(selectedGroup.value)
    if (success) {
      // 从会话列表中移除该群聊（副作用处理）
      const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value?.id)
      if (conversationIndex !== -1) {
        conversations.value.splice(conversationIndex, 1)
        conversations.value = [...conversations.value]
      }
      selectedGroup.value = null
    }
  }
}

const saveAnnouncement = async () => {
  if (selectedGroup.value) {
    // 调用 useGroup 中的实现
    const success = await groupState.updateAnnouncement(selectedGroup.value.id, editAnnouncementContent.value)
    if (success) {
      // 更新本地群聊信息（副作用处理）
      selectedGroup.value.announcement = editAnnouncementContent.value
      // 更新会话列表中的群聊信息
      const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value?.id)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].announcement = editAnnouncementContent.value
        conversations.value = [...conversations.value]
      }
      closeEditAnnouncementModal()
    }
  }
}

const closeMemberContextMenu = () => {
  showMemberContextMenuFlag.value = false
  selectedMember.value = null
  document.removeEventListener('click', closeMemberContextMenu)
}

const removeMemberFromGroup = async () => {
  if (selectedMember.value && selectedGroup.value) {
    try {
      const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${selectedMember.value.id}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        QMessage.success('成员已成功移除')
        // 从本地群聊成员列表中移除
        const index = selectedGroup.value.members.findIndex(member => member.id === selectedMember.value.id)
        if (index > -1) {
          selectedGroup.value.members.splice(index, 1)
        }
      } else {
        QMessage.error(response.message || '移除成员失败')
      }
    } catch (error) {
      console.error('移除成员失败:', error)
      QMessage.error('网络错误，移除成员失败')
    }
  }
  closeMemberContextMenu()
}

const viewMemberInfo = () => {
  if (selectedMember.value) {
    QMessage.info(`查看${selectedMember.value.name}的资料`)
    console.log('查看成员资料:', selectedMember.value)
  }
  closeMemberContextMenu()
}

const setAsAdmin = async () => {
  if (selectedMember.value && selectedGroup.value) {
    // 调用 useGroup 中的实现
    await groupState.setAsAdmin(selectedGroup.value.id, selectedMember.value.id)
    // 更新本地成员角色（副作用处理）
    const member = selectedGroup.value.members.find(m => m.id === selectedMember.value.id)
    if (member) {
      member.role = 'admin'
    }
  }
  closeMemberContextMenu()
}

const viewGroupMembers = () => {
  if (selectedGroup.value) {
    // 使用 useGroup 中的方法准备成员显示数据
    groupMembers.value = groupState.prepareGroupMembersForDisplay(selectedGroup.value, serverUrl.value)
    showGroupMembersModal.value = true
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

const viewGroupInfo = () => {
  if (selectedGroup.value) {
    // 显示群资料模态框
    showGroupInfoModal.value = true
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

const addMembersToGroup = () => {
  if (selectedGroup.value) {
    // 重置选择
    selectedAddMembers.value = []
    addMembersSearchQuery.value = ''
    // 打开添加成员模态框
    showAddMembersModal.value = true
    // 关闭群聊上下文菜单
    closeGroupContextMenu()
  }
}

// 处理邀请成员
const handleInviteMembers = (groupOrId) => {
  let group = null
  // 处理传递的是ID的情况（来自ChatWindow）
  if (typeof groupOrId === 'string') {
    group = conversations.value.find(c => c.id === groupOrId)
  } else {
    // 处理传递的是完整group对象的情况（来自GroupDetail）
    group = groupOrId
  }
  
  if (group) {
    selectedGroup.value = group
    // 重置选择
    selectedAddMembers.value = []
    addMembersSearchQuery.value = ''
    // 打开添加成员模态框
    showAddMembersModal.value = true
  }
}

// 处理切换应用
const handleSwitchApp = (app) => {
  // 切换到指定的应用
  activeOption.value = 'apps'
  selectedAppId.value = app
  console.log('切换到应用:', app)
}

// 处理切换会话
const handleSwitchConversation = async (conversationId) => {
  // 切换到最近联系人选项卡
  activeOption.value = 'recent'
  // 重新加载会话列表
  await loadConversations()
  // 选择新会话
  setCurrentConversationId(conversationId)
  // 加载新会话的消息
  await loadMessages(conversationId)
}

// 打开分享弹窗
// 关闭分享弹窗
// 加载可分享的用户和群聊
const loadShareUsersAndGroups = async () => {
  try {
    // 加载组织架构中的用户
    const orgResponse = await request('/api/v1/organization/tree')
    if (orgResponse.code === 0) {
      const users = []
      
      // 递归提取所有用户
      const extractUsers = (departments) => {
        departments.forEach(dept => {
          if (dept.employees) {
            dept.employees.forEach(emp => {
              users.push({
                id: emp.id.toString(),
                name: emp.nickname || emp.username,
                avatar: (emp.avatar && emp.avatar.startsWith('http')) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=emp'),
                department: dept.name
              })
            })
          }
          if (dept.subDepartments) {
            extractUsers(dept.subDepartments)
          }
        })
      }
      
      extractUsers(orgResponse.data)
      shareUsers.value = users
    }
    
    // 加载群聊列表
    const convResponse = await request('/api/v1/conversations')
    if (convResponse.code === 0) {
      const groups = convResponse.data.filter(conv => conv.type === 'group')
      shareGroups.value = groups.map(group => ({
        id: group.id.toString(),
        name: group.name,
        avatar: group.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=group',
        members: group.members || []
      }))
    }
  } catch (error) {
    console.error('加载分享数据失败:', error)
  }
}

// 处理分享确认
const handleShareConfirm = async (selection) => {
  try {
    const { users, groups } = selection
    const shareData = window.shareData
    console.log('分享数据:', shareData)
    // 构建分享消息内容
    let shareContent = ''
    let shareName = ''
    switch (shareType.value) {
      case 'file':
        shareContent = `分享了文件: ${shareData.name}`
        shareName = shareData.name
        break
      case 'note':
        shareContent = `分享了笔记: ${shareData.title}`
        shareName = shareData.title
        break
      case 'sticky':
        shareContent = `分享了便签: ${shareData.title}`
        shareName = shareData.title
        break
      case 'message':
        if (shareData.type === 'text') {
          shareContent = `转发了消息: ${shareData.content.substring(0, 20)}${shareData.content.length > 20 ? '...' : ''}`
          shareName = '文本消息'
        } else if (shareData.type === 'image') {
          shareContent = '转发了图片'
          shareName = '图片消息'
        } else {
          shareContent = '转发了消息'
          shareName = '消息'
        }
        break
      default:
        shareContent = '分享了内容'
        shareName = '内容'
    }
    
    // 准备分享数据
    const shareDataObj = {
      type: shareType.value,
      id: shareData.id || shareData.messageId,
      name: shareName,
      content: shareContent,
      originalContent: shareData.content, // 存储原始内容
      originalMessage: shareType.value === 'message' ? shareData : undefined // 存储原始消息数据
    }
    
    // 发送分享消息给选择的用户
    for (const userId of users) {
      // 创建私聊会话
      const convResponse = await request('/api/v1/conversations/single', {
        method: 'POST',
        body: JSON.stringify({ user_id: parseInt(userId) })
      })
      
      if (convResponse.code === 0) {
        // 发送分享消息
        let messageData = {
          type: 'share',
          content: JSON.stringify(shareDataObj)
        }

        // 如果是文件分享，直接生成文件消息（将文件信息存储在content中）
        if (shareType.value === 'file' && shareData) {
          messageData = {
            type: 'file',
            content: JSON.stringify({
              url: shareData.url || shareData.content,
              name: shareData.name,
              size: shareData.size
            })
          }
        } else if (shareType.value === 'message' && shareDataObj.originalMessage) {
          // 如果是转发消息，根据原始消息类型发送相应的消息
          const originalMessage = shareDataObj.originalMessage
          if (originalMessage.type === 'text') {
            messageData = {
              type: 'text',
              content: `[转发] ${originalMessage.content}`
            }
          } else if (originalMessage.type === 'image' || originalMessage.type === 'file' || originalMessage.type === 'miniApp' || originalMessage.type === 'share') {
            // 对于图片、文件、小程序和分享消息，直接复制消息类型和内容
            messageData = {
              type: originalMessage.type,
              content: originalMessage.content
            }
          }
        }
        
        const messageResponse = await request(`/api/v1/conversations/${convResponse.data.id}/messages`, {
          method: 'POST',
          body: JSON.stringify(messageData)
        })
        
        // 如果当前正在查看这个会话，手动添加消息到前端列表
        if (currentConversationId.value === convResponse.data.id.toString()) {
          const newMessage = {
            id: messageResponse.data.id.toString(),
            content: messageData.content,
            sender: currentUser.value,
            timestamp: Date.now(),
            type: messageData.type,
            isSelf: true,
            isRead: false
          }
          // 检查消息是否已经存在，避免重复添加
          const messageExists = messages.value.some(msg => msg.id === newMessage.id)
          if (!messageExists) {
            messages.value.push(newMessage)
          }
        }
      }
    }
    
    // 发送分享消息给选择的群聊
    for (const groupId of groups) {
      // 发送分享消息
      let messageData = {
        type: 'share',
        content: JSON.stringify(shareDataObj)
      }

      // 如果是文件分享，直接生成文件消息（将文件信息存储在content中）
      if (shareType.value === 'file' && shareData) {
        messageData = {
          type: 'file',
          content: JSON.stringify({
            url: shareData.url || shareData.content,
            name: shareData.name,
            size: shareData.size
          })
        }
      } else if (shareType.value === 'message' && shareDataObj.originalMessage) {
        // 如果是转发消息，根据原始消息类型发送相应的消息
        const originalMessage = shareDataObj.originalMessage
        if (originalMessage.type === 'text') {
          messageData = {
            type: 'text',
            content: `[转发] ${originalMessage.content}`
          }
        } else if (originalMessage.type === 'image' || originalMessage.type === 'file' || originalMessage.type === 'miniApp' || originalMessage.type === 'share') {
          // 对于图片、文件、小程序和分享消息，直接复制消息类型和内容
          messageData = {
            type: originalMessage.type,
            content: originalMessage.content
          }
        }
      }
      
      const messageResponse = await request(`/api/v1/conversations/${parseInt(groupId)}/messages`, {
        method: 'POST',
        body: JSON.stringify(messageData)
      })
      
      // 如果当前正在查看这个会话，手动添加消息到前端列表
      if (currentConversationId.value === groupId) {
        const newMessage = {
          id: messageResponse.data.id.toString(),
          content: messageData.content,
          sender: currentUser.value,
          timestamp: Date.now(),
          type: messageData.type,
          isSelf: true,
          isRead: false
        }
        // 检查消息是否已经存在，避免重复添加
        const messageExists = messages.value.some(msg => msg.id === newMessage.id)
        if (!messageExists) {
          messages.value.push(newMessage)
        }
      }
    }
    
    showMessage({ message: '分享成功', type: 'success' })
    // 不需要手动刷新会话列表，WebSocket会自动处理新消息和会话更新
    
    // 打开第一个分享对象的聊天界面
    if (users.length > 0) {
      // 打开第一个用户的聊天界面
      const firstUserId = users[0]
      // 重新加载会话列表，确保新创建的会话存在
      await loadConversations()
      // 查找对应的会话ID
      const conversation = conversations.value.find(conv => 
        conv.type === 'single' && 
        conv.members && 
        conv.members.some(member => member.id === firstUserId)
      )
      if (conversation) {
        // 调用handleSwitchConversation，确保与正常切换会话的逻辑一致
        await handleSwitchConversation(conversation.id)
      }
    } else if (groups.length > 0) {
      // 打开第一个群聊的聊天界面
      const firstGroupId = groups[0]?.id
      // 调用handleSwitchConversation，确保与正常切换会话的逻辑一致
      if (firstGroupId) {
        await handleSwitchConversation(firstGroupId)
      }
    }
  } catch (error) {
    console.error('分享失败:', error)
    showMessage({ message: '分享失败', type: 'error' })
  } finally {
    closeShareModal()
  }
}

const exitGroup = async () => {
  if (selectedGroup.value) {
    if (confirm(`确定要退出${selectedGroup.value.name}吗？`)) {
      try {
        const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/exit`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 200) {
          // 标记群聊为已退出
          const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value.id)
          if (conversationIndex !== -1) {
            // 更新会话状态为已退出
            const updatedConversation = {
              ...conversations.value[conversationIndex],
              isExited: true
            }
            conversations.value.splice(conversationIndex, 1, updatedConversation)
            // 强制触发响应式更新
            conversations.value = [...conversations.value]
            // 保存会话到本地存储
            storage.saveConversations(conversations.value)
          }
          // 关闭群聊上下文菜单
          closeGroupContextMenu()
          showMessage({ message: '退出群聊成功', type: 'success' })
        } else {
          showMessage({ message: '退出群聊失败: ' + response.message, type: 'error' })
        }
      } catch (error) {
        console.error('退出群聊失败:', error)
        showMessage({ message: '退出群聊失败，请稍后重试', type: 'error' })
      }
    }
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

// 移除成员
const removeMember = async (member) => {
  if (selectedGroup.value) {
    if (window.confirm(`确定要将${member.name}移出群聊吗？`)) {
      // 调用 useGroup 中的实现
      const success = await groupState.removeGroupMember(selectedGroup.value.id, member.id)
      if (success) {
        // 更新群成员列表（副作用处理）
        groupMembers.value = groupMembers.value.filter(m => m.id !== member.id)
        // 更新选中群的成员列表
        if (selectedGroup.value.members) {
          selectedGroup.value.members = selectedGroup.value.members.filter(m => m.id !== member.id)
        }
        // 更新会话列表中对应群聊的成员数
        const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value?.id)
        if (conversationIndex !== -1) {
          conversations.value[conversationIndex].members = selectedGroup.value?.members
          // 强制触发响应式更新
          conversations.value = [...conversations.value]
        }
      }
    }
  }
}

// 关闭添加成员模态框
// 关闭群成员模态框
// 关闭群资料模态框
// 切换成员选择状态
const toggleAddMember = (employee: any) => {
  const index = selectedAddMembers.value.findIndex(m => m.id === employee.id)
  if (index > -1) {
    selectedAddMembers.value.splice(index, 1)
  } else {
    selectedAddMembers.value.push(employee)
  }
}

// 确认添加成员
const confirmAddMembers = async (members: any[]) => {
  if (!selectedGroup.value || !members || members.length === 0) {
    return
  }
  
  try {
    const memberIDs = members.map(m => parseInt(m.id))
    const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members`, {
      method: 'POST',
      body: JSON.stringify({ member_ids: memberIDs })
    })
    
    if (response.code === 0) {
      const newMembers = (Array.isArray(response.data) ? response.data : []).map((member: any) => ({
        id: member.id?.toString() || '',
        name: member.nickname || member.username || (member.name !== undefined ? member.name : '未知用户'),
        avatar: member.avatar || ''
      }))
      
      if (selectedGroup.value.members) {
        selectedGroup.value.members = [...selectedGroup.value.members, ...newMembers]
      } else {
        selectedGroup.value.members = newMembers
      }
      
      groupMembers.value = selectedGroup.value.members || []
      
      const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value.id)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].members = selectedGroup.value.members
        conversations.value = [...conversations.value]
      }
      
      QMessage.success('添加成员成功')
      closeAddMembersModal()
    } else {
      QMessage.error('添加成员失败: ' + response.message)
    }
  } catch (error: any) {
    console.error('添加成员失败:', error)
    QMessage.error('添加成员失败，请稍后重试')
  }
}

// 点击其他地方关闭菜单由showContextMenu和showGroupContextMenu函数内部处理

const closeSettingsMenu = () => {
  showSettingsMenuFlag.value = false
  document.removeEventListener('click', closeSettingsMenu)
}

// 监听自动更新事件
if (window.electron) {
  // 检查更新中
  window.electron.ipcRenderer.on('update-checking', () => {
    isCheckingUpdate.value = true
    updateResult.value = '正在检查更新...'
  })
  
  // 发现新版本
  window.electron.ipcRenderer.on('update-available', (_event, info: any) => {
    isCheckingUpdate.value = false
    hasNewVersion.value = true
    updateResult.value = `发现新版本 v${info.version}`
  })
  
  // 当前已是最新版本
  window.electron.ipcRenderer.on('update-not-available', () => {
    isCheckingUpdate.value = false
    hasNewVersion.value = false
    updateResult.value = '当前已是最新版本'
  })
  
  // 更新错误
  window.electron.ipcRenderer.on('update-error', (_event, error: any) => {
    isCheckingUpdate.value = false
    updateResult.value = `更新错误: ${error}`
  })
  
  // 下载进度
  window.electron.ipcRenderer.on('update-progress', (_event, progressObj: any) => {
    isDownloading.value = true
    downloadProgress.value = progressObj.percent
  })
  
  // 更新下载完成
  window.electron.ipcRenderer.on('update-downloaded', (_event, _info: any) => {
    isDownloading.value = false
    updateResult.value = '下载完成，正在安装...'
    setTimeout(() => {
      updateResult.value = '升级成功，需要重启应用'
      hasNewVersion.value = false
    }, 1500)
  })
}

const checkForUpdates = () => {
  console.log('检查更新')
  // 显示检查更新对话框
  showUpdateDialog.value = true
  isCheckingUpdate.value = true
  updateResult.value = '正在检查更新...'
  hasNewVersion.value = false
  downloadProgress.value = 0
  isDownloading.value = false
  closeSettingsMenu()
  
  // 向主进程发送检查更新请求
  if (window.electron) {
    window.electron.ipcRenderer.send('check-for-updates')
  } else {
    // 模拟检查更新的过程（开发环境）
    setTimeout(() => {
      isCheckingUpdate.value = false
      // 模拟有新版本
      if (Math.random() > 0.5) {
        hasNewVersion.value = true
        updateResult.value = '发现新版本 v1.0.1'
      } else {
        hasNewVersion.value = false
        updateResult.value = '当前已是最新版本'
      }
    }, 1500)
  }
}

const downloadUpdate = () => {
  console.log('下载升级')
  isDownloading.value = true
  downloadProgress.value = 0
  
  // 向主进程发送下载更新请求
  if (window.electron) {
    window.electron.ipcRenderer.send('download-update')
  } else {
    // 模拟下载过程（开发环境）
    const interval = setInterval(() => {
      downloadProgress.value += 5
      if (downloadProgress.value >= 100) {
        clearInterval(interval)
        isDownloading.value = false
        updateResult.value = '下载完成，正在安装...'
        
        // 模拟安装过程
        setTimeout(() => {
          updateResult.value = '升级成功，需要重启应用'
          hasNewVersion.value = false
        }, 1500)
      }
    }, 100)
  }
}

const aboutApp = () => {
  console.log('关于应用')
  showAboutDialog.value = true
  closeSettingsMenu()
}

const emit = defineEmits<{
  (e: 'logout'): void
}>()

// 以下状态已从 useUI composable 导入：showLogoutDialog, showUpdateDialog, isCheckingUpdate, updateResult, hasNewVersion, downloadProgress, isDownloading
// 以下函数已从 useUI composable 导入：confirmLogout, cancelLogout, closeUpdateDialog, showThemeMenu, closeThemeMenu, showMoreMenu, closeMoreMenu, closeSettingsModal

const logout = () => {
  console.log('退出登录')
  // 显示退出登录确认弹窗
  showLogoutDialog.value = true
  closeSettingsMenu()
}

// 确认登出 - 执行实际的登出操作
const handleConfirmLogout = async () => {
  // 关闭登出确认对话框
  cancelLogout()
  
  // 使用 Promise.race 实现超时控制
  const withTimeout = (promise: Promise<any>, timeoutMs: number, label: string) => {
    return Promise.race([
      promise,
      new Promise((_, reject) => 
        setTimeout(() => reject(new Error(`${label}超时`)), timeoutMs)
      )
    ])
  }
  
  // 并行执行所有清理操作，不互相阻塞
  const cleanupTasks = [
    // 1. 调用后端登出接口（最多等待 2 秒）
    withTimeout(request('/api/v1/auth/logout', { method: 'POST' }), 2000, '登出请求')
      .catch(error => console.error('登出请求失败或超时:', error)),
    
    // 2. 清理 IndexedDB 存储（最多等待 1 秒）
    import('../utils/storage').then(({ clearAll }) => 
      withTimeout(clearAll(), 1000, '清理本地存储')
    ).catch(error => console.error('清理本地存储失败或超时:', error))
  ]
  
  // 3. 立即清理 localStorage（同步操作，无延迟）
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  
  // 等待所有清理任务完成（最多等待 2 秒）
  await Promise.allSettled(cleanupTasks)
  
  // 4. 跳转到登录页
  window.location.href = '/login'
}

// 主题菜单相关函数
// 保存系统设置 - 已从 useSettings composable 导入

// 清除缓存 - 已从 useSettings composable 导入

// 保存双因素认证设置 - 已从 useSettings composable 导入

// 浏览默认保存目录 - 已从 useSettings composable 导入

// 打开安全设置
const openSecuritySettings = () => {
  QMessage.info('打开安全设置')
  // 这里可以实现打开安全设置页面的逻辑
}

const handleSaveSettings = async (data: { profile: any; messageSettings: any; appearanceSettings: any }) => {
  settingsProfile.value = { ...settingsProfile.value, ...data.profile }
  messageSettings.value = { ...messageSettings.value, ...data.messageSettings }
  appearanceSettings.value = { ...appearanceSettings.value, ...data.appearanceSettings }
  
  if (data.appearanceSettings.theme && data.appearanceSettings.theme !== currentTheme.value) {
    setTheme(data.appearanceSettings.theme)
  }
  if (data.appearanceSettings.fontSize) {
    applyFontSize(data.appearanceSettings.fontSize)
  }
  
  await saveSettings()
  closeSettingsModal()
}

// setTheme, applyFontSize, initTheme 已从 useSettings composable 导入

// 初始化主题
initTheme()
</script>

<style>
@import url('../assets/styles/design-tokens.css');

/* Markdown 样式 */
.markdown-heading:nth-child(1) {
  margin-top: 0;
}

.markdown-heading:nth-of-type(1) {
  font-size: 1.8em;
  border-bottom: 2px solid var(--border-color);
  padding-bottom: 8px;
}

.markdown-heading:nth-of-type(2) {
  font-size: 1.5em;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 6px;
}

.markdown-heading:nth-of-type(3) {
  font-size: 1.3em;
}

.markdown-heading:nth-of-type(4) {
  font-size: 1.1em;
}

.markdown-heading:nth-of-type(5),
.markdown-heading:nth-of-type(6) {
  font-size: 1em;
  color: var(--text-secondary);
}



.markdown-code:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}


.markdown-link:hover {
  color: var(--active-color);
  text-decoration: none;
  border-bottom-color: var(--active-color);
}


.markdown-image:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: scale(1.01);
}





.markdown-table th,
.markdown-table td {
  padding: 10px 12px;
  text-align: left;
  border: 1px solid var(--border-color);
}

.markdown-table th {
  background: var(--hover-color);
  font-weight: 600;
  color: var(--text-color);
}

.markdown-table tr:hover {
  background: var(--hover-color);
}

/* 通用按钮样式 */
button {
  transition: all 0.2s ease;
}

button:hover {
  transform: translateY(-1px);
}

button:active {
  transform: translateY(0);
}

/* 模态框动画 */
.modal-content {
  animation: modalFadeIn 0.3s ease;
}

@keyframes modalFadeIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

/* 消息动画 */
@keyframes messageFadeIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes messageFadeOut {
  from {
    opacity: 1;
    transform: translateY(0);
  }
  to {
    opacity: 0;
    transform: translateY(-10px);
  }
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--sidebar-bg);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-color);
  opacity: 0.5;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .main-content {
    flex-direction: column;
  }
  
  .sidebar {
    width: 100%;
    height: 200px;
  }
  
  .right-content {
    flex: 1;
  }
}

/* 通用动画 */
@keyframes slideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

</style>

<style scoped>
/* ===== 主容器 ===== */
.im-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: var(--content-bg);
  color: var(--text-color);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

/* ===== 加载状态 ===== */
.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--content-bg);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 9999;
  animation: fadeIn 0.3s ease;
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 4px solid rgba(0, 0, 0, 0.1);
  border-radius: 50%;
  border-top-color: var(--primary-color);
  animation: spin 1s ease-in-out infinite;
}

.loading-text {
  font-size: 16px;
  color: var(--text-color);
  animation: pulse 1.5s ease-in-out infinite;
}

/* ===== 顶部区域 ===== */
.top-bar {
  display: flex;
  height: 40px;
  -webkit-app-region: drag;
}

.top-bar-left {
  width: 60px;
  background: var(--sidebar-bg);
  flex-shrink: 0;
  transition: background 0.3s ease;
}

/* ===== 主内容区域 ===== */
.main-content-area {
  flex: 1;
  display: flex;
  overflow: hidden;
  margin-left: 60px;
}

/* ===== 右侧内容 ===== */
.right-content {
  flex: 1;
  background: var(--content-bg);
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
}

.right-content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--sidebar-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.toggle-sidebar-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--border-color);
  background: var(--card-bg);
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all 0.3s ease;
  color: var(--text-color);
}

.toggle-sidebar-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
}

.right-content-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color);
  background: var(--right-content-bg);
  font-size: 14px;
  opacity: 0.7;
}

/* ===== 空状态 ===== */
.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--right-content-bg);
  opacity: 0.7;
}

.empty-content {
  text-align: center;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

/* ===== 用户应用 ===== */
.user-app-content {
  height: calc(100% - 60px);
  padding: 20px;
  overflow: hidden;
}

.user-app-iframe-container {
  height: 100%;
  width: 100%;
  overflow: hidden;
}

.user-app-iframe {
  height: 100%;
  width: 100%;
  border: none;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-user-app {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: #666;
}

.empty-user-app .empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #ccc;
}

.empty-user-app p {
  margin: 8px 0;
}

.empty-user-app .empty-hint {
  font-size: 14px;
  color: #999;
}

/* ===== 应用头部返回按钮 ===== */
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-button {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: var(--hover-color);
  color: var(--primary-color);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  font-size: 14px;
}

.back-button:hover {
  background: var(--primary-light);
  transform: scale(1.05);
  box-shadow: var(--shadow-sm);
}

/* ===== 主题样式 ===== */
.modern-light-theme {
  background: #3b82f6;
  border: 1px solid #2563eb;
}

.elegant-dark-theme {
  background: #1e293b;
  border: 1px solid #334155;
}

.ocean-blue-theme {
  background: #0ea5e9;
  border: 1px solid #0284c7;
}

.elegant-purple-theme {
  background: #8b5cf6;
  border: 1px solid #7c3aed;
}

.warm-amber-theme {
  background: linear-gradient(135deg, #f4a900 0%, #c1666b 100%);
  border: 1px solid #f4a900;
}

.crimson-red-theme {
  background: #dc2626;
  border: 1px solid #b91c1c;
}

.emerald-green-theme {
  background: #10b981;
  border: 1px solid #059669;
}

.mediterranean-dream-theme {
  background: linear-gradient(135deg, #c0392b 0%, #3498db 100%);
  border: 1px solid #c0392b;
}

.monochrome-elegance-theme {
  background: #333333;
  border: 1px solid #666666;
}

.spring-blossom-theme {
  background: linear-gradient(135deg, #f8bbd9 0%, #e1bee7 50%, #c8e6c9 100%);
  border: 1px solid #f8bbd9;
}

.green-theme {
  background: #10b981;
  border: 1px solid #059669;
}

.light-theme {
  background: #ffffff;
  border: 1px solid #e5e7eb;
}

.dark-theme {
  background: #1a1a1a;
  border: 1px solid #64ffda;
}

.netblue-theme {
  background: #0284c7;
  border: 1px solid #0369a1;
}

.sacredyellow-theme {
  background: #c9a227;
  border: 1px solid #b8973c;
}

.chinesered-theme {
  background: #c41e3a;
  border: 1px solid #a01830;
}

.grassgreen-theme {
  background: #2e8b57;
  border: 1px solid #247048;
}

/* ===== 过渡动画 ===== */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from {
  transform: translateX(-20px);
  opacity: 0;
}

.slide-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

.scale-enter-active,
.scale-leave-active {
  transition: all 0.3s ease;
}

.scale-enter-from,
.scale-leave-to {
  transform: scale(0.95);
  opacity: 0;
}

/* ===== 动画关键帧 ===== */
@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.6;
  }
}
</style>