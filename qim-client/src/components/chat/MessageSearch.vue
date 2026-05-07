<template>
  <div class="message-search">
    <!-- 搜索状态 -->
    <div v-if="isSearching" class="search-status">
      <div class="search-loading">搜索中...</div>
    </div>

    <!-- 搜索结果 -->
    <div v-else-if="searchResults.length > 0" class="search-results">
      <div class="search-results-header">
        找到 {{ searchResults.length }} 条相关消息
        <button class="clear-search-btn" @click="$emit('clear-search')">清除搜索</button>
      </div>
      <MessageItem
        v-for="message in searchResults"
        :key="message.id"
        :message="message"
        :is-self="message.isSelf"
        :is-recalled="message.isRecalled"
        :conversation-type="conversationType"
        :read-users-map="readUsersMap"
        :server-url="serverUrl"
        @contextmenu="(e: MouseEvent) => $emit('message-contextmenu', e, message)"
        @show-user-profile="(user: any) => $emit('show-user-profile', user)"
        @preview-image="(img: any) => $emit('preview-image', img)"
        @download-file="(file: any) => $emit('download-file', file)"
        @save-as="(file: any) => $emit('save-as', file)"
        @view-shared-content="(content: any) => $emit('view-shared-content', content)"
        @retry-send-message="(msg: any) => $emit('retry-send-message', msg)"
        @show-read-users="(msg: any) => $emit('show-read-users', msg)"
        @scroll-to-quoted-message="(msg: any) => $emit('scroll-to-quoted-message', msg)"
      />
    </div>

    <!-- 无搜索结果 -->
    <div v-else-if="searchQuery && !isSearching" class="search-status">
      <div class="search-no-results">没有找到相关消息</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import MessageItem from '../message/MessageItem.vue'

interface Message {
  id: string
  content: string
  isSelf: boolean
  isRecalled: boolean
  timestamp: number
  // 其他消息属性
  [key: string]: any
}

interface Props {
  searchResults: Message[]
  searchQuery: string
  isSearching: boolean
  conversationType: 'single' | 'group' | 'discussion'
  readUsersMap: Record<string, { read_users: any[], total_members: number }>
  serverUrl: string
}

defineProps<Props>()

defineEmits<{
  (e: 'clear-search'): void
  (e: 'message-contextmenu', event: MouseEvent, message: Message): void
  (e: 'show-user-profile', user: any): void
  (e: 'preview-image', image: any): void
  (e: 'download-file', file: any): void
  (e: 'save-as', file: any): void
  (e: 'view-shared-content', content: any): void
  (e: 'retry-send-message', message: any): void
  (e: 'show-read-users', message: any): void
  (e: 'scroll-to-quoted-message', message: any): void
}>()
</script>

<style scoped>
.message-search {
  width: 100%;
}

.search-status {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #666;
  font-size: 14px;
}

.search-loading {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-loading::before {
  content: '';
  width: 16px;
  height: 16px;
  border: 2px solid #e0e0e0;
  border-top: 2px solid #1976d2;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.search-results {
  padding: 16px 20px;
}

.search-results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 8px;
  font-size: 14px;
  color: #333;
}

.clear-search-btn {
  padding: 4px 12px;
  background: transparent;
  border: 1px solid #1976d2;
  border-radius: 4px;
  color: #1976d2;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s ease;
}

.clear-search-btn:hover {
  background: #1976d2;
  color: #fff;
}

.search-no-results {
  color: #999;
}
</style>
