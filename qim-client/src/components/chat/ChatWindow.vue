<template>
  <div class="chat-window">
    <!-- 头部 -->
    <ChatHeader
      ref="chatHeaderRef"
      :conversation="conversation"
      :current-user="currentUser"
      :server-url="serverUrl"
      :avatar-enabled="avatarEnabled"
      :avatar-approval-status="avatarApprovalStatus"
      @invite-members="handleInviteMembers"
      @delete-group="confirmDeleteConversation"
      @switch-conversation="handleSwitchConversation"
      @show-user-profile="showUserProfile"
      @remove-member="handleRemoveMember"
      @set-admin="handleSetAdmin"
      @transfer-owner="handleTransferOwner"
      @start-private-chat="handleStartPrivateChat"
      @update-ai-settings="handleUpdateAISettings"
      @update-avatar-enabled="handleUpdateAvatarEnabled"
      @open-group-files="openGroupFiles()"
    />

    <!-- 分身接管横幅 -->
    <AvatarTakeoverBanner
      v-if="avatarTakeoverUntil"
      :takeover-until="avatarTakeoverUntil"
      @resume="handleAvatarResume"
      @extend="handleAvatarExtend"
    />

    <!-- @消息提醒横幅 -->
    <AtMentionBanner
      v-if="unreadAtMentionCount > 0"
      :count="unreadAtMentionCount"
      @navigate="navigateToFirstAtMention"
    />

    <!-- 消息列表和成员侧边栏 -->
    <ChatBody
      ref="chatBodyRef"
      :conversation="conversation"
      :messages="messages"
      :has-more-messages="hasMoreMessages"
      :read-users-map="readUsersMap"
      :show-read-receipt="showReadReceipt"
      :server-url="serverUrl"
      :current-user-id="currentUserId"
      :selection-mode="isMessageSelectionMode"
      :selected-message-ids="selectedMessageIds"
      :is-members-sidebar-expanded="isMembersSidebarExpanded"
      :show-member-search="showMemberSearch"
      :member-search-query="memberSearchQuery"
      @message-contextmenu="showMessageContextMenu"
      @show-user-profile="showUserProfile"
      @scroll-to-quoted-message="scrollToQuotedMessage"
      @scroll="closeMessageContextMenu"
      @download-file="downloadFile"
      @save-as="saveFileAs"
      @open-mini-app="(app) => app && openMiniApp(app as MiniAppData)"
      @open-news-link="openNewsLink"
      @retry-send-message="retrySendMessage"
      @show-read-users="showReadUsers"
      @mark-read="handleMarkRead"
      @toggle-message-selection="toggleMessageSelection"
      @load-more="loadMoreMessages"
      @toggle-members-sidebar="toggleMembersSidebar"
      @toggle-member-search="toggleMemberSearch"
      @member-search-focus="() => showMemberSearch = true"
      @show-member-context-menu="handleShowMemberContextMenu"
      @start-private-chat="(member) => handleStartPrivateChat(String(member.id))"
      @update:member-search-query="(val) => memberSearchQuery = val"
    />

    <div v-if="isMessageSelectionMode" class="message-selection-actions" aria-label="多选消息操作" role="toolbar">
      <span class="message-selection-count">已选择 {{ selectedMessages.length }} 条</span>
      <button class="message-selection-cancel" type="button" @click="cancelMessageSelection">取消</button>
      <button class="message-selection-forward" type="button" :disabled="selectedMessages.length < 2" @click="forwardSelectedMessages">合并转发</button>
    </div>

    <!-- 上传进度条 -->
    <Transition name="fade">
      <div v-if="uploadProgress > 0 && uploadProgress < 100" class="upload-progress-bar">
        <div class="upload-progress-bar__inner" :style="{ width: uploadProgress + '%' }"></div>
        <span class="upload-progress-bar__text">正在上传 {{ uploadProgress }}%</span>
      </div>
    </Transition>

    <!-- 输入区域 -->
    <ChatInputArea
      ref="chatInputRef"
      :conversation="conversation"
      v-model:input-message="inputMessage"
      :pending-files="pendingFiles"
      :show-emoji-panel="showEmojiPanel"
      :show-at-members-panel="showAtMembersPanel"
      :at-members-query="atMembersQuery"
      :show-slash-panel="showSlashPanel"
      :slash-query="slashQuery"
      :slash-items="slashItems"
      :slash-title="slashMode === 'command' ? '命令' : (activeSlashCommand?.title || '')"
      :slash-item-component="slashMode === 'command' ? CommandListItemRaw : (activeSlashCommand?.itemComponent as any)"
      :slash-footer-label="slashFooterLabel"
      :slash-footer-icon="slashFooterIcon"
      v-model:show-mini-app-list="showMiniAppList"
      :quoted-message="quotedMessage"
      :is-electron="isElectron"
      :get-file-icon="getFileIcon"
      :processing-action="currentProcessingAction"
      :draft-streaming="isDraftStreaming"
      :has-draft-reply="hasDraftReply"
      :show-search="showSearch"
      v-model:search-query="searchQuery"
      @regenerate-draft="regenerateDraftReply"
      @send="handleSend"
      @input="handleInput"
      @cursor-change="handleInput"
      @toggle-emoji-panel="toggleEmojiPanel"
      @close-emoji-panel="closeEmojiPanel"
      @select-file="selectFile"
      @select-image="selectImage"
      @open-code-block="showCodeBlockEditor = true"
      @take-screenshot="takeScreenshot"
      @take-screenshot-hidden="takeScreenshotHidden"
      @open-message-manager="openMessageManager"
      @open-mini-app-list="openMiniAppList"
      @start-voice-call="startVoiceCall"
      @start-video-call="startVideoCall"
      @start-screen-share="startScreenShare"
      @insert-emoji="insertEmoji"
      @close-at-members-panel="closeAtMembersPanel"
      @select-at-member="selectAtMember"
      @select-at-all="selectAtAll"
      @select-slash-item="selectSlashItem"
      @close-slash-panel="closeSlashPanel"
      @slash-footer-action="openQuickReplyManager"
      @handle-file-select="handleFileSelect"
      @handle-paste="handlePaste"
      @handle-drop="handleDrop"
      @handle-keydown="handleKeydown"
      @remove-pending-file="removePendingFile"
      @remove-quoted-message="quotedMessage = null"
      @ai-action="handleAIAction"
      @perform-search="performSearch"
      @close-search="showSearch = false"
      @update:input-message="(val) => inputMessage = val"
      @update:show-mini-app-list="(val) => showMiniAppList = val"
      @update:search-query="(val) => searchQuery = val"
    />

    <CodeBlockEditor
      v-model:visible="showCodeBlockEditor"
      @confirm="handleSendCodeBlock"
    />

    <!-- 弹窗管理器 -->
    <OverlayManager
      ref="overlayRef"
      :conversation="conversation"
      :conversation-id="conversation?.id ?? null"
      :sender-name="conversation?.name ?? ''"
      :server-url="serverUrl"
      :current-user-id="currentUserId"
      :show-user-profile="showUserProfileFlag"
      :selected-user="selectedUser"
      :show-read-users-modal="showReadUsersModal"
      :current-read-users="currentReadUsers"
      :selected-message="selectedMessage"
      :can-manage-group-files="canManageGroupFiles"
      :selected-member="selectedMember"
      :show-message-manager="showMessageManager"
      :show-confirm-dialog="showConfirmDialog"
      :confirm-dialog-title="confirmDialogTitle"
      :confirm-dialog-message="confirmDialogMessage"
      :other-user-id="otherUserId"
      :active-mini-app="activeMiniApp"
      :get-file-icon="getFileIcon"
      :format-file-size="formatFileSize"
      :render-markdown="renderMarkdown"
      :format-time="formatTime"
      @close-user-profile="closeUserProfile"
      @send-private-message="handleSendPrivateMessage"
      @close-read-users="showReadUsersModal = false"
      @save-file-as="(d: string) => saveFileAs(d, selectedMessage ? String(selectedMessage.id) : undefined)"
      @download-file="(d: string) => downloadFile(d, selectedMessage ? String(selectedMessage.id) : undefined)"
      @copy-message="selectedMessage && copyMessage(selectedMessage, messageMenuSelection)"
      @copy-code="selectedMessage && handleCopyCode(selectedMessage)"
      @forward-message="forwardMessage"
      @select-messages="startMessageSelection"
      @quote-message="quoteMessage"
      @add-to-notes-app="addToNotesApp"
      @create-task="createTaskFromMessage"
      @recall-message="handleRecallMessage"
      @send-message-reminder="sendMessageReminder"
      @ai-summary="handleAISummary"
      @translate="handleAITranslate"
      @smart-reply="handleSmartReply"
      @save-to-group-files="saveMessageToGroupFiles"
      @remove-member="handleRemoveMemberFromOverlay"
      @set-admin="handleSetAdminFromOverlay"
      @transfer-owner="handleTransferOwnerFromOverlay"
      @view-member-info="viewMemberInfo"
      @close-message-manager="closeMessageManager"
      @scroll-to-message="scrollToMessage"
      @update-confirm-dialog="(v) => showConfirmDialog = v"
      @confirm-action="handleConfirmAction"
      @cancel-confirm-action="closeConfirmDialog"
      @close-mini-app="activeMiniApp = null"
      @mini-app-toast="handleMiniAppToast"
    />

    <!-- 群名称编辑弹窗和群公告编辑弹窗已移至 Main.vue 中的 GroupModals 组件 -->

    <!-- 快速回复管理弹窗 -->
    <QuickReplyManager v-model:visible="showQuickReplyManager" />

    <!-- AI 摘要面板 -->
    <AISummaryPanel
      v-if="conversation?.id"
      :visible="showSummaryPanel"
      :conversation-id="Number(conversation.id)"
      time-range="today"
      @close="showSummaryPanel = false"
    />

    <!-- AI 翻译面板 -->
    <AITranslatePanel
      :visible="showTranslatePanel"
      :original-text="translateContent"
      :message-type="translateMessageType"
      @close="showTranslatePanel = false"
    />

    <GroupFilesPanel
      v-if="showGroupFiles"
      :group-id="conversation.id"
      :can-manage="canManageGroupFiles"
      :reference-message-id="referenceMessageId"
      :reference-file-id="referenceFileId"
      @close="closeGroupFiles"
    />

    <!-- AI 文本预览弹窗（改写/润色/翻译） -->
    <AITextPreview
      :visible="previewVisible"
      :original-text="previewOriginal"
      :result-text="previewResult"
      :action-type="previewType"
      @replace="handlePreviewReplace"
      @cancel="previewVisible = false"
    />

  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted, defineAsyncComponent, provide, reactive, shallowRef, markRaw } from 'vue'
import type { Conversation, Message } from '../../types'
import QMessage from '../../utils/qmessage'
import ChatBody from './ChatBody.vue'
import ChatInputArea from './ChatInputArea.vue'
import CodeBlockEditor from './CodeBlockEditor.vue'
import OverlayManager from './OverlayManager.vue'
import ChatHeader from './ChatHeader.vue'
import { useServerUrl } from '../../composables/useServerUrl'
import { getCurrentUser } from '../../utils/user'
import { addWsHandlers } from '../../composables/useWebSocket'
import { useMessageActions } from '../../composables/useMessageActions'
import '../../assets/styles/modules/modals.css'
import { useChatRequest } from '../../composables/useChatRequest'
import { useMessageReminder } from '../../composables/useMessageReminder'
import { useChatUtils } from '../../composables/useChatUtils'
import { fetchUserProfile } from '../../composables/useUserProfileInfo'
import { useChatState } from '../../composables/useChatState'
import { useAIActions } from '../../composables/useAIActions'
import { useSlashCommandPanelEnabled } from '../../composables/useSlashCommandPanelEnabled'
import { getAvatarUrl, generateAvatar } from '../../utils/avatar'
// AI 快捷键已移至 Main.vue（全局生效）
// summary / quickPanel 通过事件触发
// useAISidebar 已移至 Main.vue
import AITextPreview from '../ai/AITextPreview.vue'
import { logger } from '../../utils/logger'
import { saveDraft } from '../../utils/drafts'
import {
  findActiveMentionToken,
  reconcileMentionSpans,
  replaceMentionToken,
  serializeToContent,
  type MentionSpan,
} from '../../utils/mentions'
// 大组件懒加载，按需加载减少 chat chunk 体积
const GroupModals = defineAsyncComponent(() => import('../modals/GroupModals.vue'))
const AISummaryPanel = defineAsyncComponent(() => import('../ai/AISummaryPanel.vue'))
const AITranslatePanel = defineAsyncComponent(() => import('../ai/AITranslatePanel.vue'))
import AvatarTakeoverBanner from '../avatar/AvatarTakeoverBanner.vue'
import AtMentionBanner from '../message/AtMentionBanner.vue'
import type { MiniAppData } from '../miniapp/MiniAppLoader.vue'
import { useRealtimeStore } from '../../stores/realtime'
import { useTaskStore } from '../../stores/task'
import { useSystemConfigStore } from '../../stores/systemConfig'
import { RealtimeConnectionManager, RealtimeViewerConnection } from '../../utils/realtimeConnection'
import { useAvatar } from '../../composables/useAvatar'
import { decodeToPlainText } from '../../utils/mentions'
import { replaceSlashToken, findCommandList, type SlashCommand, type SlashCommandItem, type ActiveSlashToken } from '../../utils/slashCommand'
import { getSlashCommandRegistry } from '../../utils/slashCommands'
import CommandListItem from './CommandListItem.vue'
import { isMessageSelectionEligible } from '../../utils/messageSelection'
import GroupFilesPanel from '../groups/GroupFilesPanel.vue'
import { activeMenu, activeMenuPosition, openMenu, closeMenu } from '../../composables/useUI'
import QuickReplyManager from './QuickReplyManager.vue'

// 命令列表模式下用于 <component :is>，markRaw 避免被 Vue 当作响应式对象
const CommandListItemRaw = markRaw(CommandListItem)

// 服务器地址
const { serverUrl } = useServerUrl()

// 当前用户（需要在 useScreenShare 之前声明）
const currentUser = ref(getCurrentUser())
const currentUserId = computed((): string | number => {
  const user = currentUser.value || props.currentUser
  return user?.id ?? ''
})

// 初始化 composables
const { getToken, request } = useChatRequest(serverUrl.value)
const { remind } = useMessageReminder(serverUrl.value)
const { formatTime, getFileIcon, formatFileSize, renderMarkdown } = useChatUtils()
const { $message, showConfirmDialog, confirmDialogTitle, confirmDialogMessage, openConfirmDialog, closeConfirmDialog, handleConfirmAction } = useChatState()

// 实时通信 store 和连接管理器
const realtimeStore = useRealtimeStore()
const systemConfigStore = useSystemConfigStore()
const connectionManager = new RealtimeConnectionManager()
const viewerConnection = new RealtimeViewerConnection()

// 设置观看者连接的回调
viewerConnection.setCallbacks({
  onRemoteStream: (_stream: MediaStream) => {
    logger.log('ChatWindow: 收到远程流（由 RealtimeCommunication 处理）')
  },
  onConnectionStateChange: (state: RTCPeerConnectionState) => {
    logger.log('ChatWindow: WebRTC 连接状态变化', state)
  },
  onError: (error: Error) => {
    logger.error('ChatWindow: WebRTC 连接错误', error)
  }
})

// AI 操作 composable
const {
  isProcessing: aiIsProcessing,
  translateText,
  rewriteText,
  polishText,
  draftReply,
  draftReplyStream,
  abortDraftReply,
} = useAIActions()

// AI 文本预览状态（改写/润色/翻译结果）
const previewVisible = ref(false)
const previewOriginal = ref('')
const previewResult = ref('')
const previewType = ref<'rewrite' | 'polish' | 'translate'>('rewrite')

// 当前处理中的按钮 ID（用于按钮级 loading）
const currentProcessingAction = computed(() => {
  if (isDraftStreaming.value) return 'draft-reply'
  if (aiIsProcessing.value) return currentAIAction.value
  return null
})
const currentAIAction = ref<string | null>(null)

const handlePreviewReplace = () => {
  inputMessage.value = previewResult.value
  autoResizeTextarea()
  previewVisible.value = false
  $message.success('已替换原文')
}

// 帮我回复草稿状态
const isDraftStreaming = ref(false)      // 正在流式生成
const hasDraftReply = ref(false)          // 已生成草稿、可换一个
const draftReplyTarget = ref<any>(null)   // 当前草稿回复的目标消息
const isAppendingDraft = ref(false)       // 流式追加中（用于区分用户输入与 AI 填充）

// 分身 composable
const { takeoverSession, getSession, avatarConfig, avatarApprovalStatus, fetchConfig, fetchSessions, toggleSession, isAvatarActive } = useAvatar()
const avatarEnabled = computed(() => props.conversation?.id ? isAvatarActive(props.conversation.id) : (avatarConfig.value?.enabled ?? false))

// AI 摘要面板状态
const showSummaryPanel = ref(false)

// AI 侧边栏状态（使用全局状态）
// useAISidebar 已移至 Main.vue（全局状态）

// AI 翻译面板状态
const showTranslatePanel = ref(false)
const translateContent = ref('')
const translateMessageType = ref<'image' | 'text' | undefined>(undefined)
const showGroupFiles = ref(false)
const referenceMessageId = ref<number | null>(null)
const referenceFileId = ref<number | null>(null)

// 处理分身启用状态更新（只控制当前会话，不影响全局配置）
const handleUpdateAvatarEnabled = async (enabled: boolean) => {
  try {
    if (props.conversation?.id) {
      await toggleSession(props.conversation.id, enabled)
    }
    if (enabled) {
      $message.success('分身已开启')
    } else {
      $message.success('分身已关闭')
    }
  } catch (error) {
    $message.error('切换分身状态失败')
  }
}

// AI 快捷操作处理
// 启动流式帮我回复：清空输入框并逐字填充
const startDraftReply = (conversationId: number, messageId: number, target: any) => {
  if (isDraftStreaming.value) abortDraftReply()
  draftReplyTarget.value = target
  inputMessage.value = ''
  autoResizeTextarea()
  isDraftStreaming.value = true
  hasDraftReply.value = false
  draftReplyStream(conversationId, messageId, {
    onChunk: (content: string) => {
      isAppendingDraft.value = true
      inputMessage.value += content
      isAppendingDraft.value = false
      autoResizeTextarea()
    },
    onComplete: () => {
      isDraftStreaming.value = false
      hasDraftReply.value = !!inputMessage.value
      autoResizeTextarea()
      if (inputMessage.value) {
        $message.success('回复草稿已生成')
      } else {
        $message.warning('未生成内容，请重试')
      }
    },
    onError: () => {
      isDraftStreaming.value = false
      hasDraftReply.value = false
      $message.error('生成回复失败')
    },
  })
}

// 换一个：基于同一目标消息重新生成草稿
const regenerateDraftReply = () => {
  const conversationId = props.conversation?.id
  const target = draftReplyTarget.value
  if (!conversationId || !target) return
  startDraftReply(Number(conversationId), Number(target.id), target)
}

const handleAIAction = async (actionId: string) => {
  const text = inputMessage.value.trim()

  switch (actionId) {
    case 'draft-reply': {
      const conversationId = props.conversation?.id
      if (!conversationId) return
      // 找最后一条对方发来的文本消息
      const uid = String(currentUserId.value)
      const target = [...props.messages].reverse().find(
        (m: any) => String(m.sender_id) !== uid && (m.type === 'text' || m.type === 'markdown') && !m.isRecalled
      )
      if (!target) {
        $message.warning('没有可回复的消息')
        return
      }
      startDraftReply(Number(conversationId), Number(target.id), target)
      break
    }
    case 'summary':
      if (props.conversation?.id) {
        showSummaryPanel.value = true
      }
      break
    case 'translate': {
      if (!text) {
        $message.warning('请先输入需要翻译的文本')
        return
      }
      currentAIAction.value = 'translate'
      try {
        const result = await translateText(text, 'zh')
        previewOriginal.value = text
        previewResult.value = result
        previewType.value = 'translate'
        previewVisible.value = true
      } catch {
        $message.error('翻译失败')
      } finally {
        currentAIAction.value = null
      }
      break
    }
    case 'rewrite': {
      if (!text) {
        $message.warning('请先输入需要改写的文本')
        return
      }
      currentAIAction.value = 'rewrite'
      try {
        const result = await rewriteText(text, 'concise', 'professional')
        previewOriginal.value = text
        previewResult.value = result
        previewType.value = 'rewrite'
        previewVisible.value = true
      } catch {
        $message.error('改写失败')
      } finally {
        currentAIAction.value = null
      }
      break
    }
    case 'polish': {
      if (!text) {
        $message.warning('请先输入需要润色的文本')
        return
      }
      currentAIAction.value = 'polish'
      try {
        const result = await polishText(text, 'zh')
        previewOriginal.value = text
        previewResult.value = result
        previewType.value = 'polish'
        previewVisible.value = true
      } catch {
        $message.error('润色失败')
      } finally {
        currentAIAction.value = null
      }
      break
    }
    default:
      break
  }
}

// 监听来自 Main.vue 的 AI 快捷键事件（summary / quickPanel）
const handleAISummaryAction = () => {
  if (props.conversation?.id) {
    showSummaryPanel.value = true
  }
}
const handleAIQuickPanelAction = () => {
  $message.info('AI 快捷面板')
}

interface Props {
  conversation: Conversation
  messages: Message[]
  getReadUsers?: (messageId: string) => Promise<{ read_users: any[], total_members: number }>
  currentUser: any
  hasMoreMessages: boolean
  updateConversation?: (conversation: Conversation) => void
  messageSettings?: { defaultSaveDirectory?: string; sendShortcut?: 'enter' | 'ctrl_enter' }
}

const props = defineProps<Props>()
const emit = defineEmits<{
  send: [message: { content: string; type: string; quotedMessage: unknown }]
  recall: [messageId: number]
  inviteMembers: [conversationId: string]
  'read-receipt': [conversationId: string]
  'switch-app': [app: string]
  'loadMore': [conversationId: string]
  'switchConversation': [conversationId: string]
  'retry-send': [message: any]
  'start-screen-share': []
  'start-voice-call': []
  'start-video-call': []
}>()

const unreadAtMentionCount = computed(() => {
  const currentUserId = props.currentUser?.id?.toString()
  if (!currentUserId) return 0
  return props.messages.filter(m => m.isAtMention && !m.isSelf && !m.isRead).length
})

const getConversationMemberUserId = (member: any): string => {
  const userId = member?.id ?? member?.user_id ?? member?.UserID ?? member?.user?.id ?? member?.User?.ID
  return userId === undefined || userId === null ? '' : String(userId)
}

const isSelfChatConversation = computed(() => {
  const conversation = props.conversation
  const currentId = props.currentUser?.id

  if (!conversation || currentId === undefined || currentId === null) return false
  if (conversation.type !== 'single') return false

  const currentIdText = String(currentId)
  const members = conversation.members || []
  if (members.length === 1) {
    return getConversationMemberUserId(members[0]) === currentIdText
  }

  return String((conversation as any).other_member_id || '') === currentIdText
})

const showReadReceipt = computed(() => {
  if (!systemConfigStore.enableReadReceipt) return false
  if (props.conversation?.type === 'bot') return false
  if (isSelfChatConversation.value) return false
  return true
})

const navigateToFirstAtMention = () => {
  const firstAtMention = props.messages.find(m => m.isAtMention && !m.isSelf && !m.isRead)
  if (firstAtMention) {
    scrollToMessage(String(firstAtMention.id))
    // 标记该消息为已读，横幅计数 -1（触发 unreadAtMentionCount 重算）
    firstAtMention.isRead = true
    // 同步到后端，确保刷新页面后状态保持（绕过节流，确保请求一定发出）
    if (props.conversation?.id) {
      forceMarkMessagesAsRead(props.conversation.id)
    }
  }
}

// 消息操作相关逻辑
const messageActions = useMessageActions(serverUrl, ref(props.currentUser))
const {
  readUsersMap,
  showReadUsersModal,
  currentReadUsers,
  fetchReadUsers,
  showReadUsers,
  markMessagesAsRead,
  forceMarkMessagesAsRead,
  recallMessage,
  deleteMessage,
  sendMessage,
  retrySendMessage,
  copyMessage,
  copyCode,
  loadReadUsersForMessages,
  debouncedLoadReadUsers,
  cleanup: cleanupMessageActions
} = messageActions

const remoteScreenUserId = ref<number | null>(null)

const startScreenShare = () => {
  emit('start-screen-share')
}

const inputMessage = ref('')
const quotedMessage = ref<any>(null)
const showCodeBlockEditor = ref(false)
const messageListRef = ref<HTMLDivElement>()
const chatHeaderRef = ref<any>()
const chatBodyRef = ref<InstanceType<typeof ChatBody>>()
const chatInputRef = ref<InstanceType<typeof ChatInputArea>>()
const overlayRef = ref<InstanceType<typeof OverlayManager>>()
const messageInputRef = ref<HTMLTextAreaElement>()
const showSearch = ref(false)
const searchQuery = ref('')
const searchResults = ref<Message[]>([])
const isSearching = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const lastConversationId = ref<string | null>(null)

const loadDraft = (conversationId: string) => {
  const draft = localStorage.getItem(`qim_draft_${conversationId}`)
  if (draft) {
    const { text, quoted } = JSON.parse(draft)
    inputMessage.value = text
    quotedMessage.value = quoted
  } else {
    inputMessage.value = ''
    quotedMessage.value = null
  }
}

watch(() => props.conversation?.id, (newId, oldId) => {
  if (newId && newId !== oldId) {
    closeGroupFiles()
    loadDraft(newId)

    // 切换会话时重置 mention 状态
    mentionSpans.value = []
    trackedInputMessage.value = ''
    pendingAtPosition.value = -1
    pendingAtEnd.value = -1
    atMembersQuery.value = ''
    showAtMembersPanel.value = false
    // 切换会话时清空斜杠命令面板
    clearSlashPanel()

    scrollToBottom()

    lastConversationId.value = newId
  }
})

// 标记是否正在插入系统消息（本地操作），此时不应触发滚动
const isInsertingSystemMessage = ref(false)

// 输入变化时保存草稿
watch(inputMessage, () => {
  if (props.conversation?.id) {
    saveDraft(props.conversation.id, inputMessage.value, quotedMessage.value)
  }
})

// 待发送文件
interface PendingFile {
  file: File
  name: string
}
const pendingFiles = ref<PendingFile[]>([])

// 成员上下文菜单
const selectedMember = ref<any>(null)

// 用户资料弹窗
const showUserProfileFlag = ref(false)
const selectedUser = ref<any>(null)

// 消息上下文菜单
const selectedMessage = ref<any>(null)
const messageMenuSelection = ref('')
const isMessageSelectionMode = ref(false)
const selectedMessageIds = ref(new Set<string>())
const selectedMessages = computed(() =>
  props.messages.filter(message =>
    selectedMessageIds.value.has(String(message.id)) && isMessageSelectionEligible(message)
  )
)

// 消息管理器
const showMessageManager = ref(false)

// 表情面板相关
const showEmojiPanel = ref(false)

// @成员功能相关
const showAtMembersPanel = ref(false)
const inputCursorPosition = ref(0)
const mentionSpans = ref<MentionSpan[]>([])
const trackedInputMessage = ref('')
const pendingAtPosition = ref(-1)
const pendingAtEnd = ref(-1)
const atMembersQuery = ref('')

// 斜杠命令补全相关（统一框架，支持 /task 等命令）
const slashCommandRegistry = getSlashCommandRegistry()
// 整个斜杠命令面板的开关：关闭后输入 / 不弹出任何命令面板
const { slashCommandPanelEnabled } = useSlashCommandPanelEnabled()
const showSlashPanel = ref(false)
// 'command' = 命令列表模式（输入 / 弹出所有命令）；'search' = 具体命令搜索模式（输入 /task 后）
const slashMode = ref<'command' | 'search'>('command')
const slashQuery = ref('')
const slashToken = ref<ActiveSlashToken | null>(null)
const slashItems = ref<SlashCommandItem[]>([])
// 当前活跃命令（search 模式下设置），用于面板标题/组件/getInsertText
// 用 shallowRef：命令对象本身不会变，避免对内含的 itemComponent 组件对象做深度响应式代理
const activeSlashCommand = shallowRef<{
  title: string
  itemComponent: import('vue').Component
  getInsertText: (item: SlashCommandItem) => string
  onSelect?: (item: SlashCommandItem, ctx: { conversationId: string; conversationType: string }) => import('../../utils/slashCommand').SlashCommandSelectResult
} | null>(null)

// backendSearch 命令（如 /file）的 debounce 控制状态
// - timer：debounce 定时器，每次输入变化都重置
// - seq：请求序列号，防止旧请求结果覆盖新请求结果（切换命令/关闭面板时递增使在途请求失效）
let backendSearchTimer: ReturnType<typeof setTimeout> | null = null
let backendSearchSeq = 0
const BACKEND_SEARCH_DEBOUNCE_MS = 250

/** 取消正在进行的 backendSearch：清理 timer 并让在途请求结果失效 */
const cancelBackendSearch = () => {
  if (backendSearchTimer) {
    clearTimeout(backendSearchTimer)
    backendSearchTimer = null
  }
  backendSearchSeq++
}

// 清空斜杠命令补全状态
const clearSlashPanel = () => {
  showSlashPanel.value = false
  slashMode.value = 'command'
  slashToken.value = null
  slashQuery.value = ''
  slashItems.value = []
  activeSlashCommand.value = null
  cancelBackendSearch()
}

// 快速回复管理弹窗
const showQuickReplyManager = ref(false)

// 仅当 /quick 命令激活时显示底部"管理"入口
const slashFooterLabel = computed(() =>
  slashMode.value === 'search' && activeSlashCommand.value?.title === '快速回复'
    ? '管理快速回复' : ''
)
const slashFooterIcon = computed(() => slashFooterLabel.value ? 'fas fa-cog' : '')

const openQuickReplyManager = () => {
  // 关闭补全面板，避免弹窗叠层
  clearSlashPanel()
  showQuickReplyManager.value = true
}

// 输入文本变化时同步 mention span（编辑后维持 span 与文本一致性）
watch(inputMessage, (nextText) => {
  mentionSpans.value = reconcileMentionSpans(mentionSpans.value, trackedInputMessage.value, nextText)
  trackedInputMessage.value = nextText
})

// 用户手动编辑输入框（非 AI 流式追加）时，撤销“换一个”可用态
watch(inputMessage, () => {
  if (!isAppendingDraft.value && !isDraftStreaming.value && hasDraftReply.value) {
    hasDraftReply.value = false
  }
}, { flush: 'sync' })

// 打开消息管理器
const openMessageManager = () => {
  showMessageManager.value = true
}

// 关闭消息管理器
const closeMessageManager = () => {
  showMessageManager.value = false
}

// 打开资讯链接
const openNewsLink = (url: string) => {
  if (!url) return
  // 这里可以实现打开链接的逻辑
  window.open(url, '_blank')
}

// 确认解散群聊（由 GroupManagementPanel 组件的确认对话框触发）
const confirmDeleteConversation = async () => {
  if (!props.conversation) return
  
  try {
    const response = await request(`/api/v1/groups/${props.conversation.id}`, {
      method: 'DELETE'
    })
    if (response.code === 0) {
      $message.success('群聊解散成功')
      // 触发切换会话事件，回到会话列表
      emit('switchConversation', '')
    } else {
      $message.error('解散群聊失败: ' + response.message)
    }
  } catch (error) {
    logger.error('解散群聊失败:', error)
    $message.error('解散群聊失败，请重试')
  }
}

// 切换表情面板
const toggleEmojiPanel = () => {
  showEmojiPanel.value = !showEmojiPanel.value
  // 如果显示表情面板，关闭搜索框
  if (showEmojiPanel.value) {
    showSearch.value = false
  }
}

// 插入表情
const insertEmoji = (emoji: string) => {
  inputMessage.value += emoji
  showEmojiPanel.value = false
  nextTick(() => {
    const textarea = chatInputRef.value?.messageInputRef?.messageInputRef as HTMLTextAreaElement | null
    if (textarea) {
      textarea.focus()
    }
  })
  autoResizeTextarea()
}

// 关闭表情面板
const closeEmojiPanel = () => {
  showEmojiPanel.value = false
}

const clearActiveMention = () => {
  pendingAtPosition.value = -1
  pendingAtEnd.value = -1
  atMembersQuery.value = ''
  showAtMembersPanel.value = false
}

// 处理输入事件，处理 @ 和斜杠命令补全
const handleInput = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement
  const value = textarea.value
  const cursorPos = textarea.selectionStart
  inputCursorPosition.value = cursorPos

  const convType = props.conversation?.type

  // @ 成员补全：仅群聊/讨论组启用
  if (convType === 'group' || convType === 'discussion') {
    const mentionToken = findActiveMentionToken(value, cursorPos)
    if (mentionToken) {
      pendingAtPosition.value = mentionToken.start
      pendingAtEnd.value = mentionToken.end
      atMembersQuery.value = mentionToken.query
      showAtMembersPanel.value = true
      clearSlashPanel()
      return
    }
  }
  clearActiveMention()

  // 斜杠命令补全：用户可在「设置 → 消息设置」里关闭整个斜杠命令面板
  if (!slashCommandPanelEnabled.value) {
    clearSlashPanel()
    return
  }

  const ctx = {
    conversationId: String(props.conversation?.id ?? ''),
    conversationType: String(convType ?? ''),
  }
  // 优先匹配具体命令（如 /task），命中则进入搜索模式
  const active = slashCommandRegistry.findActive(value, cursorPos, ctx)
  if (active) {
    slashToken.value = active.token
    slashQuery.value = active.token.query
    slashMode.value = 'search'
    activeSlashCommand.value = {
      title: active.command.title,
      itemComponent: active.command.itemComponent,
      getInsertText: active.command.getInsertText,
      onSelect: active.command.onSelect,
    }
    showSlashPanel.value = true
    // 切换命令或继续输入时，取消上一次 backendSearch（清理 timer + 在途请求失效）
    cancelBackendSearch()

    if (active.command.backendSearch) {
      // 后端搜索：debounce 后请求，query 变化时重新搜索
      const currentQuery = active.token.query
      const seq = backendSearchSeq
      backendSearchTimer = setTimeout(() => {
        backendSearchTimer = null
        // 取消竞态：若期间面板已关闭/命令已切换（seq 已变），不再请求
        if (seq !== backendSearchSeq) return
        void active.command.fetchItems(ctx, currentQuery).then(items => {
          // 只接受最新一次请求的结果，避免旧请求覆盖新结果
          if (seq !== backendSearchSeq) return
          slashItems.value = items
        })
      }, BACKEND_SEARCH_DEBOUNCE_MS)
      return
    }

    // 前端过滤：fetchItems 通常有缓存（只请求一次），之后用 filter 同步过滤
    void active.command.fetchItems(ctx).then(items => {
      slashItems.value = active.command.filter(items, active.token.query)
    })
    return
  }
  // 具体命令未命中，尝试命令列表模式（输入 / 弹出所有可用命令）
  const cmdList = findCommandList(value, cursorPos, ctx, slashCommandRegistry)
  if (cmdList) {
    // 从 search 模式切到 command 模式时，取消进行中的 backendSearch
    cancelBackendSearch()
    slashToken.value = cmdList.token
    slashQuery.value = cmdList.token.query
    slashMode.value = 'command'
    activeSlashCommand.value = null
    // 按 query 过滤命令列表（trigger 包含 query）
    const q = cmdList.token.query.trim().toLowerCase()
    const filtered = q
      ? cmdList.commands.filter(c => c.trigger.toLowerCase().includes(q))
      : cmdList.commands
    slashItems.value = filtered.map(c => ({ id: c.trigger, command: c }))
    showSlashPanel.value = filtered.length > 0
    return
  }
  clearSlashPanel()
}

// 处理粘贴事件
const handlePaste = async (event: ClipboardEvent) => {
  const items = event.clipboardData?.items
  if (!items) return
  
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (item.kind === 'file') {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        addPendingFile(file)
      }
    }
  }
}

// 处理拖拽文件
const handleDrop = (event: DragEvent) => {
  const files = event.dataTransfer?.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    addPendingFile(files[i])
  }
}

// 添加待发送文件
const addPendingFile = (file: File) => {
  pendingFiles.value.push({
    file: file,
    name: file.name
  })
}

// 移除待发送文件
const removePendingFile = (index: number) => {
  pendingFiles.value.splice(index, 1)
}

// 上传文件并发送
const uploadAndSendFile = async (file: File) => {
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('source', 'chat')

    const token = getToken()
    uploadProgress.value = 0

    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', `${serverUrl.value}/api/v1/upload`)
      if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          uploadProgress.value = Math.floor((e.loaded / e.total) * 100)
        }
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            const data = JSON.parse(xhr.responseText)
            if (data.code === 0) {
              const fileUrl = data.data.url
              const fileId = data.data.id
              const fileName = data.data.name || file.name
              const fileSize = data.data.size || file.size

              const messageData = {
                content: JSON.stringify({ url: fileUrl, id: fileId, name: fileName, size: fileSize }),
                type: file.type.startsWith('image/') ? 'image' : 'file',
                quotedMessage: quotedMessage.value
              }

              emit('send', messageData)
              quotedMessage.value = null
              resolve(data)
            } else {
              $message.error('上传文件失败: ' + data.message)
              reject(new Error(data.message))
            }
          } catch (e) {
            reject(e)
          }
        } else {
          $message.error('上传文件失败')
          reject(new Error(`HTTP ${xhr.status}`))
        }
      }

      xhr.onerror = () => {
        $message.error('上传文件失败')
        reject(new Error('Network error'))
      }

      xhr.onloadend = () => {
        uploadProgress.value = 0
      }

      xhr.send(formData)
    })

    // 清空文件输入，以便可以重复选择同一个文件
    if (fileInput.value) {
      fileInput.value.value = ''
    }
  } catch (error) {
    uploadProgress.value = 0
    logger.error('上传文件失败:', error)
    $message.error('上传文件失败')
  }
}



// 选择 @ 成员
const selectAtMember = (member: { id: string; name: string; avatar: string }) => {
  const textarea = chatInputRef.value?.messageInputRef?.messageInputRef as HTMLTextAreaElement | null
  if (!textarea) return

  showAtMembersPanel.value = false

  const atPosition = pendingAtPosition.value
  const atEnd = pendingAtEnd.value
  if (atPosition < 0 || atEnd < atPosition) return

  const value = inputMessage.value
  const mentionText = `@${member.name}`
  const insertText = /^\s/.test(value.slice(atEnd)) ? mentionText : `${mentionText} `
  const newText = replaceMentionToken(value, { start: atPosition, end: atEnd }, insertText)
  inputMessage.value = newText
  pendingAtPosition.value = -1
  pendingAtEnd.value = -1
  atMembersQuery.value = ''
  // 同步更新 trackedInputMessage，防止 watch 触发 reconcile 误判新 span 为跨越编辑区
  trackedInputMessage.value = newText

  // 记录 mention span（覆盖 "@姓名"，不含尾随空格）
  const span: MentionSpan = {
    start: atPosition,
    end: atPosition + mentionText.length,
    text: mentionText,
    userId: typeof member.id === 'string' ? Number(member.id) : member.id,
  }
  mentionSpans.value = [...mentionSpans.value, span]

  autoResizeTextarea()

  nextTick(() => {
    if (textarea) {
      const newPos = atPosition + insertText.length
      textarea.selectionStart = textarea.selectionEnd = newPos
      textarea.focus()
    }
  })
}

// 选择 @ 所有人
const selectAtAll = () => {
  const textarea = chatInputRef.value?.messageInputRef?.messageInputRef as HTMLTextAreaElement | null
  if (!textarea) return

  showAtMembersPanel.value = false

  const atPosition = pendingAtPosition.value
  const atEnd = pendingAtEnd.value
  if (atPosition < 0 || atEnd < atPosition) return

  const value = inputMessage.value
  const mentionText = '@所有人'
  const insertText = /^\s/.test(value.slice(atEnd)) ? mentionText : `${mentionText} `
  const newText = replaceMentionToken(value, { start: atPosition, end: atEnd }, insertText)
  inputMessage.value = newText
  pendingAtPosition.value = -1
  pendingAtEnd.value = -1
  atMembersQuery.value = ''
  // 同步更新 trackedInputMessage，防止 watch 触发 reconcile 误判新 span 为跨越编辑区
  trackedInputMessage.value = newText

  // 记录 mention span
  const span: MentionSpan = {
    start: atPosition,
    end: atPosition + mentionText.length,
    text: mentionText,
    userId: 'all',
  }
  mentionSpans.value = [...mentionSpans.value, span]

  autoResizeTextarea()

  nextTick(() => {
    if (textarea) {
      const newPos = atPosition + insertText.length
      textarea.selectionStart = textarea.selectionEnd = newPos
      textarea.focus()
    }
  })
}

// 关闭 @ 成员面板
const closeAtMembersPanel = () => {
  clearActiveMention()
}

// 选中斜杠命令候选项
// - search 模式：把 token 替换为命令的插入文本（如 #T-123 ）
// - command 模式：把 / token 替换为该命令的 trigger+空格（如 /task ），自然进入 search 模式
const selectSlashItem = (item: SlashCommandItem) => {
  const textarea = chatInputRef.value?.messageInputRef?.messageInputRef as HTMLTextAreaElement | null
  if (!textarea) return

  const token = slashToken.value
  if (!token) return

  // command 模式：item 是 CommandListEntry，取 command.trigger 进入搜索
  if (slashMode.value === 'command') {
    const entry = item as { id: string; command: SlashCommand }
    const insertText = `${entry.command.trigger} `
    const newText = replaceSlashToken(inputMessage.value, token, insertText)
    inputMessage.value = newText
    trackedInputMessage.value = newText
    clearSlashPanel()
    autoResizeTextarea()
    nextTick(() => {
      const newPos = token.start + insertText.length
      textarea.selectionStart = textarea.selectionEnd = newPos
      textarea.focus()
      const event = new Event('input', { bubbles: true })
      textarea.dispatchEvent(event)
    })
    return
  }

  // search 模式：优先走 onSelect 自定义处理（如 /note 直接发分享消息）
  const cmd = activeSlashCommand.value
  if (!cmd) return

  const ctx = {
    conversationId: props.conversation?.id ?? '',
    conversationType: props.conversation?.type ?? '',
  }

  if (cmd.onSelect) {
    const result = cmd.onSelect(item, ctx)
    if (result.action === 'send' && result.messageContent !== undefined && result.messageType) {
      // 走主发送路径（emit('send') → handleSendMessage），
      // 复用乐观追加 / 错误提示 / 断网兜底，避免自己这边不显示。
      clearSlashPanel()
      const newText = replaceSlashToken(inputMessage.value, token, '')
      inputMessage.value = newText
      trackedInputMessage.value = newText
      autoResizeTextarea()
      const messageData = {
        content: result.messageContent,
        type: result.messageType,
        quotedMessage: quotedMessage.value
      }
      emit('send', messageData)
      quotedMessage.value = null
      return
    }
    // action === 'insert' 或字段缺失，回退到 insertText
    if (result.insertText !== undefined) {
      doInsertSlashText(token, result.insertText, textarea)
      return
    }
  }

  // 默认：getInsertText 插入文本
  const insertText = cmd.getInsertText(item)
  if (!insertText) return
  doInsertSlashText(token, insertText, textarea)
}

/** 把斜杠 token 替换为 insertText，更新输入框并恢复焦点。 */
const doInsertSlashText = (token: { start: number; end: number }, insertText: string, textarea: HTMLTextAreaElement) => {
  const newText = replaceSlashToken(inputMessage.value, token, insertText)
  inputMessage.value = newText
  trackedInputMessage.value = newText
  clearSlashPanel()
  autoResizeTextarea()
  nextTick(() => {
    const newPos = token.start + insertText.length
    textarea.selectionStart = textarea.selectionEnd = newPos
    textarea.focus()
  })
}

// 关闭斜杠命令面板
const closeSlashPanel = () => {
  clearSlashPanel()
}

// 群成员搜索
const memberSearchQuery = ref('')
const showMemberSearch = ref(false)
const toggleMemberSearch = () => {
  showMemberSearch.value = !showMemberSearch.value
  // 如果显示搜索框，清空搜索内容并聚焦
  if (showMemberSearch.value) {
    memberSearchQuery.value = ''
    // 在下一个DOM更新周期聚焦输入框
    nextTick(() => {
      const searchInput = document.querySelector('.member-search-input') as HTMLInputElement
      if (searchInput) {
        searchInput.focus()
      }
    })
  }
}

// 群成员侧边栏展开/收缩状态
const isMembersSidebarExpanded = ref(true)
const toggleMembersSidebar = () => {
  isMembersSidebarExpanded.value = !isMembersSidebarExpanded.value
}

// 打开小程序列表
const openMiniAppList = () => {
  showMiniAppList.value = true
}

// 搜索相关函数
const performSearch = () => {
  const query = searchQuery.value.trim()
  if (!query) return
  
  isSearching.value = true
  
  setTimeout(() => {
    searchResults.value = props.messages.filter(message =>
      decodeToPlainText(message.content).toLowerCase().includes(query.toLowerCase())
    )
    isSearching.value = false
  }, 300)
}

const handleSend = async () => {
  // 先处理待发送文件
  const filesToSend = [...pendingFiles.value]
  pendingFiles.value = []
  
  for (const pendingFile of filesToSend) {
    try {
      await uploadAndSendFile(pendingFile.file)
    } catch (error) {
      logger.error('发送文件失败:', error)
      $message.error(`文件 ${pendingFile.name} 发送失败`)
    }
  }
  
  // 再处理文本消息
  const rawText = inputMessage.value.trim()
  if (rawText) {
    // 序列化 mention span 为 token content（单聊/bot 会话 span 为空，序列化后等于原文本）
    const content = serializeToContent(inputMessage.value, mentionSpans.value)

    // 构建消息对象，包含引用信息
    const messageData = {
      content: content,
      type: 'text',
      quotedMessage: quotedMessage.value
    }

    emit('send', messageData)
    inputMessage.value = ''
    quotedMessage.value = null
    mentionSpans.value = []
    trackedInputMessage.value = ''
    clearActiveMention()
    // 发送成功后清空草稿
    if (props.conversation?.id) {
      saveDraft(props.conversation.id, '', null)
    }
  }
}

const handleSendCodeBlock = (markdown: string) => {
  if (!markdown.trim()) return
  const messageData = {
    content: markdown,
    type: 'markdown' as const,
    quotedMessage: quotedMessage.value
  }
  emit('send', messageData)
  quotedMessage.value = null
}

const handleCopyCode = (message: Message) => {
  copyCode(message)
}

const handleKeydown = (event: KeyboardEvent) => {
  const sendShortcut = props.messageSettings?.sendShortcut || 'enter'
  if (sendShortcut === 'enter') {
    // Enter 发送，Shift+Enter 换行
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      try {
        handleSend()
      } catch (error) {
        logger.error('发送消息失败:', error)
        $message.error('发送消息失败，请重试')
      }
    }
  } else {
    // Ctrl+Enter 发送，Enter 换行
    if (event.key === 'Enter' && event.ctrlKey) {
      event.preventDefault()
      try {
        handleSend()
      } catch (error) {
        logger.error('发送消息失败:', error)
        $message.error('发送消息失败，请重试')
      }
    }
  }
}

const scrollToBottom = (instant: boolean = false) => {
  if (instant) {
    chatBodyRef.value?.scrollToBottom(true)
  } else {
    chatBodyRef.value?.scrollToBottomWithDelay(150)
  }
}

// 节流函数 - 已由 MessageListView 组件处理
const handleScroll = () => {}

// 加载更多消息的状态
const isLoadingMore = ref(false)

// 加载更多消息
const loadMoreMessages = async () => {
  if (!props.conversation || !props.hasMoreMessages) return
  
  isLoadingMore.value = true
  try {
    // 通知父组件加载更多消息，使用分页逻辑
    emit('loadMore', props.conversation.id)
  } catch (error) {
    logger.error('加载更多消息失败:', error)
  } finally {
    isLoadingMore.value = false
  }
}

// 处理标记已读
const handleMarkRead = () => {
  logger.log('[ChatWindow] handleMarkRead 被调用', {
    conversationId: props.conversation?.id
  })
  
  if (props.conversation?.id) {
    markMessagesAsRead(props.conversation.id)
  }
}

// 组件是否挂载
const isMounted = ref(true)
// 跟踪 WebSocket 消息处理器的清理函数
let wsHandlersCleanup: (() => void) | null = null
// 跟踪 context menu 的 setTimeout ID 以便清理
let memberContextMenuTimeoutId: number | null = null
let messageContextMenuTimeoutId: number | null = null

watch(() => props.messages, async (newMessages, oldMessages) => {
  if (!isMounted.value) return
  if (isInsertingSystemMessage.value) return

  const oldLength = oldMessages?.length ?? 0
  const newLength = newMessages?.length ?? 0

  // 判断是否需要滚动：新消息到达 或 最后一条消息内容有变化
  let shouldScroll = false
  if (newLength > oldLength) {
    const oldFirstId = oldMessages?.[0]?.id
    const newFirstId = newMessages?.[0]?.id
    if (oldFirstId !== newFirstId) {
      debouncedLoadReadUsers(newMessages, props.conversation?.type || 'single')
      return
    }
    shouldScroll = true
  } else if (newLength > 0 && newLength === oldLength) {
    const lastNew = newMessages[newLength - 1]
    const lastOld = oldMessages?.[newLength - 1]
    if (lastNew && lastOld && lastNew.id === lastOld.id && lastNew.content !== lastOld.content) {
      shouldScroll = true
    }
  }

  // 加载已读状态
  if (newLength > oldLength) {
    const newMessagesOnly = newMessages.slice(oldLength)
    if (newMessagesOnly.length > 0) {
      debouncedLoadReadUsers(newMessagesOnly, props.conversation?.type || 'single')
    }
  } else if (newLength === oldLength && newLength > 0) {
    const hasReadStatusChanged = oldMessages && newMessages.some((newMsg, index) => {
      const oldMsg = oldMessages[index]
      return oldMsg && newMsg.isRead !== oldMsg.isRead
    })
    if (hasReadStatusChanged) {
      debouncedLoadReadUsers(newMessages, props.conversation?.type || 'single', true)
    }
  }

  if (shouldScroll) {
    nextTick(() => {
      scrollToBottom(true)
      requestAnimationFrame(() => scrollToBottom(true))
    })
  }
}, { deep: true })

// 组件挂载时初始化
const handleForwardNote = (event: CustomEvent) => {
  const { content } = event.detail
  inputMessage.value = content
  autoResizeTextarea()
}

const handleReadReceiptUpdated = (event: Event) => {
  const { conversationId, messageIds } = (event as CustomEvent<{
    conversationId?: string
    messageIds?: Array<string | number>
  }>).detail || {}

  if (!conversationId || conversationId !== props.conversation?.id?.toString()) return
  if (!Array.isArray(messageIds) || messageIds.length === 0) return

  const messageIdSet = new Set(messageIds.map(id => id.toString()))
  const messagesToRefresh = props.messages.filter(message =>
    message.isSelf && messageIdSet.has(message.id.toString())
  )

  if (messagesToRefresh.length === 0) return

  void loadReadUsersForMessages(messagesToRefresh, props.conversation?.type || 'single', true)
}

// 组件挂载时添加事件监听器
onMounted(() => {
  isMounted.value = true
  if (messageListRef.value) {
    messageListRef.value.addEventListener('scroll', handleScroll)
  }
  initWebSocketMessageHandler()
  scrollToBottom()
  window.addEventListener('forwardNoteToChat', handleForwardNote as EventListener)
  window.addEventListener('message-read-receipt-updated', handleReadReceiptUpdated)

  if (window.electron?.ipcRenderer) {
    window.electron.ipcRenderer.on('download-progress', (_event: any, result: { downloadId?: string; percent: number; state?: string }) => {
      if (!result.downloadId) return
      if (result.state === 'progressing') {
        downloadProgress.value[result.downloadId] = result.percent
      } else {
        // completed / cancelled / interrupted：清除进度条
        delete downloadProgress.value[result.downloadId]
      }
    })

    window.electron.ipcRenderer.on('download-complete', (_event: any, result: { success: boolean; filePath?: string; error?: string; downloadId?: string; cancelled?: boolean }) => {
      if (result.success && result.filePath) {
        if (result.downloadId) {
          delete downloadProgress.value[result.downloadId]
          saveDownloadedFile(result.downloadId, result.filePath)
        }
        $message.success(`文件已下载到: ${result.filePath}`)
      } else if (!result.cancelled) {
        $message.error('文件下载失败: ' + (result.error || '未知错误'))
      }
    })

    window.electron.ipcRenderer.on('save-file-complete', (_event: any, result: { success: boolean; filePath?: string; error?: string; downloadId?: string; cancelled?: boolean }) => {
      if (result.success && result.filePath) {
        if (result.downloadId) {
          delete downloadProgress.value[result.downloadId]
          saveDownloadedFile(result.downloadId, result.filePath)
        }
        $message.success(`文件已保存到: ${result.filePath}`)
      } else if (!result.cancelled) {
        $message.error('文件保存失败: ' + (result.error || '未知错误'))
      }
    })
  }

  // 非阻塞加载分身配置和已读状态，不影响界面交互
  Promise.all([
    fetchConfig(),
    fetchSessions(),
    loadReadUsersForMessages(props.messages, props.conversation?.type || 'single')
  ])

  // 监听来自 Main.vue 的 AI 快捷键事件
  window.addEventListener('ai-action-summary', handleAISummaryAction)
  window.addEventListener('ai-action-quickpanel', handleAIQuickPanelAction)
})

// 初始化 WebSocket 消息处理
const handleRealtimeMessage = (type: string, data: any) => {
  logger.log('ChatWindow: 收到实时消息', type, 'data:', data, 'typeof data:', typeof data)
  
  switch (type) {
    case 'realtime:session:created':
      {
        logger.log('ChatWindow: data 内容:', JSON.stringify(data, null, 2))
        const session = data?.session || data
        logger.log('ChatWindow: 处理 realtime:session:created', session)
        
        if (!session) {
          logger.warn('ChatWindow: session 数据为空')
          return
        }
        
        realtimeStore.updateSession(session)
        
        const currentUserId = getCurrentUser()?.id
        logger.log('ChatWindow: 当前用户ID', currentUserId, '发起者ID', session.initiator_id)
        
        if (session.initiator_id !== currentUserId && session.type === 'screen_share') {
          logger.log('ChatWindow: 设置 remoteScreenUserId', session.initiator_id)
          remoteScreenUserId.value = session.initiator_id
          const initiatorName = session.initiator?.nickname || `用户 ${session.initiator_id}`
          $message.info(`${initiatorName} 正在共享屏幕`, 5000)
          
          // 自动请求加入会话
          logger.log('ChatWindow: 自动请求加入会话', session.id)
          realtimeStore.requestJoin(session.id).catch(err => {
            logger.error('ChatWindow: 请求加入会话失败', err)
          })
        }
      }
      break
    case 'realtime:join:requested':
      {
        logger.log('ChatWindow: 收到加入请求', data)
        const request = data?.participant || data
        if (request && request.session_id && request.user_id) {
          realtimeStore.addPendingRequest(request)
          
          // 自动批准屏幕共享的加入请求
          const session = realtimeStore.mySession
          if (session && session.type === 'screen_share') {
            logger.log('ChatWindow: 自动批准屏幕共享加入请求', request.user_id)
            realtimeStore.approveJoin(request.session_id, request.user_id).then(() => {
              // 批准成功后，创建 WebRTC 连接
              logger.log('ChatWindow: 批准成功，创建 WebRTC 连接')
              connectionManager.createConnectionForViewer(request.user_id)
            }).catch(err => {
              logger.error('ChatWindow: 批准加入失败', err)
            })
          }
        }
      }
      break
    case 'realtime:join:approved':
      {
        logger.log('ChatWindow: 加入请求被批准', data)
        // 加入请求被批准，开始建立连接
        const participant = data?.participant || data
        if (participant && participant.session_id) {
          logger.log('ChatWindow: 开始建立 WebRTC 连接')
          // 设置当前观看的会话
          const session = realtimeStore.activeSessions.find(s => s.id === participant.session_id)
          if (session) {
            realtimeStore.setCurrentViewingSession(session)
          }
          // 触发 ScreenShare 组件开始接收流
          // 注意：WebRTC 连接会在 ScreenShare 组件中建立
        }
      }
      break
    case 'realtime:join:rejected':
      // 加入请求被拒绝
      $message.warning('您的加入请求被拒绝')
      break
    case 'realtime:participant:left':
      // 参与者离开
      if (data.viewer_id) {
        connectionManager.closeConnection(data.viewer_id)
      }
      break
    case 'realtime:session:ended':
      // 会话结束
      realtimeStore.removeSession(data.session_id)
      connectionManager.closeAllConnections()
      remoteScreenUserId.value = null
      break
    case 'realtime:webrtc:offer':
      // 收到 WebRTC offer
      if (data.session_id && data.from_user_id && data.signal) {
        viewerConnection.handleOffer(data.session_id, data.from_user_id, data.signal)
      }
      break
    case 'realtime:webrtc:answer':
      // 收到 WebRTC answer
      if (data.from_user_id && data.signal) {
        connectionManager.handleAnswer(data.from_user_id, data.signal)
      }
      break
    case 'realtime:webrtc:ice':
      // 收到 ICE candidate
      if (data.from_user_id && data.signal) {
        if (realtimeStore.isViewing) {
          viewerConnection.handleIceCandidate(data.signal)
        } else if (realtimeStore.isSharing) {
          connectionManager.handleIceCandidate(data.from_user_id, data.signal)
        }
      }
      break
  }
}

const initWebSocketMessageHandler = () => {
  // 先清理旧的 handler
  if (wsHandlersCleanup) {
    wsHandlersCleanup()
    wsHandlersCleanup = null
  }
  
  const realtimeMessageTypes = [
    'realtime:session:created',
    'realtime:join:requested',
    'realtime:join:approved',
    'realtime:join:rejected',
    'realtime:participant:left',
    'realtime:session:ended',
    'realtime:webrtc:offer',
    'realtime:webrtc:answer',
    'realtime:webrtc:ice'
  ];
  
  const uniqueMessageTypes = [...new Set(realtimeMessageTypes)];
  
  // 使用新的 addWsHandlers 批量注册，返回统一清理函数
  const handlerMap: Record<string, (data: any) => void> = {};
  
  uniqueMessageTypes.forEach(type => {
    handlerMap[type] = (data: any) => {
      if (realtimeMessageTypes.includes(type)) {
        handleRealtimeMessage(type, data);
      }
    };
  });
  
  wsHandlersCleanup = addWsHandlers(handlerMap);
};

// 组件卸载时移除事件监听器
onUnmounted(() => {
  isMounted.value = false
  
  cleanupMessageActions()

  // 清理 backendSearch 的 debounce timer，避免卸载后回调执行
  cancelBackendSearch()
  
  // 移除滚动事件监听器
  if (messageListRef.value) {
    messageListRef.value.removeEventListener('scroll', handleScroll)
  }
  
  // 移除全局事件监听器
  window.removeEventListener('forwardNoteToChat', handleForwardNote as EventListener)
  window.removeEventListener('message-read-receipt-updated', handleReadReceiptUpdated)

  // 清理 WebSocket handler（使用新的清理机制）
  if (wsHandlersCleanup) {
    wsHandlersCleanup()
    wsHandlersCleanup = null
  }

  // 移除 AI 快捷键事件监听
  window.removeEventListener('ai-action-summary', handleAISummaryAction)
  window.removeEventListener('ai-action-quickpanel', handleAIQuickPanelAction)
  
  // 清理 context menu 监听器和定时器
  if (memberContextMenuTimeoutId !== null) {
    clearTimeout(memberContextMenuTimeoutId)
    memberContextMenuTimeoutId = null
  }
  document.removeEventListener('click', closeMemberContextMenu)
})

// 滚动到引用的消息位置
const scrollToQuotedMessage = (quotedMessageId: string) => {
  if (!quotedMessageId) return
  
  nextTick(() => {
    const messageElement = document.querySelector(`.message-item[data-message-id="${quotedMessageId}"]`)
    if (messageElement instanceof HTMLElement) {
      messageElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 给消息添加高亮效果
      messageElement.classList.add('highlighted-message')
      setTimeout(() => {
        messageElement.classList.remove('highlighted-message')
      }, 2000)
    }
  })
}

// 滚动到指定消息位置
const scrollToMessage = (messageId: string) => {
  if (!messageId) return
  
  // 先检查消息是否在当前的 props.messages 中
  const targetMessage = props.messages.find(m => String(m.id) === String(messageId))
  if (!targetMessage) {
    logger.warn('目标消息不在当前加载的消息列表中:', messageId)
    QMessage.info('该消息不在当前可视范围内，请加载更多历史消息后重试')
    closeMessageManager()
    return
  }
  
  nextTick(() => {
    const messageElement = document.querySelector(`.message-item[data-message-id="${messageId}"]`)
    if (messageElement instanceof HTMLElement) {
      messageElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 给消息添加高亮效果
      messageElement.classList.add('highlighted-message')
      setTimeout(() => {
        messageElement.classList.remove('highlighted-message')
      }, 2000)
    } else {
      logger.warn('未找到目标消息 DOM 元素:', messageId)
    }
  })
  // 关闭消息管理器
  closeMessageManager()
}

const autoResizeTextarea = () => {
  const textarea = chatInputRef.value?.messageInputRef?.messageInputRef as HTMLTextAreaElement | null
  if (textarea) {
    textarea.style.height = 'auto'
    const maxHeight = 200
    const scrollHeight = textarea.scrollHeight
    textarea.style.height = `${Math.min(scrollHeight, maxHeight)}px`
    textarea.style.overflowY = scrollHeight > maxHeight ? 'auto' : 'hidden'
  }
}

const computeMenuPosition = (clientX: number, clientY: number, menuWidth: number = 160, menuHeight: number = 160) => {
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

const handleShowMemberContextMenu = (event: MouseEvent, member: any) => {
  event.stopPropagation()
  const { x, y } = computeMenuPosition(event.clientX, event.clientY)
  selectedMember.value = {
    ...member,
    name: member.name || member.nickname || member.username || '未知用户'
  }
  openMenu('member', x, y)
}

const closeMemberContextMenu = () => {
  closeMenu()
  selectedMember.value = null
}

// 计算当前用户在群中的角色
const currentUserRole = computed(() => {
  var currentUser = props.currentUser
  if (!currentUser){
    currentUser = getCurrentUser()
  }
  if (!props.conversation?.members || !currentUser) return 'member'
  const member = props.conversation.members.find((m: any) => {
    return String(m.id) === String(currentUser.id)
  })
  return member?.role || 'member'
})

const canManageGroupFiles = computed(() =>
  currentUserRole.value === 'owner' || currentUserRole.value === 'admin'
)

const openGroupFiles = (messageId: number | null = null, fileId: number | null = null) => {
  referenceMessageId.value = messageId
  referenceFileId.value = fileId
  showGroupFiles.value = true
}

const closeGroupFiles = () => {
  showGroupFiles.value = false
  referenceMessageId.value = null
  referenceFileId.value = null
}

const saveMessageToGroupFiles = () => {
  if (!selectedMessage.value || !canManageGroupFiles.value) return
  try {
    const messageId = Number(selectedMessage.value.id)
    const payload = JSON.parse(selectedMessage.value.content)
    const fileId = Number(payload?.id)
    if (!Number.isInteger(messageId) || messageId <= 0 || !Number.isInteger(fileId) || fileId <= 0) {
      $message.error('无法识别聊天附件')
      return
    }
    openGroupFiles(messageId, fileId)
  } catch {
    $message.error('无法识别聊天附件')
  }
}

const viewMemberInfo = async () => {
  if (selectedMember.value) {
    const userId = selectedMember.value.user?.id || selectedMember.value.id
    if (!userId) {
      $message.error('无法获取用户ID')
      closeMemberContextMenu()
      return
    }

    const { profile, success } = await fetchUserProfile(userId, selectedMember.value)
    if (!success) {
      $message.error('获取用户信息失败')
    }
    showUserProfile(profile)
    closeMemberContextMenu()
  }
}

const showUserProfile = async (user: any) => {
  const userId = user?.id
  if (!userId) {
    selectedUser.value = user
  } else {
    const { profile } = await fetchUserProfile(userId, user)
    selectedUser.value = profile
  }
  showUserProfileFlag.value = true
}

const closeUserProfile = () => {
  showUserProfileFlag.value = false
  selectedUser.value = null
}



interface User {
  id: string | number
  name: string
  avatar?: string
}

const handleSendPrivateMessage = async (user: User | string | number) => {
  try {
    // 检查参数类型
    let processedUserId: string | number
    if (typeof user === 'object' && user !== null) {
      processedUserId = user.id
    } else {
      processedUserId = user
    }
    
    // 确保userId是数字类型
    if (typeof processedUserId === 'string') {
      // 如果是字符串格式（如 'emp1'），尝试提取数字部分
      if (processedUserId.startsWith('emp')) {
        processedUserId = processedUserId.replace('emp', '')
      }
      // 转换为数字
      processedUserId = parseInt(processedUserId)
    }
    
    const response = await request('/api/v1/conversations', {
      method: 'POST',
      body: JSON.stringify({
        type: 'single',
        user_id: processedUserId
      })
    })
    
    if (response.code === 0) {
      // 通知父组件切换到新会话
      emit('switchConversation', response.data.id.toString())
    }
  } catch (error) {
    logger.error('创建私聊失败:', error)
    $message.error('创建私聊失败，请重试')
  }
  closeUserProfile()
}

// ChatHeaderActions 事件处理方法
const handleSwitchConversation = (conversationId: string) => {
  emit('switchConversation', conversationId)
}

const handleRemoveMember = (memberId: string, memberName: string) => {
  if (!props.conversation) return

  openConfirmDialog('确认移除', `确定要移除成员 ${memberName} 吗？`, async () => {
    try {
      const response = await request(`/api/v1/groups/${props.conversation.id}/members/${memberId}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json'
        }
      })

      if (response.code === 0) {
        QMessage.success('移除成员成功')
        emit('switchConversation', props.conversation.id)
      } else {
        QMessage.error('移除成员失败: ' + response.message)
      }
    } catch (error: any) {
      logger.error('移除成员失败:', error)
      QMessage.error('移除成员失败: ' + error.message)
    }
  })
}

const handleSetAdmin = (memberId: string, memberName: string, isAdmin: boolean) => {
  if (!props.conversation) return

  const action = isAdmin ? '设为管理员' : '取消管理员'
  openConfirmDialog('确认操作', `确定要${action}成员 ${memberName} 吗？`, async () => {
    try {
      const newRole = isAdmin ? 'admin' : 'member'
      const response = await request(`/api/v1/groups/${props.conversation.id}/members/${memberId}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ role: newRole })
      })

      if (response.code === 0) {
        QMessage.success(`${action}成功`)
        emit('switchConversation', props.conversation.id)
      } else {
        QMessage.error(`${action}失败: ` + response.message)
      }
    } catch (error: any) {
      logger.error(`${action}失败:`, error)
      QMessage.error(`${action}失败: ` + error.message)
    }
  })
}

const handleTransferOwner = (memberId: string, memberName: string) => {
  if (!props.conversation) return

  openConfirmDialog('确认转让群主', `确定要将群主转让给 ${memberName} 吗？转让后您将成为管理员。`, async () => {
    try {
      const response = await request(`/api/v1/groups/${props.conversation.id}/members/${memberId}/transfer-owner`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })

      if (response.code === 0) {
        QMessage.success('群主转让成功')
        emit('switchConversation', props.conversation.id)
      } else {
        QMessage.error('群主转让失败: ' + response.message)
      }
    } catch (error: any) {
      logger.error('群主转让失败:', error)
      QMessage.error('群主转让失败: ' + error.message)
    }
  })
}

const handleStartPrivateChat = (memberId: string) => {
  handleSendPrivateMessage(memberId)
}

// OverlayManager 事件适配函数
// OverlayManager 内部组件只传出 memberId，需要适配为原处理函数需要的签名
const handleRemoveMemberFromOverlay = (memberId: string) => {
  if (!props.conversation) return
  const member = props.conversation.members?.find(m => String(m.id) === String(memberId))
  const memberName = (member as any)?.name || '未知成员'
  handleRemoveMember(memberId, memberName)
}

const handleSetAdminFromOverlay = (memberId: string) => {
  if (!props.conversation) return
  const member = props.conversation.members?.find(m => String(m.id) === String(memberId))
  const memberName = (member as any)?.name || '未知成员'
  const currentIsAdmin = (member as any)?.role === 'admin'
  handleSetAdmin(memberId, memberName, !currentIsAdmin)
}

const handleTransferOwnerFromOverlay = (memberId: string) => {
  if (!props.conversation) return
  const member = props.conversation.members?.find(m => String(m.id) === String(memberId))
  const memberName = (member as any)?.name || '未知成员'
  handleTransferOwner(memberId, memberName)
}

// 撤回消息包装函数（适配 OverlayManager 无参数事件）
const handleRecallMessage = async () => {
  if (!selectedMessage.value || !props.conversation) return
  const result = await recallMessage(String(props.conversation.id), String(selectedMessage.value.id))
  if (!result.success) {
    $message.error(result.message || '撤回失败')
  }
}

const closeMessageContextMenu = () => {
  closeMenu()
  selectedMessage.value = null
  messageMenuSelection.value = ''
}

const getForwardRecordTitle = (): string => {
  const conversationName = props.conversation?.name || props.conversation?.other_member_name
  return conversationName ? `来自「${conversationName}」的聊天记录` : '聊天记录'
}

const forwardMessage = () => {
  if (selectedMessage.value) {
    // 触发全局事件，打开分享弹窗并传递消息数据
    window.dispatchEvent(new CustomEvent('forwardMessage', {
      detail: {
        message: selectedMessage.value,
        sourceTitle: getForwardRecordTitle(),
      }
    }))
  }
  closeMessageContextMenu()
}

const startMessageSelection = () => {
  isMessageSelectionMode.value = true
  selectedMessageIds.value = selectedMessage.value && isMessageSelectionEligible(selectedMessage.value)
    ? new Set([String(selectedMessage.value.id)])
    : new Set()
}

const toggleMessageSelection = (id: string) => {
  const message = props.messages.find(item => String(item.id) === id)
  if (!message || !isMessageSelectionEligible(message)) return

  const ids = new Set(selectedMessageIds.value)
  ids.has(id) ? ids.delete(id) : ids.add(id)
  selectedMessageIds.value = ids
}

const cancelMessageSelection = () => {
  isMessageSelectionMode.value = false
  selectedMessageIds.value = new Set()
}

const forwardSelectedMessages = () => {
  if (selectedMessages.value.length < 2) return
  window.dispatchEvent(new CustomEvent('forwardMessage', {
    detail: { messages: selectedMessages.value, sourceTitle: getForwardRecordTitle() }
  }))
  cancelMessageSelection()
}

// AI 总结消息
const handleAISummary = () => {
  if (!selectedMessage.value || !props.conversation?.id) {
    closeMessageContextMenu()
    return
  }
  showSummaryPanel.value = true
  closeMessageContextMenu()
}

// AI 翻译消息
const handleAITranslate = () => {
  if (!selectedMessage.value || !selectedMessage.value.content) {
    closeMessageContextMenu()
    return
  }
  translateContent.value = selectedMessage.value.content
  translateMessageType.value = selectedMessage.value.type === 'image' ? 'image' : undefined
  showTranslatePanel.value = true
  closeMessageContextMenu()
}

// 智能回复（右键菜单：带上下文起草）
const handleSmartReply = async () => {
  if (!selectedMessage.value || !selectedMessage.value.id) {
    closeMessageContextMenu()
    return
  }
  const msg = selectedMessage.value
  const conversationId = props.conversation?.id
  closeMessageContextMenu()
  if (!conversationId) return

  try {
    const reply = await draftReply(Number(conversationId), Number(msg.id))
    if (reply) {
      inputMessage.value = reply
      autoResizeTextarea()
      $message.success('回复草稿已生成')
    }
  } catch {
    $message.error('生成回复失败')
  }
}

// 发送消息提醒（右键菜单路径；铃铛直接调 composable 的 remind）
const sendMessageReminder = () => {
  if (!selectedMessage.value) {
    closeMessageContextMenu()
    return
  }
  remind(selectedMessage.value)
  closeMessageContextMenu()
}

// 消息引用
const quoteMessage = () => {
  if (selectedMessage.value) {
    // 设置引用消息
    quotedMessage.value = selectedMessage.value
    // 聚焦到输入框
    const input = document.querySelector('.message-input') as HTMLTextAreaElement
    if (input) {
      input.focus()
    }
  }
  closeMessageContextMenu()
}

const addToNotesApp = async () => {
  if (selectedMessage.value) {
    const message = selectedMessage.value

    if (message.type !== 'text' && message.type !== 'markdown' && !message.isAIMessage && !message.is_ai_message) {
      $message.warning('仅支持文本类型的消息添加到笔记')
      closeMessageContextMenu()
      return
    }

    const rawContent = message.content || ''
    const maxNoteLength = 2000
    const truncatedContent = rawContent.length > maxNoteLength
      ? rawContent.slice(0, maxNoteLength) + `\n...(原文共 ${rawContent.length} 字，已截断)`
      : rawContent

    const noteContent = `【聊天记录】
发送者：${message.sender.name}
时间：${formatTime(message.timestamp)}
内容：${truncatedContent}`

    try {
      const { useNotes } = await import('../../composables/useNotes')
      const { createNote } = useNotes()
      const result = await createNote({
        title: `聊天记录 ${formatTime(message.timestamp)}`,
        content: noteContent,
        type: 'note',
        tags: ['聊天记录']
      })
      if (result) {
        $message.success('消息已添加到笔记')
      } else {
        $message.error('添加到笔记失败')
      }
    } catch {
      $message.error('添加到笔记失败')
    }
  }
  closeMessageContextMenu()
}

// 从消息创建任务
const createTaskFromMessage = async () => {
  if (!selectedMessage.value) {
    closeMessageContextMenu()
    return
  }

  const message = selectedMessage.value
  const messageText = message.content || ''

  try {
    const taskStore = useTaskStore()
    await taskStore.createTask({
      title: messageText.slice(0, 50) + (messageText.length > 50 ? '...' : ''),
      description: messageText,
      priority: 'medium',
      status: 'todo'
    })
    $message.success('已创建为任务')
  } catch (error: any) {
    logger.error('创建任务失败:', error)
    $message.error('创建任务失败: ' + (error.message || '未知错误'))
  }

  closeMessageContextMenu()
}

// 截图相关状态
type ScreenshotStatus = 'idle' | 'preparing' | 'capturing' | 'processing' | 'failed'
interface ScreenshotErrorPayload {
  message?: string
  code?: string
  diagnostics?: Record<string, unknown>
}

const showScreenshotPreview = ref(false)
const screenshotImageData = ref('')
const screenshotStatus = ref<ScreenshotStatus>('idle')
const isScreenshotBusy = computed(() =>
  screenshotStatus.value === 'preparing' ||
  screenshotStatus.value === 'capturing' ||
  screenshotStatus.value === 'processing'
)

const getScreenshotErrorMessage = (payload?: string | ScreenshotErrorPayload) => {
  if (typeof payload === 'string') return payload
  return payload?.message || '截图失败，请稍后重试'
}

// 获取对方用户ID（单聊）
const otherUserId = computed(() => {
  if (props.conversation?.type === 'single' && props.conversation?.members && props.conversation.members.length === 2) {
    const currentUserId = currentUser.value?.id?.toString() || ''
    return props.conversation.members.find(member => String(member.id) !== currentUserId)?.id ?? null
  }
  return null
})

// 分身接管状态
const avatarTakeoverUntil = computed(() => {
  if (!props.conversation) return null
  const session = getSession(props.conversation.id)
  return session?.takeoverUntil || null
})

// 处理分身恢复
async function handleAvatarResume() {
  if (!props.conversation) return
  await takeoverSession(props.conversation.id)
}

// 处理分身延长
async function handleAvatarExtend() {
  if (!props.conversation) return
  await takeoverSession(props.conversation.id)
}

// 检测是否在Electron环境中
const isElectron = computed(() => {
  return window.electron && window.electron.ipcRenderer && typeof window.electron.ipcRenderer.once === 'function'
})

const takeScreenshot = () => {
  // 检查是否在Electron环境中
  if (window.electron && window.electron.ipcRenderer) {
    logger.log('[Screenshot] takeScreenshot called')

    if (isScreenshotBusy.value) {
      $message.info('截图正在进行中，请稍候...')
      return
    }
    screenshotStatus.value = 'preparing'

    // 超时保护：截图 overlay 若异常（崩溃/被强制关闭/权限问题）可能既不触发 ok 也不触发 cancel/error，
    // 导致 screenshotStatus 永远卡在 preparing/capturing，用户无法再次截图，必须重启应用。
    // 60 秒后强制恢复 idle（用户在 overlay 框选的时间不受限，但 60 秒足够覆盖异常卡死场景）。
    const screenshotTimeout = setTimeout(() => {
      if (screenshotStatus.value !== 'idle') {
        logger.warn('[Screenshot] Timeout, forcing status reset to idle')
        screenshotStatus.value = 'idle'
        window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
        window.electron.ipcRenderer.removeAllListeners('screenshot-loading')
        window.electron.ipcRenderer.removeAllListeners('screenshot-error')
        $message.warning('截图超时，请重试')
      }
    }, 60000)

    // 移除所有之前的监听器，确保不会有重复监听
    logger.log('[Screenshot] Removing all previous listeners')
    window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
    window.electron.ipcRenderer.removeAllListeners('screenshot-loading')
    window.electron.ipcRenderer.removeAllListeners('screenshot-error')

    try {
      // 定义处理函数
      const screenshotHandler = async (_event: any, imageData: string | ArrayBuffer | Uint8Array) => {
        logger.log('[Screenshot] screenshotHandler triggered, imageData exists:', !!imageData)
        clearTimeout(screenshotTimeout)
        screenshotStatus.value = 'processing'
        
        // 监听器触发后立即移除所有监听器，避免重复触发
        logger.log('[Screenshot] Removing all listeners after trigger')
        window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
        window.electron.ipcRenderer.removeAllListeners('screenshot-loading')
        window.electron.ipcRenderer.removeAllListeners('screenshot-error')
        
        // 确保组件仍然挂载
        if (!isMounted.value) {
          screenshotStatus.value = 'idle'
          return
        }
        
        try {
          // 处理截图结果
          if (imageData) {
            logger.log('[Screenshot] Processing screenshot, adding to pendingFiles')
            const file = await screenshotPayloadToFile(imageData, 'screenshot.png')
            pendingFiles.value.push({
              file,
              name: 'screenshot.png'
            })
            logger.log('[Screenshot] Screenshot added to pendingFiles, count:', pendingFiles.value.length)
          } else {
            logger.log('[Screenshot] User cancelled screenshot')
          }
          screenshotStatus.value = 'idle'
        } catch (error) {
          screenshotStatus.value = 'failed'
          logger.error('[Screenshot] Error processing screenshot:', error)
          $message.error('处理截图失败')
        }
      }
      
      // 截图组件正在初始化的提示
      const loadingHandler = () => {
        logger.log('[Screenshot] screenshot-loading received')
        window.electron.ipcRenderer.removeAllListeners('screenshot-loading')
        if (isMounted.value) {
          screenshotStatus.value = 'capturing'
          $message.info('正在准备截图组件，请稍候...')
        }
      }

      const errorHandler = (_event: any, payload?: string | ScreenshotErrorPayload) => {
        logger.error('[Screenshot] screenshot-error received:', payload)
        clearTimeout(screenshotTimeout)
        screenshotStatus.value = 'failed'
        window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
        window.electron.ipcRenderer.removeAllListeners('screenshot-loading')
        window.electron.ipcRenderer.removeAllListeners('screenshot-error')
        if (isMounted.value) {
          $message.error(getScreenshotErrorMessage(payload))
        }
      }

      logger.log('[Screenshot] Registering new listener')
      // 注册新的监听器
      window.electron.ipcRenderer.on('screenshot-taken', screenshotHandler)
      window.electron.ipcRenderer.on('screenshot-loading', loadingHandler)
      window.electron.ipcRenderer.on('screenshot-error', errorHandler)
      
      logger.log('[Screenshot] Sending take-screenshot request')
      // 发送截图请求到主进程
      window.electron.ipcRenderer.send('take-screenshot')
    } catch (error) {
      screenshotStatus.value = 'failed'
      logger.error('[Screenshot] Error triggering screenshot:', error)
      $message.error('截图功能不可用')
    }
  } else {
    $message.warning('截图功能仅在客户端环境中可用')
  }
}

// 隐藏窗口截图
const takeScreenshotHidden = () => {
  if (window.electron && window.electron.ipcRenderer) {
    logger.log('[Screenshot] takeScreenshotHidden called')
    if (isScreenshotBusy.value) {
      $message.info('截图正在进行中，请稍候...')
      return
    }
    screenshotStatus.value = 'preparing'
    window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
    window.electron.ipcRenderer.removeAllListeners('screenshot-error')

    const screenshotHandler = async (_event: any, imageData: string | ArrayBuffer | Uint8Array) => {
      screenshotStatus.value = 'processing'
      window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
      window.electron.ipcRenderer.removeAllListeners('screenshot-error')
      if (!isMounted.value) {
        screenshotStatus.value = 'idle'
        return
      }
      try {
        if (imageData) {
          const file = await screenshotPayloadToFile(imageData, 'screenshot.png')
          pendingFiles.value.push({ file, name: 'screenshot.png' })
        }
        screenshotStatus.value = 'idle'
      } catch (error) {
        screenshotStatus.value = 'failed'
        logger.error('[Screenshot] Error processing hidden screenshot:', error)
        $message.error('处理截图失败')
      }
    }

    const errorHandler = (_event: any, payload?: string | ScreenshotErrorPayload) => {
      screenshotStatus.value = 'failed'
      window.electron.ipcRenderer.removeAllListeners('screenshot-taken')
      window.electron.ipcRenderer.removeAllListeners('screenshot-error')
      if (isMounted.value) {
        $message.error(getScreenshotErrorMessage(payload))
      }
    }

    window.electron.ipcRenderer.on('screenshot-taken', screenshotHandler)
    window.electron.ipcRenderer.on('screenshot-error', errorHandler)
    window.electron.ipcRenderer.send('take-screenshot-hidden')
  } else {
    $message.warning('截图功能仅在客户端环境中可用')
  }
}

const screenshotPayloadToFile = async (
  payload: string | ArrayBuffer | Uint8Array,
  filename: string
): Promise<File> => {
  if (typeof payload === 'string') {
    const response = await fetch(payload)
    const blob = await response.blob()
    return new File([blob], filename, { type: blob.type || 'image/png' })
  }

  const bytes = payload instanceof ArrayBuffer ? new Uint8Array(payload) : payload
  const data = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
  return new File([data], filename, { type: 'image/png' })
}


// 小程序列表
const showMiniAppList = ref(false)
const activeMiniApp = ref<MiniAppData | null>(null)

// 打开小程序
const openMiniApp = (miniApp: MiniAppData) => {
  activeMiniApp.value = miniApp
}

// 处理小程序 Toast 消息
const handleMiniAppToast = (message: string) => {
  $message.info(message)
}

// 开始语音通话 - 由 RealtimeCommunication 组件统一处理
const startVoiceCall = () => {
  emit('start-voice-call')
}

// 开始视频通话 - 由 RealtimeCommunication 组件统一处理
const startVideoCall = () => {
  emit('start-video-call')
}

const selectFile = () => {
  // 先清空，允许重复选择同一文件
  if (fileInput.value) fileInput.value.value = ''
  fileInput.value?.click()
}
const selectImage = () => {
  // 创建一个临时的文件输入元素
  const imageInput = document.createElement('input')
  imageInput.type = 'file'
  imageInput.accept = 'image/*'
  imageInput.multiple = true
  
  // 监听文件选择事件（使用一次性监听避免内存泄漏）
  const handleChange = (event: Event) => {
    const target = event.target as HTMLInputElement
    const files = target.files
    
    // 清理事件监听器和 DOM 引用
    imageInput.removeEventListener('change', handleChange)
    imageInput.remove()
    
    if (files && files.length > 0) {
      // 添加到待发送文件列表,而不是直接发送
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        addPendingFile(file)
      }
    }
  }
  
  imageInput.addEventListener('change', handleChange)
  
  // 触发图片选择对话框
  imageInput.click()
}

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = target.files
  
  if (files && files.length > 0) {
    // 添加到待发送文件列表,而不是直接发送
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      addPendingFile(file)
    }
    
    // 清空文件输入，以便可以重复选择同一个文件
    if (fileInput.value) {
      fileInput.value.value = ''
    }
  }
}

// 文件下载进度：key = 消息 id，value = 百分比(0-100)；完成/失败后删除
const downloadProgress = ref<Record<string, number>>({})
provide('downloadProgress', downloadProgress)

// 上传进度：0=不上传，1-100=上传中
const uploadProgress = ref(0)

// 已下载文件记录：key = 消息 id，value = 本地文件路径（刷新不丢）
const DOWNLOAD_CACHE_KEY = 'qim_downloaded_files'
const downloadedFiles = ref<Record<string, string>>({})
// 初始化时从 localStorage 读取
try {
  const cached = localStorage.getItem(DOWNLOAD_CACHE_KEY)
  if (cached) downloadedFiles.value = JSON.parse(cached)
} catch {}
provide('downloadedFiles', downloadedFiles)

// 持久化下载记录
const saveDownloadedFile = (id: string, path: string) => {
  downloadedFiles.value[id] = path
  localStorage.setItem(DOWNLOAD_CACHE_KEY, JSON.stringify(downloadedFiles.value))
}

const downloadFile = async (fileContent: string, messageId?: string) => {
  try {
    const parsedContent = JSON.parse(fileContent)
    const finalFileName = parsedContent.name || parsedContent.fileName || parsedContent.url.split('/').pop() || '文件'
    let fileUrl = parsedContent.url

    if (fileUrl && !fileUrl.startsWith('http')) {
      const cleanServerUrl = serverUrl.value.replace(/\/$/, '')
      const cleanFileUrl = fileUrl.replace(/^\//, '')
      fileUrl = `${cleanServerUrl}/${cleanFileUrl}`
    }

    if (!fileUrl) {
      $message.error('文件URL为空，无法下载')
      return
    }

    // Electron：交给主进程用内置下载管理器流式写盘，避免大文件整块读入内存
    if (window.electron?.ipcRenderer) {
      if (messageId) downloadProgress.value[messageId] = 0
      window.electron.ipcRenderer.send('download-file', {
        url: fileUrl,
        token: getToken() || '',
        fileName: finalFileName,
        saveDir: props.messageSettings?.defaultSaveDirectory,
        downloadId: messageId
      })
      return
    }

    // 浏览器环境：fetch + blob 直链下载
    const response = await fetch(fileUrl, {
      method: 'GET',
      cache: 'no-store',
      headers: {
        ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
      }
    })
    if (!response.ok) {
      $message.error(response.status === 403 ? '文件下载失败: 权限不足，请检查您的权限' : '文件下载失败: 服务器错误')
      return
    }
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = finalFileName
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
    $message.success(`文件 ${finalFileName} 已下载到默认目录`)
  } catch (error) {
    logger.error('文件下载失败:', error)
    $message.error('文件下载失败: 网络错误')
  }
}

const saveFileAs = async (fileContent: string, messageId?: string) => {
  try {
    const parsedContent = JSON.parse(fileContent)
    const finalFileName = parsedContent.name || parsedContent.fileName || parsedContent.url.split('/').pop() || '文件'
    let fileUrl = parsedContent.url

    if (fileUrl && !fileUrl.startsWith('http')) {
      const cleanServerUrl = serverUrl.value.replace(/\/$/, '')
      const cleanFileUrl = fileUrl.replace(/^\//, '')
      fileUrl = `${cleanServerUrl}/${cleanFileUrl}`
    }

    if (!fileUrl) {
      $message.error('文件URL为空，无法保存')
      return
    }

    // Electron：先弹保存对话框，用户确认后才开始下载（避免"还没选文件夹就开始下载"的困惑）
    if (window.electron?.ipcRenderer) {
      const dialogResult = await window.electron.ipcRenderer.invoke('show-save-dialog', { defaultPath: finalFileName })
      if (dialogResult.canceled || !dialogResult.filePath) return

      if (messageId) downloadProgress.value[messageId] = 0
      window.electron.ipcRenderer.send('save-file-as', {
        url: fileUrl,
        token: getToken() || '',
        fileName: finalFileName,
        savePath: dialogResult.filePath,
        downloadId: messageId
      })
      return
    }

    // 浏览器环境：fetch + blob 直链下载
    const response = await fetch(fileUrl, {
      method: 'GET',
      cache: 'no-store',
      headers: {
        ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
      }
    })
    if (!response.ok) {
      $message.error(response.status === 403 ? '文件保存失败: 权限不足，请检查您的权限' : '文件保存失败: 服务器错误')
      return
    }
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = finalFileName
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
    $message.success(`文件 ${finalFileName} 已保存`)
  } catch (error) {
    logger.error('文件保存失败:', error)
    $message.error('文件保存失败: 网络错误')
  }
}


// 消息右键菜单添加文件相关选项
const showMessageContextMenu = (event: MouseEvent, message: Message) => {
  event.stopPropagation()

  // 已撤回的消息不显示右键菜单
  if (message.isRecalled) {
    return
  }

  // 关闭其他菜单系统
  window.dispatchEvent(new CustomEvent('qim:close-all-menus'))

  messageMenuSelection.value = window.getSelection?.()?.toString() ?? ''

  selectedMessage.value = message
  openMenu('message', event.clientX, event.clientY)

  // 检查消息类型
  if (message.type === 'file' || message.type === 'image') {
    // 可以在这里添加文件或图片特定的菜单选项
  }
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeMessageContextMenu)
  }, 0)
}

// 处理邀请成员
const handleInviteMembers = () => {
  if (props.conversation?.id) {
    emit('inviteMembers', props.conversation.id)
  }
}

const handleUpdateAISettings = async (settings: any) => {
  if (!props.conversation?.id) return

  const triggerKeywords = Array.isArray(settings.aiTriggerKeywords) 
    ? settings.aiTriggerKeywords.join(',') 
    : ''

  try {
    const data = await request(`/api/v1/groups/${props.conversation.id}/ai-settings`, {
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
        ai_trigger_keywords: triggerKeywords,
        ai_learn_enabled: settings.aiLearnEnabled,
        ai_extract_todos: settings.aiExtractTodos
      })
    })

    if (data.code === 0) {
      // 更新本地状态，实现回显
      if (props.updateConversation && props.conversation) {
        props.updateConversation({
          ...props.conversation,
          ai_config: {
            ai_enabled: settings.aiEnabled,
            ai_assistant_name: settings.aiAssistantName,
            ai_reply_mode: settings.aiReplyMode,
            ai_personality: settings.aiPersonality,
            ai_custom_prompt: settings.aiCustomPrompt,
            ai_language: settings.aiLanguage,
            ai_max_length: settings.aiMaxLength,
            ai_mention_reply_mode: settings.aiMentionReplyMode,
            ai_anti_spam_interval: settings.aiAntiSpamInterval,
            ai_trigger_keywords: triggerKeywords,
            ai_learn_enabled: settings.aiLearnEnabled,
            ai_extract_todos: settings.aiExtractTodos
          }
        })
      }
      if (data.message && data.message.includes('等待审批')) {
        QMessage.info(data.message)
      } else {
        QMessage.success('AI 设置已更新')
      }
    } else {
      QMessage.error(data.message || 'AI 设置更新失败')
    }
  } catch (error: any) {
    if (error.data && error.data.message && (error.data.message.includes('等待审批') || error.data.message.includes('已有待审批'))) {
      QMessage.info(error.data.message)
    } else {
      QMessage.error(error.data?.message || 'AI 设置更新失败')
    }
  }
}

defineExpose({
  startScreenShare,
  scrollToBottom,
  startMessageSelection,
  toggleMessageSelection,
  forwardSelectedMessages,
})
</script>

<style scoped>
/* ===== ChatWindow 组件自身使用的样式 ===== */
/* 基础布局样式已移至全局 chat.css */

.message-selection-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 12px max(20px, env(safe-area-inset-right)) calc(12px + env(safe-area-inset-bottom)) max(20px, env(safe-area-inset-left));
  border-top: 1px solid var(--border-color);
  background: color-mix(in srgb, var(--card-bg) 92%, transparent);
  box-shadow: 0 -6px 18px rgba(0, 0, 0, 0.08);
}

.message-selection-count {
  margin-right: auto;
  color: var(--text-color);
  font-size: 14px;
  font-weight: 600;
}

.message-selection-actions button {
  min-height: 40px;
  padding: 0 16px;
  border-radius: 8px;
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease, opacity 0.15s ease;
}

.message-selection-cancel {
  border: 1px solid var(--border-color);
  background: transparent;
  color: var(--text-color);
}

.message-selection-cancel:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.message-selection-forward {
  border: 1px solid var(--primary-color);
  background: var(--primary-color);
  color: #fff;
  box-shadow: 0 4px 10px color-mix(in srgb, var(--primary-color), transparent 70%);
}

.message-selection-forward:not(:disabled):hover {
  background: var(--primary-hover, var(--primary-color));
  border-color: var(--primary-hover, var(--primary-color));
}

.message-selection-forward:disabled {
  cursor: not-allowed;
  opacity: 0.5;
  box-shadow: none;
}

@media (max-width: 640px) {
  .message-selection-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
  }

  .message-selection-count {
    grid-column: 1 / -1;
    margin-right: 0;
  }

  .message-selection-actions button {
    width: 100%;
  }
}

.upload-progress-bar {
  width: 300px;
  height: 18px;
  margin: 0 auto 4px;
  border-radius: 9px;
  background: color-mix(in srgb, var(--primary-color), transparent 88%);
  color: var(--primary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 500;
  position: relative;
  overflow: hidden;
}

.upload-progress-bar__inner {
  position: absolute;
  left: 0;
  top: 0;
  height: 100%;
  background: color-mix(in srgb, var(--primary-color), transparent 70%);
  border-radius: 12px;
  transition: width 0.15s ease;
}

.upload-progress-bar__text {
  position: relative;
  z-index: 1;
}

/* fade 动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

/* ===== 小程序面板样式 ===== */

.mini-app-panel-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mini-app-panel-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
}

.mini-app-panel {
  position: relative;
  background: var(--sidebar-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.mini-app-panel-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mini-app-panel-header h4 {
  margin: 0;
  color: var(--text-color);
  font-size: 16px;
}

.mini-app-grid {
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.mini-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.mini-app-item-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s;
}

.mini-app-item-icon:hover {
  transform: scale(1.05);
}

.mini-app-item-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-app-item-name {
  font-size: 12px;
  color: var(--text-color);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100px;
}

.mini-app-item-actions {
  margin-top: 4px;
}

.mini-app-action-btn {
  font-size: 10px;
  padding: 2px 8px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s;
}

.mini-app-action-btn:hover {
  background: var(--primary-hover);
}
</style>
