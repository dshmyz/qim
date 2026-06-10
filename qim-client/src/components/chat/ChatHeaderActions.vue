<template>
  <div class="group-panel-container">
    <!-- 头部操作区域 -->
    <div class="group-header-actions">
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
    <Teleport to="body">
      <Transition name="dropdown">
        <div v-if="showHeaderMenu" class="header-menu-teleport" :style="headerMenuPosition" @click.stop>
          <!-- 群聊相关菜单项 -->
          <div v-if="isGroupOrDiscussion" class="menu-item" @click="handleEditGroupInfo">
            <i class="fas fa-edit"></i> 修改群名称
          </div>
          <div v-if="isGroupOrDiscussion" class="menu-item" @click="handleEditGroupAnnouncement">
            <i class="fas fa-bullhorn"></i> 编辑群公告
          </div>
          <div v-if="systemConfigStore.enableAI && isGroupOrDiscussion" class="menu-item" @click="handleOpenAISettings">
            <i class="fas fa-robot"></i> AI 助手设置
          </div>
          <!-- 私聊相关菜单项 -->
          <div v-if="systemConfigStore.enableAI && showAvatarToggle" class="avatar-toggle-menu-item">
            <div class="avatar-toggle-header">
              <i class="fas fa-user-circle"></i>
              <span class="avatar-toggle-title">对其启用AI分身</span>
              <Switch
                :model-value="avatarEnabled ?? false"
                :size="'small'"
                :disabled="avatarApprovalStatus !== 'approved'"
                title="开启后，AI分身将在当前会话中代替你回复消息"
                @change="(value) => handleToggleAvatar(value)"
              />
            </div>
            <div v-if="avatarApprovalStatus && avatarApprovalStatus !== 'approved'" class="avatar-toggle-hint">
              {{ avatarApprovalStatus === 'pending' ? '⏳ 审批中...' : avatarApprovalStatus === 'rejected' ? '❌ 审批未通过' : '' }}
            </div>
            <div v-else class="avatar-toggle-hint">
              仅对当前会话生效
            </div>
          </div>
          <!-- 群聊解散 -->
          <div v-if="isGroupOrDiscussion && isOwner" class="menu-item" @click="handleConfirmDeleteGroup">
            <i class="fas fa-trash"></i> 解散群聊
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 本地确认对话框 -->
    <Teleport to="body">
      <div v-if="localShowConfirmDialog" class="confirm-dialog-modal" @click="closeLocalConfirmDialog">
        <div class="confirm-dialog-content" @click.stop>
          <div class="confirm-dialog-header">
            <h3>{{ localConfirmDialogTitle }}</h3>
            <button class="close-btn" @click="closeLocalConfirmDialog">&times;</button>
          </div>
          <div class="confirm-dialog-body">
            <p>{{ localConfirmDialogMessage }}</p>
          </div>
          <div class="confirm-dialog-footer">
            <button class="cancel" @click="closeLocalConfirmDialog">取消</button>
            <button class="confirm" @click="executeConfirmCallback">确定</button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- AI 助手设置模态框 -->
    <ModalContainer
      :visible="showAISettingsModal"
      title="AI 助手设置"
      @close="handleCloseAISettingsModal"
      @cancel="handleCloseAISettingsModal"
      :show-footer="false"
      :content-style="{ width: '480px', minWidth: '480px' }"
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
import MemberContextMenu from './MemberContextMenu.vue'
import GroupAIPanel from '../ai/GroupAIPanel.vue'
import Switch from '../common/Switch.vue'
import ModalContainer from '../shared/ModalContainer.vue'

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
  showHeaderMenu: boolean
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
  'update:showHeaderMenu': [value: boolean]
  'invite-members': []
  'delete-group': []
  'switch-conversation': [conversationId: string]
  'show-user-profile': [user: any]
  'remove-member': [memberId: string, memberName: string]
  'set-admin': [memberId: string, memberName: string, isAdmin: boolean]
  'transfer-owner': [memberId: string, memberName: string]
  'start-private-chat': [memberId: string]
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
}>()

// 注入群管理操作（来自 Main.vue）
const groupActions = inject('groupActions', null) as {
  openEditGroupName: (group: any) => void
  openEditAnnouncement: (group: any) => void
} | null

// Refs
const moreButtonRef = ref<HTMLElement | null>(null)
const headerMenuPosition = ref<Record<string, string>>({})

// 使用 request composable
const { request } = useRequest()

// 本地确认对话框状态
const localShowConfirmDialog = ref(false)
const localConfirmDialogTitle = ref('确认操作')
const localConfirmDialogMessage = ref('')
const localConfirmDialogCallback = ref<(() => void) | null>(null)

// 成员管理状态
const showMemberContextMenuFlag = ref(false)
const memberContextMenuPosition = ref({ x: 0, y: 0 })
const selectedMember = ref<GroupMember | null>(null)
const isMembersSidebarExpanded = ref(true)
const showMemberSearch = ref(false)
const memberSearchQuery = ref(false)

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

// 切换头部下拉菜单
function handleToggleHeaderMenu() {
  const newValue = !props.showHeaderMenu
  emit('update:showHeaderMenu', newValue)
  if (newValue) {
    nextTick(() => {
      if (moreButtonRef.value) {
        const rect = moreButtonRef.value.getBoundingClientRect()
        const menuWidth = 180
        const menuHeight = 150
        const viewportWidth = window.innerWidth
        const viewportHeight = window.innerHeight

        let right = viewportWidth - rect.right
        if (right + menuWidth > viewportWidth) {
          right = 16
        }

        let top = rect.bottom + 8
        if (top + menuHeight > viewportHeight) {
          top = rect.top - menuHeight - 8
        }

        headerMenuPosition.value = {
          position: 'fixed',
          top: `${top}px`,
          right: `${right}px`
        }
      }
      document.addEventListener('click', closeHeaderMenu)
    })
  } else {
    document.removeEventListener('click', closeHeaderMenu)
  }
}

function closeHeaderMenu() {
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 邀请成员
function handleInviteMembers() {
  emit('invite-members')
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 编辑群信息
function handleEditGroupInfo() {
  if (props.conversation && groupActions) {
    groupActions.openEditGroupName(props.conversation)
  }
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 编辑群公告
function handleEditGroupAnnouncement() {
  if (props.conversation && groupActions) {
    groupActions.openEditAnnouncement(props.conversation)
  }
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

// 确认解散群聊
function handleConfirmDeleteGroup() {
  if (!props.conversation) return

  closeHeaderMenu()

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

function executeConfirmCallback() {
  if (localConfirmDialogCallback.value) {
    localConfirmDialogCallback.value()
  }
  closeLocalConfirmDialog()
}

// 成员侧边栏操作
function toggleMembersSidebar() {
  isMembersSidebarExpanded.value = !isMembersSidebarExpanded.value
}

function toggleMemberSearch() {
  showMemberSearch.value = !showMemberSearch.value
  if (showMemberSearch.value) {
    memberSearchQuery.value = false
  }
}

// AI 设置相关方法
function handleOpenAISettings() {
  showAISettingsModal.value = true
  emit('update:showHeaderMenu', false)
  document.removeEventListener('click', closeHeaderMenu)
}

function handleCloseAISettingsModal() {
  showAISettingsModal.value = false
}

function handleUpdateAISettings(settings: any) {
  emit('update-ai-settings', settings)
  showAISettingsModal.value = false
}

// 成员右键菜单
function computeMenuPosition(clientX: number, clientY: number, menuWidth: number = 160, menuHeight: number = 160) {
  const windowWidth = window.innerWidth
  const windowHeight = window.innerHeight

  let x = clientX
  let y = clientY

  if (x + menuWidth > windowWidth) {
    x = windowWidth - menuWidth - 10
  }
  if (x < 0) {
    x = 10
  }

  if (y + menuHeight > windowHeight) {
    y = windowHeight - menuHeight - 10
  }
  if (y < 0) {
    y = 10
  }

  return { x, y }
}

function handleShowMemberContextMenu(event: MouseEvent, member: GroupMember) {
  event.stopPropagation()

  const { x, y } = computeMenuPosition(event.clientX, event.clientY)

  memberContextMenuPosition.value = { x, y }
  selectedMember.value = member
  showMemberContextMenuFlag.value = true

  setTimeout(() => {
    document.addEventListener('click', closeMemberContextMenu)
  }, 0)
}

function closeMemberContextMenu() {
  showMemberContextMenuFlag.value = false
  selectedMember.value = null
  document.removeEventListener('click', closeMemberContextMenu)
}

function handleStartPrivateChat(member: GroupMember) {
  emit('start-private-chat', member.id)
  closeMemberContextMenu()
}

// 成员操作 - 转发给父组件处理
function handleViewMemberInfo() {
  if (selectedMember.value) {
    emit('show-user-profile', selectedMember.value)
  }
  closeMemberContextMenu()
}

function handleRemoveMember() {
  if (selectedMember.value) {
    emit('remove-member', selectedMember.value.id, selectedMember.value.name)
  }
  closeMemberContextMenu()
}

function handleSetAdmin() {
  if (selectedMember.value) {
    const isAdmin = selectedMember.value.role === 'admin'
    emit('set-admin', selectedMember.value.id, selectedMember.value.name, !isAdmin)
  }
  closeMemberContextMenu()
}

function handleTransferOwner() {
  if (selectedMember.value) {
    emit('transfer-owner', selectedMember.value.id, selectedMember.value.name)
  }
  closeMemberContextMenu()
}

function handleSendPrivateMessage() {
  if (selectedMember.value) {
    emit('start-private-chat', selectedMember.value.id)
  }
  closeMemberContextMenu()
}

// 清理
onUnmounted(() => {
  document.removeEventListener('click', closeHeaderMenu)
  document.removeEventListener('click', closeMemberContextMenu)
})
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
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.03) 0%, rgba(139, 92, 246, 0.03) 100%);
}

.avatar-toggle-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.avatar-toggle-header i {
  color: #3b82f6;
  font-size: 14px;
  margin-right: 8px;
}

.avatar-toggle-title {
  flex: 1;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.avatar-toggle-hint {
  margin-top: 6px;
  padding-left: 22px;
  font-size: 11px;
  color: var(--text-tertiary);
}

/* 模态框样式 */
.modal-overlay {
  position: fixed !important;
  top: 0 !important;
  left: 0 !important;
  right: 0 !important;
  bottom: 0 !important;
  background: rgba(0, 0, 0, 0.5) !important;
  display: flex !important;
  align-items: center !important;
  justify-content: center !important;
  z-index: 2000 !important;
  opacity: 1 !important;
  visibility: visible !important;
}

.modal-content {
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.modal-header .close-btn {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.modal-header .close-btn:hover {
  background: var(--hover-bg);
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  background: var(--background-light);
}

/* 表单样式 */
.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.form-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--background-light);
  color: var(--text-color);
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  background: var(--background-light);
  color: var(--text-color);
  resize: vertical;
  min-height: 100px;
  transition: border-color 0.2s;
  box-sizing: border-box;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 0;
}

/* 按钮样式 */
.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  background: var(--background-light);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--hover-bg);
}

/* 确认对话框样式 */
.confirm-dialog-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(8px);
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.confirm-dialog-content {
  background: var(--panel-bg, #ffffff);
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.25), 0 0 0 1px rgba(0, 0, 0, 0.05);
  width: 90%;
  max-width: 420px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.3s ease;
  overflow: hidden;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.confirm-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px 0;
  background: transparent;
  flex-shrink: 0;
}

.confirm-dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.confirm-dialog-header .close-btn {
  background: var(--color-gray-100, #f5f5f5);
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  transition: all 0.2s;
}

.confirm-dialog-header .close-btn:hover {
  background: var(--color-gray-200, #e5e5e5);
  color: var(--text-color);
}

.confirm-dialog-body {
  padding: 24px;
  background: transparent;
  flex: 1;
  overflow-y: auto;
}

.confirm-dialog-body p {
  margin: 0;
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.6;
  text-align: center;
}

.confirm-dialog-footer {
  display: flex;
  justify-content: center;
  gap: 12px;
  padding: 0 24px 24px;
  background: transparent;
  flex-shrink: 0;
}

.confirm-dialog-footer button {
  padding: 10px 28px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 100px;
}

.confirm-dialog-footer button.cancel {
  background: var(--color-gray-100, #f5f5f5);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.confirm-dialog-footer button.cancel:hover {
  background: var(--color-gray-200, #e5e5e5);
  border-color: var(--color-gray-300, #d4d4d4);
}

.confirm-dialog-footer button.confirm {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: #fff;
  box-shadow: 0 2px 8px rgba(239, 68, 68, 0.3);
}

.confirm-dialog-footer button.confirm:hover {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.4);
  transform: translateY(-1px);
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

/* AI 设置模态框 */
.ai-settings-modal {
  max-width: 480px;
}

.ai-settings-modal .modal-body {
  padding: 0;
}
</style>
