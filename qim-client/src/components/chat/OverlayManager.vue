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
    @preview-image="emit('preview-image', $event)"
    @save-file-as="emit('save-file-as', $event)"
    @download-file="emit('download-file', $event)"
    @copy-message="emit('copy-message')"
    @forward-message="emit('forward-message')"
    @quote-message="emit('quote-message')"
    @add-to-note="emit('add-to-note')"
    @recall-message="emit('recall-message')"
    @send-message-reminder="emit('send-message-reminder')"
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

  <!-- 通话模态框 -->
  <CallModal
    :visible="showCallModal"
    :call-type="callType"
    :status="callStatus"
    :avatar="callAvatar"
    :name="callName"
    @reject-call="emit('reject-call')"
    @answer-call="emit('answer-call')"
    @end="emit('end-call')"
    @close="emit('close-call-modal')"
  />

  <!-- 图片预览弹窗 -->
  <ImagePreviewDialog
    :visible="showImagePreview"
    :image-url="previewImageUrl"
    @close="emit('close-image-preview')"
  />

  <!-- 分享内容预览弹窗 -->
  <SharePreviewDialog
    v-if="showSharePreview"
    :visible="showSharePreview"
    :preview-data="sharePreviewData"
    :get-file-icon="getFileIcon"
    :format-file-size="formatFileSize"
    :render-markdown="renderMarkdown"
    :format-time="formatTimeWithCoerce"
    @close="emit('close-share-preview')"
    @download-file="emit('download-file', $event)"
    @save-file-as="emit('save-file-as', $event)"
  />

  <!-- 屏幕共享组件 -->
  <ScreenShare
    :receiver-id="otherUserId ?? undefined"
    :sender-id="remoteScreenUserId"
    :sender-name="senderName"
    :conversation-id="conversationId ?? undefined"
    ref="screenShareRef"
    @screen-share-start="(data) => emit('screen-share-start', data)"
    @screen-share-stop="emit('screen-share-stop')"
    @screen-share-join="emit('screen-share-join')"
    @screen-share-leave="emit('screen-share-leave')"
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
import { ref } from 'vue'
import type { Conversation, Message, User } from '../../types'
import UserProfile from '../modals/UserProfile.vue'
import ReadUsersModal from './ReadUsersModal.vue'
import MessageContextMenu from './MessageContextMenu.vue'
import MemberContextMenu from './MemberContextMenu.vue'
import MessageManager from './MessageManager.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'
import ScreenshotPreviewDialog from './ScreenshotPreviewDialog.vue'
import CallModal from './CallModal.vue'
import ImagePreviewDialog from './ImagePreviewDialog.vue'
import SharePreviewDialog from './SharePreviewDialog.vue'
import ScreenShare from '../shared/ScreenShare.vue'
import MiniAppLoader from '../miniapp/MiniAppLoader.vue'
import type { MiniAppData } from '../miniapp/MiniAppLoader.vue'

interface SharePreviewData {
  type: 'file' | 'note' | 'sticky'
  name: string
  content?: string
  url?: string
  path?: string
  size?: number
  created_at?: string
}

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
  showCallModal: boolean
  callType: 'voice' | 'video' | ''
  callStatus: 'ringing' | 'answered' | 'ended' | ''
  callAvatar: string
  callName: string
  showImagePreview: boolean
  previewImageUrl: string
  showSharePreview: boolean
  sharePreviewData: SharePreviewData
  otherUserId: string | number | null
  remoteScreenUserId: number | null
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
  'preview-image': [data: string]
  'save-file-as': [data: string]
  'download-file': [data: string]
  'copy-message': []
  'forward-message': []
  'quote-message': []
  'add-to-note': []
  'recall-message': []
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
  'reject-call': []
  'answer-call': []
  'end-call': []
  'close-call-modal': []
  'close-image-preview': []
  'close-share-preview': []
  'screen-share-start': [data: { conversationId: string | number }]
  'screen-share-stop': []
  'screen-share-join': []
  'screen-share-leave': []
  'close-mini-app': []
  'mini-app-toast': [message: string]
}>()

const screenShareRef = ref<InstanceType<typeof ScreenShare>>()

const formatTimeWithCoerce = (timestamp: string | number | null | undefined) => {
  if (timestamp == null) return ''
  return props.formatTime(Number(timestamp))
}

defineExpose({
  stopReceiving: () => screenShareRef.value?.stopReceiving()
})
</script>
