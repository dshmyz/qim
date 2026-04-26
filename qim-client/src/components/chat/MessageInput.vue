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
      <EmojiPanel @select="handleEmojiSelect" />
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
import EmojiPanel from './EmojiPanel.vue'
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

const handleEmojiSelect = (emoji: string) => {
  emit('insert-emoji', emoji)
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

defineExpose({ messageInputRef })
</script>

<style scoped>
.chat-input-area {
  padding: 12px 20px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 150px;
  position: relative;
  box-shadow: 0 -1px 2px rgba(0, 0, 0, 0.03);
}

.input-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
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
  border-radius: 8px;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  min-height: 120px;
  max-height: 200px;
  overflow-y: hidden;
  box-sizing: border-box;
  background: var(--sidebar-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.message-input:focus {
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

.at-members-search {
  margin-bottom: 12px;
}

.at-members-search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
}

.at-members-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
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

.at-member-item:hover {
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

.empty-at-members {
  padding: 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
}

/* 搜索相关样式 */
.search-container {
  display: flex;
  align-items: center;
  padding: 8px 0;
  margin-bottom: 12px;
  gap: 8px;
  background: var(--sidebar-bg);
  padding: 8px 12px;
  border-radius: 8px;
}

.search-input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  background: var(--sidebar-bg);
  color: var(--text-color);
}

.search-input:focus {
  border-color: var(--primary-color);
}

.search-btn {
  padding: 6px 16px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}

.search-btn:hover {
  opacity: 0.9;
}

.close-search-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.close-search-btn:hover {
  background: var(--hover-color);
}

/* 待发送文件样式 */
.pending-files {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px;
  background: var(--list-bg);
  border-radius: 8px;
  border: 1px solid var(--border-color);
}

.pending-file-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  background: var(--content-bg);
  border-radius: 6px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.pending-file-item:hover {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.05);
}

.pending-file-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--primary-light);
  border-radius: 4px;
  color: var(--primary-color);
  font-size: 14px;
  flex-shrink: 0;
}

.pending-file-name {
  flex: 1;
  font-size: 14px;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pending-file-remove {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.pending-file-remove:hover {
  background: rgba(244, 67, 54, 0.1);
  color: #f44336;
}

/* ==================== 暗黑主题样式 ==================== */

[data-theme="dark"] .message-input {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .emoji-panel {
  background: var(--sidebar-bg) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2) !important;
}

[data-theme="dark"] .search-container {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .search-input {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

/* 炫酷黑主题 - 发送按钮样式 */
[data-theme="dark"] .send-btn {
  background: #2d3748 !important;
  color: rgba(229, 231, 235, 1) !important;
  border: 1px solid rgba(229, 231, 235, 0.3) !important;
}

[data-theme="dark"] .send-btn:hover:not(:disabled) {
  background: #374151 !important;
  opacity: 1 !important;
}

[data-theme="dark"] .send-btn:disabled {
  background: #1a1a1a !important;
  color: rgba(229, 231, 235, 0.7) !important;
  border: 1px solid rgba(229, 231, 235, 0.3) !important;
}

[data-theme="dark"] .toolbar-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .send-btn {
  color: white !important;
}

/* 暗黑主题下的待发送文件样式 */
[data-theme="dark"] .pending-files {
  background: var(--secondary-color);
  border-color: var(--border-color);
}

[data-theme="dark"] .pending-file-item {
  background: var(--sidebar-bg);
  border-color: var(--border-color);
}

[data-theme="dark"] .pending-file-item:hover {
  border-color: var(--primary-color);
  background: rgba(59, 130, 246, 0.1);
}

[data-theme="dark"] .pending-file-icon {
  background: rgba(59, 130, 246, 0.2);
  color: var(--primary-color);
}

[data-theme="dark"] .pending-file-name {
  color: var(--text-color);
}

[data-theme="dark"] .pending-file-remove:hover {
  background: rgba(244, 67, 54, 0.15);
  color: #f44336;
}
</style>
