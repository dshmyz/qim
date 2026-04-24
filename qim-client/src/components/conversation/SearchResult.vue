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
            <img :src="item.avatar || generateAvatar(item.name)" :alt="item.name" />
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
          <button v-if="item.type === 'group' || item.type === 'discussion'" class="search-popup-btn" @click.stop="$emit('select', item)">
            <i class="fas fa-arrow-right"></i>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { generateAvatar } from '../../utils/avatar'

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
}>()
</script>

<style scoped>
.search-popup {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  z-index: 10;
  background: var(--sidebar-bg, #fff);
  border-bottom: 1px solid var(--border-color, #e8e8e8);
  max-height: 400px;
  overflow-y: auto;
}

.search-popup-content {
  padding: 8px 0;
}

.search-popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color, #333);
  border-bottom: 1px solid var(--border-color, #e8e8e8);
}

.search-popup-count {
  font-size: 12px;
  color: var(--text-secondary, #999);
  font-weight: normal;
}

.search-popup-list {
  padding: 4px 0;
}

.search-popup-item {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
  transition: background 0.2s;
  gap: 12px;
}

.search-popup-item:hover {
  background: var(--hover-color, rgba(0, 0, 0, 0.05));
}

.search-popup-avatar {
  position: relative;
  flex-shrink: 0;
}

.search-popup-avatar img {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  background: var(--primary-color, #1976d2);
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
  min-width: 0;
}

.search-popup-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color, #333);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.search-popup-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  font-size: 12px;
  color: var(--text-secondary, #999);
  margin-top: 2px;
}

.search-popup-username {
  opacity: 0.7;
}

.search-popup-status {
  padding: 0 4px;
  border-radius: 2px;
  font-size: 10px;
}

.search-popup-status.online {
  color: var(--color-success-500);
  background: var(--color-success-50);
}

.search-popup-status.offline {
  color: var(--text-secondary);
  background: var(--color-gray-100);
}

.search-popup-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: none;
  border-radius: 4px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary, #666);
  transition: all 0.2s;
  flex-shrink: 0;
}

.search-popup-btn:hover {
  background: var(--hover-color, rgba(0, 0, 0, 0.05));
  color: var(--primary-color, #1976d2);
}
</style>
