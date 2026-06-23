<template>
  <div
    class="chat-input-area"
    :class="{ 'drag-over': isDragOver }"
    @dragover.prevent="handleDragOver"
    @dragleave.prevent="handleDragLeave"
    @drop.prevent="handleDrop"
  >
    <ChatToolbar
      :is-electron="isElectron"
      :show-ai-actions="localShowAIActions"
      @start-voice-call="$emit('start-voice-call')"
      @start-video-call="$emit('start-video-call')"
      @start-screen-share="$emit('start-screen-share')"
      @toggle-emoji-panel="$emit('toggle-emoji-panel')"
      @select-file="handleSelectFile"
      @select-image="$emit('select-image')"
      @take-screenshot="$emit('take-screenshot')"
      @take-screenshot-hidden="$emit('take-screenshot-hidden')"
      @open-message-manager="$emit('open-message-manager')"
      @open-mini-app-list="$emit('open-mini-app-list')"
      @toggle-ai-actions="toggleAI"
    />

    <transition name="ai-actions-slide">
      <div v-if="localShowAIActions" class="ai-actions-bar">
        <AIQuickActions
          :is-processing="isProcessing"
          @action="$emit('ai-action', $event)"
        />
      </div>
    </transition>
    
    <div v-if="showEmojiPanel" class="emoji-panel-container">
      <div class="emoji-panel-backdrop" @click="$emit('close-emoji-panel')"></div>
      <EmojiPanel @select="handleEmojiSelect" />
    </div>
    
    <div v-if="showAtMembersPanel && (conversation?.type === 'group' || conversation?.type === 'discussion')" class="at-members-panel-container">
      <div class="at-members-panel-backdrop" @click="$emit('close-at-members-panel')"></div>
      <div
        class="at-members-panel"
        role="listbox"
        aria-label="选择提及成员"
        @keydown="handleAtMembersKeyDown"
      >
        <div class="at-members-header"><h4>选择成员</h4></div>
        <div ref="atMembersListRef" class="at-members-list" role="list">
          <div
            class="at-member-item"
            :class="{ 'at-member-item--active': atMemberActiveIndex === -1 }"
            role="option"
            aria-selected="false"
            @click="$emit('select-at-all')"
          >
            <img :src="generateAvatar('所有人')" alt="所有人" class="at-member-avatar" />
            <span class="at-member-name">所有人</span>
          </div>
          <div
            v-for="(member, index) in filteredAtMembers"
            :key="member.id"
            class="at-member-item"
            :class="{ 'at-member-item--active': atMemberActiveIndex === index }"
            role="option"
            aria-selected="false"
            @click="$emit('select-at-member', member)"
          >
            <Avatar :src="member.avatar" :name="member.name || '未知用户'" :alt="member.name || '未知用户'" size="sm" class="at-member-avatar" />
            <span class="at-member-identity">
              <span class="at-member-name">{{ member.name || '未知用户' }}</span>
              <span v-if="member.username && member.username !== member.name" class="at-member-username">@{{ member.username }}</span>
            </span>
          </div>
          <div v-if="filteredAtMembers.length === 0" class="empty-at-members"><p>没有找到匹配的成员</p></div>
        </div>
      </div>
    </div>
    
    <MiniAppManager v-model:showMiniAppList="showMiniAppListLocal" @send-mini-app-message="$emit('send-mini-app-message', $event)" />
    
    <input type="file" ref="fileInputRef" style="display: none" @change="$emit('handle-file-select', $event)" multiple />

    <QuotedMessageInput v-if="quotedMessage" :quoted-message="quotedMessage" @remove="$emit('remove-quoted-message')" />

    <!-- 统一的 composer 容器：预览区 + textarea 融合在一个容器内 -->
    <div class="composer">
      <PendingFilesPreview
        :pending-files="pendingFiles"
        :get-file-icon="getFileIcon"
        @remove="$emit('remove-pending-file', $event)"
      />
      <textarea
        ref="messageInputRef"
        v-model="inputMessageLocal"
        class="message-input"
        placeholder="输入消息..."
        rows="4"
        @keydown="handleTextareaKeydown"
        @input="handleInputAndResize"
        @keyup="$emit('cursor-change', $event)"
        @click="$emit('cursor-change', $event)"
        @paste="$emit('handle-paste', $event)"
      />
    </div>
    
    <div class="input-actions">
      <span class="input-tip">按 Enter 发送，Shift+Enter 换行</span>
      <button class="send-btn" :disabled="!inputMessageLocal.trim() && pendingFiles.length === 0" @click="$emit('send')">发送</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import EmojiPanel from './EmojiPanel.vue'
import MiniAppManager from '../apps/MiniAppManager.vue'
import QuotedMessageInput from '../message/QuotedMessageInput.vue'
import AIQuickActions from '../ai/AIQuickActions.vue'
import ChatToolbar from './ChatToolbar.vue'
import PendingFilesPreview from './PendingFilesPreview.vue'
import { generateAvatar } from '../../utils/avatar'
import Avatar from '../shared/Avatar.vue'

interface PendingFile { file: File; name: string }
interface Member { id: string; name: string; username?: string; avatar: string }
interface Conversation { id: string; type: 'single' | 'group' | 'discussion'; members?: Member[] }

interface Props {
  conversation: Conversation | null
  inputMessage: string
  pendingFiles: PendingFile[]
  showEmojiPanel: boolean
  showAtMembersPanel: boolean
  atMembersQuery: string
  showMiniAppList: boolean
  isProcessing?: boolean
  quotedMessage: any
  isElectron: boolean
  getFileIcon: (fileUrl: string) => string
}

const props = withDefaults(defineProps<Props>(), {
  isProcessing: false
})

const emit = defineEmits<{
  (e: 'update:inputMessage', value: string): void
  (e: 'send'): void
  (e: 'toggle-emoji-panel'): void
  (e: 'close-emoji-panel'): void
  (e: 'select-file'): void
  (e: 'select-image'): void
  (e: 'take-screenshot'): void
  (e: 'take-screenshot-hidden'): void
  (e: 'open-message-manager'): void
  (e: 'open-mini-app-list'): void
  (e: 'toggle-ai-actions'): void
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
  (e: 'send-mini-app-message', miniApp: any): void
  (e: 'ai-action', actionId: string): void
  (e: 'update:showMiniAppList', value: boolean): void
  (e: 'input', event: Event): void
  (e: 'cursor-change', event: Event): void
  (e: 'handle-drop', event: DragEvent): void
}>()

const fileInputRef = ref<HTMLInputElement | null>(null)
const messageInputRef = ref<HTMLTextAreaElement | null>(null)
const atMembersListRef = ref<HTMLDivElement | null>(null)
const atMemberActiveIndex = ref(-1)
const localShowAIActions = ref(false)
const isDragOver = ref(false)
const showMiniAppListLocal = computed({ get: () => props.showMiniAppList, set: (val) => emit('update:showMiniAppList', val) })
const inputMessageLocal = computed({ get: () => props.inputMessage, set: (val) => emit('update:inputMessage', val) })

const filteredAtMembers = computed(() => {
  if (!props.conversation) return []
  if (!props.atMembersQuery) return props.conversation.members || []
  const query = props.atMembersQuery.toLowerCase()
  return (props.conversation.members || []).filter(member =>
    member.name.toLowerCase().includes(query) || member.username?.toLowerCase().includes(query)
  )
})

const handleEmojiSelect = (emoji: string) => {
  emit('insert-emoji', emoji)
}

const handleSelectFile = () => {
  fileInputRef.value?.click()
}

const toggleAI = () => {
  localShowAIActions.value = !localShowAIActions.value
}

const handleDragOver = (event: DragEvent) => {
  if (event.dataTransfer?.types.includes('Files')) {
    isDragOver.value = true
  }
}

const handleDragLeave = (event: DragEvent) => {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  if (event.clientX <= rect.left || event.clientX >= rect.right || event.clientY <= rect.top || event.clientY >= rect.bottom) {
    isDragOver.value = false
  }
}

const handleDrop = (event: DragEvent) => {
  isDragOver.value = false
  if (event.dataTransfer?.files.length) {
    emit('handle-drop', event)
  }
}

const scrollToActiveAtMember = () => {
  nextTick(() => {
    const items = atMembersListRef.value?.querySelectorAll('.at-member-item')
    if (!items || !items[atMemberActiveIndex.value + 1]) return
    
    const activeItem = items[atMemberActiveIndex.value + 1] as HTMLElement
    const container = atMembersListRef.value
    if (!container) return

    const containerRect = container.getBoundingClientRect()
    const itemRect = activeItem.getBoundingClientRect()

    if (itemRect.top < containerRect.top) {
      container.scrollTop = container.scrollTop + (itemRect.top - containerRect.top)
    } else if (itemRect.bottom > containerRect.bottom) {
      container.scrollTop = container.scrollTop + (itemRect.bottom - containerRect.bottom)
    }
  })
}

const handleAtMembersKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp' || event.key === 'Enter' || event.key === 'Escape') {
    event.preventDefault()
  }

  const totalOptions = filteredAtMembers.value.length + 1

  switch (event.key) {
    case 'ArrowDown':
      atMemberActiveIndex.value = (atMemberActiveIndex.value + 1) % totalOptions
      scrollToActiveAtMember()
      break
    case 'ArrowUp':
      atMemberActiveIndex.value = (atMemberActiveIndex.value - 1 + totalOptions) % totalOptions
      scrollToActiveAtMember()
      break
    case 'Enter':
      if (atMemberActiveIndex.value === -1) {
        emit('select-at-all')
      } else if (atMemberActiveIndex.value >= 0 && atMemberActiveIndex.value < filteredAtMembers.value.length) {
        emit('select-at-member', filteredAtMembers.value[atMemberActiveIndex.value])
      }
      break
    case 'Escape':
      emit('close-at-members-panel')
      break
  }
}

const handleTextareaKeydown = (event: KeyboardEvent) => {
  if (props.showAtMembersPanel && ['ArrowDown', 'ArrowUp', 'Enter', 'Escape'].includes(event.key)) {
    handleAtMembersKeyDown(event)
    return
  }

  emit('handle-keydown', event)
}

const handleInputAndResize = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement
  textarea.style.height = 'auto'
  const maxHeight = 200
  const scrollHeight = textarea.scrollHeight
  textarea.style.height = `${Math.min(scrollHeight, maxHeight)}px`
  textarea.style.overflowY = scrollHeight > maxHeight ? 'auto' : 'hidden'
  emit('input', event)
}

watch(
  () => props.showAtMembersPanel,
  (newVal) => {
    if (newVal) {
      atMemberActiveIndex.value = -1
    }
  }
)

watch(
  () => props.atMembersQuery,
  () => {
    atMemberActiveIndex.value = -1
  }
)

defineExpose({ messageInputRef })
</script>

<style scoped>
.chat-input-area {
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  /* gap: 10px; */
  min-height: 150px;
  position: relative;
  transition: all 0.2s ease;
}

.chat-input-area.drag-over {
  background: var(--primary-light, rgba(59, 130, 246, 0.08));
  border: 2px dashed var(--primary-color);
  border-radius: 8px;
}

.input-toolbar {
  display: flex;
  gap: 8px;
}

.toolbar-btn {
  width: 30px;
  height: 30px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.toolbar-btn:hover {
  background: var(--hover-color);
}

.toolbar-divider {
  width: 1px;
  height: 20px;
  background: var(--border-color);
  margin: 0 4px;
  align-self: center;
}

.ai-toolbar-btn svg {
  width: 16px;
  height: 16px;
  fill: currentColor;
  transition: transform 0.3s ease;
}

.ai-toolbar-btn.ai-active svg {
  transform: rotate(180deg);
}

.ai-toolbar-btn.ai-active {
  background: var(--primary-light);
  color: var(--primary-color);
}

.ai-actions-bar {
  padding: 8px 12px;
  background: var(--card-bg);
  border-radius: 6px;
  /* margin-bottom: 8px; */
  /* border: 1px solid var(--border-color); */
}

.ai-actions-slide-enter-active,
.ai-actions-slide-leave-active {
  transition: all 0.3s ease;
  max-height: 100px;
  opacity: 1;
}

.ai-actions-slide-enter-from,
.ai-actions-slide-leave-to {
  max-height: 0;
  opacity: 0;
  padding: 0 12px;
  margin-bottom: 0;
}

/* 表情面板容器样式 */
.emoji-panel-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
}

.emoji-panel-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: transparent;
}

/* 表情面板样式 */
.emoji-panel {
  position: absolute;
  bottom: 100%;
  left: 20px;
  right: 20px;
  margin-bottom: 8px;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px;
  max-height: 280px;
  overflow-y: auto;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 101;
}

.emoji-category {
  margin-bottom: 12px;
}

.emoji-category-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-color);
  opacity: 0.7;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(16, 1fr);
  gap: 2px;
}

.emoji-item {
  font-size: 20px;
  text-align: center;
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  transition: background 0.2s ease;
  min-width: 24px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.emoji-item:hover {
  background: var(--hover-color);
}

.message-input {
  width: 100%;
  padding: 10px 12px;
  border: none;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  min-height: 120px;
  max-height: 200px;
  overflow-y: hidden;
  box-sizing: border-box;
  background: transparent;
  color: var(--text-color);
}

.message-input:focus {
  outline: none;
}

/* composer：统一容器，包裹预览区和 textarea */
.composer {
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.2s ease;
}

.composer:focus-within {
  border-color: var(--primary-color);
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}

.input-tip {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.6;
}

.send-btn {
  padding: 8px 24px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.send-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.send-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}

/* @成员面板 */
.at-members-panel-container {
  position: relative;
  z-index: 1000;
  margin-top: 8px;
}

.at-members-panel-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.1);
  z-index: -1;
}

.at-members-panel {
  background: var(--list-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  max-height: 300px;
  overflow-y: auto;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  min-width: 200px;
}

.at-members-header {
  margin-bottom: 12px;
}

.at-members-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.at-members-list {
  max-height: 200px;
  overflow-y: auto;
}

.at-member-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-bottom: 4px;
}

.at-member-item:hover,
.at-member-item--active {
  background: var(--hover-color);
}

.at-member-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  margin-right: 8px;
  object-fit: cover;
}

.at-member-name {
  font-size: 14px;
  color: var(--text-color);
}

.at-member-identity {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.at-member-username {
  margin-top: 2px;
  color: var(--text-secondary);
  font-size: 12px;
}

.empty-at-members {
  padding: 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
}


</style>
