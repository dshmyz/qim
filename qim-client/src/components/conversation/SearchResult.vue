<template>
  <div v-if="searchQuery && searchResults.length > 0" class="search-popup">
    <div class="search-popup-content">
      <div class="search-popup-header">
        <span>搜索结果</span>
        <span class="search-popup-count">{{ searchResults.length }} 个结果</span>
      </div>
      <div class="search-popup-list">
        <div
          v-for="item in searchResults"
          :key="item.id + item.type"
          class="search-popup-item"
          @click="$emit('select', item)"
        >
          <div class="search-popup-avatar">
            <img :src="getAvatarUrl(item.avatar, item.name, serverUrl)" :alt="item.name" />
            <span v-if="item.type === 'group'" class="group-badge">群</span>
            <span v-if="item.type === 'discussion'" class="discussion-badge group-badge"><i class="fas fa-comments"></i></span>
          </div>
          <div class="search-popup-info">
            <div class="search-popup-name">{{ item.name }}</div>
            <div class="search-popup-meta">
              <span v-if="item.type === 'user'" class="search-popup-username">{{ item.username }}</span>
              <span v-if="item.type === 'group'" class="search-popup-type">群聊</span>
              <span v-if="item.type === 'discussion'" class="search-popup-type">讨论组</span>
              <span v-if="item.type === 'user' && item.status === 'online'" class="search-popup-status online">在线</span>
              <span v-if="item.type === 'user' && item.status !== 'online'" class="search-popup-status offline">离线</span>
              <span v-if="(item.type === 'group' || item.type === 'discussion') && item.isMember" class="search-popup-status online">已加入</span>
              <span v-if="(item.type === 'group' || item.type === 'discussion') && !item.isMember" class="search-popup-status offline">未加入</span>
            </div>
          </div>
          <button v-if="item.type === 'user'" class="search-popup-btn" @click.stop="$emit('privateChat', item)">
            <i class="fas fa-comment"></i>
          </button>
          <button v-if="(item.type === 'group' || item.type === 'discussion') && item.isMember" class="search-popup-btn" @click.stop="$emit('select', item)">
            <i class="fas fa-arrow-right"></i>
          </button>
          <button v-if="(item.type === 'group' || item.type === 'discussion') && !item.isMember" class="search-popup-btn apply-btn" @click.stop="$emit('applyJoin', item)">
            <i class="fas fa-user-plus"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { getAvatarUrl } from '../../utils/avatar'
import { useServerUrl } from '../../composables/useServerUrl'

const { serverUrl } = useServerUrl()

interface SearchResultItem {
  id: string
  name: string
  type: 'user' | 'group' | 'discussion'
  username?: string
  avatar?: string
  status?: 'online' | 'offline'
  isMember?: boolean
}

defineProps<{
  searchQuery: string
  searchResults: SearchResultItem[]
}>()

defineEmits<{
  (e: 'select', item: SearchResultItem): void
  (e: 'privateChat', item: SearchResultItem): void
  (e: 'applyJoin', item: SearchResultItem): void
}>()
</script>

<style scoped>
.search-popup {
  position: relative;
  width: 100%;
  max-height: 300px;
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  z-index: 100;
  overflow: hidden;
  margin-bottom: 8px;
  border: 1px solid var(--border-color);
}

.search-popup-content {
  display: flex;
  flex-direction: column;
}

.search-popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color);
  font-weight: 600;
  font-size: 14px;
  color: var(--text-color);
}

.search-popup-count {
  font-weight: normal;
  font-size: 12px;
  color: var(--text-secondary);
}

.search-popup-list {
  max-height: 340px;
  overflow-y: auto;
}

.search-popup-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.search-popup-item:hover {
  background-color: var(--hover-color);
}

.search-popup-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  position: relative;
}

.search-popup-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  background: var(--primary-color);
  color: white;
  font-size: 9px;
  padding: 0 3px;
  border-radius: 3px;
  line-height: 1.2;
}

.discussion-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 8px;
  padding: 1px 2px;
}

.search-popup-info {
  flex: 1;
  margin-left: 12px;
  min-width: 0;
}

.search-popup-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.search-popup-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.search-popup-username {
  font-size: 12px;
  color: var(--text-secondary);
}

.search-popup-type {
  font-size: 11px;
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 4px;
}

.search-popup-status {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 8px;
}

.search-popup-status.online {
  background-color: #dcfce7;
  color: #16a34a;
}

.search-popup-status.offline {
  background-color: #fef2f2;
  color: #dc2626;
}

.search-popup-btn {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--primary-color);
  color: white;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 0.2s ease;
}

.search-popup-btn:hover {
  background: var(--active-color);
  transform: scale(1.05);
}

.search-popup-btn.apply-btn {
  background: #10b981;
}

.search-popup-btn.apply-btn:hover {
  background: #059669;
}
</style>
