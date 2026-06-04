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
    
    <!-- 小程序面板 -->
    <MiniAppManager
      v-model:showMiniAppList="showMiniAppList"
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
        ref="sidebarRef"
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
        :hasMoreConversations="hasMoreConversations"
        :isLoadingConversations="isLoadingConversations"
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
        @resetApp="selectedAppId = ''"
        @toggleCategory="categoryExpanded[$event] = !categoryExpanded[$event]"
        @searchResultSelect="handleSearchItemClick"
        @searchResultPrivateChat="startPrivateChat"
        @searchResultApplyJoin="handleApplyJoinGroup"
        @createChannel="showCreateChannelModal = true"
        @loadMoreConversations="loadMoreConversations"
      />
      
      <!-- 实时通信全局组件（屏幕共享、视频通话） -->
    <RealtimeCommunication
      ref="realtimeRef"
      :current-conversation="currentConversation"
      :current-user-id="currentUser?.id"
      :conversations="conversations"
      :on-conversation-switch="handleConversationSelect"
      :on-send-message="handleSendMessage"
      @screen-share.start="logger.log('===== 屏幕共享已开始 =====', $event)"
      @screen-share.stop="sendMessage({ type: 'screen-share-stop', data: $event })"
      @screen-share.data="handleScreenShareData"
    />

    <!-- 聊天窗口 -->
    <template v-if="activeOption === 'recent'">
      <Suspense timeout="0">
        <template #default>
          <ChatWindow
            v-if="currentConversation"
            ref="chatWindowRef"
            :conversation="currentConversation"
            :messages="messages"
            :getReadUsers="getMessageReadUsers"
            :currentUser="currentUser"
            :hasMoreMessages="hasMoreMessages"
            :updateConversation="updateConversation"
            :fileSettings="fileSettings"
            @send="handleSendMessage"
            @recall="handleRecallMessage"
            @inviteMembers="handleInviteMembers"
            @switchConversation="handleSwitchConversation"
            @switch-app="handleSwitchApp"
            @loadMore="loadMessages($event, false)"
            @retry-send="handleRetrySendMessage"
            @start-voice-call="handleStartVoiceCall"
            @start-video-call="handleStartVideoCall"
            @start-screen-share="handleStartScreenShare"
          />
          <div v-else class="right-content">
            <div class="panel-header">
              <div class="header-left-group">
                <ToggleSidebarBtn
                  icon="fas fa-compress"
                  title="收起侧边栏"
                  @click="toggleSidebar"
                />
                <h2>{{ getPageTitle() }}</h2>
              </div>
            </div>
            <div class="empty-state">
              <div class="empty-content">
                <div class="empty-icon"><i class="fas fa-comments"></i></div>
                <p>选择一个会话开始聊天</p>
              </div>
            </div>
          </div>
        </template>
        <template #fallback>
          <ContentSkeleton type="recent" :count="6" />
        </template>
      </Suspense>
    </template>
      
      <!-- 频道页面布局 -->
      <template v-else-if="activeOption === 'channels'">
        <Suspense timeout="0">
          <template #default>
            <div class="channel-content-area">
              <ChannelDetailNew
                v-if="channelStore.selectedChannel"
                :channel="channelStore.selectedChannel"
                :isCreator="isChannelCreator(channelStore.selectedChannel)"
                :displayMode="channelStore.messageMode"
                :sortOrder="'desc'"
                :loading="channelStore.messagesLoading"
                @subscribe="handleChannelSubscribe"
                @unsubscribe="handleChannelUnsubscribe"
                @sendMessage="handleChannelSendMessage"
                @update:displayMode="handleDisplayModeChange"
                @like="handleMessageLike"
                @unlike="handleMessageUnlike"
                @comment="handleMessageComment"
              />
              <div v-else class="channel-empty-state-wrapper">
                <div class="panel-header">
                  <div class="header-left-group">
                    <ToggleSidebarBtn
                      icon="fas fa-compress"
                      title="收起侧边栏"
                      @click="toggleSidebar"
                    />
                    <h2>频道</h2>
                  </div>
                </div>
                <div class="channel-empty-state">
                  <div class="empty-icon"><i class="fas fa-bullhorn"></i></div>
                  <p>选择一个频道查看详情</p>
                </div>
              </div>
            </div>
          </template>
          <template #fallback>
            <ContentSkeleton type="channels" :count="8" />
          </template>
        </Suspense>
      </template>
      
      <!-- 组织架构用户信息 -->
      <template v-else-if="activeOption === 'org' && selectedUser">
        <Suspense timeout="0">
          <template #default>
            <UserDetailPanel
              :user="selectedUser"
              :serverUrl="serverUrl"
              @toggleSidebar="toggleSidebar"
              @privateChat="startPrivateChat"
              @showProfile="showUserProfile = true"
              @open-avatar-settings="selectedAppId = 'avatar'; activeOption = 'apps'"
            />
          </template>
          <template #fallback>
            <ContentSkeleton type="groups" :count="4" />
          </template>
        </Suspense>
      </template>
      
      <!-- 应用面板 -->
      <template v-else-if="activeOption === 'apps'">
        <!-- 应用分类列表（不需要 Suspense，因为 AppsPanel 是同步加载的） -->
        <AppsPanel
          v-if="!selectedAppId"
          :mainApps="mainApps"
          :quickTools="quickTools"
          :customApps="customApps"
          :systemApps="systemApps"
          :pageTitle="getPageTitle()"
          @toggleSidebar="toggleSidebar"
          @openApp="openApp"
        />

        <!-- 具体应用（懒加载组件，各自独立使用 Suspense） -->
        <div v-else-if="selectedAppId === 'file_manager'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <FileManagementApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 笔记应用 -->
        <div v-else-if="selectedAppId === 'notes'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <NotesApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 任务管理应用 -->
        <div v-else-if="selectedAppId === 'task_manager'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <TaskManagementApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 日历应用 -->
        <div v-else-if="selectedAppId === 'calendar'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <CalendarApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 便签应用 -->
        <div v-else-if="selectedAppId === 'sticky_notes'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <StickyNotesApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 用户创建的应用 -->
        <div v-else-if="selectedAppId === 'user-app' && currentUserApp" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <UserAppContainer
                :app="currentUserApp"
                @back="selectedAppId = ''"
                @toggleSidebar="toggleSidebar"
              />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 应用管理 -->
        <div v-else-if="selectedAppId === 'app-management'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <AppManagementApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- AI 助手 -->
        <div v-else-if="systemConfigStore.enableAI && selectedAppId === 'ai_assistant'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <AIAssistantApp @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- AI 分身 -->
        <div v-else-if="systemConfigStore.enableAI && selectedAppId === 'avatar'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <AvatarSettingsPanel @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>

        <!-- 短链接管理应用 -->
        <div v-else-if="selectedAppId === 'short_link'" class="right-content">
          <Suspense timeout="0">
            <template #default>
              <ShortLinkManager @back="selectedAppId = ''" @toggleSidebar="toggleSidebar" />
            </template>
            <template #fallback>
              <ContentSkeleton type="settings" />
            </template>
          </Suspense>
        </div>
      </template>
      
      <!-- 群聊详情 -->
      <template v-else-if="activeOption === 'groups' && selectedGroup">
        <Suspense timeout="0">
          <template #default>
            <div class="right-content">
              <div class="panel-header">
                <div class="header-left-group">
                  <ToggleSidebarBtn
                    icon="fas fa-compress"
                    title="收起侧边栏"
                    @click="toggleSidebar"
                  />
                  <h2>{{ selectedGroup.name }}</h2>
                </div>
              </div>
              <GroupDetail
                :group="selectedGroup"
                @enter="handleConversationSelect($event)"
                @invite="handleInviteMembers($event)"
                @editAnnouncement="editAnnouncement"
                @editGroupName="editGroupNameAction"
                @openAISettings="showAISettingsModal = true"
                @showMemberContextMenu="(event, member) => showMemberContextMenu(event, member)"
                @startPrivateChat="startPrivateChat"
              />
            </div>
          </template>
          <template #fallback>
            <ContentSkeleton type="groups" :count="5" />
          </template>
        </Suspense>
      </template>
      
      <div v-else class="right-content">
        <div class="panel-header">
          <div class="header-left-group">
            <ToggleSidebarBtn
              icon="fas fa-compress"
              title="收起侧边栏"
              @click="toggleSidebar"
            />
            <h2>{{ getPageTitle() }}</h2>
          </div>
        </div>
        <div class="right-content-body">
          <p>选择左侧的{{ getPageTitle() }}查看详情</p>
        </div>
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
      :selectedGroupForContextMenu="selectedGroupForContextMenu"
      :isGroupOwner="isGroupOwner(selectedGroupForContextMenu)"
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
    
    <!-- 创建频道弹窗 -->
    <ModalContainer
      :visible="showCreateChannelModal"
      title="创建频道"
      width="500px"
      @close="closeCreateChannelModal"
      @cancel="closeCreateChannelModal"
      @confirm="handleCreateChannelSubmit"
    >
      <div class="form-group">
        <label>频道名称 *</label>
        <input 
          v-model="createChannelForm.name" 
          type="text" 
          placeholder="输入频道名称" 
          class="form-input"
        />
      </div>
      <div class="form-group">
        <label>频道描述</label>
        <textarea 
          v-model="createChannelForm.description" 
          placeholder="输入频道描述" 
          rows="3" 
          class="form-textarea"
        ></textarea>
      </div>
      <div class="form-group">
        <label>频道头像URL</label>
        <input 
          v-model="createChannelForm.avatar" 
          type="text" 
          placeholder="输入头像URL（可选）" 
          class="form-input"
        />
      </div>
    </ModalContainer>
    
    <!-- 群模态框 -->
    <GroupModals
      :showGroupMembersModal="showGroupMembersModal"
      :showGroupInfoModal="showGroupInfoModal"
      :showAddMembersModal="showAddMembersModal"
      :showEditAnnouncementModal="showEditAnnouncementModal"
      :showEditGroupNameModal="showEditGroupNameModal"
      :selectedGroup="selectedGroup"
      :groupMembers="groupMembers"
      :allEmployees="allEmployees"
      :addMembersSearchQuery="addMembersSearchQuery"
      :selectedAddMembers="selectedAddMembers"
      :editGroupName="editGroupName"
      :editAnnouncementContent="editAnnouncementContent"
      :currentUserId="currentUser?.id"
      :formatTime="formatTime"
      @closeGroupMembers="closeGroupMembersModal"
      @closeGroupInfo="closeGroupInfoModal"
      @closeAddMembers="closeAddMembersModal"
      @closeEditAnnouncement="closeEditAnnouncementModal"
      @closeEditGroupName="closeEditGroupNameModal"
      @removeMember="removeMember"
      @confirmAddMembers="confirmAddMembers"
      @saveAnnouncement="saveAnnouncement"
      @saveGroupName="saveGroupName"
    />

    <!-- AI 助手设置模态框 -->
    <ModalContainer
      :visible="showAISettingsModal"
      title="AI 助手设置"
      @close="closeAISettings"
      @cancel="closeAISettings"
      :show-footer="false"
      :content-style="{ width: '480px', minWidth: '480px' }"
    >
      <GroupAIPanel
        :group-id="Number(selectedGroup?.id)"
        :server-url="serverUrl"
        :ai-enabled="selectedGroup?.ai_config?.ai_enabled ?? false"
        :ai-assistant-name="selectedGroup?.ai_config?.ai_assistant_name ?? 'AI助手'"
        :ai-reply-mode="selectedGroup?.ai_config?.ai_reply_mode ?? 'mention_only'"
        :ai-personality="selectedGroup?.ai_config?.ai_personality ?? 'professional'"
        :ai-custom-prompt="selectedGroup?.ai_config?.ai_custom_prompt ?? ''"
        :ai-language="selectedGroup?.ai_config?.ai_language ?? 'auto'"
        :ai-max-length="selectedGroup?.ai_config?.ai_max_length ?? 'medium'"
        :ai-mention-reply-mode="selectedGroup?.ai_config?.ai_mention_reply_mode ?? 'mention'"
        :ai-anti-spam-interval="selectedGroup?.ai_config?.ai_anti_spam_interval ?? 5"
        :ai-trigger-keywords="parseTriggerKeywords(selectedGroup?.ai_config?.ai_trigger_keywords)"
        :ai-learn-enabled="selectedGroup?.ai_config?.ai_learn_enabled ?? false"
        :ai-extract-todos="selectedGroup?.ai_config?.ai_extract_todos ?? false"
        @update="updateAISettings"
      />
    </ModalContainer>
    
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
      :isCheckingUpdate="isCheckingUpdate"
      :isDownloading="isDownloading"
      :downloadProgress="downloadProgress"
      :hasNewVersion="hasNewVersion"
      :updateResult="updateResult"
      :groupConversations="conversations.filter(c => c.type === 'group')"
      :allEmployees="allEmployees"
      :orgStructure="orgStructure"
      :systemMessage="systemMessage"
      @closeAbout="closeAboutDialog"
      @cancelLogout="cancelLogout"
      @confirmLogout="handleConfirmLogout"
      @closeUpdate="closeUpdateDialog"
      @downloadUpdate="downloadUpdate"
      @closeSystemMessage="closeSystemMessageModal"
      @sendSystemMessage="sendSystemMessage"
      @openFeedback="openFeedbackModal"
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
    @openSecurity="QMessage.info('打开安全设置')"
    @browseDirectory="browseDefaultSaveDirectory"
    @openFeedback="openFeedbackModal"
  />

  <!-- 意见反馈弹窗 -->
  <FeedbackModal
    :visible="showFeedbackModal"
    @close="closeFeedbackModal"
    @success="handleFeedbackSuccess"
  />
</template>

<script setup lang="ts">
import { ref, computed, defineComponent, defineAsyncComponent, onMounted, onUnmounted, watch, nextTick, provide } from 'vue'
import type { Conversation, Message, User } from '../types'
import QMessage from '../utils/qmessage'
import QMessageBox from '../utils/qmessagebox'
import axios from 'axios'
import { logger } from '../utils/logger'
const CalendarApp = defineAsyncComponent(() => import('../components/apps/CalendarApp.vue'))
const StickyNotesApp = defineAsyncComponent(() => import('../components/apps/StickyNotesApp.vue'))
const NotesApp = defineAsyncComponent(() => import('../components/apps/NotesApp.vue'))
const TaskManagementApp = defineAsyncComponent(() => import('../components/apps/task/TaskManagementApp.vue'))
const FileManagementApp = defineAsyncComponent(() => import('../components/apps/FileManagementApp.vue'))
const AppManagementApp = defineAsyncComponent(() => import('../components/apps/AppManagementApp.vue'))
const AIAssistantApp = defineAsyncComponent(() => import('../components/apps/AIAssistantApp.vue'))
const ShortLinkManager = defineAsyncComponent(() => import('../components/apps/ShortLinkManager.vue'))
import MiniAppManager from '../components/apps/MiniAppManager.vue'
const UserAppContainer = defineAsyncComponent(() => import('../components/apps/UserAppContainer.vue'))
const AvatarSettingsPanel = defineAsyncComponent(() => import('../components/avatar/AvatarSettingsPanel.vue'))
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
import RealtimeCommunication from '../components/realtime/RealtimeCommunication.vue'
const GroupDetail = defineAsyncComponent(() => import('../components/shared/GroupDetail.vue'))
import ModalContainer from '../components/shared/ModalContainer.vue'
import ToggleSidebarBtn from '../components/shared/ToggleSidebarBtn.vue'
import ShareModal from '../components/modals/ShareModal.vue'
const UserProfile = defineAsyncComponent(() => import('../components/modals/UserProfile.vue'))
const NotificationCenter = defineAsyncComponent(() => import('../components/notification/NotificationCenter.vue'))
import { mapNotification } from '../utils/notificationMapper'
const CreateGroupModal = defineAsyncComponent(() => import('../components/modals/CreateGroupModal.vue'))
const ChannelDetailNew = defineAsyncComponent(() => import('../components/channel/ChannelDetailNew.vue'))
const UserDetailPanel = defineAsyncComponent(() => import('../components/user/UserDetailPanel.vue'))
import AppsPanel from '../components/apps/AppsPanel.vue'
import SelfProfileModal from '../components/modals/SelfProfileModal.vue'
const GroupModals = defineAsyncComponent(() => import('../components/modals/GroupModals.vue'))
const GroupAIPanel = defineAsyncComponent(() => import('../components/ai/GroupAIPanel.vue'))
import MainContextMenus from '../components/menus/MainContextMenus.vue'
import MainDialogs from '../components/modals/MainDialogs.vue'
import FeedbackModal from '../components/modals/FeedbackModal.vue'
const SettingsPanel = defineAsyncComponent(() => import('../components/settings/SettingsPanel.vue'))
import ContentSkeleton from '../components/skeleton/ContentSkeleton.vue'
import { useServerUrl } from '../composables/useServerUrl'
import { generateAvatar, getAvatarUrl, isAbsoluteUrl } from '../utils/avatar'
import { request, getToken } from '../composables/useRequest'
import { useChannelStore } from '../stores/channel'
import { useChatStore } from '../stores/chat'
import { useSystemConfigStore } from '../stores/systemConfig'
import { useCurrentUser } from '../composables/useCurrentUser'
import { useProcessConversation } from '../composables/useProcessConversation'
import { useSettings } from '../composables/useSettings'
import { fetchUserProfile } from '../composables/useUserProfileInfo'
import { useNetwork } from '../composables/useNetwork'
import { useWebSocketManager } from '../composables/useWebSocketManager'
import { useGroup } from '../composables/useGroup'
import { useMessageActions } from '../composables/useMessageActions'
import { getProductName, APP_CONFIG } from '../config/appConfig'
import { useNotifications } from '../composables/useNotifications'
import { useAppState } from '../composables/useAppState'
import { useUI } from '../composables/useUI'
import { useConversation } from '../composables/useConversation'
import { useOrganizationLogic } from '../composables/useOrganizationLogic'
import { useMainWebSocketHandlers } from '../composables/useMainWebSocketHandlers'
import { useMainConversationLogic } from '../composables/useMainConversationLogic'
import { useMainGroupHandlers } from '../composables/useMainGroupHandlers'
import { useMainMessageHandlers } from '../composables/useMainMessageHandlers'
import { useMainMessageLoading } from '../composables/useMainMessageLoading'
import { useMainMessageSending } from '../composables/useMainMessageSending'
import { useShareLogic } from '../composables/useShareLogic'
import { useUserProfile } from '../composables/useUserProfile'
import { useStreamMessage } from '../composables/useStreamMessage'
import { useAppLogic } from '../composables/useAppLogic'
// useUIState 已被 useAppState 替代

// 服务器地址
const { serverUrl } = useServerUrl()

// 显示消息提示（兼容现有 showMessage 调用方式）
const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
  const { message, type = 'info', duration } = options
  if (type === 'success') QMessage.success(message, duration)
  else if (type === 'error') QMessage.error(message, duration)
  else if (type === 'warning') QMessage.warning(message, duration)
  else QMessage.info(message, duration)
}

// 当前用户信息
const { currentUser, userProfile, syncUserProfile, getProfileAvatar, refreshUser } = useCurrentUser()

// 使用消息操作
const messageActions = useMessageActions(serverUrl, currentUser)
const { markMessagesAsRead } = messageActions

// 使用频道 store
const channelStore = useChannelStore()
const { isChannelCreator, sendChannelMessage } = channelStore

// 使用聊天 store（镜像同步）
const chatStore = useChatStore()

// 使用系统配置 store
const systemConfigStore = useSystemConfigStore()

// 服务器地址变更时重新获取系统配置
watch(serverUrl, () => {
  systemConfigStore.fetchPublicConfig()
})

// 会话数据处理
const { processConversation } = useProcessConversation(serverUrl, currentUser)


// 使用 composable
const {
  unreadNotificationCount,
  showNotificationCenter,
  notificationCenterPosition,
  handleNotificationCenter,
  closeNotificationCenter,
  handleNotificationClick: _handleNotificationClick,
  handleNewNotification: _handleNewNotification,
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
  cleanupNetwork,
  handleReconnect
} = useNetwork()

// 手动重连：重新连接 WebSocket
const handleManualReconnect = () => {
    systemConfigStore.fetchPublicConfig()
    connectWebSocket()
  }

// WebSocket 管理
const {
  ws,
  isConnected,
  connectWebSocket: connectWSManager,
  disconnectWebSocket,
  sendMessage,
  addHandler,
  setOnConnectedCallback
} = useWebSocketManager(serverUrl)

// 组织架构逻辑
const orgLogic = useOrganizationLogic()

// UI 状态已在 useAppState 中管理

// 监听用户状态变化并更新会话列表
const handleUserStatusChange = (data: any) => {
  logger.log('[Main] 收到用户状态变化:', data)
  const userId = data.user_id || data.userId
  const status = data.status
  if (!userId || !status) return
  
  // 更新会话列表中对应的 single 类型会话
  conversations.value = conversations.value.map(conv => {
    const memberId = conv.other_member_id
    if (conv.type === 'single' && Number(memberId) === Number(userId)) {
      return { ...conv, status }
    }
    return conv
  })
}

// 群组相关（使用别名避免与 useUI 中的同名变量冲突）
const groupState = useGroup()
const {
  isGroupOwnerCheck,
  isGroupAdmin: isGroupAdminCheck
} = groupState

// 检查当前用户是否是群聊所有者
const isGroupOwner = (group: any) => {
  return isGroupOwnerCheck(group, currentUser.value?.id?.toString())
}

// 检查当前用户是否是群聊管理员
const isGroupAdmin = (group: any) => {
  return isGroupAdminCheck(group, currentUser.value?.id?.toString())
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
  selectedGroupForContextMenu,
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
  shareData,
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
  // 编辑群名称
  showEditGroupNameModal,
  editGroupName,
  openEditGroupNameModal,
  closeEditGroupNameModal,
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
  registerUpdateEventListeners,
  unregisterUpdateEventListeners,
  // 设置
  showSettingsModal,
  activeSettingsTab,
  openSettings,
  closeSettingsModal,
  switchSettingsTab,
  handleClickOutside
} = ui

// 意见反馈弹窗状态
const showFeedbackModal = ref(false)

const openFeedbackModal = () => {
  showFeedbackModal.value = true
}

const closeFeedbackModal = () => {
  showFeedbackModal.value = false
}

const handleFeedbackSuccess = () => {
  QMessage.success('感谢您的反馈！我们会尽快处理。')
}

// 创建频道弹窗状态
const showCreateChannelModal = ref(false)
const createChannelForm = ref({
  name: '',
  description: '',
  avatar: ''
})

// 使用 useConversation composable
const conversation = useConversation()

// 解构会话状态
const {
  conversations,
  currentConversationId,
  messages,
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
  updateConversation,
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

// Main.vue 专用的 WebSocket handlers（需要 currentConversationId 和 messages）
const mainWsHandlers = useMainWebSocketHandlers(currentConversationId, messages)

// Main.vue 专用的会话逻辑
const mainConvLogic = useMainConversationLogic(updateConversations, processConversation, conversations)

// Main.vue 专用的群组 handlers
const mainGroupHandlers = useMainGroupHandlers(conversations, currentConversationId, messages)

// Main.vue 专用的消息 handlers
const mainMessageHandlers = useMainMessageHandlers()
const { processMessage } = mainMessageHandlers

// Main.vue 专用的消息加载逻辑
const mainMessageLoading = useMainMessageLoading(conversations, processMessage)
const { loadMessages, getMessageReadUsers, messagePage, messagePageSize, hasMoreMessages } = mainMessageLoading


const { 
  loadConversations, 
  loadMoreConversations, 
  hasMoreConversations, 
  isLoadingConversations 
} = mainConvLogic

// 通知中心组件 ref
const notificationCenterRef = ref<any>(null)

// 聊天窗口引用
const chatWindowRef = ref<any>(null)

// 实时通信组件引用
const realtimeRef = ref<InstanceType<typeof RealtimeCommunication> | null>(null)

// 侧边栏组件引用
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)

// 重写会话选择处理，包含 Main.vue 的特定逻辑
const handleConversationSelect = (conversation: Conversation) => {
  const conversationId = String(conversation.id)
  
  logger.log('[Main.vue] handleConversationSelect 被调用', {
    conversationId,
    currentConversationId: currentConversationId.value,
    isSameConversation: currentConversationId.value === conversationId
  })
  
  if (currentConversationId.value === conversationId) {
    logger.log('[Main.vue] 相同会话，跳过处理')
    return
  }
  
  _handleConversationSelect(conversation)
  activeOption.value = 'recent'
  loadMessages(conversation.id)
  chatStore.markConversationRead(conversation.id)
  if (window.electron?.tray) {
    window.electron.tray.stopFlash()
  }
}

// 重写通知点击处理，包含 Main.vue 的特定逻辑
const handleNotificationClick = (notification: any) => {
  _handleNotificationClick(notification)
  if (notification.category === 'message' && notification.data?.conversationId) {
    activeOption.value = 'recent'
    const conversationId = String(notification.data.conversationId)
    setCurrentConversationId(conversationId)
    loadMessages(conversationId)
  } else if (notification.category === 'group' && notification.data?.groupId) {
    activeOption.value = 'groups'
  }
}

// 重写新通知处理，包含 Main.vue 的特定逻辑
const handleNewNotification = (notification: any) => {
  _handleNewNotification(notification)
  logger.log('收到新通知:', notification)

  showMessage({
    message: notification.content || notification.title || '您有一条新通知',
    type: 'info',
    duration: 5000
  })

  if (notificationCenterRef.value) {
    const mapped = mapNotification(notification)
    const currentNotifications = notificationCenterRef.value.notifications || []
    notificationCenterRef.value.notifications = [mapped, ...currentNotifications]
  }
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


// 自定义事件处理器（提升到组件作用域以便清理）
const handleShareStickyNote = async (event: Event) => {
  const customEvent = event as CustomEvent
  const note = customEvent.detail
  await loadShareUsersAndGroups()
  openShareModal('sticky', note)
}

const handleForwardMessage = async (event: Event) => {
  const customEvent = event as CustomEvent
  const message = customEvent.detail.message
  if (message) {
    await loadShareUsersAndGroups()
    openShareModal('message', message)
  }
}

const handleOpenShareModal = async (event: Event) => {
  const customEvent = event as CustomEvent
  const { type, data } = customEvent.detail
  await loadShareUsersAndGroups()
  openShareModal(type, data)
}

const handleRefreshUserApps = async () => {
  await loadUserApps()
}

const handleOpenUserApp = (event: any) => {
  const app = event.detail
  openUserApp(app)
}

// 注册自定义事件监听器
const registerCustomEventListeners = () => {
  window.addEventListener('shareStickyNote', handleShareStickyNote)
  window.addEventListener('forwardMessage', handleForwardMessage)
  window.addEventListener('openShareModal', handleOpenShareModal)
  window.addEventListener('refresh-user-apps', handleRefreshUserApps)
  window.addEventListener('open-user-app', handleOpenUserApp)
}

// 移除自定义事件监听器
const unregisterCustomEventListeners = () => {
  window.removeEventListener('shareStickyNote', handleShareStickyNote)
  window.removeEventListener('forwardMessage', handleForwardMessage)
  window.removeEventListener('openShareModal', handleOpenShareModal)
  window.removeEventListener('refresh-user-apps', handleRefreshUserApps)
  window.removeEventListener('open-user-app', handleOpenUserApp)
}

// 初始化数据
let isFirstConnect = true
onMounted(async () => {
  isLoading.value = true
  
  try {
    // ========== 阶段1：核心数据（必须等待，阻塞渲染）==========
    logger.log('[Main] 阶段1: 加载核心数据...')
    await refreshUser()
    await loadConversations()
    
    // 核心数据加载完成，立即展示主界面
    isLoading.value = false
    logger.log('[Main] 核心数据加载完成，主界面已展示')
    
    // ========== 阶段2：重要数据（后台并行加载，不阻塞首屏）=========
    logger.log('[Main] 阶段2: 后台加载次要数据...')
    Promise.allSettled([
      loadOrganizationTree(),
      loadUserApps(),
      loadAppCategories()
    ]).then(results => {
      results.forEach((result, index) => {
        if (result.status === 'rejected') {
          const names = ['组织架构', '用户应用', '内置应用']
          logger.warn(`[Main] ${names[index]}加载失败:`, result.reason)
        }
      })
      logger.log('[Main] 次要数据加载完成')
    })
    
    // ========== 阶段3：连接与注册（异步执行）==========
    setupPostLoadTasks()
    
  } catch (error) {
    logger.error('[Main] 核心数据加载失败:', error)
    isLoading.value = false
    showNetworkError.value = true
    networkErrorMsg.value = '核心数据加载失败，请检查网络连接'
  }
})

// 提取为独立函数：加载后任务
const setupPostLoadTasks = () => {
  setOnConnectedCallback(() => {
    if (!isFirstConnect) {
      loadConversations()
      systemConfigStore.fetchPublicConfig()
      fetchMissedMessages()
    }
    isFirstConnect = false
  })
  
  connectWebSocket()
  
  setTimeout(() => {
    systemConfigStore.fetchPublicConfig()
  }, 1000)
  
  registerUpdateEventListeners()
  registerCustomEventListeners()
}

const fetchMissedMessages = async () => {
  const chatStore = useChatStore()
  const currentConvId = chatStore.currentConversationId
  if (!currentConvId) return

  const lastMsgId = chatStore.getLastMessageId(currentConvId)
  if (!lastMsgId) return

  try {
    const response = await request(`/api/v1/conversations/${currentConvId}/messages?after_id=${lastMsgId}&page_size=50`)
    if (response.code === 0 && response.data?.messages) {
      const missedMessages = response.data.messages
        .map((msg: any) => processMessage(msg, currentConvId))
        .filter((m: any) => m !== null)
      if (missedMessages.length > 0) {
        chatStore.appendMessagesSilent(currentConvId, missedMessages)
        logger.log(`[离线补偿] 会话 ${currentConvId} 拉取到 ${missedMessages.length} 条离线消息`)
      }
    }
  } catch (error) {
    logger.error('[离线补偿] 拉取离线消息失败:', error)
  }
}



// WebSocket连接

// 处理通话和屏幕共享通知
const handleCallNotification = (type: string, data: any) => {
  const fromUserId = data.from_user_id || data.user_id
  const fromUserName = data.from_user_name || data.sender_name || '对方'
  const fromUserAvatar = data.from_user_avatar || data.sender_avatar || ''
  
  let title = ''
  let body = ''
  
  switch (type) {
    case '来电':
      title = '来电提醒'
      body = `${fromUserName} 正在呼叫您`
      break
    case '屏幕共享':
      title = '屏幕共享'
      body = `${fromUserName} 开始了屏幕共享`
      break
    case '屏幕共享请求':
      title = '屏幕共享请求'
      body = `${fromUserName} 请求与您屏幕共享`
      break
    default:
      title = type
      body = data.content || '您有一条新通知'
  }
  
  showMessage({
    message: body,
    type: 'info',
    duration: 5000
  })
  
  if (window.electron?.tray) {
    window.electron.tray.flash()
  }
  
  if (messageSettings.value.desktopNotificationsEnabled && 'Notification' in window) {
    const notificationIcon = fromUserAvatar 
      ? getAvatarUrl(fromUserAvatar, fromUserName, serverUrl.value)
      : undefined
    
    if (Notification.permission === 'granted') {
      new Notification(title, {
        body: body,
        icon: notificationIcon
      })
    } else if (Notification.permission !== 'denied') {
      Notification.requestPermission().then(permission => {
        if (permission === 'granted') {
          new Notification(title, {
            body: body,
            icon: notificationIcon
          })
        }
      })
    }
  }
}

// 连接WebSocket
const connectWebSocket = () => {
  // 为每种消息类型添加专门的处理器
  // 使用 Main.vue 专用的 handlers
  const { handleReadReceipt, handleMessageRecalled } = mainWsHandlers
  const {
    handleGroupInvitation,
    handleAddedToGroup,
    handleGroupMemberLeft,
    handleGroupMemberJoined,
    handleGroupAnnouncementUpdated,
    handleGroupMemberRoleUpdated,
    handleGroupOwnerTransferred
  } = mainGroupHandlers
  
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
    'system_config_updated': (data: any) => systemConfigStore.updateFromServer(data),
    'system_message': handleSystemMessage,
    // 用户状态变化
    'user_status_changed': (data: any) => handleUserStatusChange(data),
    // 屏幕共享消息路由到 RealtimeCommunication 组件
    'screen-share.start': (data: any) => {
      realtimeRef.value?.handleScreenShareStart(data)
      handleCallNotification('屏幕共享', data)
    },
    'screen-share.stop': (data: any) => realtimeRef.value?.handleScreenShareStop(data),
    'screen-share.data': (data: any) => realtimeRef.value?.handleScreenShareMessage('screen-share.data', data),
    'screen-share.request': (data: any) => {
      realtimeRef.value?.handleScreenShareRequest(data)
      handleCallNotification('屏幕共享请求', data)
    },
    'screen-share.accepted': (data: any) => realtimeRef.value?.handleScreenShareAccepted(data),
    'screen-share.rejected': (data: any) => realtimeRef.value?.handleScreenShareRejected(data),
    // WebRTC 信令消息
    'webrtc.offer': (data: any) => realtimeRef.value?.handleWebRTCOffer(data),
    'webrtc.answer': (data: any) => realtimeRef.value?.handleWebRTCAnswer(data),
    'webrtc.ice-candidate': (data: any) => realtimeRef.value?.handleWebRTCIceCandidate(data),
    // 实时会话消息
    'realtime:session:created': (data: any) => realtimeRef.value?.handleRealtimeSessionCreated(data),
    // 视频通话消息
    'call.start': (msg: any) => {
      realtimeRef.value?.handleVideoCallSignaling({ type: 'call.start', data: msg })
      handleCallNotification('来电', msg)
    },
    'call.answer': (msg: any) => realtimeRef.value?.handleVideoCallSignaling({ type: 'call.answer', data: msg }),
    'call.reject': (msg: any) => realtimeRef.value?.handleVideoCallSignaling({ type: 'call.reject', data: msg }),
    'call.end': (msg: any) => realtimeRef.value?.handleVideoCallSignaling({ type: 'call.end', data: msg })
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


// 处理系统消息
const handleSystemMessage = (data: any) => {
  logger.log('收到系统消息:', data)
  
  showMessage({
    message: `系统消息: ${data.title}`,
    type: 'info',
    duration: 5000
  })
  
  if (notificationCenterRef.value) {
    const newNotification = {
      id: Date.now().toString(),
      title: data.title,
      content: data.content,
      timestamp: Date.now(),
      read: false,
      type: 'system' as const
    }
    
    const currentNotifications = notificationCenterRef.value.notifications || []
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    
    unreadNotificationCount.value++
  }
}

// 处理消息删除
const handleMessageDeleted = (data: any) => {
  logger.log('消息被删除:', data)
  if (currentConversationId.value) {
    chatStore.deleteMessage(String(currentConversationId.value), String(data.message_id))
  }
}

// 处理通知
const handleNotification = (data: any) => {
  logger.log('收到通知:', data)
  showMessage({
    message: data.content,
    type: 'info',
    duration: 5000
  })

  if (notificationCenterRef.value) {
    const mapped = mapNotification(data)
    const currentNotifications = notificationCenterRef.value.notifications || []
    notificationCenterRef.value.notifications = [mapped, ...currentNotifications]
    unreadNotificationCount.value++
  }

  if (data.type && data.type.includes('_approval')) {
    const entityType = data.type.replace('_approval', '')
    if (entityType === 'avatar' && chatWindowRefs[currentConversationId.value]) {
      chatWindowRefs[currentConversationId.value]?.fetchConfig?.()
    }
  }
}

// 处理会话更新
const handleConversationUpdated = (data: any) => {
  logger.log('会话更新:', data)
  
  if (!data || !data.id) {
    logger.warn('会话更新数据无效:', data)
    return
  }
  
  try {
    const normalizedData = {
      ...data,
      id: data.id.toString()
    }
    chatStore.patchConversation(normalizedData.id, normalizedData)
  } catch (error) {
    logger.error('处理会话更新失败:', error)
    QMessage.error('处理会话更新失败')
  }
}
const formatNotificationContent = (message: any): string => {
  if (message.type === 'share' && message.shareData) {
    const shareType = message.shareData.type === 'file' ? '文件' : 
                      message.shareData.type === 'note' ? '笔记' : 
                      message.shareData.type === 'sticky' ? '便签' : '分享'
    return `[${shareType}] ${message.shareData.name || '分享内容'}`
  }
  
  if (message.type === 'miniApp' && message.miniAppData) {
    return `[小程序] ${message.miniAppData.name || '小程序'}`
  }
  
  if (message.type === 'news' && message.newsData) {
    return `[资讯] ${message.newsData.title || '资讯'}`
  }
  
  if (message.type === 'image') {
    return '[图片]'
  }
  
  if (message.type === 'file') {
    try {
      const fileData = JSON.parse(message.content || '{}')
      return `[文件] ${fileData.name || fileData.fileName || '文件'}`
    } catch {
      return '[文件]'
    }
  }
  
  return message.content || '无内容'
}

// 处理新消息
const handleNewMessage = (msg: any) => {
  const data = msg.data || msg
  const conversationId = data.conversation_id.toString()
  
  const newMessage = processMessage(data, conversationId)
  const isCurrentConv = currentConversationId.value === conversationId
  
  
  
  // 非当前会话且非流式消息，且未设置免打扰，触发提示音和桌面通知
  if (!isCurrentConv && !newMessage.isStreaming) {
    const conv = conversations.value.find(c => c.id === conversationId)
    if (messageSettings.value.notificationsEnabled && !conv?.muted) {
      if (messageSettings.value.soundEnabled) {
        playMessageSound()
      }

      if (window.electron?.tray) {
        window.electron.tray.flash()
      }
      
      if (messageSettings.value.desktopNotificationsEnabled && 'Notification' in window) {
        const notificationBody = formatNotificationContent(newMessage)
        if (Notification.permission === 'granted') {
          new Notification('新消息', {
            body: notificationBody,
            icon: getAvatarUrl(newMessage.sender.avatar, newMessage.sender.name || 'user', serverUrl.value)
          })
        } else if (Notification.permission !== 'denied') {
          Notification.requestPermission().then(permission => {
            if (permission === 'granted') {
              new Notification('新消息', {
                body: notificationBody,
                icon: getAvatarUrl(newMessage.sender.avatar, newMessage.sender.name || 'user', serverUrl.value)
              })
            }
          })
        }
      }
    }
  }
  
  // 如果会话不存在于列表中，重新加载会话列表
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex === -1) {
    loadConversations()
  }
  
  // 通过 Store 统一更新会话信息（lastMessage、timestamp、未读数）
  chatStore.receiveMessage(conversationId, newMessage, isCurrentConv)
}

let conversationSortTimer: number | null = null
const scheduleConversationSort = () => {
  if (conversationSortTimer) return
  conversationSortTimer = window.setTimeout(() => {
    conversationSortTimer = null
    conversations.value.sort((a, b) => b.timestamp - a.timestamp)
  }, 100)
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
    logger.error('播放消息提示音失败:', error)
  }
}



// 组件销毁时关闭WebSocket连接
onUnmounted(() => {
  // 关闭WebSocket连接
  disconnectWebSocket()
  // 清除重连定时器
  cleanupNetwork()
  // 清理更新事件监听器
  unregisterUpdateEventListeners()
  // 清理自定义事件监听器
  unregisterCustomEventListeners()
  // 清理会话排序定时器
  if (conversationSortTimer !== null) {
    clearTimeout(conversationSortTimer)
    conversationSortTimer = null
  }
})

const sortedConversations = computed(() => {
  return [...conversations.value].sort((a, b) => {
    if (a.is_pinned && !b.is_pinned) return -1
    if (!a.is_pinned && b.is_pinned) return 1
    return b.timestamp - a.timestamp
  })
})

const filteredConversations = computed(() => {
  if (!searchQuery.value) {
    return sortedConversations.value
  }
  
  const query = searchQuery.value.toLowerCase()
  return sortedConversations.value.filter(conv => 
    conv.name.toLowerCase().includes(query) ||
    (conv.lastMessage?.content && conv.lastMessage.content.toLowerCase().includes(query)) ||
    (conv.members && conv.members.some(member => 
      member.name.toLowerCase().includes(query)
    )) ||
    (conv.type === 'group' && '群聊'.includes(query)) ||
    (conv.type === 'single' && '用户'.includes(query))
  )
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
    logger.error('搜索失败:', error)
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

// 处理申请加入群组
const handleApplyJoinGroup = async (item) => {
  if (!item || !item.id) {
    QMessage.error('群组信息无效')
    return
  }

  try {
    const response = await request(`/api/v1/groups/${item.id}/apply`, {
      method: 'POST'
    })

    if (response.code === 0 || response.code === 200) {
      QMessage.success('申请已发送，请等待管理员审批')
      // 关闭搜索悬浮框
      searchQuery.value = ''
      searchResults.value = []
    } else {
      QMessage.error(response.message || '申请加入失败')
    }
  } catch (error) {
    logger.error('申请加入群组失败:', error)
    QMessage.error('网络错误，申请加入失败')
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




// 播放消息提示音

// 处理流式消息
const streamMessage = useStreamMessage(messages, serverUrl)
const { handleStreamMessage } = streamMessage

// 使用 Main.vue 专用的消息发送 composable
const mainMessageSending = useMainMessageSending(currentConversationId, messages, currentConversation, isConnected, sessionExpired, handleStreamMessage, () => {
  nextTick(() => chatWindowRef.value?.scrollToBottom())
})
const { handleSendMessage, handleRecallMessage, handleRetrySendMessage } = mainMessageSending




// 处理屏幕共享数据
const handleScreenShareData = (data: { conversationId: number; data: string }) => {
  sendMessage({
    type: 'screen-share.data',
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
      return getProductName()
  }
}

// 开始私聊
const startPrivateChat = async (user: any) => {
  // 关闭搜索悬浮框
  searchQuery.value = ''
  searchResults.value = []
  
  // 关闭成员右键菜单
  hideMemberContextMenu()
  
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
    
    const response = await request('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({
        type: 'single',
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
    logger.error('创建私聊失败:', error)
    QMessage.error('创建私聊失败')
    // 模拟创建会话（当API调用失败时）
    activeOption.value = 'recent'
    // 创建一个模拟的会话
    const mockConversation = {
      id: `conv_${Date.now()}`,
      name: user.name,
      avatar: user.avatar,
      lastMessage: null,
      unread_count: 0,
      timestamp: Date.now(),
      type: 'single',
      members: [
        { id: currentUser.value?.id || 'me', name: currentUser.value?.nickname || currentUser.value?.username || '我', avatar: getAvatarUrl(currentUser.value?.avatar, '我', serverUrl.value) },
        { id: user.id, name: user.name, avatar: user.avatar }
      ]
    }
    // 添加到会话列表
    conversations.value.unshift(mockConversation)
    // 选择新创建的会话
    setCurrentConversationId(mockConversation.id)
    // 初始化消息列表
    if (mockConversation.id) {
      chatStore.clearMessages(String(mockConversation.id))
    }
  }
  hideUserContextMenu()
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


const { orgStructure, loadOrganizationTree, collectEmployees } = orgLogic

const handleUserClick = (employee: any) => {
  selectedUser.value = employee
}

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
    logger.error('加载最近使用的应用失败:', error)
    QMessage.error('加载最近使用的应用失败')
  }
  // 默认最近使用的应用为空
  return []
}

const recentApps = ref(loadRecentApps())

// 所有应用列表（包括内置应用和自定义应用）
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

// 快速工具列表（分类为 tool 的内置应用，如短链接管理）
const quickTools = computed(() => {
  return builtInApps.value
    .filter(app => app.category === 'tool')
    .map(app => ({
      id: app.code || String(app.id),
      name: app.name,
      icon: app.icon || 'fas fa-cube',
      description: '',
    }))
})

// 主要应用列表（全部从后端内置应用动态加载）
const mainApps = computed(() => {
  return builtInApps.value
    .filter(app => app.category !== 'tool')
    .map(app => ({
      id: app.code || String(app.id),
      name: app.name,
      icon: app.icon || 'fas fa-cube',
    }))
})

// 系统应用列表
const systemApps = computed(() => {
  const apps = [
    { id: 'app-management', name: '应用管理', icon: 'fas fa-cog' }
  ]
  return apps
})

// 自定义应用列表
const customApps = ref<any[]>([])

// 内置应用列表已从 useAppLogic 中获取

// 应用分类列表（从内置应用动态加载）
const appCategories = computed(() => {
  return [
    {
      id: '1',
      name: '内置应用',
      expanded: categoryExpanded.value['1'],
      apps: builtInApps.value.map(app => ({
          id: app.code || String(app.id),
          name: app.name,
          code: app.code || '',
          icon: app.icon || 'fas fa-cube',
          url: app.url,
          openType: app.open_type || app.openType || 'in-app'
        }))
    },
    {
      id: '2',
      name: '自定义应用',
      expanded: categoryExpanded.value['2'],
      apps: customApps.value.map(app => ({
        id: String(app.id),
        name: app.name,
        code: app.code || '',
        icon: app.icon,
        url: app.url,
        openType: app.openType
      }))
    },
    {
      id: '3',
      name: '应用管理',
      expanded: categoryExpanded.value['3'],
      apps: [
        { id: 'app-management', name: '管理应用', icon: 'fas fa-cog' }
      ]
    }
  ]
})

// 应用分类展开/折叠状态（独立管理，避免 computed 重建）
const categoryExpanded = ref<Record<string, boolean>>({
  '1': true,
  '2': false,
  '3': false
})
// 加载用户创建的应用
const loadUserApps = async () => {
  try {
    const response = await request('/api/v1/apps')
    if (response.code === 0) {
      const userApps = response.data.list || response.data
      // 更新自定义应用列表
      customApps.value = userApps.map((app: any) => ({
        id: 'user-' + app.id.toString(),
        name: app.name,
        icon: app.icon,
        url: app.url,
        openType: app.open_type || app.openType || 'in-app'
      }))
    }
  } catch (error) {
    logger.error('加载用户应用失败:', error)
  }
}

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
    logger.error('加载应用失败:', error)
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
    logger.error('创建应用失败:', error)
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
    logger.error('更新应用失败:', error)
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
      logger.error('删除应用失败:', error)
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


// 当前打开的用户应用
const currentUserApp = ref<any>(null)

// 小程序面板状态
const showMiniAppList = ref(false)

// 笔记数据


// 应用逻辑（使用 useAppLogic composable，共享 selectedAppId 等状态）
const appLogic = useAppLogic({ selectedAppId, recentApps, currentUserApp, showMiniAppList, externalCustomApps: customApps })
const { openApp, openUserApp, openExternalApp, loadBuiltInApps: loadAppCategories, builtInApps } = appLogic

// 创建新笔记

// 创建新笔记


// 返回应用列表



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


// 处理会话创建成功
const handleConversationCreated = (newConversation: any) => {
  // 重新加载会话列表
  loadConversations()
  
  // 如果传递了新创建的会话对象，直接切换到新会话
  if (newConversation && newConversation.id) {
    const conversationId = String(newConversation.id)
    setCurrentConversationId(conversationId)
    chatStore.clearMessages(conversationId)
    messagePage.value = 1
    hasMoreMessages.value = true
    
    // 加载新会话的消息
    loadMessages(conversationId)
  }
}

// 打开系统消息发布模态框
// 关闭系统消息发布模态框
// 发送系统消息
const sendSystemMessage = async (msg: { title: string; content: string; target: string; groupId?: string; userId?: string; targetIds?: (string | number)[] }) => {
  if (!msg.title || !msg.content) {
    showMessage({ message: '请填写标题和内容', type: 'warning' })
    return
  }
  
  const payload: any = {
    title: msg.title,
    content: msg.content,
    target_type: msg.target || 'all',
  }
  
  if (msg.target === 'group' && msg.groupId) {
    payload.target_type = 'group'
    payload.target_id = parseInt(String(msg.groupId))
  } else if (msg.target === 'department') {
    payload.target_type = 'department'
    payload.target_ids = (msg.targetIds || []).map((id: any) => parseInt(String(id)))
  } else if (msg.target === 'user') {
    payload.target_type = 'user'
    payload.target_ids = (msg.targetIds || []).map((id: any) => parseInt(String(id)))
  }
  
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/system-messages`, payload, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (response.data.code === 0) {
      showMessage({ message: '系统消息发布成功', type: 'success' })
      systemMessage.value = { title: '', content: '', target: 'all', targetIds: [] }
      closeSystemMessageModal()
    } else {
      showMessage({ message: '系统消息发布失败: ' + response.data.message, type: 'error' })
    }
  } catch (error: any) {
    logger.error('发布系统消息失败:', error)
    const errMsg = error.response?.data?.message || '网络异常'
    showMessage({ message: '系统消息发布失败: ' + errMsg, type: 'error' })
  }
}


const createChannel = () => {
  hideActionMenu()
  activeOption.value = 'channels'
  nextTick(() => {
    showCreateChannelModal.value = true
  })}

// 新的频道交互方法

// 创建频道
const handleCreateChannelSubmit = async () => {
  if (!createChannelForm.value.name) {
    QMessage.error('请输入频道名称')
    return
  }

  try {
    const response = await request('/api/v1/channels', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(createChannelForm.value)
    })

    if (response.code === 0) {
      showCreateChannelModal.value = false
      createChannelForm.value = { name: '', description: '', avatar: '' }
      await channelStore.fetchChannels()
      QMessage.success('频道创建成功')
    } else {
      QMessage.error(response.message || '创建频道失败')
    }
  } catch (error) {
    logger.error('创建频道失败:', error)
    const errorMessage = error instanceof Error ? error.message : '创建频道失败'
    QMessage.error(errorMessage)
  }
}

// 关闭创建频道弹窗
const closeCreateChannelModal = () => {
  showCreateChannelModal.value = false
  createChannelForm.value = { name: '', description: '', avatar: '' }
}

const handleChannelSubscribe = async (channel: any) => {
  await channelStore.subscribeChannel(channel.id)
}

const handleChannelUnsubscribe = async (channel: any) => {
  await channelStore.unsubscribeChannel(channel.id)
}

const handleChannelSendMessage = async (channel: any, message: string) => {
  await sendChannelMessage(channel, message)
}

const handleDisplayModeChange = (mode: 'card' | 'timeline') => {
  channelStore.setMessageMode(mode)
}

const handleMessageLike = async (message: any) => {
  try {
    const response = await request(`/api/v1/channels/messages/${message.id}/like`, {
      method: 'POST'
    })
    if (response.code === 0) {
      QMessage.success('点赞成功')
    }
  } catch (error) {
    logger.error('点赞失败:', error)
    QMessage.error('点赞失败')
  }
}

const handleMessageUnlike = async (message: any) => {
  try {
    const response = await request(`/api/v1/channels/messages/${message.id}/like`, { method: 'DELETE'
    })
    if (response.code === 0) {
      QMessage.success('取消点赞')
    }
  } catch (error) {
    logger.error('取消点赞失败:', error)
    QMessage.error('取消点赞失败')
  }
}

const handleMessageComment = async (message: any) => {
  // 评论功能由子组件直接处理弹窗和提交
}

const createDiscussionGroup = () => {
  hideActionMenu()
  // 打开创建群聊模态框，类型为讨论组
  openCreateGroupModal('discussion')
  logger.log('创建讨论组')
}

const viewUserProfile = () => {
  if (selectedEmployee.value) {
    openUserProfile(selectedEmployee.value)
  }
  hideUserContextMenu()
}

const editAnnouncement = (group?: any) => {
  const targetGroup = group || selectedGroup.value
  if (targetGroup) {
    editAnnouncementContent.value = targetGroup.announcement || ''
    openEditAnnouncementModal()
  }
  closeGroupContextMenu()
}

const editGroupNameAction = () => {
  if (selectedGroup.value) {
    editGroupName.value = selectedGroup.value.name || ''
    openEditGroupNameModal()
  }
  closeGroupContextMenu()
}

// 从聊天窗口头部触发的编辑群名称（供 ChatHeaderActions 通过 inject 调用）
const openEditGroupNameFromHeader = (group: any) => {
  selectedGroup.value = group
  editGroupName.value = group.name || ''
  openEditGroupNameModal()
}

// 从聊天窗口头部触发的编辑群公告（供 ChatHeaderActions 通过 inject 调用）
const openEditAnnouncementFromHeader = (group: any) => {
  selectedGroup.value = group
  editAnnouncementContent.value = group.announcement || ''
  openEditAnnouncementModal()
}

provide('groupActions', {
  openEditGroupName: openEditGroupNameFromHeader,
  openEditAnnouncement: openEditAnnouncementFromHeader
})

const dissolveGroup = async (group?: any) => {
  const targetGroup = group || selectedGroup.value
  if (targetGroup) {
    // 调用 useGroup 中的实现
    const success = await groupState.dissolveGroup(targetGroup)
    if (success) {
      // 使用 Store Action 更新会话
      chatStore.patchConversation(targetGroup.id, {
        name: '[已解散] ' + (conversations.value.find(c => c.id === targetGroup?.id)?.name || ''),
        is_deleted: true
      })
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
      // 使用 Store Action 更新会话
      chatStore.patchConversation(selectedGroup.value.id, { announcement: editAnnouncementContent.value })
      closeEditAnnouncementModal()
    }
  }
}

const saveGroupName = async (newName: string) => {
  if (selectedGroup.value && newName.trim()) {
    const success = await groupState.updateGroup(selectedGroup.value.id, { name: newName.trim() })
    if (success) {
      // 更新本地群聊信息（副作用处理）
      selectedGroup.value.name = newName.trim()
      // 使用 Store Action 更新会话
      chatStore.patchConversation(selectedGroup.value.id, { name: newName.trim() })
      closeEditGroupNameModal()
    }
  }
}

const closeMemberContextMenu = () => {
  showMemberContextMenuFlag.value = false
  selectedMember.value = null
  document.removeEventListener('click', closeMemberContextMenu)
}

// AI 设置
const showAISettingsModal = ref(false)


const closeAISettings = () => {
  showAISettingsModal.value = false
}

const parseTriggerKeywords = (raw: string | undefined): string[] => {
  if (!raw) return []
  return raw.split(',').filter(Boolean)
}

const updateAISettings = async (settings: any) => {
  if (!selectedGroup.value) return
  try {
    const response = await request(`/api/v1/groups/${selectedGroup.value.id}/ai-settings`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        ai_enabled: settings.aiEnabled,
        ai_assistant_name: settings.aiAssistantName,
        ai_reply_mode: settings.aiReplyMode,
        ai_personality: settings.aiPersonality,
        ai_custom_prompt: settings.aiCustomPrompt,
        ai_language: settings.aiLanguage,
        ai_max_length: settings.aiMaxLength,
        ai_mention_reply_mode: settings.aiMentionReplyMode,
        ai_anti_spam_interval: settings.aiAntiSpamInterval,
        ai_trigger_keywords: settings.aiTriggerKeywords.join(','),
        ai_learn_enabled: settings.aiLearnEnabled,
        ai_extract_todos: settings.aiExtractTodos
      })
    })
    if (response.code === 0) {
      QMessage.success('AI 助手设置已更新')
      selectedGroup.value.ai_config = {
        ai_enabled: settings.aiEnabled,
        ai_assistant_name: settings.aiAssistantName,
        ai_reply_mode: settings.aiReplyMode,
        ai_personality: settings.aiPersonality,
        ai_custom_prompt: settings.aiCustomPrompt,
        ai_language: settings.aiLanguage,
        ai_max_length: settings.aiMaxLength,
        ai_mention_reply_mode: settings.aiMentionReplyMode,
        ai_anti_spam_interval: settings.aiAntiSpamInterval,
        ai_trigger_keywords: settings.aiTriggerKeywords.join(','),
        ai_learn_enabled: settings.aiLearnEnabled,
        ai_extract_todos: settings.aiExtractTodos
      }
      closeAISettings()
    } else {
      QMessage.error(response.message || '更新 AI 设置失败')
    }
  } catch (error: any) {
    logger.error('更新 AI 设置失败:', error)
    QMessage.error('网络错误，更新 AI 设置失败')
  }
}

const removeMemberFromGroup = async () => {
  if (selectedMember.value && selectedGroup.value) {
    try {
      const response = await request(`/api/v1/groups/${selectedGroup.value.id}/members/${selectedMember.value.id}`, {
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
      logger.error('移除成员失败:', error)
      QMessage.error('网络错误，移除成员失败')
    }
  }
  closeMemberContextMenu()
}

const viewMemberInfo = async () => {
  if (selectedMember.value) {
    const userId = selectedMember.value.user?.id || selectedMember.value.id
    if (!userId) {
      QMessage.error('无法获取用户ID')
      closeMemberContextMenu()
      return
    }

    const { profile, success } = await fetchUserProfile(userId, selectedMember.value)
    if (!success) {
      QMessage.error('获取用户信息失败')
    }
    openUserProfile(profile)
    closeMemberContextMenu()
  }
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

const viewGroupMembers = (group?: any) => {
  const targetGroup = group || selectedGroup.value
  if (targetGroup) {
    // 使用 useGroup 中的方法准备成员显示数据
    groupMembers.value = groupState.prepareGroupMembersForDisplay(targetGroup, serverUrl.value)
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

const addMembersToGroup = (group?: any) => {
  const targetGroup = group || selectedGroup.value
  if (targetGroup) {
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
  activeOption.value = 'apps'

  if (app && typeof app === 'object' && app.code) {
    selectedAppId.value = app.code
    logger.log('切换到内置应用:', app.code)
    return
  }

  selectedAppId.value = app
  logger.log('切换到应用:', app)
}

// 处理语音通话
const handleStartVoiceCall = async () => {
  logger.log('Main: 开始语音通话')
  if (realtimeRef.value) {
    await realtimeRef.value.startCall('voice')
  } else {
    QMessage.error('实时通信组件未初始化')
  }
}

// 处理视频通话
const handleStartVideoCall = async () => {
  logger.log('Main: 开始视频通话')
  if (realtimeRef.value) {
    await realtimeRef.value.startCall('video')
  } else {
    QMessage.error('实时通信组件未初始化')
  }
}

// 处理屏幕共享
const handleStartScreenShare = async () => {
  logger.log('Main: 开始屏幕共享')
  if (realtimeRef.value) {
    await realtimeRef.value.startScreenShare()
  } else {
    QMessage.error('实时通信组件未初始化')
  }
}

// 处理切换会话
const handleSwitchConversation = async (conversationId: string) => {
  // 确保 conversationId 是字符串类型
  const id = String(conversationId)
  // 切换到最近联系人选项卡
  activeOption.value = 'recent'
  // 重新加载会话列表
  await loadConversations()
  // 选择新会话
  setCurrentConversationId(id)
  // 加载新会话的消息
  await loadMessages(id)
}

// 分享逻辑
const shareLogic = useShareLogic(
  shareData, shareType, shareUsers, shareGroups,
  conversations, currentConversationId,
  loadConversations, handleSwitchConversation, closeShareModal
)
const { loadShareUsersAndGroups, handleShareConfirm, buildFileContent } = shareLogic

const userProfileActions = useUserProfile(currentUser, closeUserProfile)
const { triggerAvatarInput, handleAvatarChange, saveUserProfile } = userProfileActions

const exitGroup = async () => {
  if (selectedGroup.value) {
    if (confirm(`确定要退出${selectedGroup.value.name}吗？`)) {
      try {
        const response = await request(`/api/v1/groups/${selectedGroup.value.id}/exit`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 200) {
          // 使用 Store Action 标记群聊为已退出
          chatStore.patchConversation(selectedGroup.value.id, { isExited: true } as any)
          // 关闭群聊上下文菜单
          closeGroupContextMenu()
          showMessage({ message: '退出群聊成功', type: 'success' })
        } else {
          showMessage({ message: '退出群聊失败: ' + response.message, type: 'error' })
        }
      } catch (error) {
        logger.error('退出群聊失败:', error)
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
        // 使用 Store Action 移除成员
        chatStore.removeGroupMember(selectedGroup.value.id, member.id)
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
    const response = await request(`/api/v1/groups/${selectedGroup.value.id}/members`, {
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
      
      // 使用 Store Action 添加成员
      newMembers.forEach(member => {
        chatStore.addGroupMember(selectedGroup.value.id, member)
      })
      
      QMessage.success('添加成员成功')
      closeAddMembersModal()
    } else {
      QMessage.error('添加成员失败: ' + response.message)
    }
  } catch (error: any) {
    logger.error('添加成员失败:', error)
    QMessage.error('添加成员失败，请稍后重试')
  }
}

// 点击其他地方关闭菜单由showContextMenu和showGroupContextMenu函数内部处理

const closeSettingsMenu = () => {
  showSettingsMenuFlag.value = false
  document.removeEventListener('click', closeSettingsMenu)
}

const checkForUpdates = () => {
  logger.log('检查更新')
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
    // 非 Electron 环境：通过后端 API 检查更新
    checkUpdateViaAPI()
  }
}

// 通过后端 API 检查更新（非 Electron 环境）
const checkUpdateViaAPI = async () => {
  try {
    const platform = detectPlatform()
    const response = await fetch(`${serverUrl.value}/api/v1/client/versions?platform=${platform}&pageSize=1`)
    if (!response.ok) throw new Error('检查更新失败')
    
    const result = await response.json()
    if (result.code === 0 && result.data?.list?.length > 0) {
      const latestVersion = result.data.list[0]
      const currentVersion = APP_CONFIG.version
      
      if (isNewerVersion(latestVersion.version, currentVersion)) {
        hasNewVersion.value = true
        updateResult.value = `发现新版本 v${latestVersion.version}`
        // 存储下载链接供后续使用
        latestDownloadUrl.value = latestVersion.downloadUrl
      } else {
        hasNewVersion.value = false
        updateResult.value = '当前已是最新版本'
      }
    } else {
      hasNewVersion.value = false
      updateResult.value = '当前已是最新版本'
    }
  } catch (error: any) {
    updateResult.value = `检查更新失败: ${error.message || '网络错误'}`
  } finally {
    isCheckingUpdate.value = false
  }
}

// 检测当前平台
const detectPlatform = (): string => {
  const ua = navigator.userAgent.toLowerCase()
  if (ua.includes('mac')) return 'macos'
  if (ua.includes('linux')) return 'linux'
  return 'windows'
}

// 版本号比较：判断新版本是否比当前版本更新
const isNewerVersion = (newVer: string, currentVer: string): boolean => {
  const newParts = newVer.split('.').map(Number)
  const currentParts = currentVer.split('.').map(Number)
  for (let i = 0; i < Math.max(newParts.length, currentParts.length); i++) {
    const n = newParts[i] || 0
    const c = currentParts[i] || 0
    if (n > c) return true
    if (n < c) return false
  }
  return false
}

const latestDownloadUrl = ref('')

const downloadUpdate = () => {
  logger.log('下载升级')
  
  // 向主进程发送下载更新请求
  if (window.electron) {
    isDownloading.value = true
    downloadProgress.value = 0
    window.electron.ipcRenderer.send('download-update')
  } else {
    // 非 Electron 环境：打开下载链接
    if (latestDownloadUrl.value) {
      window.open(latestDownloadUrl.value, '_blank')
      updateResult.value = '已在浏览器中打开下载页面'
    } else {
      updateResult.value = '暂无可用的下载链接'
    }
  }
}

const aboutApp = () => {
  logger.log('关于应用')
  showAboutDialog.value = true
  closeSettingsMenu()
}

const emit = defineEmits<{
  (e: 'logout'): void
}>()


const logout = () => {
  logger.log('退出登录')
  // 显示退出登录确认弹窗
  showLogoutDialog.value = true
  closeSettingsMenu()
}

// 确认登出 - 执行实际的登出操作
const handleConfirmLogout = async () => {
  // 关闭登出确认对话框
  cancelLogout()
  
  // 调用后端登出接口（不等待结果，避免阻塞跳转）
  const token = getToken()
  fetch(`${serverUrl.value}/api/v1/auth/logout`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  }).catch(() => {})
  
  // 跳转到登录页（gotoLogin 内部会清理 localStorage 并跳转到 /）
  gotoLogin()
}


const handleSaveSettings = async (data: { profile: any; messageSettings: any; appearanceSettings: any; fileSettings: any; avatarFile?: File }) => {
  try {
    if (data.avatarFile) {
      const formData = new FormData()
      formData.append('file', data.avatarFile)
      
      const uploadResponse = await request('/api/v1/upload', {
        method: 'POST',
        body: formData
      })
      
      if (uploadResponse.code === 0 && uploadResponse.data && uploadResponse.data.url) {
        const updateResponse = await request('/api/v1/users/me', {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            avatar: uploadResponse.data.url
          })
        })
        
        if (updateResponse.code === 0 && updateResponse.data) {
          if (currentUser.value) {
            currentUser.value.avatar = updateResponse.data.avatar || uploadResponse.data.url
            localStorage.setItem('user', JSON.stringify(currentUser.value))
          }
          showMessage({ message: '头像更新成功', type: 'success' })
        }
      } else {
        showMessage({ message: '头像上传失败: ' + uploadResponse.message, type: 'error' })
        return
      }
    }
    
    settingsProfile.value = { ...settingsProfile.value, ...data.profile }
    messageSettings.value = { ...messageSettings.value, ...data.messageSettings }
    appearanceSettings.value = { ...appearanceSettings.value, ...data.appearanceSettings }
    fileSettings.value = { ...fileSettings.value, ...data.fileSettings }
    
    if (data.appearanceSettings.theme && data.appearanceSettings.theme !== currentTheme.value) {
      setTheme(data.appearanceSettings.theme)
    }
    if (data.appearanceSettings.fontSize) {
      applyFontSize(data.appearanceSettings.fontSize)
    }
    
    await saveSettings()
    closeSettingsModal()
  } catch (error) {
    logger.error('保存设置失败:', error)
    showMessage({ message: '保存失败: ' + error.message, type: 'error' })
  }
}



// 初始化主题
initTheme()
</script>

<style scoped>
@import url('./Main.css');
</style>

