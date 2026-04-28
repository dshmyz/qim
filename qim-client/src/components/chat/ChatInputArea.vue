<template>
  <div class="chat-input-area">
    <!-- AI 快捷指令栏 -->
    <AIQuickActions
      :is-processing="isProcessing"
      @action="emit('ai-action', $event)"
    />

    <!-- 消息输入区 -->
    <MessageInput
      ref="messageInputRef"
      :conversation="conversation as InputConversation"
      v-model:input-message="inputMessage"
      :pending-files="pendingFiles"
      :show-emoji-panel="showEmojiPanel"
      :show-at-members-panel="showAtMembersPanel"
      v-model:show-mini-app-list="showMiniAppList"
      :quoted-message="quotedMessage"
      :is-electron="isElectron"
      :get-file-icon="getFileIcon"
      :show-search="showSearch"
      v-model:search-query="searchQuery"
      @send="emit('send')"
      @input="emit('input', $event)"
      @toggle-emoji-panel="emit('toggle-emoji-panel')"
      @close-emoji-panel="emit('close-emoji-panel')"
      @select-file="emit('select-file')"
      @select-image="emit('select-image')"
      @take-screenshot="emit('take-screenshot')"
      @open-message-manager="emit('open-message-manager')"
      @open-mini-app-list="emit('open-mini-app-list')"
      @start-voice-call="emit('start-voice-call')"
      @start-video-call="emit('start-video-call')"
      @start-screen-share="emit('start-screen-share')"
      @insert-emoji="emit('insert-emoji', $event)"
      @close-at-members-panel="emit('close-at-members-panel')"
      @select-at-member="emit('select-at-member', $event)"
      @select-at-all="emit('select-at-all')"
      @handle-file-select="emit('handle-file-select', $event)"
      @handle-paste="emit('handle-paste', $event)"
      @handle-keydown="emit('handle-keydown', $event)"
      @remove-pending-file="emit('remove-pending-file', $event)"
      @remove-quoted-message="emit('remove-quoted-message')"
      @send-mini-app-message="emit('send-mini-app-message', $event)"
      @perform-search="emit('perform-search')"
      @close-search="emit('close-search')"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Message } from '../../types'
import MessageInput from './MessageInput.vue'
import AIQuickActions from '../ai/AIQuickActions.vue'

// MessageInput 内部定义的 Conversation 类型不包含 'bot'
// 定义一个兼容类型用于向 MessageInput 传递
interface InputConversation {
  id: string
  type: 'single' | 'group' | 'discussion'
  members?: Array<{ id: string; name: string; avatar: string }>
}

interface Props {
  conversation: import('../../types').Conversation | null
  inputMessage: string
  pendingFiles: Array<{ file: File; name: string }>
  showEmojiPanel: boolean
  showAtMembersPanel: boolean
  showMiniAppList: boolean
  showSearch: boolean
  searchQuery: string
  quotedMessage: Message | null
  isElectron: boolean
  isProcessing: boolean
  getFileIcon: (fileName: string) => string
}

defineProps<Props>()

const emit = defineEmits<{
  'send': []
  'input': [event: Event]
  'toggle-emoji-panel': []
  'close-emoji-panel': []
  'select-file': []
  'select-image': []
  'take-screenshot': []
  'open-message-manager': []
  'open-mini-app-list': []
  'start-voice-call': []
  'start-video-call': []
  'start-screen-share': []
  'insert-emoji': [emoji: string]
  'close-at-members-panel': []
  'select-at-member': [member: { id: string; name: string; avatar: string }]
  'select-at-all': []
  'handle-file-select': [event: Event]
  'handle-paste': [event: ClipboardEvent]
  'handle-keydown': [event: KeyboardEvent]
  'remove-pending-file': [index: number]
  'remove-quoted-message': []
  'send-mini-app-message': [miniApp: Message['miniAppData']]
  'ai-action': [actionId: string]
  'perform-search': []
  'close-search': []
}>()

const messageInputRef = ref<InstanceType<typeof MessageInput>>()

defineExpose({
  messageInputRef
})
</script>

<style scoped>
.chat-input-area {
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--border-color);
}
</style>
