<template>
  <div class="chat-input-area">
    <div class="input-toolbar">
      <button class="toolbar-btn" @click="$emit('start-voice-call')"><i class="fas fa-phone-alt"></i></button>
      <button class="toolbar-btn" @click="$emit('start-video-call')"><i class="fas fa-video"></i></button>
      <button class="toolbar-btn" @click="$emit('start-screen-share')"><i class="fas fa-desktop"></i></button>
      <button class="toolbar-btn" @click="$emit('toggle-emoji-panel')"><i class="fas fa-smile"></i></button>
      <button class="toolbar-btn" @click="$emit('select-file')"><i class="fas fa-paperclip"></i></button>
      <button class="toolbar-btn" @click="$emit('select-image')"><i class="fas fa-image"></i></button>
      <button v-if="isElectron" class="toolbar-btn" @click="$emit('take-screenshot')"><i class="fas fa-scissors"></i></button>
      <button class="toolbar-btn" @click="$emit('open-message-manager')"><i class="fas fa-history"></i></button>
      <button class="toolbar-btn" @click="$emit('open-mini-app-list')"><i class="fas fa-th-large"></i></button>
    </div>
    
    <div v-if="showEmojiPanel" class="emoji-panel-container">
      <div class="emoji-panel-backdrop" @click="$emit('close-emoji-panel')"></div>
      <div class="emoji-panel">
        <div class="emoji-category">
          <div class="emoji-category-title">常用表情</div>
          <div class="emoji-grid">
            <div v-for="emoji in commonEmojis" :key="emoji" class="emoji-item" @click="$emit('insert-emoji', emoji)">{{ emoji }}</div>
          </div>
        </div>
        <div class="emoji-category">
          <div class="emoji-category-title">表情符号</div>
          <div class="emoji-grid">
            <div v-for="emoji in faceEmojis" :key="emoji" class="emoji-item" @click="$emit('insert-emoji', emoji)">{{ emoji }}</div>
          </div>
        </div>
        <div class="emoji-category">
          <div class="emoji-category-title">动物与自然</div>
          <div class="emoji-grid">
            <div v-for="emoji in animalEmojis" :key="emoji" class="emoji-item" @click="$emit('insert-emoji', emoji)">{{ emoji }}</div>
          </div>
        </div>
      </div>
    </div>
    
    <div v-if="showAtMembersPanel && (conversation?.type === 'group' || conversation?.type === 'discussion')" class="at-members-panel-container">
      <div class="at-members-panel-backdrop" @click="$emit('close-at-members-panel')"></div>
      <div class="at-members-panel">
        <div class="at-members-header"><h4>选择成员</h4></div>
        <div class="at-members-search">
          <input v-model="atMembersSearchQuery" type="text" placeholder="搜索成员..." class="at-members-search-input" />
        </div>
        <div class="at-members-list">
          <div class="at-member-item" @click="$emit('select-at-all')">
            <img src="https://api.dicebear.com/7.x/avataaars/svg?seed=all" alt="所有人" class="at-member-avatar" />
            <span class="at-member-name">所有人</span>
          </div>
          <div v-for="member in filteredAtMembers" :key="member.id" class="at-member-item" @click="$emit('select-at-member', member)">
            <img :src="member.avatar" :alt="member.name || '未知用户'" class="at-member-avatar" />
            <span class="at-member-name">{{ member.name || '未知用户' }}</span>
          </div>
          <div v-if="filteredAtMembers.length === 0" class="empty-at-members"><p>没有找到匹配的成员</p></div>
        </div>
      </div>
    </div>
    
    <MiniAppManager v-model:showMiniAppList="showMiniAppListLocal" @send-mini-app-message="$emit('send-mini-app-message', $event)" />
    
    <input type="file" ref="fileInputRef" style="display: none" @change="$emit('handle-file-select', $event)" multiple />

    <div v-if="showSearch" class="search-container">
      <input v-model="searchQueryLocal" type="text" placeholder="搜索历史消息..." class="search-input" @keyup.enter="$emit('perform-search')" />
      <button class="search-btn" @click="$emit('perform-search')">搜索</button>
      <button class="close-search-btn" @click="$emit('close-search')">×</button>
    </div>
    
    <QuotedMessageInput v-if="quotedMessage" :quoted-message="quotedMessage" @remove="$emit('remove-quoted-message')" />

    <div v-if="pendingFiles.length > 0" class="pending-files">
      <div v-for="(file, index) in pendingFiles" :key="index" class="pending-file-item">
        <span class="pending-file-icon"><i :class="getFileIcon(file.name)"></i></span>
        <span class="pending-file-name">{{ file.name }}</span>
        <button class="pending-file-remove" @click="$emit('remove-pending-file', index)">×</button>
      </div>
    </div>
    
    <textarea ref="messageInputRef" v-model="inputMessageLocal" class="message-input" placeholder="输入消息..." rows="4" @keydown.enter="$emit('handle-keydown', $event)" @input="handleInputAndResize" @paste="$emit('handle-paste', $event)" />
    
    <div class="input-actions">
      <span class="input-tip">按 Enter 发送，Shift+Enter 换行</span>
      <button class="send-btn" :disabled="!inputMessageLocal.trim() && pendingFiles.length === 0" @click="$emit('send')">发送</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import MiniAppManager from '../apps/MiniAppManager.vue'
import QuotedMessageInput from '../message/QuotedMessageInput.vue'

interface PendingFile { file: File; name: string }
interface Member { id: string; name: string; avatar: string }
interface Conversation { id: string; type: 'single' | 'group' | 'discussion'; members?: Member[] }

interface Props {
  conversation: Conversation | null
  inputMessage: string
  pendingFiles: PendingFile[]
  showEmojiPanel: boolean
  showAtMembersPanel: boolean
  showMiniAppList: boolean
  showSearch: boolean
  searchQuery: string
  quotedMessage: any
  commonEmojis: string[]
  faceEmojis: string[]
  animalEmojis: string[]
  isElectron: boolean
  getFileIcon: (fileUrl: string) => string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:inputMessage', value: string): void
  (e: 'send'): void
  (e: 'toggle-emoji-panel'): void
  (e: 'close-emoji-panel'): void
  (e: 'select-file'): void
  (e: 'select-image'): void
  (e: 'take-screenshot'): void
  (e: 'open-message-manager'): void
  (e: 'open-mini-app-list'): void
  (e: 'start-voice-call'): void
  (e: 'start-video-call'): void
  (e: 'start-screen-share'): void
  (e: 'insert-emoji', emoji: string): void
  (e: 'close-at-members-panel'): void
  (e: 'select-at-member', member: Member): void
  (e: 'select-at-all'): void
  (e: 'handle-file-select', event: Event): void
  (e: 'handle-paste', event: ClipboardEvent): void
  (e: 'handle-keydown', event: KeyboardEvent): void
  (e: 'remove-pending-file', index: number): void
  (e: 'remove-quoted-message'): void
  (e: 'perform-search'): void
  (e: 'close-search'): void
  (e: 'send-mini-app-message', miniApp: any): void
  (e: 'update:searchQuery', value: string): void
  (e: 'update:showMiniAppList', value: boolean): void
  (e: 'input', event: Event): void
}>()

const fileInputRef = ref<HTMLInputElement | null>(null)
const messageInputRef = ref<HTMLTextAreaElement | null>(null)
const atMembersSearchQuery = ref('')
const showMiniAppListLocal = computed({ get: () => props.showMiniAppList, set: (val) => emit('update:showMiniAppList', val) })
const inputMessageLocal = computed({ get: () => props.inputMessage, set: (val) => emit('update:inputMessage', val) })
const searchQueryLocal = computed({ get: () => props.searchQuery, set: (val) => emit('update:searchQuery', val) })

const filteredAtMembers = computed(() => {
  if (!props.conversation) return []
  if (!atMembersSearchQuery.value) return props.conversation.members || []
  const query = atMembersSearchQuery.value.toLowerCase()
  return (props.conversation.members || []).filter(member => member.name.toLowerCase().includes(query))
})

const handleInputAndResize = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement
  textarea.style.height = 'auto'
  const maxHeight = 200
  const scrollHeight = textarea.scrollHeight
  textarea.style.height = `${Math.min(scrollHeight, maxHeight)}px`
  textarea.style.overflowY = scrollHeight > maxHeight ? 'auto' : 'hidden'
  emit('input', event)
}

defineExpose({ messageInputRef })
</script>
