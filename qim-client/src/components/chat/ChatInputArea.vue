<template>
  <!-- 消息输入区 -->
  <MessageInput
    ref="messageInputRef"
    :conversation="conversation as InputConversation"
    v-model:input-message="inputMessageLocal"
    :pending-files="pendingFiles"
    :show-emoji-panel="showEmojiPanel"
    :show-at-members-panel="showAtMembersPanel"
    v-model:show-mini-app-list="showMiniAppListLocal"
    :quoted-message="quotedMessage"
    :is-electron="isElectron"
    :get-file-icon="getFileIcon"
    :show-search="showSearch"
    v-model:search-query="searchQueryLocal"
    :is-processing="props.isProcessing ?? false"
    @send="emit('send')"
    @input="emit('input', $event)"
    @toggle-emoji-panel="emit('toggle-emoji-panel')"
    @close-emoji-panel="emit('close-emoji-panel')"
    @select-file="emit('select-file')"
    @select-image="emit('select-image')"
    @take-screenshot="emit('take-screenshot')"
    @take-screenshot-hidden="emit('take-screenshot-hidden')"
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
    @ai-action="emit('ai-action', $event)"
    @handle-drop="emit('handle-drop', $event)"
  />
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Message } from '../../types'
import MessageInput from './MessageInput.vue'

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
  isProcessing?: boolean
  getFileIcon: (fileName: string) => string
}

const props = withDefaults(defineProps<Props>(), {
  isProcessing: false
})

const emit = defineEmits<{
  'send': []
  'input': [event: Event]
  'toggle-emoji-panel': []
  'close-emoji-panel': []
  'select-file': []
  'select-image': []
  'take-screenshot': []
  'take-screenshot-hidden': []
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
  'handle-drop': [event: DragEvent]
  'perform-search': []
  'close-search': []
  'update:inputMessage': [value: string]
  'update:showMiniAppList': [value: boolean]
  'update:searchQuery': [value: string]
}>()

const messageInputRef = ref<InstanceType<typeof MessageInput>>()

const inputMessageLocal = computed({
  get: () => props.inputMessage,
  set: (val) => emit('update:inputMessage', val)
})

const showMiniAppListLocal = computed({
  get: () => props.showMiniAppList,
  set: (val) => emit('update:showMiniAppList', val)
})

const searchQueryLocal = computed({
  get: () => props.searchQuery,
  set: (val) => emit('update:searchQuery', val)
})

defineExpose({
  messageInputRef
})
</script>
