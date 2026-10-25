<template>
  <div v-if="searchQuery && searchResults.length > 0" class="search-popup">
    <div class="search-popup-header">
      <span>搜索结果</span>
      <span class="search-popup-count">{{ searchResults.length }} 个结果</span>
    </div>
    <div class="search-popup-list">
      <div
        v-for="item in visibleResults"
        :key="item.id + item.type"
        class="search-popup-item"
        @click="$emit('select', item)"
      >
        <div class="search-popup-avatar">
          <Avatar :src="item.avatar" :name="item.name" :server-url="serverUrl" :alt="item.name" :badge="searchBadge(item)" size="md" />
        </div>
        <div class="search-popup-info">
          <div class="search-popup-name">{{ item.name }}</div>
          <div class="search-popup-meta">
            <span v-if="isPrivateChatSearchResult(item)" class="search-popup-username">{{ item.username }}</span>
            <span v-if="item.type === 'group'" class="search-popup-type">群聊</span>
            <span v-if="item.type === 'discussion'" class="search-popup-type">讨论组</span>
            <span v-if="item.type === 'bot'" class="search-popup-type">AI 助手</span>
            <span v-if="isPrivateChatSearchResult(item) && item.type !== 'bot' && item.status === 'online'" class="search-popup-status online">在线</span>
            <span v-if="isPrivateChatSearchResult(item) && item.type !== 'bot' && item.status !== 'online'" class="search-popup-status offline">离线</span>
            <span v-if="(item.type === 'group' || item.type === 'discussion') && item.isMember" class="search-popup-status online">已加入</span>
            <span v-if="(item.type === 'group' || item.type === 'discussion') && !item.isMember" class="search-popup-status offline">未加入</span>
          </div>
        </div>
        <button v-if="isPrivateChatSearchResult(item)" class="search-popup-btn" @click.stop="$emit('privateChat', item)">
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
    <button
      v-if="hiddenCount > 0"
      class="search-popup-more"
      @click="expanded = !expanded"
    >
      {{ expanded ? '收起' : `展开全部（${searchResults.length}）` }}
      <i :class="expanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down'"></i>
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Avatar from '../shared/Avatar.vue'
import { useServerUrl } from '../../composables/useServerUrl'
import { isPrivateChatSearchResult } from '../../utils/privateChatTarget'
import { buildSearchResultBadge } from '../../utils/user'

const { serverUrl } = useServerUrl()

interface SearchResultItem {
  id: string
  name: string
  type: string
  username?: string
  avatar?: string
  status?: 'online' | 'offline'
  isMember?: boolean
}

const props = defineProps<{
  searchQuery: string
  searchResults: SearchResultItem[]
}>()

defineEmits<{
  (e: 'select', item: SearchResultItem): void
  (e: 'privateChat', item: SearchResultItem): void
  (e: 'applyJoin', item: SearchResultItem): void
}>()

// 人员/群组分组默认只展开前 N 条：命中很多时会把下方的聊天记录分组
// 推到首屏之外，用户根本不知道还有聊天记录结果。展开不影响父级滚动结构。
const COLLAPSE_LIMIT = 5
const expanded = ref(false)
watch(() => props.searchQuery, () => {
  expanded.value = false
})
const visibleResults = computed(() =>
  expanded.value ? props.searchResults : props.searchResults.slice(0, COLLAPSE_LIMIT)
)
const hiddenCount = computed(() => Math.max(0, props.searchResults.length - COLLAPSE_LIMIT))

// 搜索结果头像角标：统一由 buildSearchResultBadge 构造。
const searchBadge = (item: SearchResultItem) => buildSearchResultBadge(item)
</script>

<style scoped>
/* 搜索结果分组：只负责内容渲染，高度自适应。
   滚动与高度策略统一由父级 .search-results 负责——这里再设 max-height
   会与同级的聊天记录分组在固定高度容器里互相挤压（曾把结果行压成半行）。 */
.search-popup {
  box-sizing: border-box;
  background: var(--sidebar-bg);
  /* 水平边距与上方搜索框（padding 8px）对齐，不贴侧栏边缘 */
  margin: 0 8px 8px;
}

.search-popup-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  /* 分组标题常驻：长列表滚动时保持分组上下文（sticky 相对父级滚动容器） */
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-color);
  /* 标题分层条：比 hover-color 浅一档（条目 hover 为 50% 混合），与聊天记录分组统一。
     必须不透明：吸顶后行会从其下方滚过，半透明会让文字透出来。 */
  background: color-mix(in srgb, var(--hover-color) 70%, var(--sidebar-bg));
  font-weight: 600;
  font-size: var(--font-size-sm);
  color: var(--text-color);
}

.search-popup-count {
  font-weight: normal;
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
}

.search-popup-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.search-popup-item:hover {
  /* 比标题条的常驻 --hover-color 浅一档，避免与头部撞色 */
  background-color: color-mix(in srgb, var(--hover-color), transparent 50%);
}

/* 折叠展开：与条目同宽内缩（16px），避免误触到下方聊天记录分组 */
.search-popup-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: calc(100% - 32px);
  margin: 0 16px 8px;
  padding: 8px 0;
  border: none;
  border-radius: 4px;
  background: none;
  color: var(--primary-color);
  font-size: var(--font-size-xs);
  cursor: pointer;
}

.search-popup-more:hover {
  background-color: color-mix(in srgb, var(--hover-color), transparent 50%);
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

.search-popup-info {
  flex: 1;
  margin-left: 12px;
  min-width: 0;
}

.search-popup-name {
  font-size: var(--font-size-sm);
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
  /* 账号 + 徽标是单行 flex：必须允许收缩，否则长账号会让整行溢出、
     被私聊按钮或弹层边界裁掉（表现为「一行显示不全」） */
  min-width: 0;
}

.search-popup-username {
  font-size: var(--font-size-xxs);
  color: var(--text-secondary);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-popup-type {
  font-size: var(--font-size-xxxs);
  color: var(--text-secondary);
  background: var(--hover-color);
  padding: 1px 6px;
  border-radius: 4px;
  flex-shrink: 0;
}

.search-popup-status {
  font-size: var(--font-size-xxxs);
  padding: 1px 6px;
  border-radius: 8px;
  flex-shrink: 0;
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
