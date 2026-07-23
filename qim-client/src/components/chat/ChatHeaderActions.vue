<template>
  <div class="group-panel-container">
    <!-- 头部操作区域 -->
    <div class="group-header-actions">
      <span v-if="conversation?.type === 'group'" class="header-icon" title="群资料" @click.stop="emit('open-group-files')">
        <i class="fas fa-folder-open"></i>
      </span>
      <span v-if="isGroupOrDiscussion" class="header-icon" title="邀请成员" @click.stop="handleInviteMembers">
        <i class="fas fa-user-plus"></i>
      </span>
      <span 
        v-if="isGroupOrDiscussion || (systemConfigStore.enableAI && showAvatarToggle)" 
        class="header-icon" 
        title="更多选项" 
        @click.stop="handleToggleHeaderMenu" 
        ref="moreButtonRef"
      >
        <i class="fas fa-ellipsis-v"></i>
      </span>
    </div>

    <!-- 头部下拉菜单 -->
    <UniversalContextMenu menuId="header" :items="headerMenuItems">
      <!-- AI 分身开关（自定义内容） -->
      <div v-if="systemConfigStore.enableAI && showAvatarToggle" class="ucm-divider"></div>
      <div v-if="systemConfigStore.enableAI && showAvatarToggle" class="avatar-toggle-menu-item">
        <div class="avatar-toggle-header">
          <span class="ucm-icon"><i class="fas fa-user-circle" style="color: #3b82f6"></i></span>
          <span class="avatar-toggle-title">AI 分身</span>
          <Switch
            :model-value="avatarEnabled ?? false"
            :size="'small'"
            :disabled="avatarApprovalStatus !== 'approved'"
            @change="(value) => handleToggleAvatar(value)"
          />
        </div>
        <div class="avatar-toggle-hint">
          <template v-if="avatarApprovalStatus === 'pending'"><i class="fas fa-hourglass-half"></i> 审批中</template>
          <template v-else-if="avatarApprovalStatus === 'rejected'"><i class="fas fa-circle-xmark"></i> 审批未通过</template>
          <template v-else>代替你回复消息</template>
        </div>
      </div>
    </UniversalContextMenu>

    <!-- 确认对话框 -->
    <ConfirmDialog
      :visible="localShowConfirmDialog"
      :title="localConfirmDialogTitle"
      :message="localConfirmDialogMessage"
      confirm-text="确定"
      cancel-text="取消"
      @update:visible="handleConfirmDialogVisibleChange"
      @confirm="executeConfirmCallback"
    />

    <!-- AI 助手设置模态框 -->
    <ModalContainer
      :visible="showAISettingsModal"
      title="AI 助手设置"
      @close="handleCloseAISettingsModal"
      @cancel="handleCloseAISettingsModal"
      :show-footer="false"
      :content-style="{ width: '640px', minWidth: '640px' }"
    >
      <GroupAIPanel
        :group-id="groupId"
        :server-url="serverUrl"
        :ai-enabled="aiEnabled"
        :ai-assistant-name="aiAssistantName"
        :ai-reply-mode="aiReplyMode"
        :ai-personality="aiPersonality"
        :ai-custom-prompt="aiCustomPrompt"
        :ai-language="aiLanguage"
        :ai-max-length="aiMaxLength"
        :ai-mention-reply-mode="aiMentionReplyMode"
        :ai-anti-spam-interval="aiAntiSpamInterval"
        :ai-trigger-keywords="aiTriggerKeywords"
        :ai-learn-enabled="aiLearnEnabled"
        @update="handleUpdateAISettings"
      />
    </ModalContainer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted, inject } from 'vue'
import QMessage from '../../utils/qmessage'
import type { Conversation } from '../../types'
import { getCurrentUser } from '../../utils/user'
import { useRequest } from '../../composables/useRequest'
import { useSystemConfigStore } from '../../stores/systemConfig'
import MemberSidebar from './MemberSidebar.vue'
import GroupAIPanel from '../ai/GroupAIPanel.vue'
import Switch from '../common/Switch.vue'
import ModalContainer from '../shared/ModalContainer.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'
import UniversalContextMenu from '../shared/UniversalContextMenu.vue'
import type { ContextMenuItem } from '../shared/context-menu-types'
import { activeMenu, openMenu, closeMenu } from '../../composables/useUI'

const systemConfigStore = useSystemConfigStore()

// 类型定义
interface GroupMember {
  id: string
  name: string
  role?: 'owner' | 'admin' | 'member' | 'user' | 'guest'
  avatar?: string
  [key: string]: unknown
}

interface GroupConversation extends Omit<Conversation, 'members'> {
  announcement?: string
  members?: GroupMember[]
}

// Props 定义
interface Props {
  conversation: Conversation | null
  currentUser: any
  serverUrl: string
  aiEnabled?: boolean
  aiAssistantName?: string
  aiReplyMode?: string
  contextMessages?: number
  aiPersonality?: string
  aiCustomPrompt?: string
  aiLanguage?: string
  aiMaxLength?: string
  aiMentionReplyMode?: string
  aiAntiSpamInterval?: number
  aiTriggerKeywords?: string[]
  aiLearnEnabled?: boolean
  avatarEnabled?: boolean
  avatarApprovalStatus?: string
}

const props = withDefaults(defineProps<Props>(), {
  aiEnabled: false,
  aiAssistantName: 'AI助手',
  aiReplyMode: 'mention_only',
  contextMessages: 10,
  aiPersonality: 'professional',
  aiCustomPrompt: '',
  aiLanguage: 'auto',
  aiMaxLength: 'medium',
  aiMentionReplyMode: 'mention',
  aiAntiSpamInterval: 5,
  aiTriggerKeywords: () => [],
  aiLearnEnabled: false
})

// Emits 定义
const emit = defineEmits<{
  'invite-members': []
  'delete-group': []
  'open-group-files': []
  'update-avatar-enabled': [enabled: boolean]
  'update-ai-settings': [settings: {
    aiEnabled: boolean;
    aiAssistantName: string;
    aiReplyMode: string;
    aiPersonality: string;
    aiCustomPrompt: string;
    aiLanguage: string;
    aiMaxLength: string;
    aiMentionReplyMode: string;
    aiAntiSpamInterval: number;
    aiTriggerKeywords: string[];
    aiLearnEnabled: boolean;
  }]
  'update-avatar-enabled': [value: boolean]
  'open-group-files': []
}>()

// 注入群管理操作（来自 Main.vue）
const groupActions = inject('groupActions', null) as {
  openEditGroupName: (group: any) => void
  openEditAnnouncement: (group: any) => void
} | null

// Refs
const moreButtonRef = ref<HTMLElement | null>(null)

// 使用 request composable
const { request } = useRequest()

// 本地确认对话框状态
const localShowConfirmDialog = ref(false)
const localConfirmDialogTitle = ref('确认操作')
const localConfirmDialogMessage = ref('')
const localConfirmDialogCallback = ref<(() => void) | null>(null)

// AI 设置状态
const showAISettingsModal = ref(false)

// Computed
const isGroupOrDiscussion = computed(() => {
  return props.conversation?.type === 'group' || props.conversation?.type === 'discussion'
})

const members = computed(() => {
  return props.conversation?.members || []
})

const isOwner = computed(() => {
  return isGroupOwner(props.conversation)
})

const currentUserId = computed((): string | number => {
  const user = props.currentUser || getCurrentUser()
  return user?.id ?? ''
})

const groupId = computed((): number => {
  return typeof props.conversation?.id === 'string'
    ? parseInt(props.conversation.id, 10)
    : (props.conversation?.id ?? 0)
})

const currentUserRole = computed((): string => {
  let currentUser = props.currentUser
  if (!currentUser) {
    currentUser = getCurrentUser()
  }
  if (!props.conversation?.members || !currentUser) return 'member'
  const member = props.conversation.members.find((m) => String(m.id) === String(currentUser.id))
  return (member?.role as string) || 'member'
})

// 是否显示分身开关（私聊时显示）
const showAvatarToggle = computed(() => {
  return props.conversation?.type === 'single'
})

// 处理分身开关切换
function handleToggleAvatar(enabled: boolean) {
  emit('update-avatar-enabled', enabled)
}

// 方法
function isGroupOwner(conversation: Conversation | null): boolean {
  if (!conversation || !conversation.members) return false
  const currentUser = getCurrentUser()
  if (!currentUser) return false
  const currentUserId = currentUser.id?.toString() || ''
  const owner = conversation.members.find((member: any) => String(member.id) === currentUserId)
  return owner ? (owner.role as string) === 'owner' : false
}

// 头部下拉菜单（使用全局 activeMenu 协调）
const showHeaderMenu = computed(() => activeMenu.value === 'header')

const headerMenuItems = computed(() => {
  const items: ContextMenuItem[] = []
  if (isGroupOrDiscussion.value) {
    items.push(
      { label: '修改群名称', icon: 'fas fa-edit', action: handleEditGroupInfo },
      { label: '编辑群公告', icon: 'fas fa-volume-high', action: handleEditGroupAnnouncement }
    )
  }
  if (systemConfigStore.enableAI && isGroupOrDiscussion.value) {
    items.push({ label: 'AI 助手设置', icon: 'fas fa-robot', action: handleOpenAISettings })
  }
  if (isGroupOrDiscussion.value && isOwner.value) {
    items.push({ label: '解散群聊', icon: 'fas fa-trash', danger: true, action: handleConfirmDeleteGroup })
  }
  return items
})

function handleToggleHeaderMenu() {
  if (showHeaderMenu.value) {
    closeMenu()
  } else {
    if (moreButtonRef.value) {
      const rect = moreButtonRef.value.getBoundingClientRect()
      openMenu('header', rect.right - 180, rect.bottom + 4)
    }
  }
}

// 邀请成员
function handleInviteMembers() {
  emit('invite-members')
  closeMenu()
}

// 编辑群信息
function handleEditGroupInfo() {
  if (props.conversation && groupActions) {
    groupActions.openEditGroupName(props.conversation)
  }
  closeMenu()
}

// 编辑群公告
function handleEditGroupAnnouncement() {
  if (props.conversation && groupActions) {
    groupActions.openEditAnnouncement(props.conversation)
  }
  closeMenu()
}

// 确认解散群聊
function handleConfirmDeleteGroup() {
  if (!props.conversation) return

  closeMenu()

  openLocalConfirmDialog(
    '确认解散群聊',
    '确定要解散此群聊吗？解散后所有消息和成员数据将被删除。',
    () => {
      emit('delete-group')
    }
  )
}

// 本地确认对话框方法
function openLocalConfirmDialog(title: string, message: string, callback: () => void) {
  localConfirmDialogTitle.value = title
  localConfirmDialogMessage.value = message
  localConfirmDialogCallback.value = callback
  localShowConfirmDialog.value = true
}

function closeLocalConfirmDialog() {
  localShowConfirmDialog.value = false
  localConfirmDialogCallback.value = null
}

// ConfirmDialog 的 visible 变化处理（关闭按钮/遮罩点击触发）
function handleConfirmDialogVisibleChange(value: boolean) {
  localShowConfirmDialog.value = value
  if (!value) {
    localConfirmDialogCallback.value = null
  }
}

function executeConfirmCallback() {
  if (localConfirmDialogCallback.value) {
    localConfirmDialogCallback.value()
  }
  closeLocalConfirmDialog()
}

// AI 设置相关方法
function handleOpenAISettings() {
  showAISettingsModal.value = true
  closeMenu()
}

function handleCloseAISettingsModal() {
  showAISettingsModal.value = false
}

function handleUpdateAISettings(settings: any) {
  emit('update-ai-settings', settings)
  showAISettingsModal.value = false
}

</script>

<style scoped>
.group-panel-container {
  position: relative;
  display: flex;
  align-items: center;
  flex: 1;
  justify-content: flex-end;
}

.group-header-actions {
  display: flex;
  gap: 8px;
  position: relative;
}

.header-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-color);
  opacity: 0.7;
  font-size: 14px;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;
}

.header-icon:hover {
  background: var(--hover-color);
  opacity: 1;
}

/* 头部下拉菜单 */
.header-menu-teleport {
  position: fixed;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 9999;
  min-width: 180px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 13px;
}

.menu-item:hover {
  background-color: var(--hover-bg);
}

.menu-item i {
  margin-right: 8px;
  color: var(--text-secondary);
}

/* AI 分身开关菜单项 */
.avatar-toggle-menu-item {
  padding: 8px 16px;
}

.avatar-toggle-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.avatar-toggle-title {
  font-size: 13px;
  color: var(--text-color);
}

.avatar-toggle-hint {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-secondary);
  opacity: 0.7;
}

.avatar-toggle-hint i {
  margin-right: 4px;
}

/* 下拉动画 */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
