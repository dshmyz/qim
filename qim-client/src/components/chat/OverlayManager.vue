<template>
  <!-- 用户资料弹窗 -->
  <UserProfile
    v-if="selectedUser"
    :visible="showUserProfile"
    :user="selectedUser"
    @close="emit('close-user-profile')"
    @send-private-message="emit('send-private-message', selectedUser.id)"
  />

  <!-- 已读用户列表弹窗 -->
  <ReadUsersModal
    :visible="showReadUsersModal"
    :read-users="currentReadUsers"
    :server-url="serverUrl"
    @close="emit('close-read-users')"
  />

  <!-- 消息右键菜单 -->
  <MessageContextMenu
    :visible="showMessageContextMenu"
    :position="messageContextMenuPosition"
    :message="selectedMessage"
    @save-file-as="emit('save-file-as', $event); closeMessageMenu()"
    @download-file="emit('download-file', $event); closeMessageMenu()"
    @copy-message="emit('copy-message'); closeMessageMenu()"
    @forward-message="emit('forward-message'); closeMessageMenu()"
    @quote-message="emit('quote-message'); closeMessageMenu()"
    @add-to-notes-app="emit('add-to-notes-app'); closeMessageMenu()"
    @create-task="emit('create-task'); closeMessageMenu()"
    @recall-message="emit('recall-message'); closeMessageMenu()"
    @send-message-reminder="emit('send-message-reminder'); closeMessageMenu()"
    @ai-summary="emit('ai-summary'); closeMessageMenu()"
    @translate="emit('translate'); closeMessageMenu()"
    @smart-reply="emit('smart-reply'); closeMessageMenu()"
  />

  <!-- 成员右键菜单 -->
  <MemberContextMenu
    :visible="showMemberContextMenu"
    :position="memberContextMenuPosition"
    :member="selectedMember"
    :current-user-id="currentUserId ?? ''"
    :conversation="conversation ?? undefined"
    @close="emit('close-member-context-menu')"
    @remove-member="(memberId, _memberName) => emit('remove-member', String(memberId))"
    @set-admin="(_memberId, _memberName, _isAdmin) => emit('set-admin', String(_memberId))"
    @transfer-owner="(memberId, _memberName) => emit('transfer-owner', String(memberId))"
    @view-member-info="emit('view-member-info')"
    @send-private-message="emit('send-private-message', selectedMember?.id ?? '')"
  />

  <!-- 消息管理器 -->
  <MessageManager
    v-if="showMessageManager && conversationId"
    :visible="showMessageManager"
    :conversation-id="String(conversationId)"
    @close="emit('close-message-manager')"
    @scroll-to-message="emit('scroll-to-message', $event)"
  />

  <!-- 确认对话框 -->
  <ConfirmDialog
    :visible="showConfirmDialog"
    :title="confirmDialogTitle"
    :message="confirmDialogMessage"
    @update:visible="(v) => emit('update-confirm-dialog', v)"
    @confirm="emit('confirm-action')"
    @cancel="emit('cancel-confirm-action')"
  />

  <!-- 截图预览对话框 -->
  <ScreenshotPreviewDialog
    :visible="showScreenshotPreview"
    :image-data="screenshotImageData"
    @cancel="emit('cancel-screenshot')"
    @retake="emit('retake-screenshot')"
    @send="emit('send-screenshot')"
  />

  <!-- 小程序加载器 -->
  <div style="display: contents">
    <MiniAppLoader
      :mini-app="activeMiniApp"
      @close="emit('close-mini-app')"
      @show-toast="emit('mini-app-toast', $event)"
    />
  </div>
</template>

<script setup lang="ts">
import type { Conversation, Message, User } from '../../types'
import UserProfile from '../modals/UserProfile.vue'
import ReadUsersModal from './ReadUsersModal.vue'
import MessageContextMenu from './MessageContextMenu.vue'
import MemberContextMenu from './MemberContextMenu.vue'
import MessageManager from './MessageManager.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'
import ScreenshotPreviewDialog from './ScreenshotPreviewDialog.vue'
import MiniAppLoader from '../miniapp/MiniAppLoader.vue'
import type { MiniAppData } from '../miniapp/MiniAppLoader.vue'

interface Props {
  conversation: Conversation | null
  conversationId: string | number | null
  senderName: string
  serverUrl: string
  currentUserId: string | number | null
  showUserProfile: boolean
  selectedUser: User | null
  showReadUsersModal: boolean
  currentReadUsers: { read_users: User[]; total_members: number }
  showMessageContextMenu: boolean
  messageContextMenuPosition: { x: number; y: number }
  selectedMessage: Message | null
  showMemberContextMenu: boolean
  memberContextMenuPosition: { x: number; y: number }
  selectedMember: User | null
  showMessageManager: boolean
  showConfirmDialog: boolean
  confirmDialogTitle: string
  confirmDialogMessage: string
  showScreenshotPreview: boolean
  screenshotImageData: string
  otherUserId: string | number | null
  activeMiniApp: MiniAppData | null
  getFileIcon: (fileName: string) => string
  formatFileSize: (size: number) => string
  renderMarkdown: (content: string) => string
  formatTime: (timestamp: number) => string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'close-user-profile': []
  'send-private-message': [userId: string | number]
  'close-read-users': []
  'save-file-as': [data: string]
  'download-file': [data: string]
  'copy-message': []
  'forward-message': []
  'quote-message': []
  'add-to-notes-app': []
  'create-task': []
  'recall-message': []
  'close-message-menu': []
  'send-message-reminder': []
  'close-member-context-menu': []
  'remove-member': [memberId: string]
  'set-admin': [memberId: string]
  'transfer-owner': [memberId: string]
  'view-member-info': []
  'close-message-manager': []
  'scroll-to-message': [messageId: string]
  'update-confirm-dialog': [visible: boolean]
  'confirm-action': []
  'cancel-confirm-action': []
  'cancel-screenshot': []
  'retake-screenshot': []
  'send-screenshot': []
  'close-mini-app': []
  'mini-app-toast': [message: string]
  'ai-summary': []
  'translate': []
  'smart-reply': []
}>()

const closeMessageMenu = () => {
  emit('close-message-menu')
}

const formatTimeWithCoerce = (timestamp: string | number | null | undefined) => {
  if (timestamp == null) return ''
  return props.formatTime(Number(timestamp))
}

defineExpose({})
</script>
